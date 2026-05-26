"""
faster-whisper — segment gốc (seg.start / seg.end).
"""
import logging
from dataclasses import dataclass

import torch
from faster_whisper import WhisperModel

from app.config import get_settings
from app.segment_filter import is_spam_or_hallucination

logger = logging.getLogger(__name__)
settings = get_settings()

_whisper: WhisperModel | None = None
_whisper_model_name: str = ""


@dataclass
class TranscriptSegment:
    text: str
    start_sec: float
    end_sec: float


def _segments_from_whisper(raw_segments) -> list[TranscriptSegment]:
    out: list[TranscriptSegment] = []
    dropped_spam = 0

    for seg in raw_segments:
        text = (seg.text or "").strip()
        if not text:
            continue
        if is_spam_or_hallucination(text):
            dropped_spam += 1
            logger.info("Dropped spam segment: %s", text[:80])
            continue

        start = float(seg.start)
        end = float(seg.end)
        if end <= start:
            end = start + 0.1
        out.append(TranscriptSegment(text=text, start_sec=start, end_sec=end))

    if dropped_spam:
        logger.info("Filtered %d spam/outro segment(s)", dropped_spam)
    return out


def _resolve_compute_type(device: str) -> str:
    raw = (settings.whisper_compute_type or "auto").strip().lower()
    if raw != "auto":
        return raw
    if device == "cuda":
        return "float16"
    return "int8_float16"


def load_whisper() -> WhisperModel:
    global _whisper, _whisper_model_name

    device = "cuda" if torch.cuda.is_available() else "cpu"
    compute_type = _resolve_compute_type(device)
    _whisper_model_name = settings.whisper_model_size.strip() or "large-v3"

    logger.info(
        "Loading faster-whisper '%s' on %s (%s)…",
        _whisper_model_name,
        device,
        compute_type,
    )
    try:
        _whisper = WhisperModel(
            _whisper_model_name,
            device=device,
            compute_type=compute_type,
            cpu_threads=settings.whisper_cpu_threads,
            num_workers=settings.whisper_num_workers,
        )
    except ValueError as exc:
        if compute_type == "int8_float16" and device == "cpu":
            logger.warning("int8_float16 unavailable (%s), falling back to int8", exc)
            _whisper = WhisperModel(
                _whisper_model_name,
                device=device,
                compute_type="int8",
                cpu_threads=settings.whisper_cpu_threads,
                num_workers=settings.whisper_num_workers,
            )
        else:
            raise

    logger.info("faster-whisper loaded successfully")
    return _whisper


def get_whisper() -> WhisperModel:
    if _whisper is None:
        raise RuntimeError("Whisper model not loaded. Did lifespan run?")
    return _whisper


def get_whisper_model_name() -> str:
    return _whisper_model_name or settings.whisper_model_size


def _run_transcribe(model: WhisperModel, audio_path: str, *, use_vad: bool) -> tuple[list, object]:
    kwargs: dict = {
        "language": settings.whisper_language,
        "word_timestamps": True,
        "vad_filter": use_vad,
        "condition_on_previous_text": False,
        "temperature": 0.0,
        "compression_ratio_threshold": 2.4,
        "no_speech_threshold": 0.5,
        "beam_size": settings.whisper_beam_size,
        "best_of": settings.whisper_beam_size,
    }
    if use_vad:
        kwargs["vad_parameters"] = {
            "min_silence_duration_ms": 500,
            "speech_pad_ms": 300,
        }

    prompt = (settings.whisper_initial_prompt or "").strip()
    if prompt:
        kwargs["initial_prompt"] = prompt

    segments_iter, info = model.transcribe(audio_path, **kwargs)
    return list(segments_iter), info


def transcribe_audio(audio_path: str) -> list[TranscriptSegment]:
    model = get_whisper()
    logger.info("Transcribing with %s: %s", get_whisper_model_name(), audio_path)

    raw, info = _run_transcribe(model, audio_path, use_vad=True)
    sentences = _segments_from_whisper(raw)

    if not sentences:
        logger.warning("0 segments with VAD — retry without VAD")
        raw, info = _run_transcribe(model, audio_path, use_vad=False)
        sentences = _segments_from_whisper(raw)

    if sentences:
        sample = [(round(s.start_sec, 2), round(s.end_sec, 2), s.text[:50]) for s in sentences[:5]]
        logger.info("Sample timings: %s", sample)

    logger.info(
        "Whisper done — lang=%s (prob=%.2f) raw=%d kept=%d duration=%.1fs",
        info.language,
        info.language_probability,
        len(raw),
        len(sentences),
        getattr(info, "duration", 0.0),
    )
    return sentences
