# Offline RAG Evaluation (RAGAS-style)

离线评估工具（RAGAS 风格），基于 LLM 评估以下 4 个维度：
- Faithfulness（忠实度）
- Answer Relevancy（答案相关性）
- Context Precision（上下文精确度）
- Context Recall（上下文召回率）

## 输入数据格式
JSON 或 JSONL，每条样本包含：
```json
{
  "id": "case_id",
  "question": "...",
  "answer": "...",
  "reference_answer": "...",
  "contexts": ["chunk1", "chunk2"],
  "metadata": {"file_names": ["a.pdf", "b.pdf"]}
}
```

`reference_answer` 为空时，会跳过 Context Recall 计算。

## 快速开始
```bash
cd evals/ragas_offline
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 设置 LLM（OpenAI 兼容）
export RAG_EVAL_API_BASE=http://localhost:8080/v1
export RAG_EVAL_API_KEY=YOUR_KEY
export RAG_EVAL_MODEL=gpt-4o-mini

# 运行评估
python3 eval.py \
  --input /path/to/dataset.jsonl \
  --output /path/to/report.json \
  --report /path/to/report.md
```

## 从 Explore benchmark 事件生成数据集
```bash
python3 extract_from_bench.py \
  --cases-dir ../../tests/benchmarks/explor/cases_pdf \
  --events-dir ../../tests/benchmarks/explor/reports/explore_bench_20260309_160732_events \
  --output /tmp/explore_eval_dataset.jsonl
```

如果在 case YAML 里补充 `reference_answer` 字段，评估会自动使用。

## 环境变量
- `RAG_EVAL_API_BASE`：OpenAI 兼容地址（默认 `http://localhost:8080/v1`）
- `RAG_EVAL_API_KEY`：API Key（可空）
- `RAG_EVAL_MODEL`：模型名（默认 `gpt-4o-mini`）
- `RAG_EVAL_MAX_CONTEXT_CHARS`：上下文最大字符数（默认 12000）
- `RAG_EVAL_CACHE_PATH`：缓存文件路径（默认 `.cache.jsonl`）

## 输出
- `report.json`：每条样本的分数 + 汇总均值
- `report.md`：可读性报告（表格）
