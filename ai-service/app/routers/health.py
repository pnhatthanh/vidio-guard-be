"""
Health check router.
"""
from fastapi import APIRouter
from app.model import _model
from app.schemas import HealthResponse
from app.config import get_settings

router = APIRouter(tags=["health"])
settings = get_settings()


@router.get("/health", response_model=HealthResponse, summary="Service health check")
def health() -> HealthResponse:
    return HealthResponse(
        status="ok",
        model_loaded=_model is not None,
        labels=settings.labels,
    )
