"""
Configuration management via environment variables.
"""
from pydantic_settings import BaseSettings
from functools import lru_cache


class Settings(BaseSettings):
    # Model
    model_path: str = "models/efficientnet.keras"
    # EfficientNetB3 standard input size is 300x300
    img_size: int = 300
    batch_size: int = 32
    labels: list[str] = ["nsfw", "safe", "violence"]

    # Prediction thresholds — frames exceeding these are flagged directly
    nsfw_threshold: float = 0.6
    violence_threshold: float = 0.6

    # Service
    host: str = "0.0.0.0"
    port: int = 8000
    workers: int = 1
    log_level: str = "info"

    # CORS (allow Golang API / Worker to call)
    cors_origins: list[str] = ["*"]

    class Config:
        env_file = ".env"
        env_prefix = "AI_"


@lru_cache
def get_settings() -> Settings:
    return Settings()
