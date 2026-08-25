from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable, List, Tuple

from .config import SETTINGS

try:
    import torch
except Exception:  # pragma: no cover
    torch = None

try:
    import onnxruntime as ort
except Exception:  # pragma: no cover
    ort = None

try:
    from transformers import AutoTokenizer, AutoModelForSequenceClassification
except Exception:  # pragma: no cover
    AutoTokenizer = None
    AutoModelForSequenceClassification = None


Pair = Tuple[str, str]


@dataclass
class ScoreResult:
    scores: List[float]


class BackendError(RuntimeError):
    pass


class RerankBackend:
    def score(self, pairs: Iterable[Pair], max_length: int, normalize: bool) -> ScoreResult:
        raise NotImplementedError


class TorchBackend(RerankBackend):
    def __init__(self, model_name: str, device: str) -> None:
        if AutoTokenizer is None or AutoModelForSequenceClassification is None or torch is None:
            raise BackendError("torch/transformers not available")

        if device == "auto":
            device = "cuda" if torch.cuda.is_available() else "cpu"

        self.device = device
        self.tokenizer = AutoTokenizer.from_pretrained(model_name)
        self.model = AutoModelForSequenceClassification.from_pretrained(model_name)
        if self.device == "cuda":
            self.model = self.model.half().to(self.device)
        else:
            self.model = self.model.to(self.device)
        self.model.eval()

    def score(self, pairs: Iterable[Pair], max_length: int, normalize: bool) -> ScoreResult:
        pairs = list(pairs)
        if not pairs:
            return ScoreResult(scores=[])

        enc = self.tokenizer(
            pairs,
            padding=True,
            truncation=True,
            max_length=max_length,
            return_tensors="pt",
        )
        if self.device == "cuda":
            enc = {k: v.to(self.device) for k, v in enc.items()}
            with torch.no_grad(), torch.cuda.amp.autocast():
                logits = self.model(**enc).logits
        else:
            with torch.no_grad():
                logits = self.model(**enc).logits
        scores = logits.squeeze(-1)
        if normalize:
            scores = torch.sigmoid(scores)
        scores = scores.detach().cpu().tolist()
        return ScoreResult(scores=scores)


class OnnxBackend(RerankBackend):
    def __init__(self, model_name: str, onnx_path: str, device: str) -> None:
        if AutoTokenizer is None or ort is None:
            raise BackendError("onnxruntime/transformers not available")
        if not onnx_path:
            raise BackendError("RERANK_ONNX_PATH is required for onnx backend")

        providers = ["CPUExecutionProvider"]
        if device == "auto":
            device = "cuda" if ort.get_device() == "GPU" else "cpu"
        if device == "cuda":
            providers = [
                "TensorrtExecutionProvider",
                "CUDAExecutionProvider",
                "CPUExecutionProvider",
            ]
        self.device = device
        self.tokenizer = AutoTokenizer.from_pretrained(model_name)
        self.session = ort.InferenceSession(onnx_path, providers=providers)
        self.input_names = {i.name for i in self.session.get_inputs()}

    def score(self, pairs: Iterable[Pair], max_length: int, normalize: bool) -> ScoreResult:
        pairs = list(pairs)
        if not pairs:
            return ScoreResult(scores=[])
        enc = self.tokenizer(
            pairs,
            padding=True,
            truncation=True,
            max_length=max_length,
            return_tensors="np",
        )
        feed = {}
        for key in ("input_ids", "attention_mask", "token_type_ids"):
            if key in self.input_names and key in enc:
                feed[key] = enc[key]
        outputs = self.session.run(None, feed)
        logits = outputs[0].squeeze(-1)
        if normalize:
            import numpy as np

            logits = 1 / (1 + np.exp(-logits))
        return ScoreResult(scores=logits.tolist())


def create_backend() -> RerankBackend:
    backend = SETTINGS.backend.lower()
    device = SETTINGS.device.lower()

    if backend == "onnx":
        return OnnxBackend(SETTINGS.model_name, SETTINGS.onnx_path, device)
    if backend == "torch":
        return TorchBackend(SETTINGS.model_name, device)
    raise BackendError(f"unknown backend: {backend}")
