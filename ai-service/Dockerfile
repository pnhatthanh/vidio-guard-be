# ----- Builder stage -----
FROM python:3.11-slim AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Install system build tools + runtime libs needed to compile some wheels
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        libgl1 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

# Build wheels into a temp dir (no pip cache, no install overhead)
RUN pip wheel --no-cache-dir --wheel-dir /app/wheels -r requirements.txt

# ----- Runtime stage -----
FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Only runtime system libs — no gcc/build-essential
RUN apt-get update && apt-get install -y --no-install-recommends \
        libgl1 \
    && rm -rf /var/lib/apt/lists/*

# Install from pre-built wheels (no compiler, no internet needed)
COPY --from=builder /app/wheels /wheels
RUN pip install --no-cache-dir /wheels/* \
    && rm -rf /wheels

# Copy only application source (models/ is excluded via .dockerignore)
COPY app/ ./app/
COPY run.py .

# Mount point for model files — actual files are provided via Docker volume
RUN mkdir -p /app/models

EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "1"]
