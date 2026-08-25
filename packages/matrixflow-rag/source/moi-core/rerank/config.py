from __future__ import annotations

import os
from dataclasses import dataclass


def _get_int(name: str, default: int) -> int:
    raw = os.getenv(name)
    if raw is None or raw == "":
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def _get_float(name: str, default: float) -> float:
    raw = os.getenv(name)
    if raw is None or raw == "":
        return default
    try:
        return float(raw)
    except ValueError:
        return default


def _get_bool(name: str, default: bool) -> bool:
    raw = os.getenv(name)
    if raw is None or raw == "":
        return default
    return raw.lower() in {"1", "true", "yes", "y", "on"}


@dataclass(frozen=True)
class Settings:
    model_name: str = os.getenv("RERANK_MODEL", "BAAI/bge-reranker-base")
    backend: str = os.getenv("RERANK_BACKEND", "torch")  # torch | onnx
    onnx_path: str = os.getenv("RERANK_ONNX_PATH", "")
    device: str = os.getenv("RERANK_DEVICE", "auto")  # auto | cuda | cpu

    max_length: int = _get_int("RERANK_MAX_LENGTH", 256)
    normalize: bool = _get_bool("RERANK_NORMALIZE", False)

    # micro-batching
    batch_max_size: int = _get_int("RERANK_BATCH_MAX_SIZE", 32)
    batch_min_size: int = _get_int("RERANK_BATCH_MIN_SIZE", 4)
    batch_max_wait_ms: int = _get_int("RERANK_BATCH_MAX_WAIT_MS", 10)
    batch_target_ms: int = _get_int("RERANK_BATCH_TARGET_MS", 50)
    batch_step: int = _get_int("RERANK_BATCH_STEP", 4)

    # GPU scheduling
    enable_gpu_metrics: bool = _get_bool("RERANK_GPU_METRICS", False)
    gpu_util_high: int = _get_int("RERANK_GPU_UTIL_HIGH", 90)
    gpu_util_low: int = _get_int("RERANK_GPU_UTIL_LOW", 40)

    # safety limits
    max_docs: int = _get_int("RERANK_MAX_DOCS", 200)
    max_text_chars: int = _get_int("RERANK_MAX_TEXT_CHARS", 4000)


SETTINGS = Settings()
