"""
POST /audio/predict — faster-whisper ASR + PhoBERT.
"""
import logging
import os
import tempfile

from fastapi import APIRouter, File, HTTPException, UploadFile

from app import model as phobert
from app.model import FLAGGED_LABELS
from app.schemas import AudioPredictResponse, SentencePrediction
from app.transcriber import transcribe_audio

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/audio", tags=["predict"])

_LABEL_PRIORITY = {"Toxic": 1, "Clean": 0}


def _overall_label(sentences: list[SentencePrediction]) -> str:
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
        "Upload WAV 16 kHz mono (from video-api). "
        "faster-whisper ASR + PhoBERT **Clean** / **Toxic** per segment."
    ),
)
async def predict_audio(
    file: UploadFile = File(..., description="Audio file (WAV 16 kHz mono from video-api)"),
) -> AudioPredictResponse:
    suffix = os.path.splitext(file.filename or "audio.wav")[1] or ".wav"
    try:
        with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
            tmp.write(await file.read())
            tmp_path = tmp.name
    except Exception as exc:
        logger.error("Failed to save upload: %s", exc)
        raise HTTPException(status_code=500, detail=f"Could not save audio file: {exc}")

    try:
        logger.info("[audio_predict] Transcribing: %s", file.filename)
        segments = transcribe_audio(tmp_path)

        if not segments:
            logger.warning("[audio_predict] No speech detected")
            return AudioPredictResponse(
                total_sentences=0,
                flagged_count=0,
                overall_label="Clean",
                sentences=[],
            )

        texts = [s.text for s in segments]
        logger.info("[audio_predict] PhoBERT on %d segment(s)", len(texts))
        preds = phobert.predict(texts)

        sentence_results: list[SentencePrediction] = [
            SentencePrediction(
                text=seg.text,
                start_sec=seg.start_sec,
                end_sec=seg.end_sec,
                **pred,
            )
            for seg, pred in zip(segments, preds)
        ]

        flagged = sum(1 for s in sentence_results if s.label in FLAGGED_LABELS)
        verdict = _overall_label(sentence_results)

        logger.info(
            "[audio_predict] Done — sentences=%d flagged=%d verdict=%s",
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
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
