package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func RunFFmpeg(args []string) error {
	cmdLine := "ffmpeg " + strings.Join(args, " ")
	log.Printf("[ffmpeg] running: %s", cmdLine)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[ffmpeg] command failed: %v", err)
		return err
	}
	return nil
}

type ffprobeFormat struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func ProbeDurationSec(path string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var probe ffprobeFormat
	if err := json.Unmarshal(out, &probe); err != nil {
		return 0, fmt.Errorf("ffprobe parse: %w", err)
	}

	durStr := strings.TrimSpace(probe.Format.Duration)
	if durStr == "" {
		return 0, fmt.Errorf("ffprobe: empty duration")
	}
	d, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("ffprobe: invalid duration %v", d)
	}
	return d, nil
}

// GetVideoFPS uses ffprobe to get the frame rate of a video stream.
func GetVideoFPS(path string) (float64, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=r_frame_rate",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe fps: %w", err)
	}

	fpsStr := strings.TrimSpace(string(out))
	parts := strings.Split(fpsStr, "/")
	if len(parts) != 2 {
		return strconv.ParseFloat(fpsStr, 64)
	}

	num, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe parse fps num: %w", err)
	}
	den, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe parse fps den: %w", err)
	}

	if den == 0 {
		return 0, fmt.Errorf("ffprobe: invalid fps denominator 0")
	}

	return num / den, nil
}
