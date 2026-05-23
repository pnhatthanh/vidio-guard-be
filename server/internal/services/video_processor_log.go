package services

import (
	"log"

	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
)

func logPredictionResult(r *dto.PredictionResult) {
	if r == nil {
		return
	}

	earlyStr := ""
	if r.FlaggedEarly {
		earlyStr = " (EARLY EXIT)"
	}
	log.Printf("[ai_result] ============================================================")
	log.Printf("[ai_result] Video          : %s", r.VideoID)
	log.Printf("[ai_result] Frames checked : %d  |  Flagged: %d%s", r.Total, r.FlaggedCount, earlyStr)
	log.Printf("[ai_result] ------------------------------------------------------------")
	log.Printf("[ai_result] %-20s  %-10s  %-6s  %s", "Frame", "Label", "Conf", "Scores")
	log.Printf("[ai_result] ------------------------------------------------------------")
	for _, p := range r.Predictions {
		flagMark := "  "
		if dto.IsFlaggedFrameLabel(p.Label) {
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

func logAudioResult(r *dto.AudioResult) {
	if r == nil {
		return
	}

	log.Printf("[audio_result] ============================================================")
	log.Printf("[audio_result] Video          : %s", r.VideoID)
	log.Printf("[audio_result] Sentences      : %d  |  Flagged: %d", r.TotalSentences, r.FlaggedCount)
	log.Printf("[audio_result] VERDICT         : %s", r.OverallLabel)
	log.Printf("[audio_result] ------------------------------------------------------------")
	log.Printf("[audio_result] %-10s  %-10s  %-6s  %s", "Label", "Conf", "Flag", "Text")
	log.Printf("[audio_result] ------------------------------------------------------------")
	for _, s := range r.Sentences {
		flagMark := "  "
		if dto.IsFlaggedAudioLabel(s.Label) {
			flagMark = "! "
		}
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
