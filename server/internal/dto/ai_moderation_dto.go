package dto

type FrameResult struct {
	Frame      string             `json:"frame"`
	Label      string             `json:"label"`
	Confidence float64            `json:"confidence"`
	Scores     map[string]float64 `json:"scores"`
}

type PredictionResult struct {
	VideoID      string        `json:"video_id"`
	Total        int           `json:"total"`
	FlaggedCount int           `json:"flagged_count"`
	FlaggedEarly bool          `json:"flagged_early"`
	OverallLabel string        `json:"overall_label"`
	Predictions  []FrameResult `json:"predictions"`
}

type AudioSentence struct {
	Text       string             `json:"text"`
	Label      string             `json:"label"`
	LabelID    int                `json:"label_id"`
	Confidence float64            `json:"confidence"`
	Scores     map[string]float64 `json:"scores"`
}

type AudioResult struct {
	VideoID        string          `json:"video_id"`
	TotalSentences int             `json:"total_sentences"`
	FlaggedCount   int             `json:"flagged_count"`
	OverallLabel   string          `json:"overall_label"`
	Sentences      []AudioSentence `json:"sentences"`
}

func IsFlaggedFrameLabel(label string) bool {
	return label == "nsfw" || label == "violence"
}

func IsFlaggedAudioLabel(label string) bool {
	return label == "Offensive" || label == "Hate"
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
