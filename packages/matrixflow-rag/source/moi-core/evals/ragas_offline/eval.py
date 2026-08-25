import argparse
import hashlib
import json
import os
import re
import time
from typing import Any, Dict, List, Optional

import requests


def _load_jsonl(path: str) -> List[Dict[str, Any]]:
    items = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            items.append(json.loads(line))
    return items


def _load_json(path: str) -> List[Dict[str, Any]]:
    with open(path, "r", encoding="utf-8") as f:
        data = json.load(f)
    if isinstance(data, list):
        return data
    if isinstance(data, dict) and "cases" in data:
        return data["cases"]
    raise ValueError("unsupported json format")


def load_dataset(path: str) -> List[Dict[str, Any]]:
    if path.endswith(".jsonl"):
        return _load_jsonl(path)
    return _load_json(path)


def save_json(path: str, payload: Dict[str, Any]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        json.dump(payload, f, ensure_ascii=False, indent=2)


def save_md(path: str, payload: Dict[str, Any]) -> None:
    lines = []
    summary = payload.get("summary", {})
    lines.append("# RAG Evaluation Report\n")
    lines.append("## Summary")
    for k in ["faithfulness", "answer_relevancy", "context_precision", "context_recall"]:
        if k in summary:
            lines.append(f"- {k}: {summary[k]:.4f}")
    lines.append("\n## Per Case")
    lines.append("| id | faithfulness | answer_relevancy | context_precision | context_recall |")
    lines.append("|---|---:|---:|---:|---:|")
    for item in payload.get("cases", []):
        def fmt(v):
            return "-" if v is None else f"{v:.4f}"
        lines.append(
            "| {id} | {f} | {ar} | {cp} | {cr} |".format(
                id=item.get("id", ""),
                f=fmt(item.get("faithfulness")),
                ar=fmt(item.get("answer_relevancy")),
                cp=fmt(item.get("context_precision")),
                cr=fmt(item.get("context_recall")),
            )
        )
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))


