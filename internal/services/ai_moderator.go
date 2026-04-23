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
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

// ─────────────────────────────────────────────────────────────────────────────
// AI service response shapes (mirrors ai-service/app/schemas.py)
// ─────────────────────────────────────────────────────────────────────────────

type aiFramePrediction struct {
	Frame      string             `json:"frame"`
	Label      string             `json:"label"`
	Confidence float64            `json:"confidence"`
	Scores     map[string]float64 `json:"scores"`
}

type aiBatchPredictResponse struct {
	Total       int                 `json:"total"`
	Predictions []aiFramePrediction `json:"predictions"`
}

// isFlagged reports whether a label counts as harmful content.
func isFlagged(label string) bool {
	return label == "nsfw" || label == "violence"
}

// overallLabel derives the video-level verdict from individual frame labels.
// Priority: nsfw > violence > safe
func overallLabel(predictions []model.FrameResult) string {
	hasViolence := false
	for _, p := range predictions {
		if p.Label == "nsfw" {
			return "nsfw"
		}
		if p.Label == "violence" {
			hasViolence = true
		}
	}
	if hasViolence {
		return "violence"
	}
	return "safe"
}

// ─────────────────────────────────────────────────────────────────────────────
// AIModerator — calls /predict/batch on the Python FastAPI AI service
// ─────────────────────────────────────────────────────────────────────────────

type AIModerator struct {
	cfg    config.AIServiceConfig
	client *http.Client
}

func NewAIModerator(cfg config.AIServiceConfig) *AIModerator {
	return &AIModerator{
		cfg: cfg,
		client: &http.Client{
			// Timeout per chunk request: generous enough for CPU inference on
			// up to 32 frames, but bounded to avoid hanging forever.
			Timeout: 3 * time.Minute,
		},
	}
}

// PredictFramesDir reads all JPEG frames from framesDir, splits them into
// chunks of cfg.ChunkSize, and sends each chunk as a separate HTTP request to
// the AI service.
//
// Early-exit: if cfg.EarlyExitCount > 0 and the accumulated flagged-frame
// count reaches that threshold, remaining chunks are skipped immediately.
// This keeps latency bounded even for long videos.
func (a *AIModerator) PredictFramesDir(videoID, framesDir string) (*model.PredictionResult, error) {
	// ── 1. Collect & sort frame file paths ───────────────────────────────
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		return nil, fmt.Errorf("read frames dir %s: %w", framesDir, err)
	}

	var framePaths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			framePaths = append(framePaths, filepath.Join(framesDir, e.Name()))
		}
	}
	sort.Strings(framePaths) // deterministic order: frame_0001, frame_0002, …

	totalFrames := len(framePaths)
	if totalFrames == 0 {
		log.Printf("[ai_moderator] video=%s: no frames found in %s, skipping", videoID, framesDir)
		return &model.PredictionResult{
			VideoID:      videoID,
			OverallLabel: "safe",
		}, nil
	}

	// ── 2. Determine chunk size ───────────────────────────────────────────
	chunkSize := a.cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 32 // safe fallback
	}
	earlyExit := a.cfg.EarlyExitCount // 0 = disabled

	totalChunks := (totalFrames + chunkSize - 1) / chunkSize
	log.Printf("[ai_moderator] video=%s: %d frames → %d chunk(s) of ≤%d (early-exit after %d flagged)",
		videoID, totalFrames, totalChunks, chunkSize, earlyExit)

	// ── 3. Process chunks ─────────────────────────────────────────────────
	result := &model.PredictionResult{
		VideoID:     videoID,
		Predictions: make([]model.FrameResult, 0, totalFrames),
	}

	for chunkIdx := 0; chunkIdx < totalChunks; chunkIdx++ {
		start := chunkIdx * chunkSize
		end := start + chunkSize
		if end > totalFrames {
			end = totalFrames
		}
		chunk := framePaths[start:end]

		log.Printf("[ai_moderator] video=%s: sending chunk %d/%d (%d frames)",
			videoID, chunkIdx+1, totalChunks, len(chunk))

		chunkPreds, err := a.sendChunk(chunk)
		if err != nil {
			// Non-fatal: log and continue with next chunk
			log.Printf("[ai_moderator] video=%s: chunk %d error: %v (skipping chunk)",
				videoID, chunkIdx+1, err)
			continue
		}

		// Accumulate results
		for _, p := range chunkPreds {
			fr := model.FrameResult{
				Frame:      p.Frame,
				Label:      p.Label,
				Confidence: p.Confidence,
				Scores:     p.Scores,
			}
			result.Predictions = append(result.Predictions, fr)
			result.Total++
			if isFlagged(p.Label) {
				result.FlaggedCount++
			}
		}

		// ── Early exit check ─────────────────────────────────────────────
		if earlyExit > 0 && result.FlaggedCount >= earlyExit {
			remaining := totalFrames - end
			log.Printf("[ai_moderator] video=%s: early exit triggered — %d flagged frames reached threshold %d (%d frames unchecked)",
				videoID, result.FlaggedCount, earlyExit, remaining)
			result.FlaggedEarly = true
			break
		}
	}

	// ── 4. Compute overall video-level verdict ────────────────────────────
	result.OverallLabel = overallLabel(result.Predictions)

	log.Printf("[ai_moderator] video=%s: done — checked %d/%d frames, flagged=%d, early=%v, verdict=%s",
		videoID, result.Total, totalFrames, result.FlaggedCount, result.FlaggedEarly, result.OverallLabel)

	return result, nil
}

// sendChunk POSTs a single chunk of frame files to the AI service /predict/batch
// and returns the raw predictions.
func (a *AIModerator) sendChunk(framePaths []string) ([]aiFramePrediction, error) {
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
			return nil, fmt.Errorf("copy frame data %s: %w", fp, err)
		}
		_ = f.Close()
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart: %w", err)
	}

	url := a.cfg.URL + "/predict/batch"
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
		return nil, fmt.Errorf("AI service HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var aiResp aiBatchPredictResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return aiResp.Predictions, nil
}
