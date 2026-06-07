package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

type VideoProcessor interface {
	Process(ctx context.Context, job dto.VideoJob, progress *VideoProgress) (*dto.ProcessingOutput, error)
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

func (p *ffmpegVideoProcessor) Process(ctx context.Context, job dto.VideoJob, progress *VideoProgress) (*dto.ProcessingOutput, error) {
	videoID := job.VideoID
	framesDir := filepath.Join(p.outputBasePath, videoID.String(), "frames")
	audioDir := filepath.Join(p.outputBasePath, videoID.String(), "audio")

	for _, dir := range []string{framesDir, audioDir} {
		if err := utils.EnsureDir(dir); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	originalFPS, err := utils.GetVideoFPS(job.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("get video FPS: %w", err)
	}

	targetFPS := int(originalFPS / 3)
	if targetFPS < 1 {
		targetFPS = 1
	}
	log.Printf("[processor] video=%s: original FPS=%.2f, target FPS=%d", videoID, originalFPS, targetFPS)

	framesPattern := filepath.Join(framesDir, "frame_%05d.jpg")
	frameArgs := []string{
		"-i", job.VideoPath,
		"-vf", fmt.Sprintf("fps=%d,select='gt(scene,0.3)+not(mod(n,10))'", targetFPS),
		"-vsync", "vfr",
		"-q:v", "2",
		framesPattern,
	}

	// faster-whisper: mono 16 kHz PCM + light normalize (dynaudnorm) for stable ASR levels.
	audioOut := filepath.Join(audioDir, "audio.wav")
	audioArgs := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-fflags", "+genpts",
		"-i", job.VideoPath,
		"-map", "0:a:0?",
		"-vn",
		"-af", "pan=mono|c0=0.5*c0+0.5*c1,highpass=f=80,dynaudnorm=f=150:g=15,aresample=16000:resampler=swr,aformat=sample_fmts=s16:channel_layouts=mono",
		"-ar", "16000",
		"-ac", "1",
		"-c:a", "pcm_s16le",
		"-f", "wav",
		"-y",
		audioOut,
	}

	var (
		wg           sync.WaitGroup
		moderationWG sync.WaitGroup
		framesErr    error
		audioErr     error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = progress.Update(ctx, videoID, constants.StageFrameExtraction)
		log.Printf("[processor] video=%s: extracting frames", videoID)
		framesErr = utils.RunFFmpeg(frameArgs)
	}()
	go func() {
		defer wg.Done()
		_ = progress.Update(ctx, videoID, constants.StageAudioExtraction)
		log.Printf("[processor] video=%s: extracting audio", videoID)
		audioErr = utils.RunFFmpeg(audioArgs)
	}()
	wg.Wait()

	if framesErr != nil {
		return nil, fmt.Errorf("extract frames: %w", framesErr)
	}
	if audioErr != nil {
		log.Printf("[processor] video=%s: audio extraction failed: %v", videoID, audioErr)
	}

	out := &dto.ProcessingOutput{}

	if p.ai == nil {
		log.Printf("[processor] video=%s: AI moderator not configured", videoID)
		return out, nil
	}

	moderationWG.Add(1)
	go func() {
		defer moderationWG.Done()
		err := progress.Update(ctx, videoID, constants.StageFrameAnalysis)
		if err != nil {
			log.Printf("[processor] video=%s: update progress error: %v", videoID, err)
			return
		}
		frames, err := p.ai.PredictFramesDir(videoID.String(), framesDir)
		if err != nil {
			log.Printf("[processor] video=%s: frame moderation error: %v", videoID, err)
			return
		}
		out.Frames = frames
		logPredictionResult(frames)

	}()

	if audioErr == nil {
		moderationWG.Add(1)
		go func() {
			defer moderationWG.Done()
			if err := progress.Update(ctx, videoID, constants.StageAudioAnalysis); err != nil {
				log.Printf("[processor] video=%s: update progress error: %v", videoID, err)
				return
			}
			audio, err := p.ai.PredictAudioFile(videoID.String(), audioOut)
			if err != nil {
				log.Printf("[processor] video=%s: audio moderation error: %v", videoID, err)
			} else {
				out.Audio = audio
				logAudioResult(audio)
			}
		}()
	} else {
		log.Printf("[processor] video=%s: skipping audio moderation", job.VideoID)
	}
	moderationWG.Wait()

	return out, nil
}
