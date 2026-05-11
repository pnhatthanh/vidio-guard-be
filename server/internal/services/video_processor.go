package services

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

type VideoProcessor interface {
	Process(job model.VideoJob) error
}

type FFmpegVideoProcessor struct {
	OutputBasePath string
	AIModerator    *AIModerator
}

func NewFFmpegVideoProcessor(outputBasePath string, ai *AIModerator) *FFmpegVideoProcessor {
	return &FFmpegVideoProcessor{
		OutputBasePath: outputBasePath,
		AIModerator:    ai,
	}
}

func (p *FFmpegVideoProcessor) Process(job model.VideoJob) error {

	framesDir := filepath.Join(p.OutputBasePath, job.VideoID, "frames")
	audioDir := filepath.Join(p.OutputBasePath, job.VideoID, "audio")
	audioChunksDir := filepath.Join(p.OutputBasePath, job.VideoID, "audio_chunks")

	if err := utils.EnsureDir(framesDir); err != nil {
		return err
	}
	if err := utils.EnsureDir(audioDir); err != nil {
		return err
	}
	if err := utils.EnsureDir(audioChunksDir); err != nil {
		return err
	}

	framesPattern := filepath.Join(framesDir, "frame_%05d.jpg")
	frameArgs := []string{
		"-i", job.VideoPath,
		"-vf", "fps=1,select='gt(scene,0.3)+not(mod(n,10))'",
		"-vsync", "vfr",
		"-q:v", "2",
		framesPattern,
	}

	audioOut := filepath.Join(audioDir, "audio.wav")
	audioArgs := []string{
		"-i", job.VideoPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-acodec", "pcm_s16le",
		audioOut,
	}

	var (
		wg        sync.WaitGroup
		framesErr error
		audioErr  error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Printf("[processor] Extracting frames to %s", framesPattern)
		framesErr = utils.RunFFmpeg(frameArgs)
	}()
	go func() {
		defer wg.Done()
		log.Printf("[processor] Extracting audio to %s", audioOut)
		audioErr = utils.RunFFmpeg(audioArgs)
	}()

	wg.Wait()

	if framesErr != nil {
		return fmt.Errorf("extracting frames: %w", framesErr)
	}
	if audioErr != nil {
		log.Printf("[processor] Warning: audio extraction failed: %v", audioErr)
	} else {
		log.Printf("[processor] Audio extracted successfully")
	}

	log.Printf("[processor] Done processing video: %s", job.VideoID)

	if p.AIModerator == nil {
		log.Printf("[processor] video=%s: AIModerator not configured, skipping predict step", job.VideoID)
		return nil
	}

	result, err := p.AIModerator.PredictFramesDir(job.VideoID, framesDir)
	if err != nil {
		log.Printf("[processor] video=%s: image AI moderation error: %v", job.VideoID, err)
	} else {
		logPredictionResult(result)
	}
	
	if audioErr == nil {
		audioResult, err := p.AIModerator.PredictAudioFile(job.VideoID, audioOut)
		if err != nil {
			log.Printf("[processor] video=%s: audio AI moderation error: %v", job.VideoID, err)
		} else {
			logAudioResult(audioResult)
		}
	} else {
		log.Printf("[processor] video=%s: skipping audio moderation (audio extraction failed)", job.VideoID)
	}

	return nil
}

// logPredictionResult prints all frame predictions in a readable table format.
func logPredictionResult(r *model.PredictionResult) {
	earlyStr := ""
	if r.FlaggedEarly {
		earlyStr = " (EARLY EXIT)"
	}
	log.Printf("[ai_result] ============================================================")
	log.Printf("[ai_result] Video          : %s", r.VideoID)
	log.Printf("[ai_result] Frames checked : %d  |  Flagged: %d%s", r.Total, r.FlaggedCount, earlyStr)
	log.Printf("[ai_result] VERDICT         : %s", r.OverallLabel)
	log.Printf("[ai_result] ------------------------------------------------------------")
	log.Printf("[ai_result] %-20s  %-10s  %-6s  %s", "Frame", "Label", "Conf", "Scores")
	log.Printf("[ai_result] ------------------------------------------------------------")
	for _, p := range r.Predictions {
		flagMark := "  "
		if isFlagged(p.Label) {
			flagMark = "! "
		}
		log.Printf("[ai_result] %s%-18s  %-10s  %.4f  nsfw=%.3f safe=%.3f violence=%.3f",
			flagMark,
			p.Frame,
			p.Label,
			p.Confidence,
			p.Scores["nsfw"],
			p.Scores["safe"],
			p.Scores["violence"],
		)
	}
	log.Printf("[ai_result] ============================================================")
}

// logAudioResult prints per-sentence PhoBERT results in a readable table,
func logAudioResult(r *model.AudioResult) {
	log.Printf("[audio_result] ============================================================")
	log.Printf("[audio_result] Video          : %s", r.VideoID)
	log.Printf("[audio_result] Sentences      : %d  |  Flagged: %d", r.TotalSentences, r.FlaggedCount)
	log.Printf("[audio_result] VERDICT         : %s", r.OverallLabel)
	log.Printf("[audio_result] ------------------------------------------------------------")
	log.Printf("[audio_result] %-10s  %-10s  %-6s  %s", "Label", "Conf", "Flag", "Text")
	log.Printf("[audio_result] ------------------------------------------------------------")
	for _, s := range r.Sentences {
		flagMark := "  "
		if s.Label == "Offensive" || s.Label == "Hate" {
			flagMark = "! "
		}
		// Truncate long sentences for readability
		text := s.Text
		if len(text) > 80 {
			text = text[:77] + "..."
		}
		log.Printf("[audio_result] %s%-10s  %.4f    clean=%.3f offensive=%.3f hate=%.3f",
			flagMark,
			s.Label,
			s.Confidence,
			s.Scores["Clean"],
			s.Scores["Offensive"],
			s.Scores["Hate"],
		)
		log.Printf("[audio_result]    ↳ %s", text)
	}
	log.Printf("[audio_result] ============================================================")
}
