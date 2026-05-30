# Audio Moderation Service

**Preprocess** (DeepFilterNet + Silero VAD) → **faster-whisper** (`large-v3`) → **PhoBERT** (Clean / Toxic).

## Pre-ASR pipeline

| Step | Env | Default |
|------|-----|---------|
| DeepFilterNet denoise | `AUDIO_PREPROCESS_DENOISE` | `true` |
| Silero VAD chunking | `AUDIO_PREPROCESS_SILERO_VAD` | `true` |
| Demucs vocals (heavy) | `AUDIO_PREPROCESS_DEMUCS` | `false` |
| Whisper built-in VAD | `AUDIO_WHISPER_USE_BUILTIN_VAD` | `false` when Silero on |

Go server normalizes level at extract time (`dynaudnorm` + `highpass` in FFmpeg).

## ASR — faster-whisper

Uses [faster-whisper](https://github.com/SYSTRAN/faster-whisper) with standard CTranslate2 models (auto-download):

| `AUDIO_WHISPER_MODEL_SIZE` | Notes |
|----------------------------|--------|
| `large-v3` (default) | Best quality, needs GPU for reasonable speed |
| `medium` | Lighter |

## PhoBERT

Binary classification **Clean** / **Toxic** on each transcribed segment.

## Run locally

```bash
cp .env.example .env
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8001
```

Optional Demucs (only if `AUDIO_PREPROCESS_DEMUCS=true`):

```bash
pip install demucs
```

## Colab + ngrok

See `audio_moderation_colab.ipynb` — Colab pipeline: **Silero VAD → chunking → Whisper → PhoBERT** (no DeepFilterNet on Python 3.12). Upload **mono 16 kHz WAV** from the Go server.
