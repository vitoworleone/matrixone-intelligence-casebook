"""知识库 CRUD 用例。"""
import allure
import pytest

import moi_product_sdk as product_sdk

from utils.data_factory import random_name
from utils.github_issues import github_issues

pytestmark = pytest.mark.core


@allure.feature("知识库模块")
@allure.story("CRUD")
@pytest.mark.knowledge
class TestKnowledgeCRUD:

    @allure.title("创建知识库成功，并深度校验记录存在")
    @allure.severity(allure.severity_level.CRITICAL)
    @github_issues(11516)
    def test_create_knowledge_base(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        name = random_name("kb")
        model, data = moi_tester_product_sdk_workspace.create_semantic_model(name)
        assert data.id, "创建响应中缺少 id"

        try:
            listed = moi_tester_product_sdk_workspace.knowledge().list()
            assert any(item.name == name for item in listed.items), f"创建后列表未命中 name={name}"
        finally:
            model.delete()

    @allure.title("创建知识库并删除 → 列表校验不存在")
    @allure.severity(allure.severity_level.CRITICAL)
    @github_issues(11149)
    def test_create_and_delete_knowledge_base(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        name = random_name("kb")
        model, data = moi_tester_product_sdk_workspace.create_semantic_model(name)
        assert data.id, "创建响应中缺少 id"

        listed = moi_tester_product_sdk_workspace.knowledge().list()
        assert any(item.name == name for item in listed.items), f"创建后列表未命中 name={name}"

        model.delete()

        listed_after = moi_tester_product_sdk_workspace.knowledge().list()
        assert not any(str(item.id) == model.id for item in listed_after.items), f"删除后列表仍命中 id={model.id}"

    @allure.title("获取知识库列表 → 返回结构化数据")
    @allure.severity(allure.severity_level.CRITICAL)
    @github_issues(11148)
    def test_list_knowledge_bases(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        data = moi_tester_product_sdk_workspace.knowledge().list()
        assert isinstance(list(data.items), list), f"列表项应为 list，实际: {type(data.items)}"

    @allure.title("知识库名称创建后不可修改，改名返回参数错误且名称不变")
    @allure.severity(allure.severity_level.CRITICAL)
    @github_issues(11149)
    def test_update_knowledge_base_name_immutable(self, moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle):
        name = random_name("kb")
        model, data = moi_tester_product_sdk_workspace.create_semantic_model(name)
        assert data.id, "创建响应中缺少 id"

        try:
            new_name = random_name("kb-upd")
            with pytest.raises(product_sdk.Error) as exc_info:
                model.update(new_name)
            err = exc_info.value
            assert err.code == "ErrParamInvalid", f"改名应返回 ErrParamInvalid，实际: {err!r}"
            detail = model.info()
            assert detail.name == name, f"改名后名称应不变：id={model.id}，期望 {name!r}，实际 {detail.name!r}"
        finally:
            model.delete()
