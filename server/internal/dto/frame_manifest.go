package dto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FrameManifestFilename = "manifest.json"

type FrameManifestEntry struct {
	File         string  `json:"file"`
	TimestampSec float64 `json:"timestamp_sec"`
}

type FrameManifest struct {
	VideoDurationSec float64              `json:"video_duration_sec"`
	TargetFPS        int                  `json:"target_fps"`
	Frames           []FrameManifestEntry `json:"frames"`
}

func (m *FrameManifest) TimestampForFile(name string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	base := filepath.Base(name)
	for _, e := range m.Frames {
		if e.File == name || e.File == base {
			return e.TimestampSec, true
		}
	}
	return 0, false
}

func LoadFrameManifest(path string) (*FrameManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read frame manifest: %w", err)
	}
	var m FrameManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse frame manifest: %w", err)
	}
	return &m, nil
}

func SaveFrameManifest(path string, m *FrameManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal frame manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write frame manifest: %w", err)
	}
	return nil
}
