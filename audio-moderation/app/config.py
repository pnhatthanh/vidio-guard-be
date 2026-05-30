"""
Configuration via environment variables (prefix AUDIO_).
"""
from functools import lru_cache

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # ── faster-whisper (OpenAI Whisper CT2) ───────────────────────────────────
    whisper_model_size: str = "large-v3"
    whisper_language: str = "vi"
    whisper_compute_type: str = "auto"
    whisper_initial_prompt: str = ""
    whisper_beam_size: int = 5
    whisper_cpu_threads: int = 4
    whisper_num_workers: int = 1
    whisper_use_builtin_vad: bool = False

    # ── Pre-ASR pipeline ──────────────────────────────────────────────────────
    preprocess_enabled: bool = True
    preprocess_denoise: bool = True
    preprocess_silero_vad: bool = True
    preprocess_demucs: bool = False
    silero_threshold: float = 0.5
    silero_min_speech_ms: int = 250
    silero_min_silence_ms: int = 300
    silero_speech_pad_ms: int = 200
    chunk_max_sec: float = 30.0

    # ── PhoBERT (CustomPhoBERT binary: Clean / Toxic) ─────────────────────────
    phobert_model_path: str = "models"
    phobert_base_model: str = "vinai/phobert-base-v2"
    phobert_num_labels: int = 2
    phobert_dropout: float = 0.2
    phobert_unfreeze_last_n: int = 8
    phobert_max_length: int = 128
    phobert_batch_size: int = 32

    # ── Service ───────────────────────────────────────────────────────────────
    host: str = "0.0.0.0"
    port: int = 8001
    workers: int = 1
    log_level: str = "info"
    cors_origins: list[str] = ["*"]

    class Config:
        env_file = ".env"
        env_prefix = "AUDIO_"


@lru_cache
def get_settings() -> Settings:
    return Settings()
