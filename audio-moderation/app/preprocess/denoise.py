"""
DeepFilterNet speech enhancement (noise reduction).
"""
from __future__ import annotations

import logging
import os
import tempfile
logger = logging.getLogger(__name__)

_df_model = None
_df_state = None


def load_deepfilter() -> bool:
    """Load DeepFilterNet once. Returns True if ready."""
    global _df_model, _df_state
    if _df_model is not None:
        return True
    try:
        from df.enhance import init_df

        logger.info("Loading DeepFilterNet…")
        _df_model, _df_state, _ = init_df()
        logger.info("DeepFilterNet loaded")
        return True
    except Exception as exc:
        logger.error("DeepFilterNet load failed: %s", exc)
        return False


def is_deepfilter_loaded() -> bool:
    return _df_model is not None and _df_state is not None


def deepfilter_enhance(input_path: str) -> str:
    """
    Denoise input WAV; returns path to enhanced WAV (temp file).
  Falls back to input_path if model unavailable.
    """
    if not is_deepfilter_loaded() and not load_deepfilter():
        logger.warning("DeepFilterNet unavailable — skipping denoise")
        return input_path

    from df.enhance import enhance, load_audio, save_audio

    audio, _ = load_audio(input_path, sr=_df_state.sr())
    enhanced = enhance(_df_model, _df_state, audio)
    fd, out_path = tempfile.mkstemp(suffix="_denoised.wav", prefix="vg_audio_")
    os.close(fd)
    save_audio(out_path, enhanced, sr=_df_state.sr())
    logger.info("DeepFilterNet wrote %s", out_path)
    return out_path
