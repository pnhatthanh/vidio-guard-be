package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type ffprobeFormat struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// ProbeDurationSec returns media duration in seconds via ffprobe, or 0 on failure.
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
