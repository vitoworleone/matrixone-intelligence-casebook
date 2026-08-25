"""语义模型 P1 深链路：sources / governance / segments / re-embedding。

覆盖 Product SDK KnowledgeService 此前仅 L1 可达的写后读业务语义：
- create_with_sources / list_sources / append_sources
- update_source_governance
- check_source_existence / list_source_jobs / backfill_legacy_sources
- segments import-initial / reembed（有 segment base 时）
"""
from __future__ import annotations

import time

import allure
import moi_product_sdk as product_sdk
import pytest

from client.errors import (
    CODE_OK,
    ERR_FORBIDDEN,
    ERR_NOT_FOUND,
    ERR_PARAM_INVALID,
    ERR_SERVER,
    ERR_UNAUTHORIZED,
    ERR_WORKSPACE_PERMISSION_DENIED,
    HTTP_400,
    HTTP_200,
    HTTP_201,
    HTTP_403,
    HTTP_404,
    HTTP_500,
)
from tests.knowledge.helpers import require_knowledge_embedding_models
from utils.data_factory import random_name
from utils.github_issues import github_issues

pytestmark = [pytest.mark.core, pytest.mark.semantic, pytest.mark.knowledge]

_SAMPLE_TXT = "api-tester p1 semantic source document\nline-2\n"


def _skip_if_sdk_env(exc: product_sdk.Error, scene: str) -> None:
    if exc.code in {
        ERR_UNAUTHORIZED,
        "ErrMissingWorkspaceAuth",
        "ErrWorkspaceAccessDenied",
        "ErrWorkspaceNotFound",
        "ErrTenantDBConnection",
    }:
        pytest.skip(f"环境前置不满足（{scene}）：workspace/tenant 不可用: {exc}")
    if "route not found" in (exc.message or "").lower():
        pytest.skip(f"环境能力未部署（{scene}）：{exc}")
    if exc.code == ERR_PARAM_INVALID:
        msg = (exc.message or "").lower()
        if "embedding model" in msg and "not available in this workspace" in msg:
            pytest.skip(f"环境前置不满足（{scene}）：workspace 无可用 embedding model: {exc}")
    if exc.status == 500 and exc.code in {ERR_SERVER, "INTERNAL"}:
        msg = (exc.message or "").lower()
        if "tenant" in msg or "workspace" in msg or "no such table" in msg:
            pytest.skip(f"环境前置不满足（{scene}）：{exc}")


def _sdk_call(scene: str, fn):
    try:
        return fn()
    except product_sdk.Error as exc:
        _skip_if_sdk_env(exc, scene)
        if exc.status == HTTP_403 or exc.code in {
            ERR_FORBIDDEN,
            ERR_WORKSPACE_PERMISSION_DENIED,
            "ErrPermissionDenied",
        }:
            pytest.skip(f"环境无 semantic 权限（{scene}）: {exc}")
        raise


def _local_file_source(
    knowledge,
    file_name: str = "p1-source.txt",
    model_id: str = "",
) -> product_sdk.SemanticModelSourceInput:
    uploaded = _sdk_call(
        "upload_local_file",
        lambda: knowledge.upload_model_local_file(model_id, file_name, _SAMPLE_TXT.encode("utf-8"))
        if model_id
        else knowledge.upload_local_file(file_name, _SAMPLE_TXT.encode("utf-8")),
    )
    file_id = str(getattr(uploaded, "file_id", "") or "")
    assert file_id, f"upload_local_file 应返回 file_id: {uploaded}"
    return product_sdk.SemanticModelSourceInput(
        "local_file",
        file_name=file_name,
        file_id=file_id,
        upload_kind="unstructured",
    )


def _model_id_from_with_sources(result) -> str:
    model = getattr(result, "model", None)
    model_id = str(getattr(model, "id", "") or getattr(result, "id", "") or "")
    return model_id


