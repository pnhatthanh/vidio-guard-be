package services

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

const (
	labelSafe     = "safe"
	labelNsfw     = "nsfw"
	labelViolence = "violence"
	labelClean    = "Clean"
	labelToxic    = "Toxic"

	weightSafe     = 0
	weightToxic    = 2
	weightNsfw     = 4
	weightViolence = 5
)

type framePoint struct {
	num        int
	label      string
	confidence float64
	timestamp  float64
}

// ModerationScorer computes fused risk scores and final verdict.
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
		totalFrames = len(frames.Predictions)
		if frames.Total > 0 {
			totalFrames = frames.Total
		}
	}

	duration := videoDurationSec
	if duration <= 0 {
		duration = inferDuration(frames, audio, totalFrames)
	}

	frameScore := s.computeFrameScore(frames)
	audioScore := s.computeAudioScore(audio, duration)
	finalScore := s.fuse(frameScore, audioScore)
	// Audio-heavy content: don't let long video duration dilute fusion below audio risk.
	if audioScore > finalScore {
		finalScore = audioScore
	}

	reason := s.evaluateHardRules(frames, audio, duration)
	verdict := s.verdictFromScore(finalScore)
	if reason != "" {
		verdict = constants.VerdictViolation
	}

	return &model.FinalVerdict{
		VideoID:           videoID,
		Verdict:           verdict.String(),
		Transcript:        joinAudioTranscript(audio),
		RiskScore:         finalScore,
		FrameScore:        frameScore,
		AudioScore:        audioScore,
		FinalScore:        finalScore,
		TotalFrames:       totalFrames,
		VideoDurationSec:  duration,
		HardRuleTriggered: reason != "",
		HardRuleReason:    reason,
	}
}

func (s *ModerationScorer) computeFrameScore(frames *dto.PredictionResult) float64 {
	if frames == nil || len(frames.Predictions) == 0 {
		return 0
	}
	var sum float64
	for _, p := range frames.Predictions {
		sum += float64(frameLabelWeight(p.Label)) * p.Confidence
	}
	raw := sum / float64(len(frames.Predictions))
	return normalizeScore(raw, s.cfg.MaxLabelWeight)
}

func (s *ModerationScorer) computeAudioScore(audio *dto.AudioResult, durationSec float64) float64 {
	if audio == nil || len(audio.Sentences) == 0 || durationSec <= 0 {
		return 0
	}

	toxicDur, avgConf, _ := toxicAudioStats(audio)
	if toxicDur <= 0 {
		return 0
	}

	// Coverage × confidence: reflects how much of the video is toxic (not diluted by silence).
	coverageScore := (toxicDur / durationSec) * avgConf

	// Legacy density across full timeline (kept for short clips).
	var weightedSum float64
	for _, sent := range audio.Sentences {
		dur := sent.EndSec - sent.StartSec
		if dur <= 0 {
			dur = 0.5
		}
		weightedSum += float64(audioLabelWeight(sent.Label)) * sent.Confidence * dur
	}
	densityScore := normalizeScore(weightedSum/durationSec, s.cfg.MaxLabelWeight)

	return clamp01(maxFloat(coverageScore, densityScore))
}

func (s *ModerationScorer) fuse(frameScore, audioScore float64) float64 {
	fw := s.cfg.FrameWeight
	aw := s.cfg.AudioWeight
	if fw+aw <= 0 {
		return frameScore
	}
	total := fw + aw
	return (fw*frameScore + aw*audioScore) / total
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
	if reason := s.hardRuleViolenceConsecutive(frames); reason != "" {
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
	points := sortedFramePoints(frames)
	if len(points) == 0 {
		return ""
	}
	threshold := s.cfg.HardNsfwConfidence
	required := s.cfg.HardNsfwSec

	var runStart, runEnd float64
	var inRun bool

	for _, p := range points {
		if p.label == labelNsfw && p.confidence >= threshold {
			tsEnd := p.timestamp + visualFrameDurationSec
			if !inRun {
				inRun = true
				runStart = p.timestamp
				runEnd = tsEnd
				continue
			}
			if p.timestamp <= runEnd+visualMergeGapSec {
				if tsEnd > runEnd {
					runEnd = tsEnd
				}
			} else {
				if runEnd-runStart >= required {
					return "nsfw_sustained_5s"
				}
				runStart = p.timestamp
				runEnd = tsEnd
			}
			continue
		}
		if inRun {
			if runEnd-runStart >= required {
				return "nsfw_sustained_5s"
			}
			inRun = false
		}
	}
	if inRun && runEnd-runStart >= required {
		return "nsfw_sustained_5s"
	}
	return ""
}

func (s *ModerationScorer) hardRuleViolenceConsecutive(frames *dto.PredictionResult) string {
	points := sortedFramePoints(frames)
	if len(points) == 0 {
		return ""
	}
	required := s.cfg.HardViolenceFrames
	streak := 0
	for _, p := range points {
		if p.label == labelViolence {
			streak++
			if streak >= required {
				return "violence_consecutive_frames"
			}
			continue
		}
		streak = 0
	}
	return ""
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

func (s *ModerationScorer) hardRuleToxicSustained(audio *dto.AudioResult) string {
	if audio == nil {
		return ""
	}
	required := s.cfg.HardToxicSec
	spans := mergeToxicSpans(audio, audioMergeGapSec)
	for _, sp := range spans {
		if sp.end-sp.start >= required {
			return "toxic_sustained_15s"
		}
	}
	return ""
}

func sortedFramePoints(frames *dto.PredictionResult) []framePoint {
	if frames == nil {
		return nil
	}
	out := make([]framePoint, 0, len(frames.Predictions))
	for _, p := range frames.Predictions {
		num := parseFrameNumber(p.Frame)
		ts := float64(num - 1)
		if ts < 0 {
			ts = 0
		}
		out = append(out, framePoint{
			num:        num,
			label:      strings.ToLower(strings.TrimSpace(p.Label)),
			confidence: p.Confidence,
			timestamp:  ts,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].num != out[j].num {
			return out[i].num < out[j].num
		}
		return out[i].timestamp < out[j].timestamp
	})
	return out
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

func audioLabelWeight(label string) int {
	switch strings.TrimSpace(label) {
	case labelToxic:
		return weightToxic
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

func joinAudioTranscript(audio *dto.AudioResult) string {
	if audio == nil {
		return ""
	}
	var parts []string
	for _, s := range audio.Sentences {
		if t := strings.TrimSpace(s.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

func inferDuration(frames *dto.PredictionResult, audio *dto.AudioResult, totalFrames int) float64 {
	var maxEnd float64
	if audio != nil {
		for _, s := range audio.Sentences {
			if s.EndSec > maxEnd {
				maxEnd = s.EndSec
			}
		}
	}
	if maxEnd > 0 {
		return maxEnd
	}
	if totalFrames > 0 {
		return float64(totalFrames) * visualFrameDurationSec
	}
	if frames != nil && len(frames.Predictions) > 0 {
		return float64(len(frames.Predictions)) * visualFrameDurationSec
	}
	return 1
}
