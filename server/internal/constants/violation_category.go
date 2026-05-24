package constants

type ViolationCategory string

const (
	CategoryNudity    ViolationCategory = "nudity"
	CategoryViolence  ViolationCategory = "violence"
	CategoryHateSpeech ViolationCategory = "hate_speech"
)