"""
Silero VAD — speech region detection at 16 kHz.
"""
from __future__ import annotations

import logging
from dataclasses import dataclass

import torch

from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

_silero_model = None
_get_speech_timestamps = None
_read_audio = None


@dataclass(frozen=True)
class SpeechRegion:
    start_sec: float
    end_sec: float


def load_silero() -> bool:
    global _silero_model, _get_speech_timestamps, _read_audio
    if _silero_model is not None:
        return True
    try:
        model, utils = torch.hub.load(
            repo_or_dir="snakers4/silero-vad",
            model="silero_vad",
            force_reload=False,
            onnx=False,
            trust_repo=True,
        )
        (
            get_speech_timestamps,
            _,
            read_audio,
            *_,
        ) = utils
        _silero_model = model
        _get_speech_timestamps = get_speech_timestamps
        _read_audio = read_audio
        logger.info("Silero VAD loaded")
        return True
    except Exception as exc:
        logger.error("Silero VAD load failed: %s", exc)
        return False


def is_silero_loaded() -> bool:
    return _silero_model is not None


def silero_speech_regions(audio_path: str, sample_rate: int = 16000) -> list[SpeechRegion]:
    """Return speech intervals in seconds. Empty if VAD unavailable or no speech."""
    if not is_silero_loaded() and not load_silero():
        return []

    wav = _read_audio(audio_path, sampling_rate=sample_rate)
    stamps = _get_speech_timestamps(
        wav,
        _silero_model,
        sampling_rate=sample_rate,
        threshold=settings.silero_threshold,
        min_speech_duration_ms=settings.silero_min_speech_ms,
        min_silence_duration_ms=settings.silero_min_silence_ms,
        speech_pad_ms=settings.silero_speech_pad_ms,
        return_seconds=True,
    )
    regions = [
        SpeechRegion(start_sec=float(s["start"]), end_sec=float(s["end"]))
        for s in stamps
        if float(s["end"]) > float(s["start"])
    ]
    logger.info("Silero VAD: %d speech region(s) in %s", len(regions), audio_path)
    return regions


def split_long_regions(
    regions: list[SpeechRegion],
    max_sec: float,
) -> list[SpeechRegion]:
    """Split regions longer than max_sec into smaller chunks (for Whisper)."""
    if max_sec <= 0:
        return regions
    out: list[SpeechRegion] = []
    for r in regions:
        start = r.start_sec
        while start < r.end_sec:
            end = min(start + max_sec, r.end_sec)
            if end > start:
                out.append(SpeechRegion(start_sec=start, end_sec=end))
            start = end
    return out
