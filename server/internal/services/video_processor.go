package services

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

// VideoProcessor runs ffmpeg extraction and AI moderation for a video job.
type VideoProcessor interface {
	Process(job model.VideoJob) error
}

type ffmpegVideoProcessor struct {
	outputBasePath string
	ai             AIModerator
}

func NewFFmpegVideoProcessor(outputBasePath string, ai AIModerator) VideoProcessor {
	return &ffmpegVideoProcessor{
		outputBasePath: outputBasePath,
		ai:             ai,
	}
}

func (p *ffmpegVideoProcessor) Process(job model.VideoJob) error {
	framesDir := filepath.Join(p.outputBasePath, job.VideoID, "frames")
	audioDir := filepath.Join(p.outputBasePath, job.VideoID, "audio")
	audioChunksDir := filepath.Join(p.outputBasePath, job.VideoID, "audio_chunks")

	for _, dir := range []string{framesDir, audioDir, audioChunksDir} {
		if err := utils.EnsureDir(dir); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
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
		log.Printf("[processor] extracting frames to %s", framesPattern)
		framesErr = utils.RunFFmpeg(frameArgs)
	}()
	go func() {
		defer wg.Done()
		log.Printf("[processor] extracting audio to %s", audioOut)
		audioErr = utils.RunFFmpeg(audioArgs)
	}()
	wg.Wait()

	if framesErr != nil {
		return fmt.Errorf("extract frames: %w", framesErr)
	}
	if audioErr != nil {
		log.Printf("[processor] video=%s: audio extraction failed: %v", job.VideoID, audioErr)
	} else {
		log.Printf("[processor] video=%s: audio extracted", job.VideoID)
	}

	log.Printf("[processor] video=%s: ffmpeg done", job.VideoID)

	if p.ai == nil {
		log.Printf("[processor] video=%s: AI moderator not configured, skipping", job.VideoID)
		return nil
	}

	var moderationWG sync.WaitGroup

	moderationWG.Add(1)
	go func() {
		defer moderationWG.Done()
		result, err := p.ai.PredictFramesDir(job.VideoID, framesDir)
		if err != nil {
			log.Printf("[processor] video=%s: image moderation error: %v", job.VideoID, err)
			return
		}
		logPredictionResult(result)
	}()

	if audioErr == nil {
		moderationWG.Add(1)
		go func() {
			defer moderationWG.Done()
			audioResult, err := p.ai.PredictAudioFile(job.VideoID, audioOut)
			if err != nil {
				log.Printf("[processor] video=%s: audio moderation error: %v", job.VideoID, err)
				return
			}
			logAudioResult(audioResult)
		}()
	} else {
		log.Printf("[processor] video=%s: skipping audio moderation", job.VideoID)
	}

	moderationWG.Wait()

	return nil
}
