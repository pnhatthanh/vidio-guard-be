"""
Pre-ASR pipeline: optional Demucs → DeepFilterNet → Silero VAD regions.
"""
from __future__ import annotations

import logging
import os
import tempfile
from dataclasses import dataclass, field

import numpy as np
import soundfile as sf

from app.config import get_settings
from app.preprocess import demucs_sep, denoise, vad
from app.preprocess.vad import SpeechRegion

logger = logging.getLogger(__name__)
settings = get_settings()

SAMPLE_RATE = 16000


@dataclass
class PreprocessResult:
    """Paths and speech regions for chunked transcription."""

    audio_path: str
    regions: list[SpeechRegion] = field(default_factory=list)
    temp_files: list[str] = field(default_factory=list)


def load_preprocess_models() -> None:
    """Warm up optional preprocess models (called from app lifespan)."""
    if not settings.preprocess_enabled:
        return
    if settings.preprocess_denoise:
        denoise.load_deepfilter()
    if settings.preprocess_silero_vad:
        vad.load_silero()


def preprocess_for_asr(wav_path: str) -> PreprocessResult:
    """
    Run configured preprocess steps. Returns working audio path + speech regions.
    Caller must call cleanup_preprocess() when done.
    """
    if not settings.preprocess_enabled:
        return PreprocessResult(audio_path=wav_path, regions=[])

    working = wav_path
    temps: list[str] = []

    if settings.preprocess_demucs:
        vocals = demucs_sep.separate_vocals(working)
        if vocals != working:
            temps.append(vocals)
            working = vocals

    if settings.preprocess_denoise:
        enhanced = denoise.deepfilter_enhance(working)
        if enhanced != working:
            temps.append(enhanced)
            working = enhanced

    regions: list[SpeechRegion] = []
    if settings.preprocess_silero_vad:
        regions = vad.silero_speech_regions(working, sample_rate=SAMPLE_RATE)
        regions = vad.split_long_regions(regions, settings.chunk_max_sec)

    return PreprocessResult(audio_path=working, regions=regions, temp_files=temps)


def extract_region_wav(
    audio_path: str,
    region: SpeechRegion,
    *,
    prefix: str = "vg_chunk_",
) -> str:
    """Write a single speech region to a temp WAV file."""
    data, sr = sf.read(audio_path, dtype="float32", always_2d=False)
    if data.ndim > 1:
        data = data.mean(axis=1)
    if sr != SAMPLE_RATE:
        duration = len(data) / sr
        new_len = int(duration * SAMPLE_RATE)
        data = np.interp(
            np.linspace(0, len(data) - 1, new_len),
            np.arange(len(data)),
            data,
        ).astype(np.float32)
        sr = SAMPLE_RATE

    i0 = max(0, int(region.start_sec * sr))
    i1 = min(len(data), int(region.end_sec * sr))
    chunk = data[i0:i1]
    if len(chunk) == 0:
        chunk = np.zeros(1, dtype=np.float32)

    fd, path = tempfile.mkstemp(suffix=".wav", prefix=prefix)
    os.close(fd)
    sf.write(path, chunk, sr, subtype="PCM_16")
    return path


def cleanup_preprocess(result: PreprocessResult) -> None:
    for p in result.temp_files:
        try:
            os.unlink(p)
        except OSError:
            pass
