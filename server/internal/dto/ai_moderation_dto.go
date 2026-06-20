package dto

type FrameResult struct {
	Frame         string             `json:"frame"`
	Label         string             `json:"label"`
	Confidence    float64            `json:"confidence"`
	Scores        map[string]float64 `json:"scores"`
	TimestampSec  float64            `json:"timestamp_sec,omitempty"`
}

type AudioSentence struct {
	Text       string             `json:"text"`
	Label      string             `json:"label"`
	LabelID    int                `json:"label_id"`
	Confidence float64            `json:"confidence"`
	Scores     map[string]float64 `json:"scores"`
	StartSec   float64            `json:"start_sec"`
	EndSec     float64            `json:"end_sec"`
}

type AudioResult struct {
	VideoID        string          `json:"video_id"`
	TotalSentences int             `json:"total_sentences"`
	FlaggedCount   int             `json:"flagged_count"`
	OverallLabel   string          `json:"overall_label"`
	Sentences      []AudioSentence `json:"sentences"`
}
type PredictionResult struct {
	VideoID      string        `json:"video_id"`
	Total        int           `json:"total"`
	FlaggedCount int           `json:"flagged_count"`
	OverallLabel string        `json:"overall_label"`
	TargetFPS    int           `json:"target_fps,omitempty"`
	Predictions  []FrameResult `json:"predictions"`
}

type AIImagePredictResponse struct {
	Total       int           `json:"total"`
	Predictions []FrameResult `json:"predictions"`
}

type AIAudioPredictResponse struct {
	TotalSentences int             `json:"total_sentences"`
	FlaggedCount   int             `json:"flagged_count"`
	OverallLabel   string          `json:"overall_label"`
	Sentences      []AudioSentence `json:"sentences"`
}

func (r *AIAudioPredictResponse) ToAudioResult(videoID string) *AudioResult {
	if r == nil {
		return nil
	}
	return &AudioResult{
		VideoID:        videoID,
		TotalSentences: r.TotalSentences,
		FlaggedCount:   r.FlaggedCount,
		OverallLabel:   r.OverallLabel,
		Sentences:      r.Sentences,
	}
}



func IsFlaggedFrameLabel(label string) bool {
	return label == "nsfw" || label == "violence"
}

func IsFlaggedAudioLabel(label string) bool {
	return label == "Toxic"
}

func OverallFrameLabel(predictions []FrameResult) string {
	hasViolence := false
	for _, p := range predictions {
		if p.Label == "nsfw" {
			return "nsfw"
		}
		if p.Label == "violence" {
			hasViolence = true
		}
	}
	if hasViolence {
		return "violence"
	}
	return "safe"
}
