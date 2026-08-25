"""知识库集成场景共用的严格前置检查。"""
from __future__ import annotations

import moi_product_sdk as product_sdk
import pytest

from client.errors import CODE_OK, HTTP_200


def require_knowledge_embedding_models(
    workspace: product_sdk.WorkspaceHandle,
) -> None:
    """确认后端创建知识库时固定依赖的文本与图片 embedding 模型均可用。"""

    result = workspace._client._product_result(  # noqa: SLF001 - typed 响应可能落后于后端扩展字段
        "GET",
        "/embedding/models",
        workspace_id=workspace.id,
    )
    body = result.body if isinstance(result.body, dict) else {}
    assert result.status == HTTP_200 and body.get("code") == CODE_OK, (
        "读取知识库 embedding 模型应成功: "
        f"status={result.status}, body={body!r}"
    )
    data = body.get("data")
    assert isinstance(data, dict), f"embedding models 应返回对象 data: {body!r}"
    models = data.get("models")
    assert isinstance(models, list), f"embedding models 应为列表: {data!r}"
    names = {
        str(item.get("model") or "")
        for item in models
        if isinstance(item, dict) and item.get("model")
    }
    missing = {"bge-m3", "efficientnet-b3"} - names
    if missing:
        pytest.skip(
            "环境前置不满足：workspace 缺少知识库固定 embedding model: "
            f"missing={sorted(missing)}, available={sorted(names)}"
        )
