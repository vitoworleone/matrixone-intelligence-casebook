"""图片 embedding 环境与产品发现契约回归。"""
from __future__ import annotations

import allure
import moi_product_sdk as product_sdk
import pytest

from utils.github_issues import github_issues


@allure.feature("知识库模块")
@allure.story("图片 Embedding 模型服务")
@allure.title("环境暴露可路由的 efficientnet-b3 多模态 embedding 模型")
@pytest.mark.knowledge
@pytest.mark.integration
@github_issues(12507)
def test_image_embedding_model_is_registered_with_multimodal_routing_metadata(
    moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
):
    """模型缺失或模态错误必须失败，不能把图片模型当文本模型或静默跳过。"""

    workspace = moi_tester_product_sdk_workspace
    models = list(workspace.models().list_embedding_models().models)
    matches = [
        item
        for item in models
        if item.model == "efficientnet-b3"
    ]
    assert len(matches) == 1, (
        "环境必须唯一注册 efficientnet-b3 图片 embedding 模型: "
        f"matches={matches!r}, models={models!r}"
    )
    model = matches[0]
    assert model.model_type == "embedding_multimodal", (
        f"efficientnet-b3 必须声明多模态 embedding 类型: {model!r}"
    )
    assert model.backend_id != 0, (
        f"efficientnet-b3 缺少可路由 backend_id: {model!r}"
    )
    assert model.backend_name.strip(), (
        f"efficientnet-b3 缺少 backend_name: {model!r}"
    )
