# Rerank Service (FastAPI)

轻量本地部署的 rerank 服务，支持 Torch/ONNX 后端、FP16、micro-batching、adaptive batch size 与 GPU 利用率调度。

## 功能覆盖
- `max_length` 截断策略：tokenizer 级别截断，避免超长文本拖垮吞吐。
- micro-batching：聚合请求批次，降低 GPU kernel 启动开销。
- FP16：CUDA 下默认 `half()` + autocast。
- ONNX/TensorRT：可切换 ONNX Runtime（支持 TensorRT Provider）。
- batch size 自适应：基于延迟与 backlog 自动增减 batch。
- GPU 利用率调度：可选读取 NVML，按 GPU 利用率调节 batch。

## 快速启动（Torch）
```bash
cd rerank
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# 运行
uvicorn rerank.app:app --host 0.0.0.0 --port 8010
```

请求示例：
```bash
curl -s http://localhost:8010/v1/rerank \
  -H 'content-type: application/json' \
  -d '{
    "query": "How to request USB access?",
    "documents": [
      {"id": "d1", "text": "USB access requires approval..."},
      {"id": "d2", "text": "VPN access policy..."}
    ]
  }' | jq .
```

## Docker 运行
GPU 版（推荐 CUDA 环境）：
```bash
docker build -f rerank/Dockerfile -t moi-rerank:gpu .
docker run --gpus all -p 8010:8010 moi-rerank:gpu
```

CPU 版：
```bash
docker build -f rerank/Dockerfile.cpu -t moi-rerank:cpu .
docker run -p 8010:8010 moi-rerank:cpu
```

## ONNX / TensorRT
1. 导出 ONNX（示例）：
```bash
python -m transformers.onnx \
  --model=BAAI/bge-reranker-base \
  --feature=sequence-classification \
  onnx
```
2. 启动 ONNX 后端：
```bash
export RERANK_BACKEND=onnx
export RERANK_ONNX_PATH=onnx/model.onnx
export RERANK_DEVICE=cuda  # 如果有 GPU
uvicorn rerank.app:app --host 0.0.0.0 --port 8010
```

> 需要 TensorRT 时，请安装带 TensorRT 的 `onnxruntime-gpu` 并确保 NVIDIA 驱动与 CUDA 正常。

## 配置参数（环境变量）
- `RERANK_MODEL`：默认 `BAAI/bge-reranker-base`
- `RERANK_BACKEND`：`torch` | `onnx`
- `RERANK_ONNX_PATH`：ONNX 模型路径（ONNX 后端必需）
- `RERANK_DEVICE`：`auto` | `cuda` | `cpu`
- `RERANK_MAX_LENGTH`：token 最大长度（默认 256）
- `RERANK_NORMALIZE`：是否 sigmoid 归一化（默认 false）
- `RERANK_BATCH_MAX_SIZE` / `RERANK_BATCH_MIN_SIZE` / `RERANK_BATCH_STEP`
- `RERANK_BATCH_MAX_WAIT_MS`：等待聚合批次的最大时间
- `RERANK_BATCH_TARGET_MS`：期望单次 batch 延迟
- `RERANK_GPU_METRICS`：开启 GPU 利用率调度（需 `pynvml`）
- `RERANK_GPU_UTIL_HIGH` / `RERANK_GPU_UTIL_LOW`
- `RERANK_MAX_DOCS` / `RERANK_MAX_TEXT_CHARS`

## 说明
- 请求里的 `max_length` / `normalize` 目前不会覆盖全局配置，避免破坏 micro-batching 的一致性。
- 没有 GPU 时可直接使用 CPU，但吞吐会显著下降。
