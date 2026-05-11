"""
API router — POST /audio/predict

Accepts a WAV/MP3 audio file upload, runs:
  1. Faster-Whisper  → list of Vietnamese sentence segments
  2. PhoBERT         → per-sentence label (Clean / Offensive / Hate)

Returns an AudioPredictResponse with per-sentence details + overall verdict.
"""
import logging
import os
import tempfile

from fastapi import APIRouter, File, UploadFile, HTTPException

from app.transcriber import transcribe_audio
from app import model as phobert
from app.schemas import AudioPredictResponse, SentencePrediction

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/audio", tags=["predict"])

# Labels that count as flagged content
_FLAGGED_LABELS = {"Offensive", "Hate"}

# Worst-case priority for overall_label
_LABEL_PRIORITY = {"Hate": 2, "Offensive": 1, "Clean": 0}


def _overall_label(sentences: list[SentencePrediction]) -> str:
    """Return the worst label seen across all sentences."""
    worst = "Clean"
    for s in sentences:
        if _LABEL_PRIORITY.get(s.label, 0) > _LABEL_PRIORITY.get(worst, 0):
            worst = s.label
    return worst


@router.post(
    "/predict",
    response_model=AudioPredictResponse,
    summary="Transcribe audio and classify each sentence",
    description=(
        "Upload a WAV/MP3 audio file. "
        "Faster-Whisper transcribes it into segments; "
        "each segment is classified by PhoBERT as **Clean**, **Offensive**, or **Hate**."
    ),
)
async def predict_audio(
    file: UploadFile = File(..., description="Audio file (WAV / MP3, 16 kHz mono recommended)"),
) -> AudioPredictResponse:

    # ── 1. Save upload to a temp file ─────────────────────────────────────────
    suffix = os.path.splitext(file.filename or "audio.wav")[1] or ".wav"
    try:
        with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
            tmp.write(await file.read())
            tmp_path = tmp.name
    except Exception as exc:
        logger.error("Failed to save upload: %s", exc)
        raise HTTPException(status_code=500, detail=f"Could not save audio file: {exc}")

    try:
        # ── 2. Transcribe ──────────────────────────────────────────────────────
        logger.info("[audio_predict] Transcribing uploaded file: %s", file.filename)
        sentences_text = transcribe_audio(tmp_path)

        if not sentences_text:
            logger.warning("[audio_predict] No speech detected in audio file")
            return AudioPredictResponse(
                total_sentences=0,
                flagged_count=0,
                overall_label="Clean",
                sentences=[],
            )

        # ── 3. PhoBERT inference ───────────────────────────────────────────────
        logger.info("[audio_predict] Running PhoBERT on %d sentence(s)", len(sentences_text))
        preds = phobert.predict(sentences_text)

        # ── 4. Assemble response ───────────────────────────────────────────────
        sentence_results: list[SentencePrediction] = [
            SentencePrediction(text=txt, **pred)
            for txt, pred in zip(sentences_text, preds)
        ]

        flagged = sum(1 for s in sentence_results if s.label in _FLAGGED_LABELS)
        verdict = _overall_label(sentence_results)

        logger.info(
            "[audio_predict] Done — sentences=%d  flagged=%d  verdict=%s",
            len(sentence_results),
            flagged,
            verdict,
        )

        return AudioPredictResponse(
            total_sentences=len(sentence_results),
            flagged_count=flagged,
            overall_label=verdict,
            sentences=sentence_results,
        )

    finally:
        # Always clean up the temp file
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
