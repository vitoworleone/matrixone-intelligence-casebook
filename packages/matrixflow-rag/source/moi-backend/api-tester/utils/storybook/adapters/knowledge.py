"""Knowledge source 建模与索引任务适配器。"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import moi_product_sdk as product_sdk

from utils.data_factory import random_name
from utils.polling import assert_poll_success, poll_until
from utils.storybook.adapters.base import ActionAdapter, required_list, required_string

_SUCCESS_STATUSES = {"succeeded", "success", "completed"}
_FAILURE_STATUSES = {"failed", "cancelled", "canceled"}


@dataclass
class _KnowledgeSnapshot:
    sources: Any
    jobs: Any


class KnowledgeAdapter(ActionAdapter):
    """通过 Product SDK 创建知识库并等待真实 source jobs。"""

    actions = {
        "create_from_assets": "create_from_assets",
        "append_from_assets": "append_from_assets",
        "wait_until_indexed": "wait_until_indexed",
    }

    def _sources_from_assets(
        self,
        context: Any,
        assets: list[Any],
        model_id: str = "",
    ) -> list[product_sdk.SemanticModelSourceInput]:
        service = context.workspace.knowledge()
        sources = []
        for raw_asset in assets:
            asset = required_string({"asset": raw_asset}, "asset")
            path = context.asset_path(asset)
            content = path.read_bytes()
            if not content:
                raise AssertionError(f"Knowledge 测试资产不能为空: {path}")
            if model_id:
                uploaded = service.upload_model_local_file(model_id, path.name, content)
            else:
                uploaded = service.upload_local_file(path.name, content)
            file_id = str(uploaded.file_id or "")
            if not file_id:
                raise AssertionError(f"Knowledge 本地文件上传未返回 file_id: {uploaded}")
            sources.append(
                product_sdk.SemanticModelSourceInput(
                    "local_file",
                    file_name=path.name,
                    file_id=file_id,
                    upload_kind="unstructured",
                )
            )
        return sources

    def _image_index_create_options(self, params: dict[str, Any]) -> list[Any]:
        """Build create_with_sources options for optional image index enablement.

        Supports:
        - image_index_enabled: bool — pass through to Product SDK
        - require_image_index: true — force image_index_enabled=true

        When image index is required or image_index_enabled is explicitly set
        (true or false), missing SDK support fails closed instead of omitting
        the field and relying on backend defaults.

        Call this before uploading assets so invalid config fails without
        leaving persistent catalog files.
        """
        has_require = "require_image_index" in params
        require_image_index = False
        if has_require:
            require_raw = params.get("require_image_index")
            if not isinstance(require_raw, bool):
                raise AssertionError(
                    "knowledge.create_from_assets require_image_index 必须是 bool, "
                    f"got {type(require_raw).__name__}: {require_raw!r}"
                )
            require_image_index = require_raw

        has_explicit = "image_index_enabled" in params
        # Only a fully absent image-index intent may omit the option.
        if not require_image_index and not has_explicit:
            return []

        if has_explicit:
            enabled = params.get("image_index_enabled")
            if not isinstance(enabled, bool):
                raise AssertionError(
                    "knowledge.create_from_assets image_index_enabled 必须是 bool, "
                    f"got {type(enabled).__name__}: {enabled!r}"
                )
        else:
            enabled = True

        if require_image_index and not enabled:
            raise AssertionError(
                "knowledge.create_from_assets require_image_index=true 时 "
                "image_index_enabled 不能为 false"
            )

        option_fn = getattr(
            product_sdk,
            "with_semantic_model_with_sources_image_index_enabled",
            None,
        )
        if option_fn is None:
            # has_explicit or require_image_index is always true here; never
            # silently convert an explicit false into an omitted request field.
            raise RuntimeError(
                "Product SDK 未暴露 with_semantic_model_with_sources_image_index_enabled；"
                "无法透传 image_index_enabled，禁止降级为省略字段/后端默认"
            )

        return [option_fn(enabled)]

    def create_from_assets(
        self,
        context: Any,
        params: dict[str, Any],
        expect: dict[str, Any],
    ) -> dict[str, Any]:
        assets = required_list(params, "assets")
        # Validate image-index options before upload so invalid config does not
        # leave persistent catalog files that Storybook cleanup cannot discover.
        create_opts = self._image_index_create_options(params)
        sources = self._sources_from_assets(context, assets)
        name = random_name(str(params.get("name_prefix") or "storybook_kb"))
        service = context.workspace.knowledge()
        try:
            created = service.create_with_sources(name, sources, *create_opts)
        except Exception as original:
            try:
                listed = service.list(product_sdk.with_limit(200))
            except Exception as discovery_error:
                raise RuntimeError(
                    f"{original}; Knowledge 部分创建资源发现失败: {discovery_error}"
                ) from original
            for item in listed.items:
                if item.name != name:
                    continue
                model_id = str(item.id or "")
                if model_id:
                    context.register_cleanup(
                        "semantic_model_partial",
                        model_id,
                        context.workspace.semantic_model(model_id).delete,
                    )
            raise
        model_id = str(created.model.id or "")
        if not model_id:
            raise AssertionError(f"Knowledge 创建未返回 model.id: {created}")
        model = context.workspace.semantic_model(model_id)
        context.register_cleanup("semantic_model", model_id, model.delete)
        detail = model.info()
        if str(detail.id) != model_id or detail.name != name:
            raise AssertionError(f"Knowledge 写后读不一致: created={created}, detail={detail}")
        expected_sources = int(expect.get("source_count") or len(assets))
        if len(created.sources) != expected_sources:
            raise AssertionError(
                f"Knowledge 创建响应 source 数量不正确: expected={expected_sources}, "
                f"actual={len(created.sources)}"
            )
        return {
            "model_id": model_id,
            "name": name,
            "source_ids": [source.source_id or source.row_id for source in created.sources],
            "job_ids": [job.job_id for job in created.jobs],
        }

    def append_from_assets(
        self,
        context: Any,
        params: dict[str, Any],
        expect: dict[str, Any],
    ) -> dict[str, Any]:
        model_id = required_string(params, "model_id")
        assets = required_list(params, "assets")
        sources = self._sources_from_assets(context, assets, model_id)
        appended = context.workspace.knowledge().append_sources(model_id, sources)
        expected_sources = int(expect.get("source_count") or len(assets))
        if len(appended.sources) < expected_sources:
            raise AssertionError(
                f"Knowledge 追加响应 source 数量不正确: expected_at_least={expected_sources}, "
                f"actual={len(appended.sources)}"
            )
        return {
            "model_id": model_id,
            "source_ids": [source.source_id or source.row_id for source in appended.sources],
            "job_ids": [job.job_id for job in appended.jobs],
        }

    def wait_until_indexed(
        self,
        context: Any,
        params: dict[str, Any],
        expect: dict[str, Any],
    ) -> dict[str, Any]:
        model_id = required_string(params, "model_id")
        minimum_sources = int(expect.get("source_count") or 1)
        minimum_jobs = int(expect.get("job_count") or minimum_sources)
        service = context.workspace.knowledge()

        def snapshot() -> _KnowledgeSnapshot:
            return _KnowledgeSnapshot(
                sources=service.list_sources(model_id),
                jobs=service.list_source_jobs(model_id),
            )

        def statuses(value: _KnowledgeSnapshot) -> list[str]:
            return [str(item.job_status or "").lower() for item in value.jobs.items]

        polled = poll_until(
            snapshot,
            lambda value: (
                len(value.sources.items) >= minimum_sources
                and len(value.jobs.items) >= minimum_jobs
                and all(status in _SUCCESS_STATUSES for status in statuses(value))
            ),
            timeout_s=float(params.get("timeout_s") or 300),
            interval_s=3,
            is_terminal_failure=lambda value: any(
                status in _FAILURE_STATUSES for status in statuses(value)
            ),
            snapshot_fn=lambda value: (
                f"model_id={model_id}, sources={len(value.sources.items)}, "
                f"jobs={[(item.job_id, item.job_status, item.error) for item in value.jobs.items]}"
            ),
        )
        assert_poll_success(polled, reason=f"Knowledge source jobs {model_id}")
        value = polled.last_value
        return {
            "model_id": model_id,
            "source_ids": [item.source_id or item.row_id for item in value.sources.items],
            "jobs": [
                {
                    "job_id": item.job_id,
                    "source_id": item.source_id,
                    "status": item.job_status,
                    "workflow_execution_id": getattr(item, "workflow_execution_id", ""),
                    "error": item.error,
                }
                for item in value.jobs.items
            ],
        }
