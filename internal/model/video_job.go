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
	VideoID string `json:"video_id"`

	// Total number of frames actually sent for inference.
	Total int `json:"total"`

	// FlaggedCount is how many frames were labelled nsfw or violence.
	FlaggedCount int `json:"flagged_count"`

	// FlaggedEarly is true when processing stopped early because
	// FlaggedCount reached the configured EarlyExitCount threshold.
	FlaggedEarly bool `json:"flagged_early"`

	// OverallLabel is the video-level verdict derived from frame results.
	// Priority: nsfw > violence > safe
	OverallLabel string `json:"overall_label"`

	Predictions []FrameResult `json:"predictions"`
}
