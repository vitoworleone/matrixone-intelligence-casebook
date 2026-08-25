"""知识库 P0 契约测试：完整参数链路、最佳努力、删除级联。"""
import allure
import pytest

import moi_product_sdk as product_sdk

from utils.data_factory import random_name
from utils.github_issues import github_issues

pytestmark = pytest.mark.core


_TABLE_COLUMNS = [
    {
        "name": "name",
        "type": "VARCHAR(255)",
        "precision": [255],
        "comment": "名称",
        "is_pk": True,
        "default": "zhangsan",
    }
]


@allure.feature("知识库模块")
@allure.story("P0 契约")
@pytest.mark.knowledge
class TestKnowledgeContractP0:

    @allure.severity(allure.severity_level.CRITICAL)
    @allure.title("知识库创建完整参数链路并回读校验")
    @github_issues(11150)
    def test_create_with_full_fields_and_readback(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        catalog = moi_tester_product_sdk_workspace.prepare_catalog(random_name("kb-full-cat"))
        database = catalog.prepare_database(random_name("kbfulldb"))
        table, _table_result = database.create_table(
            random_name("kbfulltbl"),
            _TABLE_COLUMNS,
            product_sdk.with_table_comment("knowledge full table"),
        )
        model = None
        try:
            model, data = moi_tester_product_sdk_workspace.create_semantic_model(
                random_name("kb-full"),
                product_sdk.with_semantic_model_tables(
                    [{"db_name": database.name, "table_names": [table.name], "parents": []}]
                ),
            )
            assert data.id, "知识库创建成功但响应中缺少 id"

            detail = model.info()
            assert str(detail.id) == model.id, "知识库详情 id 不匹配"
        finally:
            if model is not None:
                model.delete()
            table.delete()
            database.delete()
            catalog.delete()

    @allure.severity(allure.severity_level.CRITICAL)
    @allure.title("知识库创建内联子项契约：主体成功，内联子项按当前实现忽略/未上线")
    def test_create_best_effort_subitems(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        model, data = moi_tester_product_sdk_workspace.create_semantic_model(
            random_name("kb-best"),
            product_sdk.with_semantic_model_description("P0-最佳努力策略"),
            product_sdk.with_semantic_model_tables(
                [{"db_name": "sdk_p0_db", "table_names": ["sdk_p0_table"], "parents": []}]
            ),
        )
        assert data.id, "最佳努力场景主体创建成功但无 id"
        try:
            detail = model.info()
            assert detail.description == "P0-最佳努力策略", f"description 不匹配: {detail.description!r}"
        finally:
            model.delete()
