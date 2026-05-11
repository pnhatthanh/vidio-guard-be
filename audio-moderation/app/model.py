"""
PhoBERT singleton — loads the model once at startup and exposes `predict()`.

Label map:
  0 → Clean
  1 → Offensive
  2 → Hate
"""
import logging
import numpy as np
import torch
from transformers import AutoTokenizer, AutoModelForSequenceClassification
from pyvi import ViTokenizer

from app.config import get_settings

logger = logging.getLogger(__name__)
settings = get_settings()

# ── Label map ─────────────────────────────────────────────────────────────────
LABEL_MAP: dict[int, str] = {
    0: "Clean",
    1: "Offensive",
    2: "Hate",
}

# ── Global singletons ─────────────────────────────────────────────────────────
_tokenizer: AutoTokenizer | None = None
_model: AutoModelForSequenceClassification | None = None
_device: torch.device | None = None


def load_phobert() -> None:
    """Load tokenizer + model at startup (called once from lifespan)."""
    global _tokenizer, _model, _device

    _device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    model_path = settings.phobert_model_path

    logger.info("📦 Loading PhoBERT tokenizer from: %s", model_path)
    _tokenizer = AutoTokenizer.from_pretrained(model_path)

    logger.info("🧠 Loading PhoBERT model from: %s  (device=%s)", model_path, _device)
    _model = AutoModelForSequenceClassification.from_pretrained(model_path).to(_device)
    _model.eval()

    logger.info("✅ PhoBERT model loaded successfully (device=%s)", _device)


def is_loaded() -> bool:
    return _model is not None and _tokenizer is not None


def _word_segment(text: str) -> str:
    """Vietnamese word segmentation — must match training pipeline."""
    return ViTokenizer.tokenize(str(text).strip())


@torch.no_grad()
def predict(texts: list[str]) -> list[dict]:
    """
    Classify a list of Vietnamese sentences.

    Returns a list of dicts, one per input text:
    {
        "label":      "Clean" | "Offensive" | "Hate",
        "label_id":   0 | 1 | 2,
        "confidence": float,
        "scores":     {"Clean": float, "Offensive": float, "Hate": float},
    }
    """
    if not is_loaded():
        raise RuntimeError("PhoBERT model not loaded. Did lifespan run?")

    batch_size = settings.phobert_batch_size
    segmented = [_word_segment(t) for t in texts]
    all_results: list[dict] = []

    for i in range(0, len(segmented), batch_size):
        batch = segmented[i : i + batch_size]

        encoded = _tokenizer(
            batch,
            truncation=True,
            padding=True,
            max_length=settings.phobert_max_length,
            return_tensors="pt",
        )
        encoded = {k: v.to(_device) for k, v in encoded.items()}

        outputs = _model(**encoded)
        probs = torch.softmax(outputs.logits, dim=-1).cpu().numpy()

        for prob in probs:
            pred_id = int(np.argmax(prob))
            all_results.append(
                {
                    "label":      LABEL_MAP[pred_id],
                    "label_id":   pred_id,
                    "confidence": float(prob[pred_id]),
                    "scores": {
                        "Clean":     float(prob[0]),
                        "Offensive": float(prob[1]),
                        "Hate":      float(prob[2]),
                    },
                }
            )

    return all_results
