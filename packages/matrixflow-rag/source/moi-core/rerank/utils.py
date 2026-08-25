from __future__ import annotations

import time
from typing import Optional


def now_ms() -> int:
    return int(time.time() * 1000)


def get_gpu_utilization() -> Optional[int]:
    try:
        import pynvml  # type: ignore

        pynvml.nvmlInit()
        handle = pynvml.nvmlDeviceGetHandleByIndex(0)
        util = pynvml.nvmlDeviceGetUtilizationRates(handle)
        return int(util.gpu)
    except Exception:
        return None
