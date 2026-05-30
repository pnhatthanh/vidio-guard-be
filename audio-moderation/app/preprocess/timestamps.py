"""
Merge Whisper segments with global timeline offsets (after Silero chunking).
"""
from __future__ import annotations

from app.segments import TranscriptSegment


def offset_segments(
    segments: list[TranscriptSegment],
    offset_sec: float,
) -> list[TranscriptSegment]:
    if offset_sec == 0:
        return segments
    return [
        TranscriptSegment(
            text=s.text,
            start_sec=s.start_sec + offset_sec,
            end_sec=s.end_sec + offset_sec,
        )
        for s in segments
    ]


def merge_segment_lists(lists: list[list[TranscriptSegment]]) -> list[TranscriptSegment]:
    merged: list[TranscriptSegment] = []
    for batch in lists:
        merged.extend(batch)
    merged.sort(key=lambda s: (s.start_sec, s.end_sec))
    return merged
