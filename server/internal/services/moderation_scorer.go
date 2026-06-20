package services

import (
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

const (
	labelNsfw     = "nsfw"
	labelViolence = "violence"

	weightSafe     = 0
	weightNsfw     = 4
	weightViolence = 5
)

type ModerationScorer struct {
	cfg config.ModerationConfig
}

func NewModerationScorer(cfg config.ModerationConfig) *ModerationScorer {
	return &ModerationScorer{cfg: cfg}
}

func (s *ModerationScorer) BuildFinalVerdict(
	videoID uuid.UUID,
	frames *dto.PredictionResult,
	audio *dto.AudioResult,
	videoDurationSec float64,
) *model.FinalVerdict {
	totalFrames := 0
	if frames != nil {
		totalFrames = frames.Total
	}

	duration := videoDurationSec
	if duration <= 0 && frames != nil && frames.TargetFPS > 0 {
		duration = float64(totalFrames) / float64(frames.TargetFPS)
	}

	frameScore := s.computeFrameScore(frames, duration)
	audioScore := s.computeAudioScore(audio, duration)
	finalScore := s.fuse(frameScore, audioScore)
	if audioScore > finalScore {
		finalScore = audioScore
	}

	reason := s.evaluateHardRules(frames, audio, duration)
	verdict := s.verdictFromScore(finalScore)
	if reason != "" {
		verdict = constants.VerdictViolation
		if finalScore < s.cfg.HardRuleFloorScore {
			finalScore = s.cfg.HardRuleFloorScore
		}
	}

	return &model.FinalVerdict{
		VideoID:           videoID,
		Verdict:           verdict.String(),
		RiskScore:         finalScore,
		FrameScore:        frameScore,
		AudioScore:        audioScore,
		TotalFrames:       totalFrames,
		VideoDurationSec:  duration,
		HardRuleTriggered: reason != "",
		HardRuleReason:    reason,
	}
}

func (s *ModerationScorer) computeFrameScore(frames *dto.PredictionResult, videoDurationSec float64) float64 {
	if frames == nil || len(frames.Predictions) == 0 || videoDurationSec <= 0 {
		return 0
	}

	targetFPS := float64(frames.TargetFPS)
	if targetFPS <= 0 {
		targetFPS = 10
	}
	events := BuildVisualEvents(frames.Predictions, targetFPS)
	gap := s.cfg.VisualMergeGapSec

	nsfwIntervals := MergeVisualIntervals(events, labelNsfw, gap, 0)
	violenceIntervals := MergeVisualIntervals(events, labelViolence, gap, 0)

	coverageNsfw := visualCoverageScore(nsfwIntervals, videoDurationSec)
	coverageViolence := visualCoverageScore(violenceIntervals, videoDurationSec)
	peak := visualPeakWindowScore(events, s.cfg.VisualPeakWindowSec, s.cfg.MaxLabelWeight)

	return clamp01(maxFloat(coverageNsfw, maxFloat(coverageViolence, peak)))
}

func (s *ModerationScorer) computeAudioScore(audio *dto.AudioResult, durationSec float64) float64 {
	if audio == nil || len(audio.Sentences) == 0 || durationSec <= 0 {
		return 0
	}
	toxicDur, avgConf, _ := toxicAudioStats(audio)
	if toxicDur <= 0 {
		return 0
	}
	coverageScore := (toxicDur / durationSec) * avgConf
	peakScore := audioPeakWindowScore(audio, durationSec, s.cfg.AudioPeakWindowSec)
	return clamp01(maxFloat(coverageScore, peakScore))
}

func audioPeakWindowScore(audio *dto.AudioResult, videoDurationSec, windowSec float64) float64 {
	if audio == nil || windowSec <= 0 || videoDurationSec <= 0 {
		return 0
	}
	toxic := make([]dto.AudioSentence, 0)
	for _, sent := range audio.Sentences {
		if dto.IsFlaggedAudioLabel(sent.Label) {
			toxic = append(toxic, sent)
		}
	}
	if len(toxic) == 0 {
		return 0
	}

	var best float64
	for i, startSent := range toxic {
		windowEnd := startSent.StartSec + windowSec
		var dur, confSum float64
		for j := i; j < len(toxic); j++ {
			sent := toxic[j]
			if sent.StartSec > windowEnd {
				break
			}
			start := sent.StartSec
			if start < startSent.StartSec {
				start = startSent.StartSec
			}
			end := sent.EndSec
			if end <= start {
				end = start + 0.5
			}
			if end > windowEnd {
				end = windowEnd
			}
			segDur := end - start
			if segDur <= 0 {
				continue
			}
			dur += segDur
			confSum += sent.Confidence * segDur
		}
		if dur <= 0 {
			continue
		}
		score := (dur / windowSec) * (confSum / dur)
		if score > best {
			best = score
		}
	}
	return best
}

func (s *ModerationScorer) fuse(frameScore, audioScore float64) float64 {
	fw := s.cfg.FrameWeight
	aw := s.cfg.AudioWeight
	if fw+aw <= 0 {
		return frameScore
	}
	return (fw*frameScore + aw*audioScore) / (fw + aw)
}

func (s *ModerationScorer) verdictFromScore(score float64) constants.ModerationVerdict {
	if score < s.cfg.SafeThreshold {
		return constants.VerdictSafe
	}
	if score < s.cfg.ViolationThreshold {
		return constants.VerdictWarning
	}
	return constants.VerdictViolation
}

func (s *ModerationScorer) evaluateHardRules(frames *dto.PredictionResult, audio *dto.AudioResult, videoDurationSec float64) string {
	if reason := s.hardRuleNsfwSustained(frames); reason != "" {
		return reason
	}
	if reason := s.hardRuleViolenceSustained(frames); reason != "" {
		return reason
	}
	if reason := s.hardRuleViolenceBurst(frames); reason != "" {
		return reason
	}
	if reason := s.hardRuleToxicSustained(audio); reason != "" {
		return reason
	}
	if reason := s.hardRuleToxicAggregate(audio, videoDurationSec); reason != "" {
		return reason
	}
	return ""
}

func (s *ModerationScorer) hardRuleToxicAggregate(audio *dto.AudioResult, videoDurationSec float64) string {
	if audio == nil {
		return ""
	}
	toxicDur, avgConf, _ := toxicAudioStats(audio)
	flaggedSentences := countFlaggedAudioSentences(audio)
	if flaggedSentences == 0 {
		return ""
	}
	if flaggedSentences >= s.cfg.HardToxicSegmentCount && avgConf >= 0.85 {
		return "toxic_many_segments"
	}
	if toxicDur >= s.cfg.HardToxicTotalSec {
		return "toxic_total_duration"
	}
	if videoDurationSec > 0 && toxicDur/videoDurationSec >= s.cfg.HardToxicCoverageRatio {
		return "toxic_coverage_ratio"
	}
	return ""
}

func (s *ModerationScorer) hardRuleNsfwSustained(frames *dto.PredictionResult) string {
	events := s.visualEvents(frames)
	intervals := MergeVisualIntervals(events, labelNsfw, s.cfg.VisualMergeGapSec, s.cfg.HardNsfwConfidence)
	for _, iv := range intervals {
		if iv.EndSec-iv.StartSec >= s.cfg.HardNsfwSec {
			return "nsfw_sustained"
		}
	}
	return ""
}

func (s *ModerationScorer) hardRuleViolenceSustained(frames *dto.PredictionResult) string {
	events := s.visualEvents(frames)
	minConf := s.cfg.HardViolenceConfidence
	if minConf <= 0 {
		minConf = 0.85
	}
	intervals := MergeVisualIntervals(events, labelViolence, s.cfg.VisualMergeGapSec, minConf)
	for _, iv := range intervals {
		if iv.EndSec-iv.StartSec >= s.cfg.HardViolenceSec {
			return "violence_sustained"
		}
	}
	return ""
}

func (s *ModerationScorer) hardRuleViolenceBurst(frames *dto.PredictionResult) string {
	events := s.visualEvents(frames)
	minConf := s.cfg.HardViolenceBurstConf
	if minConf <= 0 {
		minConf = 0.80
	}

	window := s.cfg.VisualPeakWindowSec
	if window <= 0 {
		window = 3.0
	}
	required := s.cfg.HardViolenceBurstCount
	if required <= 0 {
		required = 3
	}

	for i, startEv := range events {
		if startEv.Label != labelViolence || startEv.Confidence < minConf {
			continue
		}
		windowEnd := startEv.TimestampSec + window
		count := 0
		var confSum float64
		for j := i; j < len(events); j++ {
			e := events[j]
			if e.TimestampSec > windowEnd {
				break
			}
			if e.Label == labelViolence && e.Confidence >= minConf {
				count++
				confSum += e.Confidence
			}
		}
		if count >= required && confSum/float64(count) >= minConf {
			return "violence_burst"
		}
	}
	return ""
}

func (s *ModerationScorer) hardRuleToxicSustained(audio *dto.AudioResult) string {
	if audio == nil {
		return ""
	}
	spans := mergeToxicSpans(audio, audioMergeGapSec)
	for _, sp := range spans {
		if sp.end-sp.start >= s.cfg.HardToxicSec {
			return "toxic_sustained"
		}
	}
	return ""
}

func (s *ModerationScorer) visualEvents(frames *dto.PredictionResult) []VisualEvent {
	if frames == nil {
		return nil
	}
	targetFPS := float64(frames.TargetFPS)
	if targetFPS <= 0 {
		targetFPS = 10
	}
	return BuildVisualEvents(frames.Predictions, targetFPS)
}

func toxicAudioStats(audio *dto.AudioResult) (mergedDuration, avgConfidence float64, flaggedCount int) {
	if audio == nil {
		return 0, 0, 0
	}
	spans := mergeToxicSpans(audio, 2.0)
	var totalDur, confSum float64
	for _, sp := range spans {
		d := sp.end - sp.start
		if d <= 0 {
			d = 0.5
		}
		totalDur += d
		confSum += sp.peakConf * d
	}
	flaggedCount = len(spans)
	if totalDur <= 0 {
		return 0, 0, flaggedCount
	}
	return totalDur, confSum / totalDur, flaggedCount
}

type toxicSpan struct {
	start, end float64
	peakConf   float64
}

func countFlaggedAudioSentences(audio *dto.AudioResult) int {
	if audio == nil {
		return 0
	}
	n := 0
	for _, sent := range audio.Sentences {
		if dto.IsFlaggedAudioLabel(sent.Label) {
			n++
		}
	}
	return n
}

func mergeToxicSpans(audio *dto.AudioResult, gapSec float64) []toxicSpan {
	var spans []toxicSpan
	for _, sent := range audio.Sentences {
		if !dto.IsFlaggedAudioLabel(sent.Label) {
			continue
		}
		start, end := sent.StartSec, sent.EndSec
		if end <= start {
			end = start + 0.5
		}
		if len(spans) == 0 {
			spans = append(spans, toxicSpan{start, end, sent.Confidence})
			continue
		}
		last := &spans[len(spans)-1]
		if start <= last.end+gapSec {
			if end > last.end {
				last.end = end
			}
			if sent.Confidence > last.peakConf {
				last.peakConf = sent.Confidence
			}
			continue
		}
		spans = append(spans, toxicSpan{start, end, sent.Confidence})
	}
	return spans
}

func frameLabelWeight(label string) int {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case labelNsfw:
		return weightNsfw
	case labelViolence:
		return weightViolence
	default:
		return weightSafe
	}
}

