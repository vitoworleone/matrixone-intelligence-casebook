import argparse
import json
import os
from typing import Any, Dict, List

import yaml


def load_yaml(path: str) -> Dict[str, Any]:
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def load_events(path: str) -> List[Dict[str, Any]]:
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def extract_answer(events: List[Dict[str, Any]]) -> str:
    for e in events:
        if e.get("event_type") == "explore.synthesis.done":
            blocks = e.get("data", {}).get("blocks", [])
            parts = []
            for b in blocks:
                if b.get("type") == "text":
                    parts.append(b.get("content", ""))
            return "\n".join(parts).strip()
    return ""


def extract_contexts(events: List[Dict[str, Any]]) -> List[str]:
    # Prefer rag_chunks from synthesis.done (contexts used by the answer)
    for e in events:
        if e.get("event_type") == "explore.synthesis.done":
            chunks = e.get("data", {}).get("rag_chunks", [])
            if chunks:
                return [c.get("content", "") for c in chunks if c.get("content")]
    # Fallback: retriever chunks (may be larger set)
    for e in events:
        if e.get("event_type") == "explore.retriever.chunks":
            chunks = e.get("data", [])
            return [c.get("content", "") for c in chunks if c.get("content")]
    return []


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cases-dir", required=True)
    parser.add_argument("--events-dir", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--run-index", type=int, default=0)
    args = parser.parse_args()

    out_lines = []
    for name in sorted(os.listdir(args.cases_dir)):
        if not name.endswith(".yaml"):
            continue
        case_path = os.path.join(args.cases_dir, name)
        case = load_yaml(case_path)
        case_id = case.get("id")
        if not case_id:
            continue

        event_file = f"{case_id}_run{args.run_index}.json"
        event_path = os.path.join(args.events_dir, event_file)
        if not os.path.exists(event_path):
            continue

        events = load_events(event_path)
        question = case.get("question", "")
        reference = case.get("reference_answer", "")
        answer = extract_answer(events)
        contexts = extract_contexts(events)

        out_lines.append(
            {
                "id": case_id,
                "question": question,
                "answer": answer,
                "reference_answer": reference,
                "contexts": contexts,
                "metadata": {
                    "case_name": case.get("name"),
                    "source_files": [f.get("name") for f in (case.get("data_sources", {}).get("files", []) or [])],
                },
            }
        )

    with open(args.output, "w", encoding="utf-8") as f:
        for item in out_lines:
            f.write(json.dumps(item, ensure_ascii=False) + "\n")


if __name__ == "__main__":
    main()
