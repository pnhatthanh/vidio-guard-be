from pydantic import BaseModel, Field


# ── Per-sentence result ───────────────────────────────────────────────────────

class SentencePrediction(BaseModel):
    text:       str   = Field(..., description="Original sentence text")
    label:      str   = Field(..., description="Clean | Toxic")
    label_id:   int   = Field(..., description="0=Clean, 1=Toxic")
    confidence: float = Field(..., ge=0.0, le=1.0, description="Top-class confidence")
    scores:     dict[str, float] = Field(..., description="Softmax scores per class")
    start_sec:  float = Field(..., ge=0.0, description="Segment start time in seconds")
    end_sec:    float = Field(..., ge=0.0, description="Segment end time in seconds")


# ── Full audio predict response ────────────────────────────────────────────────

class AudioPredictResponse(BaseModel):
    total_sentences: int                  = Field(..., description="Number of segments transcribed")
    flagged_count:   int                  = Field(..., description="Sentences classified as Toxic")
    overall_label:   str                  = Field(..., description="Worst-case label across all sentences")
    sentences:       list[SentencePrediction]


# ── Health ────────────────────────────────────────────────────────────────────

class HealthResponse(BaseModel):
    status:         str
    whisper_loaded: bool
    whisper_model:  str = ""
    phobert_loaded: bool
    device:         str
