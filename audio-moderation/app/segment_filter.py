"""
Chỉ lọc câu outro/spam YouTube — KHÔNG lọc theo logprob (dễ xóa hết lời nói khi có nhạc).
"""
import re
import unicodedata

_HALLUCINATION_FRAGMENTS = (
    "subscribe",
    "đăng ký",
    "đăng kí",
    "kênh youtube",
    "ghiền mì gõ",
    "ghien mi go",
    "không bỏ lỡ",
    "video hấp dẫn",
    "cảm ơn các bạn đã theo dõi",
    "hãy like",
    "bấm chuông",
    "theo dõi kênh",
    "xem video tiếp theo",
    "tiếng việt phổ thông",
    "dấu thanh đầy đủ",
    "để không bỏ lỡ những video",
)


def _normalize(text: str) -> str:
    text = unicodedata.normalize("NFC", text.strip().lower())
    return re.sub(r"\s+", " ", text)


def is_spam_or_hallucination(text: str) -> bool:
    norm = _normalize(text)
    if len(norm) < 3:
        return True
    return any(frag in norm for frag in _HALLUCINATION_FRAGMENTS)