class LLMClient:
    def __init__(self) -> None:
        self.base = os.getenv("RAG_EVAL_API_BASE", "http://localhost:8080/v1").rstrip("/")
        self.key = os.getenv("RAG_EVAL_API_KEY", "")
        self.model = os.getenv("RAG_EVAL_MODEL", "gpt-4o-mini")
        self.timeout = float(os.getenv("RAG_EVAL_TIMEOUT", "60"))
        self.cache_path = os.getenv("RAG_EVAL_CACHE_PATH", ".cache.jsonl")
        self._cache = self._load_cache()

    def _load_cache(self) -> Dict[str, str]:
        cache: Dict[str, str] = {}
        if not os.path.exists(self.cache_path):
            return cache
        with open(self.cache_path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                obj = json.loads(line)
                cache[obj["key"]] = obj["value"]
        return cache

    def _save_cache_item(self, key: str, value: str) -> None:
        with open(self.cache_path, "a", encoding="utf-8") as f:
            f.write(json.dumps({"key": key, "value": value}, ensure_ascii=False) + "\n")

    def chat(self, system: str, user: str, temperature: float = 0.0, max_tokens: int = 512) -> str:
        payload = {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            "temperature": temperature,
            "max_tokens": max_tokens,
        }
        key = hashlib.sha256(json.dumps(payload, sort_keys=True).encode("utf-8")).hexdigest()
        if key in self._cache:
            return self._cache[key]

        headers = {"Content-Type": "application/json"}
        if self.key:
            headers["Authorization"] = f"Bearer {self.key}"

        url = f"{self.base}/chat/completions"
        resp = requests.post(url, headers=headers, json=payload, timeout=self.timeout)
        resp.raise_for_status()
        data = resp.json()
        text = data["choices"][0]["message"]["content"]
        self._cache[key] = text
        self._save_cache_item(key, text)
        return text


def _extract_json(text: str) -> Dict[str, Any]:
    text = text.strip()
    if text.startswith("{") and text.endswith("}"):
        return json.loads(text)
    match = re.search(r"\{.*\}", text, re.DOTALL)
    if match:
        return json.loads(match.group(0))
    raise ValueError("LLM did not return JSON")


def _truncate_contexts(contexts: List[str], max_chars: int) -> List[str]:
    out = []
    used = 0
    for c in contexts:
        if not c:
            continue
        if used >= max_chars:
            break
        remain = max_chars - used
        piece = c[:remain]
        out.append(piece)
        used += len(piece)
    return out


SYSTEM_JSON = "You are a strict grader. Return ONLY valid JSON."


def eval_faithfulness(client: LLMClient, question: str, answer: str, contexts: List[str]) -> Optional[float]:
    if not answer or not contexts:
        return None
    ctx = "\n\n".join([f"[Context {i}]\n{c}" for i, c in enumerate(contexts)])
    prompt = (
        "Evaluate whether the answer is fully supported by the provided contexts.\n"
        "If the answer contains any information not present in the contexts, reduce the score.\n"
        "Return JSON: {\"score\": number between 0 and 1, \"unsupported_claims\": [..], \"reason\": \"...\"}.\n\n"
        f"Question:\n{question}\n\n"
        f"Answer:\n{answer}\n\n"
        f"Contexts:\n{ctx}\n"
    )
    raw = client.chat(SYSTEM_JSON, prompt, temperature=0.0, max_tokens=512)
    data = _extract_json(raw)
    return float(data.get("score", 0.0))


def eval_answer_relevancy(client: LLMClient, question: str, answer: str) -> Optional[float]:
    if not answer:
        return None
    prompt = (
        "Evaluate how relevant the answer is to the question.\n"
        "Return JSON: {\"score\": number between 0 and 1, \"reason\": \"...\"}.\n\n"
        f"Question:\n{question}\n\n"
        f"Answer:\n{answer}\n"
    )
    raw = client.chat(SYSTEM_JSON, prompt, temperature=0.0, max_tokens=256)
    data = _extract_json(raw)
    return float(data.get("score", 0.0))


def eval_context_precision(client: LLMClient, question: str, contexts: List[str]) -> Optional[float]:
    if not contexts:
        return None
    ctx = "\n\n".join([f"[{i}] {c}" for i, c in enumerate(contexts)])
    prompt = (
        "Mark which contexts are relevant to answering the question.\n"
        "Return JSON: {\"relevant_indices\": [..], \"reason\": \"...\"}.\n\n"
        f"Question:\n{question}\n\n"
        f"Contexts:\n{ctx}\n"
    )
    raw = client.chat(SYSTEM_JSON, prompt, temperature=0.0, max_tokens=512)
    data = _extract_json(raw)
    relevant = data.get("relevant_indices", [])
    if not isinstance(relevant, list):
        return 0.0
    rel_count = len({i for i in relevant if isinstance(i, int) and 0 <= i < len(contexts)})
    return rel_count / float(len(contexts))


def eval_context_recall(client: LLMClient, question: str, reference_answer: str, contexts: List[str]) -> Optional[float]:
    if not reference_answer or not contexts:
        return None
    ctx = "\n\n".join([f"[Context {i}]\n{c}" for i, c in enumerate(contexts)])
    prompt = (
        "Extract atomic facts from the reference answer, then check which facts are supported by the contexts.\n"
        "Return JSON: {\"facts\": [{\"fact\": \"...\", \"supported\": true/false}], \"score\": number between 0 and 1}.\n\n"
        f"Question:\n{question}\n\n"
        f"Reference Answer:\n{reference_answer}\n\n"
        f"Contexts:\n{ctx}\n"
    )
    raw = client.chat(SYSTEM_JSON, prompt, temperature=0.0, max_tokens=768)
    data = _extract_json(raw)
    if "score" in data:
        try:
            return float(data.get("score", 0.0))
        except Exception:
            pass
    facts = data.get("facts", [])
    if not facts:
        return 0.0
    supported = sum(1 for f in facts if isinstance(f, dict) and f.get("supported") is True)
    return supported / float(len(facts))


def extract_answer(blocks: List[Dict[str, Any]]) -> str:
    if not blocks:
        return ""
    parts = []
    for b in blocks:
        if b.get("type") == "text":
            parts.append(b.get("content", ""))
    return "\n".join(parts).strip()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--report", default="")
    parser.add_argument("--max-cases", type=int, default=0)
    args = parser.parse_args()

    data = load_dataset(args.input)
    if args.max_cases > 0:
        data = data[: args.max_cases]

    client = LLMClient()
    max_ctx = int(os.getenv("RAG_EVAL_MAX_CONTEXT_CHARS", "12000"))

    results = []
    for item in data:
        question = item.get("question", "")
        answer = item.get("answer", "")
        ref = item.get("reference_answer", "")
        contexts = item.get("contexts", []) or []
        contexts = _truncate_contexts(contexts, max_ctx)

        case = {"id": item.get("id", "")}
        case["faithfulness"] = eval_faithfulness(client, question, answer, contexts)
        case["answer_relevancy"] = eval_answer_relevancy(client, question, answer)
        case["context_precision"] = eval_context_precision(client, question, contexts)
        case["context_recall"] = eval_context_recall(client, question, ref, contexts)
        results.append(case)
        time.sleep(0.2)

    def mean(key: str) -> Optional[float]:
        vals = [r[key] for r in results if r.get(key) is not None]
        if not vals:
            return None
        return sum(vals) / len(vals)

    payload = {
        "summary": {
            "faithfulness": mean("faithfulness"),
            "answer_relevancy": mean("answer_relevancy"),
            "context_precision": mean("context_precision"),
            "context_recall": mean("context_recall"),
        },
        "cases": results,
    }

    save_json(args.output, payload)
    if args.report:
        save_md(args.report, payload)


if __name__ == "__main__":
    main()
