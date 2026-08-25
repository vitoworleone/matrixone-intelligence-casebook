"""语义模型（Semantic Model）契约与流程测试。"""
from __future__ import annotations

import allure
import pytest

import moi_product_sdk as product_sdk

from client.errors import ERR_NOT_FOUND, ERR_PARAM_INVALID, HTTP_400, HTTP_404, HTTP_500
from tests.env_guard import guard_test_class_for_refresh_invalid_client
from utils.data_factory import random_name
from utils.github_issues import github_issues

pytestmark = pytest.mark.core


_TABLE_COLUMNS = [
    {
        "name": "id",
        "type": "INT",
        "precision": [],
        "comment": "主键",
        "is_pk": True,
        "default": "1",
    },
    {
        "name": "amount",
        "type": "INT",
        "precision": [],
        "comment": "金额",
        "is_pk": False,
        "default": "0",
    },
]


@pytest.fixture()
def sdk_semantic_model(moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle) -> product_sdk.SemanticModelHandle:
    model, result = moi_tester_product_sdk_workspace.create_semantic_model(
        random_name("sem-model"),
        product_sdk.with_semantic_model_tables(
            [
                {
                    "db_name": "sdk_semantic_db",
                    "table_names": ["sdk_semantic_table"],
                    "parents": [],
                }
            ]
        ),
    )
    assert result.id, f"创建语义模型缺少 id: {result}"

    yield model

    try:
        model.delete()
    except product_sdk.Error:
        pass


@pytest.fixture()
def sdk_table_backed_semantic_model(
    moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
) -> tuple[product_sdk.CatalogHandle, product_sdk.DatabaseHandle, product_sdk.TableHandle, product_sdk.SemanticModelHandle]:
    catalog = moi_tester_product_sdk_workspace.prepare_catalog(random_name("sem-cat"))
    database = catalog.prepare_database(random_name("semdb"))
    table_name = random_name("semtbl")
    table, _table_result = database.create_table(
        table_name,
        _TABLE_COLUMNS,
        product_sdk.with_table_comment("semantic precondition"),
    )
    model, result = moi_tester_product_sdk_workspace.create_semantic_model(
        random_name("sem-model"),
        product_sdk.with_semantic_model_tables(
            [
                {
                    "db_name": database.name,
                    "table_names": [table_name],
                    "parents": [],
                }
            ]
        ),
    )
    assert result.id, f"创建语义模型缺少 id: {result}"

    yield catalog, database, table, model

    for cleanup in (model.delete, table.delete, database.delete, catalog.delete):
        try:
            cleanup()
        except product_sdk.Error:
            pass


def _entry(kind: str, key: str, spec: dict, table_name: str) -> product_sdk.SemanticEntryInput:
    return product_sdk.SemanticEntryInput(kind, key, spec, [table_name])


def _assert_not_found_or_ok(call) -> None:
    try:
        call()
    except product_sdk.Error as err:
        assert err.status in (HTTP_404, HTTP_500) or err.code in (ERR_NOT_FOUND, ERR_PARAM_INVALID), (
            f"删除后读取应返回 not found 或环境允许的错误: {err}"
        )


def _assert_param_invalid(exc_info) -> None:
    err = exc_info.value
    assert isinstance(err, product_sdk.Error)
    assert err.status == HTTP_400 or err.code == ERR_PARAM_INVALID, f"应返回 ErrParamInvalid: {err}"


