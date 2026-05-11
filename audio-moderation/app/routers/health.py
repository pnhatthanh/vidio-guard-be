"""
Health-check router.
"""
import torch
from fastapi import APIRouter

from app.model import is_loaded as phobert_loaded
from app.transcriber import _whisper
from app.schemas import HealthResponse

router = APIRouter(tags=["health"])


@router.get("/health", response_model=HealthResponse, summary="Service health check")
def health() -> HealthResponse:
    device = "cuda" if torch.cuda.is_available() else "cpu"
    return HealthResponse(
        status="ok",
        whisper_loaded=_whisper is not None,
        phobert_loaded=phobert_loaded(),
        device=device,
    )
