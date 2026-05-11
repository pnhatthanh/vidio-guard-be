"""
API router — /predict endpoints.
"""
import logging
from fastapi import APIRouter, File, UploadFile, Depends, HTTPException
from typing import Annotated

import tensorflow as tf

from app.model import get_model, preprocess_image, predict_batch
from app.schemas import BatchPredictResponse, FramePrediction
from app.config import get_settings

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/images/predict", tags=["predict"])
settings = get_settings()

@router.post(
    "/batch",
    response_model=BatchPredictResponse,
    summary="Predict labels for a batch of frames",
    description=(
        "Upload multiple JPEG/PNG frames at once. "
        "Returns violence / nsfw / safe prediction for each frame."
    ),
)
async def predict_frames_batch(
    files: Annotated[
        list[UploadFile],
        File(description="List of frame images (JPEG/PNG)"),
    ],
    model: tf.keras.Model = Depends(get_model),
) -> BatchPredictResponse:
    if not files:
        raise HTTPException(status_code=422, detail="No files uploaded.")

    max_batch = settings.batch_size
    if len(files) > max_batch:
        raise HTTPException(
            status_code=422,
            detail=f"Too many frames. Max batch size is {max_batch}.",
        )

    images = []
    filenames = []

    for upload in files:
        raw = await upload.read()
        try:
            arr = preprocess_image(raw)
        except Exception as exc:
            logger.warning(f"Failed to decode {upload.filename}: {exc}")
            raise HTTPException(
                status_code=422,
                detail=f"Cannot decode image: {upload.filename}",
            )
        images.append(arr)
        filenames.append(upload.filename or f"frame_{len(filenames)}.jpg")

    logger.info(f"Running inference on {len(images)} frames")
    raw_results = predict_batch(images, model)

    predictions = [
        FramePrediction(frame=fname, **res)
        for fname, res in zip(filenames, raw_results)
    ]

    return BatchPredictResponse(total=len(predictions), predictions=predictions)
