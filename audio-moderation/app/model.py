"""
PhoBERT toxic classifier — CustomPhoBERT (Clean / Toxic), aligned with Colab training.
"""
import logging

import numpy as np
import torch
from pyvi import ViTokenizer
from transformers import AutoTokenizer

from app.config import get_settings
from app.phobert_arch import CustomPhoBERT

logger = logging.getLogger(__name__)
settings = get_settings()

LABEL_MAP: dict[int, str] = {
    0: "Clean",
    1: "Toxic",
}

FLAGGED_LABELS = {"Toxic"}

_tokenizer: AutoTokenizer | None = None
_model: CustomPhoBERT | None = None
_device: torch.device | None = None


def load_phobert() -> None:
    global _tokenizer, _model, _device

    _device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    model_path = settings.phobert_model_path

    logger.info("Loading PhoBERT tokenizer from: %s", model_path)
    _tokenizer = AutoTokenizer.from_pretrained(model_path)

    logger.info(
        "Loading CustomPhoBERT from %s (base=%s, labels=%d, device=%s)",
        model_path,
        settings.phobert_base_model,
        settings.phobert_num_labels,
        _device,
    )
    _model = CustomPhoBERT.from_pretrained(
        save_directory=model_path,
        base_model_name=settings.phobert_base_model,
        num_labels=settings.phobert_num_labels,
        dropout_prob=settings.phobert_dropout,
        unfreeze_last_n=settings.phobert_unfreeze_last_n,
    ).to(_device)
    _model.eval()

    logger.info("PhoBERT model loaded successfully (device=%s)", _device)


def is_loaded() -> bool:
    return _model is not None and _tokenizer is not None


def _word_segment(text: str) -> str:
    """Match training: segment + lowercase."""
    return ViTokenizer.tokenize(str(text).strip().lower())


@torch.no_grad()
def predict(texts: list[str]) -> list[dict]:
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
            label = LABEL_MAP.get(pred_id, "Clean")
            scores = {"Clean": float(prob[0])}
            if prob.shape[0] > 1:
                scores["Toxic"] = float(prob[1])
            all_results.append(
                {
                    "label": label,
                    "label_id": pred_id,
                    "confidence": float(prob[pred_id]),
                    "scores": scores,
                }
            )

    return all_results
