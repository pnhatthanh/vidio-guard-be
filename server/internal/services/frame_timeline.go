package services

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
)

var frameNumberRE = regexp.MustCompile(`(\d+)`)

func parseFrameNumber(name string) int {
	m := frameNumberRE.FindStringSubmatch(name)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

const audioMergeGapSec = 0.5

type VisualEvent struct {
	TimestampSec float64
	SpanSec      float64
	Label        string
	Confidence   float64
	Frame        string
}

type VisualInterval struct {
	Label    string
	StartSec float64
	EndSec   float64
	AvgConf  float64
	PeakConf float64
	Evidence string
}

func BuildVisualEvents(predictions []dto.FrameResult, targetFPS float64) []VisualEvent {
	if len(predictions) == 0 {
		return nil
	}

	defaultSpan := 1.0 / targetFPS
	if defaultSpan <= 0 {
		defaultSpan = 0.125
	}
	maxSpan := 2.0 / targetFPS
	if maxSpan <= 0 {
		maxSpan = defaultSpan * 2
	}

	type keyed struct {
		ts   float64
		ev   VisualEvent
	}
	items := make([]keyed, 0, len(predictions))
	for _, p := range predictions {
		ts := p.TimestampSec
		if ts < 0 {
			num := parseFrameNumber(p.Frame)
			ts = float64(num-1) / targetFPS
			if ts < 0 {
				ts = 0
			}
		}
		items = append(items, keyed{
			ts: ts,
			ev: VisualEvent{
				TimestampSec: ts,
				Label:        strings.ToLower(strings.TrimSpace(p.Label)),
				Confidence:   p.Confidence,
				Frame:        filepath.Base(p.Frame),
			},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ts < items[j].ts })

	out := make([]VisualEvent, len(items))
	for i, it := range items {
		span := defaultSpan
		if i+1 < len(items) {
			gap := items[i+1].ts - it.ts
			if gap > 0 && gap < maxSpan {
				span = gap
			}
		}
		it.ev.SpanSec = span
		out[i] = it.ev
	}
	return out
}

func MergeVisualIntervals(events []VisualEvent, label string, gapSec, minConf float64) []VisualInterval {
	label = strings.ToLower(strings.TrimSpace(label))
	var filtered []VisualEvent
	for _, e := range events {
		if e.Label != label || e.Confidence < minConf {
			continue
		}
		filtered = append(filtered, e)
	}
	if len(filtered) == 0 {
		return nil
	}

	var spans []VisualInterval
	for _, e := range filtered {
		end := e.TimestampSec + e.SpanSec
		if len(spans) == 0 {
			spans = append(spans, VisualInterval{
				Label:    label,
				StartSec: e.TimestampSec,
				EndSec:   end,
				AvgConf:  e.Confidence,
				PeakConf: e.Confidence,
				Evidence: e.Frame,
			})
			continue
		}
		last := &spans[len(spans)-1]
		if e.TimestampSec <= last.EndSec+gapSec {
			if end > last.EndSec {
				last.EndSec = end
			}
			dur := last.EndSec - last.StartSec
			if dur > 0 {
				last.AvgConf = (last.AvgConf*(dur-e.SpanSec) + e.Confidence*e.SpanSec) / dur
			}
			if e.Confidence > last.PeakConf {
				last.PeakConf = e.Confidence
				last.Evidence = e.Frame
			}
			continue
		}
		spans = append(spans, VisualInterval{
			Label:    label,
			StartSec: e.TimestampSec,
			EndSec:   end,
			AvgConf:  e.Confidence,
			PeakConf: e.Confidence,
			Evidence: e.Frame,
		})
	}
	return spans
}

func visualCoverageScore(intervals []VisualInterval, videoDurationSec float64) float64 {
	if len(intervals) == 0 || videoDurationSec <= 0 {
		return 0
	}
	var totalDur, confSum float64
	for _, iv := range intervals {
		d := iv.EndSec - iv.StartSec
		if d <= 0 {
			continue
		}
		totalDur += d
		confSum += iv.AvgConf * d
	}
	if totalDur <= 0 {
		return 0
	}
	return clamp01((totalDur / videoDurationSec) * (confSum / totalDur))
}

func visualPeakWindowScore(events []VisualEvent, windowSec, maxWeight float64) float64 {
	if len(events) == 0 || windowSec <= 0 {
		return 0
	}
	flagged := make([]VisualEvent, 0)
	for _, e := range events {
		if frameLabelWeight(e.Label) == 0 {
			continue
		}
		flagged = append(flagged, e)
	}
	if len(flagged) == 0 {
		return 0
	}

	var best float64
	for i, startEv := range flagged {
		windowEnd := startEv.TimestampSec + windowSec
		var weightedSum, covered float64
		for j := i; j < len(flagged); j++ {
			e := flagged[j]
			if e.TimestampSec > windowEnd {
				break
			}
			weightedSum += float64(frameLabelWeight(e.Label)) * e.Confidence * e.SpanSec
			end := e.TimestampSec + e.SpanSec
			if end > windowEnd {
				end = windowEnd
			}
			segStart := e.TimestampSec
			if segStart < startEv.TimestampSec {
				segStart = startEv.TimestampSec
			}
			if end > segStart {
				covered += end - segStart
			}
		}
		if covered <= 0 {
			continue
		}
		raw := weightedSum / windowSec
		if s := normalizeScore(raw, maxWeight); s > best {
			best = s
		}
	}
	return best
}

func frameLabelToViolationCategory(label string) constants.ViolationCategory {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case labelNsfw:
		return constants.CategoryNudity
	case labelViolence:
		return constants.CategoryViolence
	default:
		return constants.ViolationCategory(label)
	}
}

func enrichFrameTimestamps(result *dto.PredictionResult, manifest *dto.FrameManifest) {
	if result == nil || manifest == nil {
		return
	}
	targetFPS := float64(manifest.TargetFPS)
	if targetFPS <= 0 {
		targetFPS = 10
	}
	result.TargetFPS = manifest.TargetFPS

	for i := range result.Predictions {
		p := &result.Predictions[i]
		if ts, ok := manifest.TimestampForFile(p.Frame); ok {
			p.TimestampSec = ts
			continue
		}
		num := parseFrameNumber(p.Frame)
		p.TimestampSec = float64(num-1) / targetFPS
		if p.TimestampSec < 0 {
			p.TimestampSec = 0
		}
	}
}
