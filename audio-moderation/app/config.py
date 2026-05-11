"""
Configuration management via environment variables.
"""
from functools import lru_cache
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # ── Whisper ──────────────────────────────────────────────────────────────
    whisper_model_size: str = "medium"   # tiny | base | small | medium | large

    # ── PhoBERT ──────────────────────────────────────────────────────────────
    phobert_model_path: str = "models"   # folder with config.json, model.safetensors, …
    phobert_max_length: int = 128
    phobert_batch_size: int = 32

    # ── Service ───────────────────────────────────────────────────────────────
    host: str = "0.0.0.0"
    port: int = 8001
    workers: int = 1
    log_level: str = "info"

    # ── CORS ──────────────────────────────────────────────────────────────────
    cors_origins: list[str] = ["*"]

    class Config:
        env_file = ".env"
        env_prefix = "AUDIO_"


@lru_cache
def get_settings() -> Settings:
    return Settings()