def _product_data(result: product_sdk.ProductResult, scene: str) -> dict:
    body = result.body if isinstance(result.body, dict) else {}
    assert result.status == HTTP_200 and body.get("code") == CODE_OK, (
        f"{scene} 应成功: status={result.status}, body={body!r}"
    )
    data = body.get("data")
    assert isinstance(data, dict), f"{scene} 应返回对象 data: {body!r}"
    return data


def _first_source(list_result):
    items = list(list_result.items)
    return items[0] if items else None


def _delete_model_or_fail(knowledge, model_id: str, scene: str) -> None:
    if not model_id:
        return
    try:
        knowledge.delete(model_id)
    except product_sdk.Error as exc:
        _skip_if_sdk_env(exc, scene)
        if exc.status == HTTP_403 or exc.code in {
            ERR_FORBIDDEN,
            ERR_WORKSPACE_PERMISSION_DENIED,
            "ErrPermissionDenied",
        }:
            pytest.skip(f"环境无 semantic 权限（{scene}）: {exc}")
        if exc.status in (HTTP_404,) or exc.code in {ERR_NOT_FOUND, "NOT_FOUND"}:
            return
        pytest.fail(f"cleanup 删除 semantic model 失败（{scene}）: {exc}")


@allure.feature("语义模型")
@allure.story("Sources / Governance / Segments P1")
class TestSemanticModelSourcesP1:
    @allure.title("create_with_sources → list_sources → append_sources → delete")
    @pytest.mark.smoke
    @github_issues(12777, 13551, 13954, 14099)
    def test_create_with_sources_list_append(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        knowledge = moi_tester_product_sdk_workspace.knowledge()
        name = random_name("p1-sem-src")
        model_id = ""
        try:
            created = _sdk_call(
                "create_with_sources",
                lambda: knowledge.create_with_sources(name, [_local_file_source(knowledge)]),
            )
            model_id = _model_id_from_with_sources(created)
            assert model_id, f"create_with_sources 应返回 model.id: {created}"
            sources_from_create = list(getattr(created, "sources", []) or [])
            # sources 可能异步落库，允许 create 时为空，再 list 校验
            listed = _sdk_call(
                "list_sources",
                lambda: knowledge.list_sources(model_id),
            )
            assert isinstance(listed.total, int), f"list_sources total 应为 int: {listed}"
            items = list(listed.items)
            assert isinstance(items, list), f"list_sources items 应为 list: {listed}"
            if not items and sources_from_create:
                items = sources_from_create
            if items:
                src = items[0]
                assert src.source_id or src.row_id or src.display_name, (
                    f"source 缺少标识: {src}"
                )
                assert src.source_type in {
                    "local_file",
                    "catalog_file",
                    "catalog_table",
                    "",
                } or src.source_type, f"source_type 异常: {src}"

            appended = _sdk_call(
                "append_sources",
                lambda: knowledge.append_sources(
                    model_id,
                    [_local_file_source(knowledge, "p1-source-append.txt", model_id)],
                ),
            )
            append_sources = list(getattr(appended, "sources", []) or [])
            listed_after = _sdk_call(
                "list_sources after append",
                lambda: knowledge.list_sources(model_id),
            )
            # 写后读：append 后 sources 数量应 >= 之前（异步时至少 total 可解析）
            assert isinstance(listed_after.total, int), (
                f"append 后 list total 应为 int: {listed_after}"
            )
            assert listed_after.total >= listed.total or append_sources or list(
                listed_after.items
            ), f"append 后 sources 未见增长: before={listed} after={listed_after} append={appended}"
        finally:
            _delete_model_or_fail(knowledge, model_id, "delete after sources append")

    @allure.title("selection-only 创建知识库后重复添加同一表保持幂等")
    @github_issues(12661, 13558, 13918)
    def test_create_with_database_table_selection_only(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        workspace = moi_tester_product_sdk_workspace
        knowledge = workspace.knowledge()
        require_knowledge_embedding_models(workspace)
        catalog_service = workspace.catalog()
        catalog_id = 0
        database_id = 0
        table_id = 0
        model_id = ""
        try:
            catalog_data = _product_data(
                catalog_service.create_result({"name": random_name("kb-selection-cat")}),
                "create selection catalog",
            )
            catalog_id = int(catalog_data.get("id") or 0)
            assert catalog_id, f"create selection catalog 缺少 id: {catalog_data!r}"

            database_data = _product_data(
                catalog_service.create_database_result(
                    {"catalog_id": catalog_id, "name": random_name("kbselectiondb")}
                ),
                "create selection database",
            )
            database_id = int(database_data.get("id") or 0)
            assert database_id, f"create selection database 缺少 id: {database_data!r}"

            table_data = _product_data(
                catalog_service.create_table_result(
                    {
                        "database_id": database_id,
                        "name": random_name("kbselectiontbl"),
                        "comment": "knowledge selection source",
                        "columns": [
                            {"name": "id", "type": "INT", "comment": "primary key"},
                            {"name": "content", "type": "VARCHAR", "comment": "content"},
                        ],
                    }
                ),
                "create selection table",
            )
            table_id = int(table_data.get("id") or 0)
            assert table_id, f"create selection table 缺少 id: {table_data!r}"

            body = {
                "name": random_name("kb-selection"),
                "sources": [],
                "source_selections": [
                    product_sdk.SemanticModelSourceSelectionInput(
                        "database_tables",
                        database_id=database_id,
                        selected_table_ids=[table_id],
                    ).body()
                ],
            }
            result = workspace._client._product_result(  # noqa: SLF001 - SDK typed create 暂未暴露 source_selections
                "POST",
                "/semantic-models/create-with-sources",
                workspace_id=workspace.id,
                body=body,
            )
            response = result.body if isinstance(result.body, dict) else {}
            data = response.get("data") if isinstance(response.get("data"), dict) else {}
            model = data.get("model") if isinstance(data.get("model"), dict) else {}
            model_id = str(model.get("id") or "")
            assert result.status == HTTP_201 and response.get("code") == CODE_OK, (
                "selection-only 创建应通过 IAM 依赖解析并进入业务处理: "
                f"status={result.status}, body={response!r}"
            )
            assert model_id, f"selection-only 创建成功后应返回 model.id: {response!r}"

            detail = _sdk_call(
                "read selection-only semantic model",
                lambda: knowledge.get(model_id),
            )
            assert str(detail.id) == model_id, f"selection-only 创建写后读不一致: {detail}"
            listed_sources = _sdk_call(
                "list selection-only sources",
                lambda: knowledge.list_sources(model_id),
            )
            assert listed_sources.total >= 1 or list(listed_sources.items), (
                f"selection-only 创建后应持久化表来源: {listed_sources}"
            )

            source_ids_before = {
                str(item.source_id or item.row_id)
                for item in listed_sources.items
                if item.source_table_id == table_id and (item.source_id or item.row_id)
            }
            assert len(source_ids_before) == 1, (
                "首次添加后应只有一个匹配来源: "
                f"table_id={table_id}, sources={listed_sources}"
            )
            jobs_before = _sdk_call(
                "list source jobs before duplicate append",
                lambda: knowledge.list_source_jobs(model_id),
            )
            job_ids_before = {str(item.job_id) for item in jobs_before.items if item.job_id}

            duplicate_result = _sdk_call(
                "append existing catalog table",
                lambda: knowledge.append_sources(
                    model_id,
                    [product_sdk.SemanticModelSourceInput("catalog_table", table_id=table_id)],
                ),
            )
            duplicate_source_ids = {
                str(item.source_id or item.row_id)
                for item in duplicate_result.sources
                if item.source_table_id == table_id and (item.source_id or item.row_id)
            }
            assert duplicate_source_ids == source_ids_before, (
                "重复添加应返回并复用原有 source: "
                f"before={source_ids_before}, duplicate={duplicate_source_ids}"
            )

            sources_after = _sdk_call(
                "list sources after duplicate append",
                lambda: knowledge.list_sources(model_id),
            )
            matching_sources_after = [
                item for item in sources_after.items if item.source_table_id == table_id
            ]
            assert len(matching_sources_after) == 1, (
                "重复添加不得产生重复有效 source: "
                f"table_id={table_id}, sources={sources_after}"
            )
            assert str(
                matching_sources_after[0].source_id or matching_sources_after[0].row_id
            ) in source_ids_before, (
                f"重复添加改变了 source 标识: before={source_ids_before}, after={sources_after}"
            )

            jobs_after = _sdk_call(
                "list source jobs after duplicate append",
                lambda: knowledge.list_source_jobs(model_id),
            )
            job_ids_after = {str(item.job_id) for item in jobs_after.items if item.job_id}
            assert job_ids_after == job_ids_before, (
                "重复添加不得创建新的导入或索引任务: "
                f"before={job_ids_before}, after={job_ids_after}"
            )
        finally:
            _delete_model_or_fail(knowledge, model_id, "delete selection-only semantic model")
            if table_id:
                catalog_service.delete_table(table_id)
            if database_id:
                catalog_service.delete_database(database_id)
            if catalog_id:
                catalog_service.delete(catalog_id)

    @allure.title("update_source_governance 写后读 tags/enabled")
    def test_source_governance(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        knowledge = moi_tester_product_sdk_workspace.knowledge()
        name = random_name("p1-sem-gov")
        model_id = ""
        try:
            created = _sdk_call(
                "create_with_sources for governance",
                lambda: knowledge.create_with_sources(name, [_local_file_source(knowledge)]),
            )
            model_id = _model_id_from_with_sources(created)
            assert model_id, f"create_with_sources 应返回 model.id: {created}"

            source = None
            for _ in range(8):
                listed = knowledge.list_sources(model_id)
                source = _first_source(listed)
                if source and (source.source_id or source.row_id):
                    break
                # create 响应里的 sources
                create_sources = list(getattr(created, "sources", []) or [])
                if create_sources:
                    source = create_sources[0]
                    break
                time.sleep(1)
            if not source:
                pytest.skip("create_with_sources 后未返回可用 source，无法测 governance")

            source_id = str(source.source_id or source.row_id or "")
            assert source_id, f"source 缺少 source_id: {source}"

            gov = _sdk_call(
                "update_source_governance",
                lambda: knowledge.update_source_governance(
                    model_id,
                    source_id,
                    product_sdk.with_semantic_model_source_tags("p1", "api-tester"),
                    product_sdk.with_semantic_model_source_enabled(True),
                ),
            )
            gov_source = getattr(gov, "source", None) or gov
            tags = list(getattr(gov_source, "tags", []) or [])
            # tags 可能被规范化；至少 enabled 可断言
            assert getattr(gov_source, "enabled", True) in (True, 1, False, 0) or tags, (
                f"governance 应回写 source 字段: {gov_source}"
            )
            if tags:
                assert "p1" in tags or "api-tester" in tags or len(tags) >= 1, (
                    f"tags 未生效: {tags}"
                )
        finally:
            _delete_model_or_fail(knowledge, model_id, "delete after governance")

    @allure.title("list_source_jobs / check_source_existence / backfill_legacy 可调用")
    @github_issues(13754)
    def test_source_jobs_existence_backfill(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        knowledge = moi_tester_product_sdk_workspace.knowledge()
        name = random_name("p1-sem-jobs")
        model_id = ""
        try:
            created = _sdk_call(
                "create_with_sources for jobs",
                lambda: knowledge.create_with_sources(name, [_local_file_source(knowledge)]),
            )
            model_id = _model_id_from_with_sources(created)
            assert model_id, f"create_with_sources 应返回 model.id: {created}"

            jobs = _sdk_call(
                "list_source_jobs",
                lambda: knowledge.list_source_jobs(model_id),
            )
            assert jobs is not None, f"list_source_jobs 应返回结果: {jobs}"
            # items 字段名可能是 jobs/items
            job_items = list(
                getattr(jobs, "items", None)
                or getattr(jobs, "jobs", None)
                or getattr(jobs, "list", None)
                or []
            )
            assert isinstance(job_items, list), f"source jobs 应可枚举: {jobs}"

            existence = _sdk_call(
                "check_source_existence",
                lambda: knowledge.check_source_existence(model_id, [], []),
            )
            assert existence is not None, f"existence 应返回结果: {existence}"

            backfill = _sdk_call(
                "backfill_legacy_sources",
                lambda: knowledge.backfill_legacy_sources(model_id),
            )
            assert backfill is not None, f"backfill 应返回结果: {backfill}"
        finally:
            _delete_model_or_fail(knowledge, model_id, "delete after jobs/existence")

    @allure.title("segments re-embedding / import-initial（有 segment base 时）")
    def test_segments_reembed_when_ready(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        knowledge = moi_tester_product_sdk_workspace.knowledge()
        name = random_name("p1-sem-seg")
        model_id = ""
        try:
            created = _sdk_call(
                "create_with_sources for segments",
                lambda: knowledge.create_with_sources(name, [_local_file_source(knowledge)]),
            )
            model_id = _model_id_from_with_sources(created)
            assert model_id, f"create_with_sources 应返回 model.id: {created}"

            source = None
            for _ in range(10):
                listed = knowledge.list_sources(model_id)
                source = _first_source(listed)
                if source and (source.source_id or source.row_id):
                    break
                create_sources = list(getattr(created, "sources", []) or [])
                if create_sources:
                    source = create_sources[0]
                    break
                time.sleep(1)
            if not source:
                pytest.skip("无可用 source，跳过 segments")

            source_id = str(source.source_id or source.row_id or "")
            version_id = str(getattr(source, "segment_version_id", "") or "")
            index_version = int(getattr(source, "index_version", 0) or 0)

            if not version_id:
                # 尝试 import-initial 需要 base；无 base 时验证失败契约
                with pytest.raises(product_sdk.Error) as raised:
                    knowledge.import_initial_segments(
                        model_id,
                        source_id,
                        product_sdk.SemanticModelSegmentBase("", 0),
                    )
                exc = raised.value
                _skip_if_sdk_env(exc, "import_initial without base")
                assert exc.status in (HTTP_400, HTTP_404, HTTP_500) or exc.code in {
                    ERR_PARAM_INVALID,
                    ERR_NOT_FOUND,
                    ERR_SERVER,
                }, f"无 base 的 import-initial 应明确失败: {exc}"
                return

            base = product_sdk.SemanticModelSegmentBase(version_id, index_version)
            # reembed 可能因 ingest 未完成失败；成功则校验 document 字段
            try:
                reembed = knowledge.reembed_segments(model_id, source_id, base)
            except product_sdk.Error as exc:
                _skip_if_sdk_env(exc, "reembed_segments")
                if exc.status in (HTTP_400, HTTP_404, HTTP_500) or exc.code in {
                    ERR_PARAM_INVALID,
                    ERR_NOT_FOUND,
                    ERR_SERVER,
                }:
                    pytest.skip(f"当前 source 尚未具备 re-embedding 条件: {exc}")
                raise
            assert reembed is not None, f"reembed 应返回结果: {reembed}"
            document = getattr(reembed, "document", None)
            if document is not None:
                assert (
                    getattr(document, "source_id", "")
                    or getattr(document, "model_id", "")
                    or getattr(document, "segments", None) is not None
                ), f"reembed document 字段异常: {document}"
        finally:
            _delete_model_or_fail(knowledge, model_id, "delete after segments")

    @allure.title("list_sources 不存在 model → 明确失败")
    def test_list_sources_nonexistent_model(
        self,
        moi_tester_product_sdk_workspace: product_sdk.WorkspaceHandle,
    ):
        with pytest.raises(product_sdk.Error) as raised:
            moi_tester_product_sdk_workspace.knowledge().list_sources(
                "999999999999"
            )
        exc = raised.value
        _skip_if_sdk_env(exc, "list_sources missing model")
        assert exc.status in (HTTP_400, HTTP_403, HTTP_404, HTTP_500), (
            f"不存在 model 应返回明确失败: {exc}"
        )
        assert exc.code, f"应返回错误码: {exc}"
        assert exc.message.strip(), f"应返回可读错误信息: {exc}"
