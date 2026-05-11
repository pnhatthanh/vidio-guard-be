# ----- Builder stage -----
FROM python:3.11-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Build tools needed to compile some wheels (e.g. tokenizers Rust extension)
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        curl \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

# Pre-build all wheels into /app/wheels — avoids recompilation in runtime image
RUN pip wheel --no-cache-dir --wheel-dir /app/wheels -r requirements.txt


# ----- Runtime stage -----
FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# ffmpeg is needed by faster-whisper for audio decoding
RUN apt-get update && apt-get install -y --no-install-recommends \
        ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Install from pre-built wheels (no compiler, no internet needed at runtime)
COPY --from=builder /app/wheels /wheels
RUN pip install --no-cache-dir /wheels/* \
    && rm -rf /wheels

# Copy application source (models/ is excluded via .dockerignore)
COPY app/ ./app/
COPY run.py .

# Mount point for PhoBERT model files — provided via Docker volume at runtime
RUN mkdir -p /app/models

EXPOSE 8001

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "1"]
