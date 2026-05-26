package services

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

const (
	visualFrameDurationSec = 1.0
	visualMergeGapSec      = 1.0
	audioMergeGapSec       = 0.5 
	maxEvidenceLen         = 200
)

type interval struct {
	category  constants.ViolationCategory
	startSec  float64
	endSec    float64
	peakScore float64
	evidence  string
}

func buildViolationSegments(videoID uuid.UUID, frames *dto.PredictionResult, audio *dto.AudioResult) []model.ViolationSegment {
	var out []model.ViolationSegment
	for _, seg := range buildVisualViolationSegments(frames) {
		out = append(out, model.ViolationSegment{
			VideoID:   videoID,
			Source:    constants.VisualSource,
			Category:  seg.category,
			StartSec:  seg.startSec,
			EndSec:    seg.endSec,
			PeakScore: seg.peakScore,
			Evidence:  seg.evidence,
		})
	}
	for _, seg := range buildAudioViolationSegments(audio) {
		out = append(out, model.ViolationSegment{
			VideoID:   videoID,
			Source:    constants.AudioSource,
			Category:  seg.category,
			StartSec:  seg.startSec,
			EndSec:    seg.endSec,
			PeakScore: seg.peakScore,
			Evidence:  seg.evidence,
		})
	}
	return out
}

func buildVisualViolationSegments(frames *dto.PredictionResult) []interval {
	if frames == nil || len(frames.Predictions) == 0 {
		return nil
	}

	points := make([]interval, 0)
	for _, p := range frames.Predictions {
		if !dto.IsFlaggedFrameLabel(p.Label) {
			continue
		}
		num := parseFrameNumber(p.Frame)
		ts := float64(num - 1)
		if ts < 0 {
			ts = 0
		}
		peak := score(p.Scores, p.Label)
		if peak == 0 {
			peak = p.Confidence
		}
		points = append(points, interval{
			category:  frameLabelToViolationCategory(p.Label),
			startSec:  ts,
			endSec:    ts + visualFrameDurationSec,
			peakScore: peak,
			evidence:  p.Frame,
		})
	}

	return mergeIntervals(points, visualMergeGapSec)
}

func buildAudioViolationSegments(audio *dto.AudioResult) []interval {
	if audio == nil {
		return nil
	}

	out := make([]interval, 0)
	for _, s := range audio.Sentences {
		if !dto.IsFlaggedAudioLabel(s.Label) {
			continue
		}
		start := s.StartSec
		end := s.EndSec
		if end <= start {
			end = start + 0.5
		}
		peak := score(s.Scores, string(constants.CategoryHateSpeech))
		if peak == 0 {
			peak = s.Confidence
		}
		out = append(out, interval{
			category:  constants.CategoryHateSpeech,
			startSec:  start,
			endSec:    end,
			peakScore: peak,
			evidence:  truncateEvidence(s.Text),
		})
	}
	return mergeIntervals(out, audioMergeGapSec)
}

func mergeIntervals(points []interval, gapSec float64) []interval {
	if len(points) == 0 {
		return nil
	}

	sorted := make([]interval, len(points))
	copy(sorted, points)
	sortIntervals(sorted)

	merged := []interval{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		cur := sorted[i]
		last := &merged[len(merged)-1]
		if cur.startSec <= last.endSec+gapSec {
			if cur.endSec > last.endSec {
				last.endSec = cur.endSec
			}
			if categoryPriority(cur.category) > categoryPriority(last.category) {
				last.category = cur.category
			}
			if cur.peakScore > last.peakScore {
				last.peakScore = cur.peakScore
				last.evidence = cur.evidence
			}
			continue
		}
		merged = append(merged, cur)
	}
	return merged
}

func sortIntervals(points []interval) {
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j].startSec < points[i].startSec {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
}

func frameLabelToViolationCategory(label string) constants.ViolationCategory {
	switch label {
	case "nsfw":
		return constants.CategoryNudity
	case "violence":
		return constants.CategoryViolence
	default:
		return constants.ViolationCategory(label)
	}
}

func categoryPriority(category constants.ViolationCategory) int {
	switch category {
	case constants.CategoryNudity:
		return 2
	case constants.CategoryViolence:
		return 1
	default:
		return 0
	}
}

func truncateEvidence(text string) string {
	t := strings.TrimSpace(text)
	if !utf8.ValidString(t) {
		t = strings.ToValidUTF8(t, "")
	}
	runes := []rune(t)
	if len(runes) <= maxEvidenceLen {
		return t
	}
	return string(runes[:maxEvidenceLen-3]) + "..."
}

func mapViolationSegmentsToDTO(rows []model.ViolationSegment) []dto.ViolationSegmentSummary {
	if len(rows) == 0 {
		return []dto.ViolationSegmentSummary{}
	}
	out := make([]dto.ViolationSegmentSummary, len(rows))
	for i, r := range rows {
		out[i] = dto.ViolationSegmentSummary{
			Source:    string(r.Source),
			Category:  string(r.Category),
			StartSec:  r.StartSec,
			EndSec:    r.EndSec,
			PeakScore: r.PeakScore,
			Evidence:  r.Evidence,
		}
	}
	return out
}
