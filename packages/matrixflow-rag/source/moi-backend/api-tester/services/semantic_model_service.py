"""语义模型独立接口封装。"""
from __future__ import annotations

import allure
import requests

from client import apis
from client.base_test import BaseTest
from client.errors import CODE_OK, HTTP_200, HTTP_201
from client.http_client import MoiHttpClient


class SemanticModelService:
    def __init__(self, client: MoiHttpClient, base_test: BaseTest) -> None:
        self._client = client
        self._base_test = base_test

    @allure.step("获取语义模型列表")
    def list_models(self, cursor=None, limit=None) -> requests.Response:
        params = {}
        if cursor is not None:
            params["cursor"] = cursor
        if limit is not None:
            params["limit"] = limit
        return self._client.get(apis.SEMANTIC_MODELS_LIST, params=params)

    @staticmethod
    def _extract_table_rows(data):
        """兼容 table 接口的多种 data 结构，提取表行。"""
        if isinstance(data, dict):
            rows = data.get("list") or data.get("tables") or data.get("items")
            return rows if isinstance(rows, list) else []
        if isinstance(data, list):
            return data
        return []

    @staticmethod
    def _normalize_tables_payload(rows):
        """将 table 行结构归一化为 semantic-model 需要的 tables 结构。"""
        buckets = {}
        for row in rows or []:
            if not isinstance(row, dict):
                continue
            db_name = row.get("db_name") or row.get("database_name") or row.get("database")
            table_name = row.get("table_name") or row.get("name")
            if not db_name or not table_name:
                continue
            key = str(db_name)
            buckets.setdefault(key, set()).add(str(table_name))

        tables = []
        for db_name, table_names in buckets.items():
            tables.append(
                {
                    "db_name": db_name,
                    "table_names": sorted(table_names),
                    "parents": [],
                }
            )
        return tables

    def _probe_tables_from_database(self, database_id):
        """从 /table/database/tables 读取数据库下表信息。"""
        if database_id is None:
            return []
        db_id_value = database_id
        if isinstance(database_id, str) and database_id.isdigit():
            db_id_value = int(database_id)
        resp = self._client.post(apis.TABLE_DATABASE_TABLES, json={"database_id": db_id_value})
        if resp.status_code not in (HTTP_200, HTTP_201):
            return []
        body = resp.json() if resp.content else {}
        if body.get("code") != CODE_OK:
            return []
        return self._normalize_tables_payload(self._extract_table_rows(body.get("data")))

    def _probe_tables_from_overview(self):
        """从 /table/overview 获取可用表，作为 create 的兜底前置。"""
        resp = self._client.post(apis.TABLE_OVERVIEW, json={})
        if resp.status_code not in (HTTP_200, HTTP_201):
            return []
        body = resp.json() if resp.content else {}
        if body.get("code") != CODE_OK:
            return []
        rows = self._extract_table_rows(body.get("data"))
        tables = self._normalize_tables_payload(rows)
        if tables:
            return [tables[0]]
        return []

    def _ensure_tables_payload(self, *, tables=None, database_id=None):
        """后端契约要求 tables 非空：优先使用调用方传入，其次尝试自动探针。"""
        if isinstance(tables, list) and tables:
            return tables
        if isinstance(tables, dict) and tables:
            return [tables]

        inferred = self._probe_tables_from_database(database_id)
        if inferred:
            return inferred

        inferred = self._probe_tables_from_overview()
        if inferred:
            return inferred

        # 最后一层兜底：保证请求契约完整，避免因为旧调用方式导致硬失败。
        suffix = str(database_id or "default")
        return [
            {
                "db_name": f"auto_db_{suffix}",
                "table_names": [f"auto_table_{suffix}"],
                "parents": [],
            }
        ]

    @allure.step("创建语义模型")
    def create_model(
        self,
        name: str,
        description=None,
        database_id=None,
        tables=None,
        files=None,
        **kwargs,
    ) -> requests.Response:
        payload = {"name": name}
        if description is not None:
            payload["description"] = description
        payload["tables"] = self._ensure_tables_payload(tables=tables, database_id=database_id)
        if files is not None:
            payload["files"] = files
        if kwargs:
            payload.update(kwargs)
        return self._client.post(apis.SEMANTIC_MODELS_CREATE, json=payload)

    @allure.step("获取语义模型详情")
    def get_model(self, model_id: str) -> requests.Response:
        return self._client.get(apis.SEMANTIC_MODELS_GET.format(id=model_id))

    @allure.step("更新语义模型")
    def update_model(self, model_id: str, **kwargs) -> requests.Response:
        payload = dict(kwargs)
        if not payload.get("tables"):
            detail_resp = self.get_model(model_id)
            if detail_resp.status_code in (HTTP_200, HTTP_201):
                detail_body = detail_resp.json() if detail_resp.content else {}
                if detail_body.get("code") == CODE_OK:
                    detail = detail_body.get("data") or {}
                    existing_tables = detail.get("tables")
                    if isinstance(existing_tables, list) and existing_tables:
                        payload["tables"] = existing_tables

        if not payload.get("tables"):
            database_id = payload.get("database_id")
            payload["tables"] = self._ensure_tables_payload(database_id=database_id)

        # 源码契约没有 database_id 字段，避免发送未知字段干扰后端行为。
        payload.pop("database_id", None)
        return self._client.put(apis.SEMANTIC_MODELS_UPDATE.format(id=model_id), json=payload)

    @allure.step("删除语义模型")
    def delete_model(self, model_id: str) -> requests.Response:
        return self._client.delete(apis.SEMANTIC_MODELS_DELETE.format(id=model_id))

    @allure.step("获取语义模型条目列表")
    def list_entries(self, model_id: str, **params) -> requests.Response:
        return self._client.get(
            apis.SEMANTIC_MODELS_ENTRIES_LIST.format(id=model_id),
            params=(params or None),
        )

    @allure.step("创建语义模型条目")
    def create_entry(self, model_id: str, **kwargs) -> requests.Response:
        return self._client.post(apis.SEMANTIC_MODELS_ENTRIES_CREATE.format(id=model_id), json=kwargs)

    @allure.step("更新语义模型条目")
    def update_entry(self, model_id: str, entry_id: str, **kwargs) -> requests.Response:
        return self._client.put(
            apis.SEMANTIC_MODELS_ENTRIES_UPDATE.format(id=model_id, entry_id=entry_id), json=kwargs)

    @allure.step("删除语义模型条目")
    def delete_entry(self, model_id: str, entry_id: str) -> requests.Response:
        return self._client.delete(
            apis.SEMANTIC_MODELS_ENTRIES_DELETE.format(id=model_id, entry_id=entry_id))

    @allure.step("导入语义模型")
    def import_model(self, model_id: str, data=None, **kwargs) -> requests.Response:
        payload = data if data is not None else kwargs
        return self._client.post(apis.SEMANTIC_MODELS_IMPORT.format(id=model_id), json=payload)

    @allure.step("导出语义模型")
    def export_model(self, model_id: str) -> requests.Response:
        return self._client.get(apis.SEMANTIC_MODELS_EXPORT.format(id=model_id))

    @allure.step("校验语义模型")
    def validate_model(self, model_id: str) -> requests.Response:
        return self._client.post(apis.SEMANTIC_MODELS_VALIDATE.format(id=model_id))
