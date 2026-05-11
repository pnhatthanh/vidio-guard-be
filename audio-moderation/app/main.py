"""
FastAPI application factory with lifespan model loading.

Startup order:
  1. Load Faster-Whisper (CPU/GPU auto-detect)
  2. Load PhoBERT tokenizer + model

Both models are singletons — loaded once and reused for all requests.
"""
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.config import get_settings
from app.transcriber import load_whisper
from app.model import load_phobert
from app.routers import health, predict

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s — %(message)s",
)
logger = logging.getLogger(__name__)
settings = get_settings()


# ── Lifespan: load both models once on startup ────────────────────────────────
@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("🚀 Starting VideoGuard Audio Moderation Service…")
    load_whisper()
    load_phobert()
    logger.info("✅ All models ready — service is accepting requests")
    yield
    logger.info("🛑 Shutting down Audio Moderation Service…")


# ── App factory ───────────────────────────────────────────────────────────────
def create_app() -> FastAPI:
    app = FastAPI(
        title="VideoGuard Audio Moderation Service",
        description=(
            "Faster-Whisper + PhoBERT audio moderation pipeline. "
            "Transcribes Vietnamese audio and classifies each segment as "
            "**Clean**, **Offensive**, or **Hate**."
        ),
        version="1.0.0",
        lifespan=lifespan,
        docs_url="/docs",
        redoc_url="/redoc",
    )

    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_methods=["POST", "GET"],
        allow_headers=["*"],
    )

    app.include_router(health.router)
    app.include_router(predict.router)

    return app


app = create_app()
