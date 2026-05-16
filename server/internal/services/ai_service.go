package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
)

// AIModerator calls external image and audio moderation APIs.
type AIModerator interface {
	PredictFramesDir(videoID, framesDir string) (*dto.PredictionResult, error)
	PredictAudioFile(videoID, audioPath string) (*dto.AudioResult, error)
}

type aiModerator struct {
	cfg    config.AIServiceConfig
	client *http.Client
}

func NewAIModerator(cfg config.AIServiceConfig) AIModerator {
	return &aiModerator{
		cfg: cfg,
		client: &http.Client{
			Timeout: 3 * time.Minute,
		},
	}
}

// PredictFramesDir reads frames from framesDir, sends them in chunks to the image
// moderation service, and aggregates results. Early-exit skips remaining chunks
// once FlaggedCount reaches EarlyExitCount.
func (a *aiModerator) PredictFramesDir(videoID, framesDir string) (*dto.PredictionResult, error) {
	framePaths, err := a.listFramePaths(framesDir)
	if err != nil {
		return nil, err
	}

	totalFrames := len(framePaths)
	if totalFrames == 0 {
		log.Printf("[ai_moderator] video=%s: no frames in %s, skipping", videoID, framesDir)
		return &dto.PredictionResult{
			VideoID:      videoID,
			OverallLabel: "safe",
		}, nil
	}

	chunkSize := a.cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 32
	}
	earlyExit := a.cfg.EarlyExitCount
	totalChunks := (totalFrames + chunkSize - 1) / chunkSize

	log.Printf("[ai_moderator] video=%s: %d frames → %d chunk(s) of ≤%d (early-exit after %d flagged)",
		videoID, totalFrames, totalChunks, chunkSize, earlyExit)

	result := &dto.PredictionResult{
		VideoID:     videoID,
		Predictions: make([]dto.FrameResult, 0, totalFrames),
	}

	for chunkIdx := 0; chunkIdx < totalChunks; chunkIdx++ {
		start := chunkIdx * chunkSize
		end := min(start+chunkSize, totalFrames)
		chunk := framePaths[start:end]

		log.Printf("[ai_moderator] video=%s: chunk %d/%d (%d frames)",
			videoID, chunkIdx+1, totalChunks, len(chunk))

		preds, err := a.predictFrameChunk(chunk)
		if err != nil {
			log.Printf("[ai_moderator] video=%s: chunk %d error: %v (skipping)", videoID, chunkIdx+1, err)
			continue
		}

		for _, p := range preds {
			result.Predictions = append(result.Predictions, p)
			result.Total++
			if dto.IsFlaggedFrameLabel(p.Label) {
				result.FlaggedCount++
			}
		}

		if earlyExit > 0 && result.FlaggedCount >= earlyExit {
			log.Printf("[ai_moderator] video=%s: early exit — %d flagged (threshold %d, %d frames unchecked)",
				videoID, result.FlaggedCount, earlyExit, totalFrames-end)
			result.FlaggedEarly = true
			break
		}
	}

	result.OverallLabel = dto.OverallFrameLabel(result.Predictions)
	log.Printf("[ai_moderator] video=%s: done — checked %d/%d, flagged=%d, early=%v, verdict=%s",
		videoID, result.Total, totalFrames, result.FlaggedCount, result.FlaggedEarly, result.OverallLabel)

	return result, nil
}

func (a *aiModerator) listFramePaths(framesDir string) ([]string, error) {
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		return nil, fmt.Errorf("read frames dir %s: %w", framesDir, err)
	}

	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			paths = append(paths, filepath.Join(framesDir, e.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func (a *aiModerator) predictFrameChunk(framePaths []string) ([]dto.FrameResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, fp := range framePaths {
		f, err := os.Open(fp)
		if err != nil {
			return nil, fmt.Errorf("open frame %s: %w", fp, err)
		}

		part, err := writer.CreateFormFile("files", filepath.Base(fp))
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("create form field for %s: %w", fp, err)
		}
		if _, err = io.Copy(part, f); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("copy frame %s: %w", fp, err)
		}
		_ = f.Close()
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart: %w", err)
	}

	url := a.cfg.FrameModeratorUrl + "/images/predict/batch"
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("image moderation HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var aiResp dto.AIImagePredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return aiResp.Predictions, nil
}

func (a *aiModerator) PredictAudioFile(videoID, audioPath string) (*dto.AudioResult, error) {
	if a.cfg.AudioModeratorUrl == "" {
		return nil, fmt.Errorf("audio moderator URL is not configured")
	}

	log.Printf("[audio_moderator] video=%s: sending audio to %s", videoID, a.cfg.AudioModeratorUrl)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio %s: %w", audioPath, err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form field: %w", err)
	}
	if _, err = io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart: %w", err)
	}

	url := a.cfg.AudioModeratorUrl + "/audio/predict"
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("audio moderation HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var aiResp dto.AIAudioPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := aiResp.ToAudioResult(videoID)
	log.Printf("[audio_moderator] video=%s: done — sentences=%d flagged=%d verdict=%s",
		videoID, result.TotalSentences, result.FlaggedCount, result.OverallLabel)

	return result, nil
}
