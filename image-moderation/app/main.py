"""
FastAPI application factory with lifespan model loading.
"""
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.config import get_settings
from app.model import load_model
from app.routers import predict, health

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s — %(message)s",
)
logger = logging.getLogger(__name__)
settings = get_settings()


# ─────────────────────────────────────────────────────────────────────────────
# Lifespan: load model once on startup, release on shutdown
# ─────────────────────────────────────────────────────────────────────────────
@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("🚀 Starting VideoGuard AI Service...")
    load_model()
    yield
    logger.info("🛑 Shutting down VideoGuard AI Service...")


# ─────────────────────────────────────────────────────────────────────────────
# App factory
# ─────────────────────────────────────────────────────────────────────────────
def create_app() -> FastAPI:
    app = FastAPI(
        title="VideoGuard AI Service",
        description=(
            "EfficientNet-based image moderation service. "
            "Classifies video frames into: **violence**, **nsfw**, **safe**."
        ),
        version="1.0.0",
        lifespan=lifespan,
        docs_url="/docs",
        redoc_url="/redoc",
    )

    # CORS — allow Golang worker to call
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_methods=["POST", "GET"],
        allow_headers=["*"],
    )

    # Routers
    app.include_router(health.router)
    app.include_router(predict.router)

    return app


app = create_app()