func normalizeScore(raw, maxWeight float64) float64 {
	return clamp01(raw / maxWeight)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func buildFinalVerdict(
	videoID uuid.UUID,
	frames *dto.PredictionResult,
	audio *dto.AudioResult,
	videoDurationSec float64,
	scorer *ModerationScorer,
) *model.FinalVerdict {
	if scorer == nil {
		scorer = NewModerationScorer(config.ModerationConfig{
			FrameWeight:              0.7,
			AudioWeight:              0.3,
			SafeThreshold:            0.25,
			ViolationThreshold:       0.55,
			MaxLabelWeight:           5,
			HardRuleFloorScore:       0.85,
			HardNsfwConfidence:       0.90,
			HardNsfwSec:              5,
			HardViolenceSec:          2.0,
			HardViolenceConfidence:   0.85,
			HardViolenceBurstCount:   3,
			HardViolenceBurstConf:    0.80,
			VisualMergeGapSec:        0.5,
			VisualPeakWindowSec:      3.0,
			AudioPeakWindowSec:       10.0,
			HardToxicSec:             15,
			HardToxicCoverageRatio:   0.15,
			HardToxicSegmentCount:    8,
			HardToxicTotalSec:        30,
		})
	}
	return scorer.BuildFinalVerdict(videoID, frames, audio, videoDurationSec)
}

func score(scores map[string]float64, key string) float64 {
	if scores == nil {
		return 0
	}
	return scores[key]
}
