from __future__ import annotations

from typing import List, Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from .backend import create_backend
from .batcher import MicroBatcher
from .config import SETTINGS


class Doc(BaseModel):
    id: str = Field(..., description="Document identifier")
    text: str = Field(..., description="Document content")


class RerankRequest(BaseModel):
    query: str
    documents: List[Doc]
    max_length: Optional[int] = None
    normalize: Optional[bool] = None


class ScoreItem(BaseModel):
    id: str
    score: float


class RerankResponse(BaseModel):
    scores: List[ScoreItem]


app = FastAPI(title="Rerank Service", version="0.1.0")
_backend = create_backend()
_batcher = MicroBatcher(_backend)


@app.on_event("startup")
async def _startup() -> None:
    await _batcher.start()


@app.on_event("shutdown")
async def _shutdown() -> None:
    await _batcher.stop()


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.post("/v1/rerank", response_model=RerankResponse)
async def rerank(req: RerankRequest) -> RerankResponse:
    if not req.query.strip():
        raise HTTPException(status_code=400, detail="query is required")
    if not req.documents:
        return RerankResponse(scores=[])
    if len(req.documents) > SETTINGS.max_docs:
        raise HTTPException(status_code=400, detail="too many documents")

    # Apply global max_length and normalize to keep batching efficient.
    max_length = SETTINGS.max_length
    normalize = SETTINGS.normalize

    pairs = []
    for d in req.documents:
        text = d.text
        if len(text) > SETTINGS.max_text_chars:
            text = text[: SETTINGS.max_text_chars]
        pairs.append((req.query, text))

    scores = await _batcher.submit(pairs)
    return RerankResponse(scores=[ScoreItem(id=d.id, score=s) for d, s in zip(req.documents, scores)])
