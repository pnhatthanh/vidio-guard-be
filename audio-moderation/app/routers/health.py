"""
Health-check router.
"""
import torch
from fastapi import APIRouter

from app.config import get_settings
from app.model import is_loaded as phobert_loaded
from app.transcriber import _whisper, get_whisper_model_name
from app.schemas import HealthResponse

router = APIRouter(tags=["health"])
settings = get_settings()


@router.get("/health", response_model=HealthResponse, summary="Service health check")
def health() -> HealthResponse:
    device = "cuda" if torch.cuda.is_available() else "cpu"
    return HealthResponse(
        status="ok",
        whisper_loaded=_whisper is not None,
        whisper_model=get_whisper_model_name() if _whisper else settings.whisper_model_size,
        phobert_loaded=phobert_loaded(),
        device=device,
    )
