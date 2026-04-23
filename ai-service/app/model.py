"""
Singleton model loader — loads EfficientNet once at startup and reuses.
"""
import logging
import numpy as np
import tensorflow as tf
from PIL import Image
import io

from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

# Global model instance
_model: tf.keras.Model | None = None


def load_model() -> tf.keras.Model:
    """Load model from disk. Called once at app startup via lifespan."""
    global _model
    logger.info(f"Loading model from: {settings.model_path}")
    _model = tf.keras.models.load_model(settings.model_path)
    _model.summary(print_fn=logger.debug)
    logger.info("✅ Model loaded successfully")
    return _model


def get_model() -> tf.keras.Model:
    """FastAPI dependency — returns the cached model."""
    if _model is None:
        raise RuntimeError("Model not loaded. Did lifespan run?")
    return _model


def preprocess_image(image_bytes: bytes) -> np.ndarray:
    """
    Decode raw image bytes → numpy array ready for EfficientNetB3 inference.

    Steps:
      1. Open with Pillow (supports JPEG/PNG/WebP)
      2. Convert to RGB (drop alpha channel if any)
      3. Resize to (img_size, img_size) — EfficientNetB3 expects 300×300
      4. Normalize pixel values to [0, 1]  (model was trained with /255.0)
    """
    size = (settings.img_size, settings.img_size)  # (300, 300)
    img = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    img = img.resize(size, Image.BILINEAR)
    arr = np.array(img, dtype=np.float32) / 255.0
    return arr


def predict_batch(
    images: list[np.ndarray],
    model: tf.keras.Model,
    nsfw_threshold: float | None = None,
    violence_threshold: float | None = None,
) -> list[dict]:
    """
    Run inference on a batch of preprocessed images.

    Labelling logic (mirrors `predict_image` in training notebook):
      - If prob(nsfw)     > nsfw_threshold     → label = "nsfw"
      - Elif prob(violence) > violence_threshold → label = "violence"
      - Else                                    → label = argmax class

    Args:
        images:             list of float32 arrays shape (H, W, 3)
        model:              loaded Keras model
        nsfw_threshold:     override for nsfw cutoff  (default from settings)
        violence_threshold: override for violence cutoff (default from settings)

    Returns:
        list of dicts with keys: label, confidence, scores{nsfw, safe, violence}
    """
    thr_nsfw     = nsfw_threshold     if nsfw_threshold     is not None else settings.nsfw_threshold
    thr_violence = violence_threshold if violence_threshold is not None else settings.violence_threshold

    # Stack → (N, H, W, 3)
    batch = np.stack(images, axis=0)

    # Single forward pass — GPU-efficient
    raw_preds: np.ndarray = model.predict(batch, verbose=0)  # shape (N, num_classes)

    results = []
    for pred in raw_preds:
        prob_map: dict[str, float] = {
            label: float(pred[i])
            for i, label in enumerate(settings.labels)
        }

        # Threshold-first decision (matches training predict_image logic)
        if prob_map.get("nsfw", 0.0) > thr_nsfw:
            predicted_label = "nsfw"
        elif prob_map.get("violence", 0.0) > thr_violence:
            predicted_label = "violence"
        else:
            predicted_label = max(prob_map, key=prob_map.get)  # type: ignore[arg-type]

        results.append({
            "label":      predicted_label,
            "confidence": prob_map[predicted_label],
            "scores":     prob_map,
        })

    return results
