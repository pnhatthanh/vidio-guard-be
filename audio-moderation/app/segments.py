"""Shared transcript segment type (ASR output)."""
from dataclasses import dataclass


@dataclass
class TranscriptSegment:
    text: str
    start_sec: float
    end_sec: float
