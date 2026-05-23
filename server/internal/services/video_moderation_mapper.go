package services

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/datatypes"
)

var frameNumberRE = regexp.MustCompile(`(\d+)`)

func mapFrameResults(videoID uuid.UUID, framesDir string, pred *dto.PredictionResult) []model.FrameResult {
	if pred == nil || len(pred.Predictions) == 0 {
		return nil
	}

	out := make([]model.FrameResult, 0, len(pred.Predictions))
	for _, p := range pred.Predictions {
		num := parseFrameNumber(p.Frame)
		ts := float64(num - 1)
		if ts < 0 {
			ts = 0
		}
		out = append(out, model.FrameResult{
			VideoID:          videoID,
			FrameNumber:      num,
			FrameURL:         filepath.ToSlash(filepath.Join(framesDir, p.Frame)),
			TimestampSeconds: ts,
			ViolenceScore:    score(p.Scores, "violence"),
			NsfwScore:        score(p.Scores, "nsfw"),
			SafeScore:        score(p.Scores, "safe"),
			PredictedLabel:   p.Label,
		})
	}
	return out
}

func mapAudioResult(videoID uuid.UUID, audio *dto.AudioResult) *model.AudioResult {
	if audio == nil {
		return nil
	}

	var parts []string
	var peakOffensive, peakHate, peakClean float64
	for _, s := range audio.Sentences {
		if t := strings.TrimSpace(s.Text); t != "" {
			parts = append(parts, t)
		}
		peakOffensive = max(peakOffensive, score(s.Scores, "Offensive"))
		peakHate = max(peakHate, score(s.Scores, "Hate"))
		peakClean = max(peakClean, score(s.Scores, "Clean"))
	}

	return &model.AudioResult{
		VideoID:        videoID,
		Transcript:     strings.Join(parts, " "),
		OffensiveScore: peakOffensive,
		HateScore:      peakHate,
		CleanScore:     peakClean,
		PredictedLabel: audio.OverallLabel,
	}
}

func buildFinalVerdict(videoID uuid.UUID, frames *dto.PredictionResult, audio *dto.AudioResult) *model.FinalVerdict {
	peakViolence, peakNsfw := 0.0, 0.0
	var flaggedTS []float64

	if frames != nil {
		for _, p := range frames.Predictions {
			v := score(p.Scores, "violence")
			n := score(p.Scores, "nsfw")
			if v > peakViolence {
				peakViolence = v
			}
			if n > peakNsfw {
				peakNsfw = n
			}
			if dto.IsFlaggedFrameLabel(p.Label) {
				num := parseFrameNumber(p.Frame)
				ts := float64(num - 1)
				if ts < 0 {
					ts = 0
				}
				flaggedTS = append(flaggedTS, ts)
			}
		}
	}

	frameVerdict := "safe"
	if frames != nil {
		frameVerdict = dto.OverallFrameLabel(frames.Predictions)
	}

	audioVerdict := "Clean"
	if audio != nil && audio.OverallLabel != "" {
		audioVerdict = audio.OverallLabel
	}

	verdict := aggregateVerdict(frameVerdict, audioVerdict)
	risk := max(peakNsfw, peakViolence)
	if audio != nil {
		risk = max(risk, max(scoreFromSentences(audio, "Offensive"), scoreFromSentences(audio, "Hate")))
	}

	tsJSON, _ := json.Marshal(flaggedTS)

	return &model.FinalVerdict{
		VideoID:            videoID,
		Verdict:            verdict,
		RiskScore:          risk,
		PeakViolenceScore:  peakViolence,
		PeakNsfwScore:      peakNsfw,
		FlaggedFramesCount: len(flaggedTS),
		FlaggedTimestamps:  datatypes.JSON(tsJSON),
	}
}

func aggregateVerdict(frameLabel, audioLabel string) string {
	if frameLabel == "nsfw" || audioLabel == "Hate" {
		return "nsfw"
	}
	if frameLabel == "violence" || audioLabel == "Offensive" {
		return "violence"
	}
	if audioLabel != "Clean" && audioLabel != "" {
		return strings.ToLower(audioLabel)
	}
	return frameLabel
}

func scoreFromSentences(audio *dto.AudioResult, key string) float64 {
	var peak float64
	for _, s := range audio.Sentences {
		peak = max(peak, score(s.Scores, key))
	}
	return peak
}

func parseFrameNumber(name string) int {
	m := frameNumberRE.FindStringSubmatch(name)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func score(scores map[string]float64, key string) float64 {
	if scores == nil {
		return 0
	}
	return scores[key]
}
