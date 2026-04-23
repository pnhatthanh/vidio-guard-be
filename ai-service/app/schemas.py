"""
Pydantic schemas for request/response validation.
"""
from pydantic import BaseModel, Field


class FramePrediction(BaseModel):
    """Prediction result for a single frame."""
    frame: str = Field(..., description="Original filename of the frame")
    label: str = Field(..., description="Predicted class: nsfw | safe | violence")
    confidence: float = Field(..., ge=0.0, le=1.0, description="Confidence of top label")
    scores: dict[str, float] = Field(..., description="Softmax scores per class")


class BatchPredictResponse(BaseModel):
    """Response for /predict/batch endpoint."""
    total: int = Field(..., description="Number of frames processed")
    predictions: list[FramePrediction]


class HealthResponse(BaseModel):
    """Response for /health endpoint."""
    status: str
    model_loaded: bool
    labels: list[str]
