FROM ccr.ccs.tencentyun.com/mocloud/pytorch:2.2.2-cuda12.1-cudnn8-runtime

WORKDIR /app
ENV PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    TRANSFORMERS_CACHE=/app/.cache/huggingface \
    HF_HOME=/app/.cache/huggingface

COPY rerank/requirements-base.txt /app/requirements-base.txt
RUN pip install --no-cache-dir -r requirements-base.txt

COPY rerank /app/rerank

EXPOSE 8010
CMD ["uvicorn", "rerank.app:app", "--host", "0.0.0.0", "--port", "8010"]
