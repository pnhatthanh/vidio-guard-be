package constants

type VideoStage string

const (
	StageStarting         VideoStage = "starting"
	StageFrameExtraction  VideoStage = "frame_extraction"
	StageAudioExtraction  VideoStage = "audio_extraction"
	StageFrameAnalysis    VideoStage = "frame_analysis"
	StageAudioAnalysis    VideoStage = "audio_analysis"
	StageAggregation      VideoStage = "aggregation"
	StageCompleted        VideoStage = "completed"
	StageFailed           VideoStage = "failed"
)

var StageProgress = map[VideoStage]int{
	StageStarting:        0,
	StageFrameExtraction: 15,
	StageAudioExtraction: 35,
	StageFrameAnalysis:   50,
	StageAudioAnalysis:   65,
	StageAggregation:     90,
	StageCompleted:       100,
}
