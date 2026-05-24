package services

import (
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

var frameNumberRE = regexp.MustCompile(`(\d+)`)

func buildFinalVerdict(
	videoID uuid.UUID,
	frames *dto.PredictionResult,
	audio *dto.AudioResult,
	videoDurationSec float64,
	scorer *ModerationScorer,
) *model.FinalVerdict {
	if scorer == nil {
		scorer = NewModerationScorer(config.ModerationConfig{
			FrameWeight:        0.7,
			AudioWeight:        0.3,
			SafeThreshold:      0.3,
			ViolationThreshold: 0.6,
			MaxLabelWeight:     5,
			HardNsfwConfidence: 0.98,
			HardNsfwSec:        5,
			HardViolenceFrames: 10,
			HardToxicSec:           15,
			HardToxicCoverageRatio: 0.15,
			HardToxicSegmentCount:  8,
			HardToxicTotalSec:      45,
		})
	}
	return scorer.BuildFinalVerdict(videoID, frames, audio, videoDurationSec)
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
