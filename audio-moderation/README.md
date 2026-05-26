# Audio Moderation Service

**faster-whisper** (OpenAI Whisper) + **PhoBERT** (Clean / Toxic).

## ASR — faster-whisper

Uses [faster-whisper](https://github.com/SYSTRAN/faster-whisper) with standard CTranslate2 models (auto-download):

| `AUDIO_WHISPER_MODEL_SIZE` | Notes |
|----------------------------|--------|
| `large-v3` (default) | Best quality, needs GPU for reasonable speed |
| `medium` | Lighter |
| `small` / `base` / `tiny` | Faster, lower quality |

Set via env or Colab `WHISPER_MODEL_SIZE`.

## PhoBERT

Binary classification **Clean** / **Toxic** on each transcribed segment.

## Run locally

```bash
cp .env.example .env
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8001
```

## Colab + ngrok

See `audio_moderation_colab.ipynb` for Colab + ngrok (same pipeline: faster-whisper + PhoBERT). Upload **mono 16 kHz WAV** from the Go server.
