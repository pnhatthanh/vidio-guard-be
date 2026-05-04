package constants

type VideoStatus string

const (
	StatusUploaded   VideoStatus = "uploaded"
	StatusProcessing VideoStatus = "processing"
	StatusDone       VideoStatus = "done"
	StatusFailed     VideoStatus = "failed"
)
