#!/usr/bin/env python3
"""
Entry point — runs Uvicorn programmatically.
Usage: python run.py
"""
import uvicorn
from app.config import get_settings

settings = get_settings()

if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host=settings.host,
        port=settings.port,
        workers=settings.workers,   # keep at 1 — PyTorch doesn't fork well
        log_level=settings.log_level,
        reload=False,
    )
