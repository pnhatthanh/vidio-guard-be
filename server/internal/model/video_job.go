package model

type VideoJob struct {
	VideoID   string
	VideoPath string
}


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
