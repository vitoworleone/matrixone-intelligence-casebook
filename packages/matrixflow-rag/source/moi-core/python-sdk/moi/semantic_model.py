"""Semantic model service."""

from typing import Any, Dict, List, Optional

from ._http import HTTPClient


class SemanticModelService:
    """语义模型服务。"""

    def __init__(self, http: HTTPClient, workspace_id: str) -> None:
        self._http = http
        self._workspace_id = workspace_id

    def _base(self) -> str:
        return f"/api/v1/workspaces/{self._workspace_id}/semantic-models"

    @staticmethod
    def _validate_structured_tables(tables: Any) -> None:
        """Validate structured tables payload.

        Expected format:
        [
          {"db_name": "sales_db", "table_names": ["orders"], "parents": []}
        ]
        """
        if not isinstance(tables, list):
            raise TypeError("tables must be a list of structured table objects")
        for idx, item in enumerate(tables):
            if not isinstance(item, dict):
                raise TypeError(f"tables[{idx}] must be an object")
            if "table_names" not in item:
                raise ValueError(f"tables[{idx}].table_names is required")
            table_names = item.get("table_names")
            if not isinstance(table_names, list):
                raise TypeError(f"tables[{idx}].table_names must be a list")
            for j, name in enumerate(table_names):
                if not isinstance(name, str) or not name.strip():
                    raise ValueError(
                        f"tables[{idx}].table_names[{j}] must be a non-empty string"
                    )
            db_name = item.get("db_name")
            if db_name is not None and not isinstance(db_name, str):
                raise TypeError(f"tables[{idx}].db_name must be a string")
            parents = item.get("parents")
            if parents is not None:
                if not isinstance(parents, list):
                    raise TypeError(f"tables[{idx}].parents must be a list of strings")
                for j, parent in enumerate(parents):
                    if not isinstance(parent, str):
                        raise TypeError(
                            f"tables[{idx}].parents[{j}] must be a string"
                        )

    def create(
        self,
        name: str,
        tables: List[Dict[str, Any]],
        *,
        description: Optional[str] = None,
        files: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """创建语义模型。"""
        self._validate_structured_tables(tables)
        body: Dict[str, Any] = {"name": name, "tables": tables}
        if description is not None:
            body["description"] = description
        if files is not None:
            body["files"] = files
        return self._http.post(self._base(), body) or {}

    def list(
        self,
        *,
        page_size: Optional[int] = None,
        page_token: Optional[str] = None,
    ) -> Dict[str, Any]:
        """分页列出语义模型。"""
        path = self._base()
        params = []
        if page_size is not None:
            params.append(f"page_size={page_size}")
        if page_token:
            params.append(f"page_token={page_token}")
        if params:
            path = path + "?" + "&".join(params)
        return self._http.get(path) or {}

    def list_tags(
        self,
        *,
        page_size: Optional[int] = None,
        page_token: Optional[str] = None,
    ) -> Dict[str, Any]:
        """列出当前可见语义模型聚合标签。"""
        path = self._base() + "/tags"
        params = []
        if page_size is not None:
            params.append(f"page_size={page_size}")
        if page_token:
            params.append(f"page_token={page_token}")
        if params:
            path = path + "?" + "&".join(params)
        return self._http.get(path) or {}

    def get(self, model_id: int) -> Dict[str, Any]:
        """获取语义模型详情。"""
        return self._http.get(f"{self._base()}/{model_id}") or {}

    def update(
        self,
        model_id: int,
        *,
        name: str,
        tables: List[Dict[str, Any]],
        description: Optional[str] = None,
        files: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """更新语义模型。"""
        self._validate_structured_tables(tables)
        body: Dict[str, Any] = {"name": name, "tables": tables}
        if description is not None:
            body["description"] = description
        if files is not None:
            body["files"] = files
        return self._http.put(f"{self._base()}/{model_id}", body) or {}

    def delete(self, model_id: int) -> None:
        """删除语义模型。"""
        self._http.delete(f"{self._base()}/{model_id}")

    def import_(
        self,
        *,
        name: str,
        tables: List[Dict[str, Any]],
        entries: Optional[List[Dict[str, Any]]] = None,
        description: Optional[str] = None,
        files: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """导入语义模型（模型 + 条目）。"""
        self._validate_structured_tables(tables)
        body: Dict[str, Any] = {"name": name, "tables": tables}
        if description is not None:
            body["description"] = description
        if files is not None:
            body["files"] = files
        if entries is not None:
            body["entries"] = entries
        return self._http.post(self._base() + "/import", body) or {}

    def import_batch(self, requests: List[Dict[str, Any]]) -> Dict[str, Any]:
        """批量导入语义模型（同一 /import 接口，body 为数组）。"""
        if not requests:
            raise ValueError("requests must not be empty")
        for idx, item in enumerate(requests):
            if not isinstance(item, dict):
                raise TypeError(f"requests[{idx}] must be an object")
            if "tables" not in item:
                raise ValueError(f"requests[{idx}].tables is required")
            self._validate_structured_tables(item.get("tables"))
        return self._http.post(self._base() + "/import", requests) or {}

    def export(self, model_id: int) -> Dict[str, Any]:
        """导出语义模型（模型 + 条目）。"""
        return self._http.get(f"{self._base()}/{model_id}/export") or {}

    def validate(self, model_id: int) -> Dict[str, Any]:
        """校验语义模型。"""
        return self._http.post(f"{self._base()}/{model_id}/validate", {}) or {}

    def create_entry(
        self,
        model_id: int,
        *,
        kind: str,
        key: str,
        spec: Dict[str, Any],
        tables: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """创建语义条目。"""
        body: Dict[str, Any] = {"kind": kind, "key": key, "spec": spec}
        if tables is not None:
            body["tables"] = tables
        return self._http.post(f"{self._base()}/{model_id}/entries", body) or {}

    def list_entries(
        self,
        model_id: int,
        *,
        kind: Optional[str] = None,
        page_size: Optional[int] = None,
        page_token: Optional[str] = None,
    ) -> Dict[str, Any]:
        """分页列出语义条目。"""
        path = f"{self._base()}/{model_id}/entries"
        params = []
        if kind:
            params.append(f"kind={kind}")
        if page_size is not None:
            params.append(f"page_size={page_size}")
        if page_token:
            params.append(f"page_token={page_token}")
        if params:
            path = path + "?" + "&".join(params)
        return self._http.get(path) or {}

    def update_entry(
        self,
        model_id: int,
        entry_id: int,
        *,
        kind: str,
        key: str,
        spec: Dict[str, Any],
        tables: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """更新语义条目。"""
        body: Dict[str, Any] = {"kind": kind, "key": key, "spec": spec}
        if tables is not None:
            body["tables"] = tables
        return self._http.put(f"{self._base()}/{model_id}/entries/{entry_id}", body) or {}

    def delete_entry(self, model_id: int, entry_id: int) -> None:
        """删除语义条目。"""
        self._http.delete(f"{self._base()}/{model_id}/entries/{entry_id}")
