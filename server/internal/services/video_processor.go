package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
	out := &dto.ProcessingOutput{}

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

	videoDurationSec := 0.0
	if dur, err := utils.ProbeDurationSec(job.VideoPath); err == nil {
		videoDurationSec = dur
	}

	vfFilter := fmt.Sprintf("fps=%d,select='gt(scene,0.3)+not(mod(n,10))',showinfo", targetFPS)
	framesPattern := filepath.Join(framesDir, "frame_%05d.jpg")
	frameArgs := []string{
		"-i", job.VideoPath,
		"-vf", vfFilter,
		"-vsync", "vfr",
		"-q:v", "2",
		framesPattern,
	}

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
		frameStderr  string
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = progress.Update(ctx, videoID, constants.StageFrameExtraction)
		log.Printf("[processor] video=%s: extracting frames", videoID)
		frameStderr, framesErr = utils.RunFFmpegCaptureStderr(frameArgs)
		manifest, err := buildFrameManifest(framesDir, frameStderr, videoDurationSec, targetFPS)
		if err != nil {
			log.Printf("[processor] video=%s: frame manifest warning: %v", videoID, err)
			manifest = &dto.FrameManifest{VideoDurationSec: videoDurationSec, TargetFPS: targetFPS}
		}
		manifestPath := filepath.Join(framesDir, dto.FrameManifestFilename)
		if err := dto.SaveFrameManifest(manifestPath, manifest); err != nil {
			log.Printf("[processor] video=%s: save manifest warning: %v", videoID, err)
		}
		out.FrameManifest = manifest
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
		enrichFrameTimestamps(frames, out.FrameManifest)
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

func buildFrameManifest(framesDir, ffmpegStderr string, videoDurationSec float64, targetFPS int) (*dto.FrameManifest, error) {
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		return nil, err
	}

	var frameFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			frameFiles = append(frameFiles, e.Name())
		}
	}
	sort.Strings(frameFiles)

	ptsTimes := utils.ParseShowinfoPTSTimes(ffmpegStderr)
	manifest := &dto.FrameManifest{
		VideoDurationSec: videoDurationSec,
		TargetFPS:        targetFPS,
		Frames:           make([]dto.FrameManifestEntry, 0, len(frameFiles)),
	}

	for i, name := range frameFiles {
		ts := float64(i) / float64(targetFPS)
		if i < len(ptsTimes) {
			ts = ptsTimes[i]
		}
		manifest.Frames = append(manifest.Frames, dto.FrameManifestEntry{
			File:         name,
			TimestampSec: ts,
		})
	}
	return manifest, nil
}
