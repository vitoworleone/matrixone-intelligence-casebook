from __future__ import annotations

import asyncio
import time
from collections import deque
from dataclasses import dataclass
from typing import Deque, List, Tuple

from .backend import RerankBackend, ScoreResult
from .config import SETTINGS
from .utils import get_gpu_utilization

Pair = Tuple[str, str]


@dataclass
class BatchItem:
    pairs: List[Pair]
    future: asyncio.Future


class MicroBatcher:
    def __init__(self, backend: RerankBackend) -> None:
        self._backend = backend
        self._queue: Deque[BatchItem] = deque()
        self._cv = asyncio.Condition()
        self._running = False
        self._target_batch = SETTINGS.batch_max_size

    async def start(self) -> None:
        if self._running:
            return
        self._running = True
        asyncio.create_task(self._loop())

    async def stop(self) -> None:
        self._running = False
        async with self._cv:
            self._cv.notify_all()

    async def submit(self, pairs: List[Pair]) -> List[float]:
        loop = asyncio.get_running_loop()
        fut: asyncio.Future = loop.create_future()
        item = BatchItem(pairs=pairs, future=fut)
        async with self._cv:
            self._queue.append(item)
            self._cv.notify()
        return await fut

    async def _loop(self) -> None:
        while self._running:
            async with self._cv:
                while not self._queue and self._running:
                    await self._cv.wait()
                if not self._running:
                    break
                batch, started_at = await self._collect_batch()

            if not batch:
                continue

            # Flatten pairs
            flat_pairs: List[Pair] = []
            spans: List[Tuple[int, int]] = []
            cursor = 0
            for item in batch:
                flat_pairs.extend(item.pairs)
                spans.append((cursor, cursor + len(item.pairs)))
                cursor += len(item.pairs)

            t0 = time.time()
            try:
                result: ScoreResult = self._backend.score(
                    flat_pairs,
                    max_length=SETTINGS.max_length,
                    normalize=SETTINGS.normalize,
                )
                scores = result.scores
            except Exception as exc:
                for item in batch:
                    if not item.future.done():
                        item.future.set_exception(exc)
                continue

            for item, (s, e) in zip(batch, spans):
                if not item.future.done():
                    item.future.set_result(scores[s:e])

            latency_ms = int((time.time() - t0) * 1000)
            self._adjust_batch_size(latency_ms)

    async def _collect_batch(self) -> Tuple[List[BatchItem], float]:
        batch: List[BatchItem] = []
        started_at = time.time()
        max_wait = SETTINGS.batch_max_wait_ms / 1000.0
        target = min(self._target_batch, SETTINGS.batch_max_size)

        while True:
            if self._queue and len(batch) < target:
                batch.append(self._queue.popleft())
                if len(batch) >= target:
                    break
                continue

            elapsed = time.time() - started_at
            if elapsed >= max_wait:
                break

            timeout = max_wait - elapsed
            try:
                await asyncio.wait_for(self._cv.wait(), timeout=timeout)
            except asyncio.TimeoutError:
                break

        return batch, started_at

    def _adjust_batch_size(self, latency_ms: int) -> None:
        target_ms = SETTINGS.batch_target_ms
        step = SETTINGS.batch_step

        util = get_gpu_utilization() if SETTINGS.enable_gpu_metrics else None
        backlog = len(self._queue)

        if util is not None:
            if util < SETTINGS.gpu_util_low and backlog > 0:
                self._target_batch = min(self._target_batch + step, SETTINGS.batch_max_size)
                return
            if util > SETTINGS.gpu_util_high:
                self._target_batch = max(self._target_batch - step, SETTINGS.batch_min_size)
                return

        if latency_ms < target_ms and backlog > 0:
            self._target_batch = min(self._target_batch + step, SETTINGS.batch_max_size)
        elif latency_ms > target_ms:
            self._target_batch = max(self._target_batch - step, SETTINGS.batch_min_size)
