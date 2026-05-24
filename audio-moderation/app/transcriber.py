"""
Faster-Whisper transcription helper.

Loads the Whisper model once (singleton) and exposes `transcribe_audio(path)`
which returns one TranscriptSegment per Whisper segment (text + start/end times).
"""
import logging
from dataclasses import dataclass

import torch
from faster_whisper import WhisperModel

from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

_whisper: WhisperModel | None = None


@dataclass
class TranscriptSegment:
    text: str
    start_sec: float
    end_sec: float


def load_whisper() -> WhisperModel:
    """Load Whisper model at startup (called once from lifespan)."""
    global _whisper

    device = "cuda" if torch.cuda.is_available() else "cpu"
    compute_type = "float16" if device == "cuda" else "int8"

    logger.info(
        "Loading Faster-Whisper '%s' on %s (%s)…",
        settings.whisper_model_size,
        device,
        compute_type,
    )
    _whisper = WhisperModel(
        settings.whisper_model_size,
        device=device,
        compute_type=compute_type,
    )
    logger.info("Whisper model loaded successfully")
    return _whisper


def get_whisper() -> WhisperModel:
    if _whisper is None:
        raise RuntimeError("Whisper model not loaded. Did lifespan run?")
    return _whisper


def transcribe_audio(audio_path: str) -> list[TranscriptSegment]:
    """
    Transcribe *audio_path* and return one segment per Whisper utterance.

    Each segment includes start/end times (seconds) for timeline reporting.
    """
    model = get_whisper()
    logger.info("Transcribing: %s", audio_path)

    segments, info = model.transcribe(audio_path, beam_size=5, language="vi")

    sentences: list[TranscriptSegment] = []
    for seg in segments:
        text = seg.text.strip()
        if text:
            sentences.append(
                TranscriptSegment(
                    text=text,
                    start_sec=float(seg.start),
                    end_sec=float(seg.end),
                )
            )

    logger.info(
        "Transcription done — language=%s (prob=%.2f), segments=%d",
        info.language,
        info.language_probability,
        len(sentences),
    )
    return sentences
