package dto

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
