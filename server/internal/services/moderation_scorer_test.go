package services

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
)

func testScorer() *ModerationScorer {
	return NewModerationScorer(config.ModerationConfig{
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

func TestModerationScorer_AllSafe(t *testing.T) {
	scorer := testScorer()
	frames := &dto.PredictionResult{
		Total: 100,
		Predictions: []dto.FrameResult{
			{Frame: "frame_00001.jpg", Label: "safe", Confidence: 0.99},
		},
	}
	for i := 2; i <= 100; i++ {
		frames.Predictions = append(frames.Predictions, dto.FrameResult{
			Frame: "frame_00001.jpg", Label: "safe", Confidence: 0.99,
		})
	}

	v := scorer.BuildFinalVerdict(uuid.New(), frames, nil, 100)
	if v.Verdict != constants.VerdictSafe.String() {
		t.Fatalf("expected safe, got %s (final=%.4f)", v.Verdict, v.FinalScore)
	}
}

func TestModerationScorer_SingleViolenceFrameWarningNotViolation(t *testing.T) {
	scorer := testScorer()
	preds := make([]dto.FrameResult, 100)
	for i := range preds {
		preds[i] = dto.FrameResult{Frame: "frame_00001.jpg", Label: "safe", Confidence: 0.95}
	}
	preds[50] = dto.FrameResult{Frame: "frame_00051.jpg", Label: "violence", Confidence: 0.95}

	v := scorer.BuildFinalVerdict(uuid.New(), &dto.PredictionResult{Predictions: preds}, nil, 100)
	if v.Verdict == constants.VerdictViolation.String() {
		t.Fatalf("single violence frame should not auto-violation, got %s final=%.4f", v.Verdict, v.FinalScore)
	}
}

func TestModerationScorer_HardRuleViolenceConsecutive(t *testing.T) {
	scorer := testScorer()
	preds := make([]dto.FrameResult, 15)
	for i := range preds {
		preds[i] = dto.FrameResult{
			Frame:      frameName(i + 1),
			Label:      "violence",
			Confidence: 0.9,
		}
	}

	v := scorer.BuildFinalVerdict(uuid.New(), &dto.PredictionResult{Predictions: preds}, nil, 15)
	if !v.HardRuleTriggered || v.Verdict != constants.VerdictViolation.String() {
		t.Fatalf("expected hard rule violation, got verdict=%s hard=%v reason=%q",
			v.Verdict, v.HardRuleTriggered, v.HardRuleReason)
	}
}

func TestModerationScorer_ManyToxicSegmentsNotSafe(t *testing.T) {
	scorer := testScorer()
	sentences := make([]dto.AudioSentence, 27)
	for i := range sentences {
		start := float64(i * 9)
		sentences[i] = dto.AudioSentence{
			Label:      "Toxic",
			Confidence: 0.97,
			StartSec:   start,
			EndSec:     start + 7,
			Text:       "toxic phrase",
		}
	}
	v := scorer.BuildFinalVerdict(uuid.New(), nil, &dto.AudioResult{Sentences: sentences}, 251)
	if v.Verdict == constants.VerdictSafe.String() {
		t.Fatalf("expected warning/violation for pervasive toxic audio, got safe final=%.4f audio=%.4f reason=%q",
			v.FinalScore, v.AudioScore, v.HardRuleReason)
	}
}

func TestModerationScorer_AudioToxicWeightedByDuration(t *testing.T) {
	scorer := testScorer()
	short := scorer.BuildFinalVerdict(uuid.New(), nil, &dto.AudioResult{
		Sentences: []dto.AudioSentence{
			{Label: "Toxic", Confidence: 0.9, StartSec: 1, EndSec: 2, Text: "x"},
		},
	}, 120)
	long := scorer.BuildFinalVerdict(uuid.New(), nil, &dto.AudioResult{
		Sentences: []dto.AudioSentence{
			{Label: "Toxic", Confidence: 0.9, StartSec: 0, EndSec: 60, Text: "y"},
		},
	}, 120)
	if long.AudioScore <= short.AudioScore {
		t.Fatalf("longer toxic segment should score higher: short=%.4f long=%.4f", short.AudioScore, long.AudioScore)
	}
}

func frameName(n int) string {
	return fmt.Sprintf("frame_%05d.jpg", n)
}