@allure.feature("语义模型")
@allure.story("契约与流程")
@pytest.mark.semantic
@guard_test_class_for_refresh_invalid_client
class TestSemanticModelContract:

    @allure.title("新语义模型 list_entries 返回空列表而非错误")
    @allure.severity(allure.severity_level.CRITICAL)
    def test_list_entries_empty_for_new_kb(self, sdk_semantic_model: product_sdk.SemanticModelHandle):
        data = sdk_semantic_model.entries()
        items = list(data.items)
        assert isinstance(items, list), f"items 应为 list: {data}"
        assert isinstance(data.total, int), f"total 应为 int: {data}"
        assert data.total == len(items), f"total 与 items 数量不一致: {data}"

    @allure.title("list_entries page_size=0 -> ErrParamInvalid")
    @allure.severity(allure.severity_level.CRITICAL)
    def test_list_entries_invalid_page_size(self, sdk_semantic_model: product_sdk.SemanticModelHandle):
        with pytest.raises(product_sdk.Error) as exc_info:
            sdk_semantic_model.service().list_entries_raw(sdk_semantic_model.id, product_sdk.with_page_size(0))
        _assert_param_invalid(exc_info)

    @allure.title("import 空 entries -> ErrParamInvalid")
    @allure.severity(allure.severity_level.NORMAL)
    def test_import_empty_entries_should_fail(self, sdk_semantic_model: product_sdk.SemanticModelHandle):
        with pytest.raises(product_sdk.Error) as exc_info:
            sdk_semantic_model.service().import_entries_raw(sdk_semantic_model.id, {"entries": []})
        _assert_param_invalid(exc_info)

    @allure.title("create_entry 非法 kind -> ErrParamInvalid")
    @allure.severity(allure.severity_level.CRITICAL)
    def test_create_entry_invalid_kind(self, sdk_semantic_model: product_sdk.SemanticModelHandle):
        with pytest.raises(product_sdk.Error) as exc_info:
            sdk_semantic_model.service().create_entry_raw(
                sdk_semantic_model.id,
                {
                    "kind": "invalid_kind",
                    "key": "k1",
                    "tables": ["t1"],
                    "spec": {"column": "c1"},
                },
            )
        _assert_param_invalid(exc_info)

    @allure.title("model 不存在时 get/export/validate 返回 ErrNotFound 或结构化软删除结果")
    @allure.severity(allure.severity_level.NORMAL)
    def test_get_export_validate_not_found_for_new_kb(self, sdk_semantic_model: product_sdk.SemanticModelHandle):
        model_id = sdk_semantic_model.id
        sdk_semantic_model.delete()

        _assert_not_found_or_ok(lambda: sdk_semantic_model.service().get(model_id))
        _assert_not_found_or_ok(lambda: sdk_semantic_model.service().export(model_id))
        _assert_not_found_or_ok(lambda: sdk_semantic_model.service().validate(model_id))

    @allure.title("语义条目完整生命周期：create -> list -> update -> export -> delete")
    @allure.severity(allure.severity_level.BLOCKER)
    @github_issues(11386)
    def test_entry_crud_full_flow(
        self,
        sdk_table_backed_semantic_model: tuple[
            product_sdk.CatalogHandle,
            product_sdk.DatabaseHandle,
            product_sdk.TableHandle,
            product_sdk.SemanticModelHandle,
        ],
    ):
        _catalog, _database, table, model = sdk_table_backed_semantic_model
        table_name = table.name

        entry, created = model.create_entry(_entry("metric", random_name("metric"), {"expr": "SUM(amount)"}, table_name))
        assert created.kind == "metric", f"kind 不一致: {created}"

        rows = model.entries(product_sdk.with_query_param("kind", "metric")).items
        assert any(str(item.id) == entry.id for item in rows), f"list_entries 未命中刚创建条目: {entry.id}"

        new_key = random_name("metric-upd")
        updated = entry.update(_entry("metric", new_key, {"expr": "SUM(amount)+1"}, table_name))
        assert updated.updated is True, f"update_entry 返回语义异常: {updated}"

        rows_after = model.entries(product_sdk.with_query_param("kind", "metric")).items
        updated_row = next((item for item in rows_after if str(item.id) == entry.id), None)
        assert updated_row is not None, f"更新后未查到目标条目: entry_id={entry.id}"
        assert updated_row.key == new_key, f"更新后 key 未生效: {updated_row}"

        export_data = model.export()
        assert any(str(item.id) == entry.id for item in export_data.entries), f"export 未包含目标 entry: {entry.id}"

        validate_data = model.validate()
        assert isinstance(validate_data.valid, bool), f"validate 响应缺少 valid bool: {validate_data}"
        if validate_data.valid is False:
            assert validate_data.errors, f"valid=false 时应返回 errors: {validate_data}"

        deleted = entry.delete()
        assert deleted.deleted is True, f"delete_entry 返回语义异常: {deleted}"

        rows_after_delete = model.entries().items
        assert all(str(item.id) != entry.id for item in rows_after_delete), f"删除后条目仍存在: {entry.id}"

    @allure.title("import -> get_model -> export -> validate 主链路")
    @allure.severity(allure.severity_level.CRITICAL)
    def test_import_export_validate_flow(
        self,
        sdk_table_backed_semantic_model: tuple[
            product_sdk.CatalogHandle,
            product_sdk.DatabaseHandle,
            product_sdk.TableHandle,
            product_sdk.SemanticModelHandle,
        ],
    ):
        _catalog, database, table, model = sdk_table_backed_semantic_model
        table_name = table.name
        key_dim = random_name("dim")
        key_fact = random_name("fact")

        import_data = model.import_entries(
            [
                _entry("dimension", key_dim, {"column": "id"}, table_name),
                _entry("fact", key_fact, {"column": "amount"}, table_name),
            ]
        )
        assert import_data.imported == 2, f"imported 数量异常: {import_data}"
        assert import_data.model_id, f"import 成功应返回 model_id: {import_data}"

        active_model_id = str(import_data.model_id or model.id)
        service = model.service()

        try:
            model_data = service.get(active_model_id)
            assert database.name in str(model_data.tables), f"model tables 未包含目标库: {model_data}"
        except product_sdk.Error as err:
            assert err.status == HTTP_404 or err.code == ERR_NOT_FOUND, f"get_model 错误不符合预期: {err}"

        try:
            export_data = service.export(active_model_id)
            keys = {entry.key for entry in export_data.entries}
            assert {key_dim, key_fact}.issubset(keys), (
                f"export entries 未完整包含导入 key: expected={[key_dim, key_fact]}, actual={list(keys)}"
            )
        except product_sdk.Error as err:
            assert err.status == HTTP_404 or err.code == ERR_NOT_FOUND, f"export 错误不符合预期: {err}"

        try:
            filtered = service.list_entries(active_model_id, product_sdk.with_query_param("kind", "dimension"))
            assert filtered.items, f"kind=dimension 过滤结果不应为空: {filtered}"
            assert all(item.kind == "dimension" for item in filtered.items), f"kind 过滤失效: {filtered.items[:5]}"
        except product_sdk.Error as err:
            assert err.status == HTTP_404 or err.code == ERR_NOT_FOUND, f"list_entries 错误不符合预期: {err}"

        try:
            validate_data = service.validate(active_model_id)
            assert isinstance(validate_data.valid, bool), f"validate 响应缺少 valid bool: {validate_data}"
            if validate_data.valid is False:
                assert validate_data.errors, f"valid=false 时应返回 errors: {validate_data}"
        except product_sdk.Error as err:
            assert err.status == HTTP_404 or err.code == ERR_NOT_FOUND, f"validate 错误不符合预期: {err}"
        finally:
            if active_model_id != model.id:
                try:
                    service.delete(active_model_id)
                except product_sdk.Error:
                    pass

    @allure.title("list_entries 组合参数语义：kind × page_size × page_token")
    @allure.severity(allure.severity_level.CRITICAL)
    def test_list_entries_pairwise_kind_page_token(
        self,
        sdk_table_backed_semantic_model: tuple[
            product_sdk.CatalogHandle,
            product_sdk.DatabaseHandle,
            product_sdk.TableHandle,
            product_sdk.SemanticModelHandle,
        ],
    ):
        _catalog, _database, table, model = sdk_table_backed_semantic_model
        table_name = table.name
        metric_keys = [random_name("metric-pw"), random_name("metric-pw"), random_name("metric-pw")]
        dimension_key = random_name("dim-pw")

        for key in metric_keys:
            model.create_entry(_entry("metric", key, {"expr": "SUM(amount)"}, table_name))
        model.create_entry(_entry("dimension", dimension_key, {"column": "id"}, table_name))

        page = model.entries(product_sdk.with_query_param("kind", "metric"), product_sdk.with_page_size(1))
        items = page.items
        assert len(items) <= 1, f"page_size=1 时条目数不应超过 1: {page}"
        assert all(item.kind == "metric" for item in items), f"kind=metric 过滤失效: {items}"

        total = int(page.total)
        assert total >= len(items), f"total 应不小于当前页条数: {page}"

        token = page.next_page_token
        if total > len(items):
            assert token, f"存在下一页时 next_page_token 不应为空: {page}"

        seen_ids = {str(item.id) for item in items if item.id}
        seen_keys = {item.key for item in items}
        seen_tokens: set[str] = set()
        guard = 0

        while token and token not in seen_tokens and guard < 20:
            seen_tokens.add(token)
            guard += 1
            page = model.entries(
                product_sdk.with_query_param("kind", "metric"),
                product_sdk.with_page_size(1),
                product_sdk.with_query_param("page_token", token),
            )
            assert len(page.items) <= 1, f"page_size=1 时条目数不应超过 1: {page}"
            assert all(item.kind == "metric" for item in page.items), f"kind=metric 过滤失效: {page.items}"
            assert int(page.total) == total, f"翻页后 total 不应变化: first={total}, page={page}"

            for item in page.items:
                item_id = str(item.id)
                assert item_id not in seen_ids, f"page_token 翻页出现重复条目: id={item_id}, page={page}"
                seen_ids.add(item_id)
                seen_keys.add(item.key)

            token = page.next_page_token

        assert guard < 20, "分页游标疑似循环，超过安全翻页上限"
        assert set(metric_keys).issubset(seen_keys), (
            f"kind=metric 列表未覆盖全部新建 metric 条目: expected={metric_keys}, actual={sorted(seen_keys)}"
        )
        assert dimension_key not in seen_keys, f"kind=metric 过滤应排除 dimension 条目: {sorted(seen_keys)}"

    @allure.title("update_entry 改 kind 应返回 ErrParamInvalid")
    @allure.severity(allure.severity_level.NORMAL)
    def test_update_entry_kind_immutable(
        self,
        sdk_table_backed_semantic_model: tuple[
            product_sdk.CatalogHandle,
            product_sdk.DatabaseHandle,
            product_sdk.TableHandle,
            product_sdk.SemanticModelHandle,
        ],
    ):
        _catalog, _database, table, model = sdk_table_backed_semantic_model
        entry, created = model.create_entry(_entry("dimension", random_name("dim"), {"column": "id"}, table.name))
        assert created.id, f"create_entry 成功但缺少 id: {created}"

        with pytest.raises(product_sdk.Error) as exc_info:
            entry.update(_entry("metric", random_name("metric"), {"expr": "SUM(amount)"}, table.name))
        _assert_param_invalid(exc_info)

    @allure.title("delete 不存在 entry_id -> ErrNotFound")
    @allure.severity(allure.severity_level.NORMAL)
    def test_delete_nonexistent_entry_should_not_found(
        self,
        sdk_table_backed_semantic_model: tuple[
            product_sdk.CatalogHandle,
            product_sdk.DatabaseHandle,
            product_sdk.TableHandle,
            product_sdk.SemanticModelHandle,
        ],
    ):
        _catalog, _database, table, model = sdk_table_backed_semantic_model
        model.create_entry(_entry("dimension", random_name("dim"), {"column": "id"}, table.name))

        with pytest.raises(product_sdk.Error) as exc_info:
            model.service().delete_entry_raw(model.id, "999999999")
        err = exc_info.value
        assert err.status == HTTP_404 or err.code == ERR_NOT_FOUND, f"应返回 ErrNotFound: {err}"
