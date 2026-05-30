"""
Optional Demucs vocal separation (heavy; off by default).
"""
from __future__ import annotations

import logging
import os
import shutil
import subprocess
import tempfile
from pathlib import Path

logger = logging.getLogger(__name__)


def separate_vocals(input_path: str) -> str:
    """
    Extract vocals stem via Demucs CLI. Returns path to vocals WAV (temp).
    On failure, returns input_path unchanged.
    """
    demucs_bin = shutil.which("demucs")
    if demucs_bin is None:
        try:
            import demucs  # noqa: F401
        except ImportError:
            logger.warning("Demucs not installed — skipping vocal separation")
            return input_path
        demucs_bin = "demucs"

    out_root = tempfile.mkdtemp(prefix="vg_demucs_")
    try:
        cmd = [
            demucs_bin,
            "-n",
            "htdemucs",
            "--two-stems",
            "vocals",
            "-o",
            out_root,
            input_path,
        ]
        logger.info("Running Demucs: %s", " ".join(cmd))
        subprocess.run(cmd, check=True, capture_output=True, text=True)

        stem_name = Path(input_path).stem
        vocals = Path(out_root) / "htdemucs" / stem_name / "vocals.wav"
        if not vocals.is_file():
            logger.warning("Demucs vocals not found at %s", vocals)
            return input_path

        fd, dest = tempfile.mkstemp(suffix="_vocals.wav", prefix="vg_audio_")
        os.close(fd)
        shutil.copy2(vocals, dest)
        logger.info("Demucs vocals saved to %s", dest)
        return dest
    except subprocess.CalledProcessError as exc:
        logger.error("Demucs failed: %s %s", exc.stderr, exc.stdout)
        return input_path
    finally:
        shutil.rmtree(out_root, ignore_errors=True)
