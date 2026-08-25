# Python SDK API Reference

> 本文档从 python-sdk 源码自动生成，请勿手动编辑。
> 修改源码后运行 `make doc-update` 更新。

## Module Functions

### new

```python
new( endpoint: str, api_key: str, timeout: float = 60.0, *, response_header_timeout: Any = None, stream_timeout: Any = None, logger: Any = None, workspace_created_callback: Any = None, worker_id: Any = None, custom_headers: Any = None, **kwargs: Any, ) -> Client
```

创建 SDK 客户端，与 Go moi.New 等价。SDK 不提供 HTTP 重试。

### new_with_personal_access_token

```python
new_with_personal_access_token( endpoint: str, personal_access_token: str, **kwargs: Any, ) -> Client
```

Create a client explicitly for a UC personal access token (PAT).

    PAT values are opaque and are never classified from their prefix. The
    token must be non-empty and contain no whitespace. Requests send exactly
    one ``X-API-Key`` header and no ``Authorization`` or ``Cookie`` header.

## Client

moi-core SDK client. 与 go-sdk Client 一比一等价。

### __init__

```python
__init__(endpoint: str, api_key: str, timeout: float = 60.0, *, response_header_timeout: Optional[float] = None, stream_timeout: Optional[float] = None, logger: Optional[Any] = None, workspace_created_callback: WorkspaceCreatedCallback = None, worker_id: Optional[str] = None, custom_headers: Optional[dict] = None, )
```

初始化 moi-core SDK 客户端。

Args:
    endpoint: 服务端地址（如 http://localhost:8080）。
    api_key: 认证用 API Key。
    timeout: HTTP 请求超时秒数，默认 60。SDK 不重试，失败即返回。
    response_header_timeout: 等待服务端返回响应头的超时秒数；默认沿用 timeout。
    stream_timeout: SSE 响应开始/读超时秒数；未设置时使用 SDK 的 stream 默认值。
    logger: 可选日志记录器。
    workspace_created_callback: 工作区创建后的回调（用于测试清理）。
    worker_id: 可选 Worker ID。
    custom_headers: 可选自定义 HTTP 头。

### set_api_key

```python
set_api_key(api_key: str) -> None
```

更新 API Key，与 Go SetAPIKey 等价。

### close

```python
close() -> None
```

释放资源；当前为 no-op，与 Go Close 等价。

### logger

```python
logger() -> Any
```

返回创建 SDK 客户端时注入的 logger。

### get

```python
get(path: str, headers: Optional[Dict[str, str]] = None) -> Any
```

发送 GET 请求并返回解析后的响应。

### post

```python
post(path: str, body: Optional[dict] = None) -> Any
```

发送 POST 请求并返回解析后的响应。

### post_with_headers

```python
post_with_headers(path: str, body: Optional[dict], headers: Dict[str, str] ) -> Any
```

发送带受控协议头的 POST 请求并返回解析后的响应。

### put

```python
put(path: str, body: Optional[dict] = None) -> Any
```

发送 PUT 请求并返回解析后的响应。

### delete

```python
delete(path: str, body: Optional[dict] = None) -> None
```

发送 DELETE 请求。

### delete_with_response

```python
delete_with_response(path: str, body: Optional[dict] = None) -> Any
```

发送 DELETE 请求并返回解析后的响应。

### post_bytes

```python
post_bytes(path: str, body: bytes, content_type: str) -> Any
```

发送原始字节 POST 请求并返回解析后的响应。

### post_multipart

```python
post_multipart(path: str, files: MultipartFiles) -> Any
```

POST multipart/form-data，供 File/Volume 上传使用。

### post_multipart_with_data

```python
post_multipart_with_data(path: str, files: MultipartFiles, data_fields: list, ) -> Any
```

POST multipart/form-data，附带额外的 form fields。

Args:
    path: API 路径。
    files: 文件字段 dict，格式同 post_multipart。
    data_fields: 额外表单字段列表，每项为 (name, value) 元组。

### download

```python
download(path: str) -> bytes
```

GET 并返回原始字节，供文件下载使用。

### catalogs

```python
catalogs() -> Any
```

Catalog 服务，与 Go Catalogs() 等价。

### agents

```python
agents() -> Any
```

通用 A2A Agent 服务，与 Go Agents() 等价。

### agent_packages

```python
agent_packages(workspace_id: str) -> Any
```

Agent package load/export service for the given workspace.

### agent_versions

```python
agent_versions(workspace_id: str) -> Any
```

Agent version lifecycle service for the given workspace.

### workspaces

```python
workspaces() -> Any
```

工作区服务，与 Go Workspaces() 等价。

### workflows

```python
workflows(workspace_id: str) -> Any
```

工作流定义服务，与 Go Workflows(workspaceID) 等价。

### workflow_versions

```python
workflow_versions(workspace_id: str) -> Any
```

工作流版本服务，与 Go WorkflowVersions(workspaceID) 等价。

### tasks

```python
tasks(workspace_id: str) -> Any
```

任务服务，与 Go Tasks(workspaceID) 等价。

### traces

```python
traces(workspace_id: str) -> Any
```

Trace 服务，与 Go Traces(workspaceID) 等价。

### users

```python
users() -> Any
```

用户服务，与 Go Users() 等价。

### apikeys

```python
apikeys() -> Any
```

API Key 服务，与 Go APIKeys() 等价。

### api_keys

```python
api_keys() -> Any
```

API Key 服务别名（兼容 snake_case 命名习惯）。

### databases

```python
databases() -> Any
```

数据库服务，与 Go Databases() 等价。

### llm

```python
llm(workspace_id: str) -> LLMService
```

Return LLM service for the given workspace.

### openxml

```python
openxml(workspace_id: str) -> Any
```

OpenXML service for the given workspace.

### mineru

```python
mineru(workspace_id: str) -> Any
```

MinerU service for the given workspace.

### office_converter

```python
office_converter(workspace_id: str) -> Any
```

Office Converter service for the given workspace.

### embeddings

```python
embeddings(workspace_id: str) -> Any
```

Embedding service for the given workspace.

### parsers

```python
parsers(workspace_id: str) -> Any
```

Parser backend service for the given workspace.

### semantic_models

```python
semantic_models(workspace_id: str) -> Any
```

Semantic model service for the given workspace.

### data_assets

```python
data_assets(workspace_id: str) -> Any
```

Data asset service for the given workspace.

### mowl_lineage

```python
mowl_lineage(workspace_id: str) -> Any
```

Mowl lineage/rerun service, aligned with Go MowlLineage(workspaceID).

### volumes

```python
volumes() -> Any
```

Volume 管理服务，与 Go Volumes() 等价。

### files

```python
files() -> Any
```

工作区文件服务，与 Go Files() 等价。

### volume_files

```python
volume_files() -> Any
```

Volume 内文件关联服务，与 Go VolumeFiles() 等价。

### garbage

```python
garbage() -> Any
```

垃圾回收触发服务，与 Go Garbage() 等价。

### work_items

```python
work_items(workspace_id: str) -> Any
```

工作项服务，与 Go WorkItems(workspaceID) 等价。

### custom_operators

```python
custom_operators(workspace_id: str) -> Any
```

Workspace custom operator service.

### parse_results

```python
parse_results(workspace_id: str) -> Any
```

解析结果服务（view/modify/export）。

### volume_content

```python
volume_content() -> Any
```

Volume 内容查询构建器，与 Go VolumeContent() 等价。

### query

```python
query() -> Any
```

资源树查询构建器，与 Go QueryBuilder 等价。

### file_query

```python
file_query() -> Any
```

文件查询构建器，与 Go FileQueryBuilder 等价。

### worker

```python
worker(workspace_id: str, **kwargs: Any) -> Any
```

Worker 客户端（gRPC），与 Go Worker(workspaceID, opts...) 等价。

### cdh

```python
cdh(workspace_id: str) -> Any
```

CDH 服务，与 Go CDH(workspaceID) 等价。

提供 CDH 配置管理和元数据同步功能。

### maxcompute

```python
maxcompute(workspace_id: str) -> Any
```

MaxCompute 服务，与 Go MaxCompute(workspaceID) 等价。

提供 MaxCompute 配置管理和元数据同步功能。

### dataphin

```python
dataphin(workspace_id: str) -> Any
```

Dataphin 服务，与 Go Dataphin(workspaceID) 等价。

提供 Dataphin 配置管理和元数据同步功能。

### system_default_ai

```python
system_default_ai() -> Any
```

System default AI service config API. Requires the raw system API key.

### upgrade

```python
upgrade() -> Any
```

System auto-upgrade diagnostics API. Requires the raw system API key.

## Error

SDK 错误，包装服务端返回的错误码与详情。

    Attributes:
code: 错误码（与 ErrorCode 一致）。
message: 错误信息。
request_id: 请求 ID（若有）。
details: 额外详情（dict）。

### __init__

```python
__init__(code: int, message: str, request_id: Optional[str] = None, details: Optional[Dict[str, str]] = None, ) -> None
```

## CatalogService

Catalog 服务：工作区内 Catalog 的创建、查询、更新、删除及下属数据库列表。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### create

```python
create(workspace_id: str, name: str, *, comment: Optional[str] = None, ) -> Dict[str, Any]
```

在工作区内创建 Catalog。

Args:
    workspace_id: 工作区 ID。
    name: Catalog 名称，须遵守 Catalog 资源名称契约；服务不会自动修正输入。
    comment: 可选备注。

Returns:
    包含 id、name 等字段的 Catalog 对象（dict）。

### get

```python
get(workspace_id: str, catalog_id: int) -> Dict[str, Any]
```

根据 ID 获取 Catalog 详情。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。

Returns:
    Catalog 对象（dict）。

### list

```python
list(workspace_id: str, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出工作区内的 Catalog。

Args:
    workspace_id: 工作区 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### update

```python
update(workspace_id: str, catalog_id: int, *, name: Optional[str] = None, comment: Optional[str] = None, ) -> Dict[str, Any]
```

更新 Catalog 属性。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。
    name: 可选新名称；提交时须遵守与创建相同的 Catalog 资源名称契约。
    comment: 可选新备注。

Returns:
    更新后的 Catalog 对象（dict）。

### delete

```python
delete(workspace_id: str, catalog_id: int) -> None
```

删除 Catalog。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。

### delete_multiple

```python
delete_multiple(workspace_id: str, catalog_ids: List[int], *, continue_on_error: bool = False, batch_concurrency: Optional[int] = None, ) -> Dict[str, Any]
```

批量删除 Catalog（客户端循环调用 Delete），与 Go SDK DeleteMultiple 一致。

Args:
    workspace_id: 工作区 ID。
    catalog_ids: 要删除的 Catalog ID 列表。
    continue_on_error: 为 True 时单个失败继续处理其余项。
    batch_concurrency: 未使用，保留以兼容接口。

Returns:
    包含 success_count、failure_count、failures 的汇总结果（dict）。

### list_databases

```python
list_databases(workspace_id: str, catalog_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 Catalog 下的数据库。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### list_summaries

```python
list_summaries(workspace_id: str, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出工作区内的 Catalog 摘要（含可见 Database 数量）。

与 Go SDK CatalogService.ListSummaries 对齐。

Args:
    workspace_id: 工作区 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、total、next_page_token 等的响应（dict）。
    每个 item 含 catalog 与 database_count。

### list_iter

```python
list_iter(workspace_id: str, *, page_size: int = 100, ) -> _BaseIterator
```

返回自动分页迭代器，与 Go ListIter 等价。

Example::

    for cat in client.catalogs().list_iter(ws_id):
        print(cat["name"])

### get_stats

```python
get_stats(workspace_id: str, catalog_id: int) -> Dict[str, Any]
```

获取 Catalog 的统计信息。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。

Returns:
    包含 database_count、table_count、volume_count、file_count 字段的统计信息（dict）。

Raises:
    moi.errors.Error: 当 Catalog 不存在（404）或其他 HTTP 错误时抛出。

## DatabaseService

数据库元数据服务：同步、查询数据库及下属 Volume/表列表。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### sync_metadata

```python
sync_metadata(workspace_id: str, database_name: str, catalog_id: int, *, comment: Optional[str] = None, register_create_iam: bool = False, register_update_iam: Optional[int] = None, ) -> Dict[str, Any]
```

将 MatrixOne 中已存在的 database 元数据同步到 moi-core，与 Go SyncMetadata 一致。

Args:
    workspace_id: 工作区 ID。
    database_name: MatrixOne 中的数据库名。用户创建同步时须遵守 Catalog
        资源名称契约；普通后台同步兼容并保留已有名称。
    catalog_id: 所属 Catalog ID。
    comment: 可选数据库描述。None=不修改，""=清空。
    register_create_iam: 用户新建 Database 后同步时必须为 True；Core
        将在同步前鉴权并登记本次创建的 Direct Owner。
    register_update_iam: 用户更新或删除已有 Database 元数据时传入
        database ID；Core 会在同步前校验该 Database 的更新权限。

Returns:
    同步结果（如数据库记录）的 dict。

### get

```python
get(workspace_id: str, database_id: int) -> Dict[str, Any]
```

根据 ID 获取数据库详情。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。

Returns:
    数据库对象（dict）。

### resolve_structured_load_target_database

```python
resolve_structured_load_target_database(workspace_id: str, database_id: int, ) -> Dict[str, Any]
```

Resolve a structured-load target database through the system API.

Args:
    workspace_id: Workspace ID.
    database_id: Catalog database ID.

Returns:
    Database metadata dict.

### resolve_structured_load_target_database_runtime

```python
resolve_structured_load_target_database_runtime(workspace_id: str, database_id: int, ) -> Dict[str, Any]
```

Resolve a structured-load target database and its runtime reference.

Args:
    workspace_id: Workspace ID.
    database_id: Catalog database ID.

Returns:
    Response dict containing database metadata and the runtime reference.

### get_table

```python
get_table(workspace_id: str, table_id: int) -> Dict[str, Any]
```

根据表 ID 获取表详情，包含所属数据库和 Catalog。

Args:
    workspace_id: 工作区 ID。
    table_id: 表 ID。

Returns:
    包含 table、database、catalog 的响应（dict）。

### resolve_structured_load_target_table

```python
resolve_structured_load_target_table(workspace_id: str, table_id: int, ) -> Dict[str, Any]
```

Resolve an existing structured-load target table through the system API.

Args:
    workspace_id: Workspace ID.
    table_id: Catalog table ID.

Returns:
    Response dict containing table, database, and catalog metadata.

### list

```python
list(workspace_id: str, catalog_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 Catalog 下的数据库。

Args:
    workspace_id: 工作区 ID。
    catalog_id: Catalog ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### list_volumes

```python
list_volumes(workspace_id: str, database_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出数据库下的 Volume。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### list_tables

```python
list_tables(workspace_id: str, database_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出数据库下的表。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### list_children

```python
list_children(workspace_id: str, database_id: int, ) -> Dict[str, Any]
```

列出 Database 的可见直接子节点（table / volume）。

与 Go SDK DatabaseService.ListChildren 对齐，一次请求返回渲染列表所需字段。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。

Returns:
    包含 items 的响应（dict）；每个 item 含 id、name、comment、type 等。

### list_iter

```python
list_iter(workspace_id: str, catalog_id: int, *, page_size: int = 100, ) -> _BaseIterator
```

返回自动分页迭代器，与 Go DatabaseService.ListIter 等价。

### get_stats

```python
get_stats(workspace_id: str, database_id: int) -> Dict[str, Any]
```

获取 Database 的统计信息。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。

Returns:
    包含 table_count、volume_count、file_count 字段的统计信息（dict）。

Raises:
    moi.errors.Error: 当 Database 不存在（404）或其他 HTTP 错误时抛出。

## VolumeService

Volume 服务：Volume CRUD、子 Volume/文件列表、文件上传下载与删除。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### create

```python
create(workspace_id: str, database_id: int, name: str, *, comment: Optional[str] = None, parent_id: Optional[int] = None, ) -> Dict[str, Any]
```

在数据库下创建 Volume。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。
    name: Volume 名称，须遵守 Catalog 资源名称契约；子 Volume 使用相同规则。
    comment: 可选备注。
    parent_id: 可选父 Volume ID。

Returns:
    包含 id、name 等字段的 Volume 对象（dict）。

### get

```python
get(workspace_id: str, volume_id: int) -> Dict[str, Any]
```

根据 ID 获取 Volume 详情。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。

Returns:
    Volume 对象（dict）。

### list

```python
list(workspace_id: str, database_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出数据库下的 Volume。

Args:
    workspace_id: 工作区 ID。
    database_id: 数据库 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### update

```python
update(workspace_id: str, volume_id: int, *, name: Optional[str] = None, comment: Optional[str] = None, ) -> Dict[str, Any]
```

更新 Volume 属性。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    name: 可选新名称；提交时须遵守与创建相同的 Catalog 资源名称契约。
    comment: 可选新备注。

Returns:
    更新后的 Volume 对象（dict）。

### delete

```python
delete(workspace_id: str, volume_id: int) -> None
```

删除 Volume。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。

### delete_multiple

```python
delete_multiple(workspace_id: str, volume_ids: List[int], *, continue_on_error: bool = False, ) -> Dict[str, Any]
```

批量删除 Volume，与 Go DeleteMultiple 等价。

Args:
    workspace_id: 工作区 ID。
    volume_ids: Volume ID 列表。
    continue_on_error: 为 True 时单个失败继续处理其余项。

Returns:
    包含 success_count、failure_count、failures 的 dict。

### get_children

```python
get_children(workspace_id: str, volume_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页获取 Volume 的子 Volume 列表。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应（dict）。

### get_path

```python
get_path(workspace_id: str, volume_id: int) -> Dict[str, Any]
```

获取 Volume 的路径信息。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。

Returns:
    路径相关信息（dict）。

### upload

```python
upload(workspace_id: str, volume_id: int, filename: str, content: Union[BinaryIO, bytes], *, content_type: Optional[str] = None, metadata: Optional[Dict[str, str]] = None, parent_directory: Optional[str] = None, ) -> Dict[str, Any]
```

向 Volume 上传文件。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    filename: 文件名。
    content: 文件内容（文件对象或 bytes）。
    content_type: 可选 MIME 类型。
    metadata: 可选元数据。
    parent_directory: 可选父目录 ID，与 Go WithParentDirectory 等价。

Returns:
    上传结果（如文件记录）的 dict。

### list_files

```python
list_files(workspace_id: str, volume_id: int, *, prefix: Optional[str] = None, parent_id: Optional[str] = None, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Any
```

分页列出 Volume 下的文件，支持前缀过滤与父目录过滤。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    prefix: 可选文件名前缀过滤，与 Go WithPrefix 等价。
    parent_id: 可选父目录 ID，与 Go WithParent 等价。
    page_size: 每页条数，与 Go WithFilesPageSize 等价。
    page_token: 分页令牌，与 Go WithFilesPageToken 等价。

Returns:
    包含 items、next_page_token 等的响应。

### download

```python
download(workspace_id: str, file_id: str) -> bytes
```

下载文件内容。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

Returns:
    文件原始字节内容。

### delete_file

```python
delete_file(workspace_id: str, file_id: str) -> None
```

删除文件。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

### list_iter

```python
list_iter(workspace_id: str, database_id: int, *, page_size: int = 100, ) -> _BaseIterator
```

返回自动分页迭代器，与 Go VolumeService.ListIter 等价。

## VolumeFileService

Volume 与文件关联服务：将文件加入/移出 Volume、在 Volume 间移动文件。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### add_files

```python
add_files(workspace_id: str, volume_id: int, file_ids: List[str], *, file_names: Optional[Dict[str, str]] = None, ) -> None
```

将已有文件关联到 Volume。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    file_ids: 文件 ID 列表。
    file_names: 可选，文件 ID 到展示名的映射。

### move_files

```python
move_files(workspace_id: str, source_volume_id: int, target_volume_id: int, file_ids: List[str], ) -> None
```

将文件从源 Volume 移动到目标 Volume。

Args:
    workspace_id: 工作区 ID。
    source_volume_id: 源 Volume ID。
    target_volume_id: 目标 Volume ID。
    file_ids: 要移动的文件 ID 列表。

### trigger_files

```python
trigger_files(workspace_id: str, volume_id: int, file_ids: List[str], ) -> Dict[str, Any]
```

为 Volume 中已有文件重新触发工作流。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    file_ids: 要触发的文件 ID 列表。

Returns:
    包含 triggered 的响应（dict）。

### remove_files

```python
remove_files(workspace_id: str, volume_id: int, file_ids: List[str], ) -> None
```

解除文件与 Volume 的关联（不删除文件本身）。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    file_ids: 要解除关联的文件 ID 列表。

### list_files

```python
list_files(workspace_id: str, volume_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, order_by: Optional[str] = None, order: Optional[str] = None, filters: Optional[Dict[str, str]] = None, fuzzy_filters: Optional[Dict[str, str]] = None, ) -> Dict[str, Any]
```

分页列出 Volume 下的文件，支持排序与过滤。

与 Go VolumeFileService.ListFiles 等价。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    page_size: 每页条数，与 Go WithVolumeFilesPageSize 等价。
    page_token: 分页令牌，与 Go WithVolumeFilesPageToken 等价。
    order_by: 排序字段，与 Go WithVolumeFilesOrderBy 等价。
    order: 排序方向 ("asc"/"desc")，与 Go WithVolumeFilesOrder 等价。
    filters: 精确过滤条件 {field: value}，与 Go WithVolumeFilesFilter 等价。
    fuzzy_filters: 模糊过滤条件 {field: value}，与 Go WithVolumeFilesFuzzyFilter 等价。

Returns:
    包含 items、total、next_page_token 等的响应（dict）。

### list_files_detail

```python
list_files_detail(workspace_id: str, volume_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, order_by: Optional[str] = None, order: Optional[str] = None, filters: Optional[Dict[str, str]] = None, fuzzy_filters: Optional[Dict[str, str]] = None, ) -> Dict[str, Any]
```

分页列出 Volume 下的文件（含文件详情），连表查询 file 表返回完整文件元数据。

与 Go VolumeFileService.ListFilesDetail 等价。
返回的每条记录除了 volume_files 表字段外，还包含 file 表的
original_name、md5、size、ref_count。

Args:
    workspace_id: 工作区 ID。
    volume_id: Volume ID。
    page_size: 每页条数。
    page_token: 分页令牌。
    order_by: 排序字段 (file_id, file_name, file_path, created_at, updated_at, id)。
    order: 排序方向 ("asc"/"desc")。
    filters: 精确过滤条件 {field: value}，支持 file_id, file_name, file_path, file_ext。
    fuzzy_filters: 模糊过滤条件 {field: value}。

Returns:
    包含 items、total、next_page_token 的响应（dict）。

## VolumeContentBuilder

Volume 内容查询构建器：按条件查询子 Volume 与文件，支持过滤、排序、分页。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化构建器。

### workspace

```python
workspace(workspace_id: str) -> "VolumeContentBuilder"
```

指定工作区 ID。返回 self 以链式调用。

### volume

```python
volume(volume_id: int) -> "VolumeContentBuilder"
```

指定 Volume ID。返回 self 以链式调用。

### with_all

```python
with_all() -> "VolumeContentBuilder"
```

包含子 Volume 与文件。返回 self 以链式调用。

### with_volumes

```python
with_volumes() -> "VolumeContentBuilder"
```

仅包含子 Volume。返回 self 以链式调用。

### with_files

```python
with_files() -> "VolumeContentBuilder"
```

仅包含文件。返回 self 以链式调用。

### filter_by_name

```python
filter_by_name(pattern: str) -> "VolumeContentBuilder"
```

按名称模式过滤。返回 self 以链式调用。

### filter_by_type

```python
filter_by_type(item_type: str) -> "VolumeContentBuilder"
```

按类型过滤。返回 self 以链式调用。

### order_by

```python
order_by(field: str) -> "VolumeContentBuilder"
```

指定排序字段。返回 self 以链式调用。

### order_desc

```python
order_desc() -> "VolumeContentBuilder"
```

降序排序。返回 self 以链式调用。

### order_asc

```python
order_asc() -> "VolumeContentBuilder"
```

升序排序。返回 self 以链式调用。

### page_size

```python
page_size(size: int) -> "VolumeContentBuilder"
```

设置每页条数。返回 self 以链式调用。

### page_token

```python
page_token(token: str) -> "VolumeContentBuilder"
```

设置分页令牌。返回 self 以链式调用。

### get

```python
get() -> Dict[str, Any]
```

执行查询，返回当前页的结果。

Returns:
    包含 items、next_page_token、stats 等的 dict。

Raises:
    ValueError: 未设置 workspace_id 或 volume_id 时。

### count

```python
count() -> Dict[str, Any]
```

返回统计信息（如总数）。

### first

```python
first() -> Optional[Dict[str, Any]]
```

只取第一条结果，若无则返回 None。

### all

```python
all() -> List[Dict[str, Any]]
```

拉取全部页并合并为一条列表。

## FileService

工作区级文件服务：上传、下载、查询、删除文件（不绑定 Volume）。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### upload

```python
upload(workspace_id: str, filename: str, content: Union[BinaryIO, bytes], *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

上传文件到工作区。

Args:
    workspace_id: 工作区 ID。
    filename: 文件名。
    content: 文件内容（文件对象或 bytes）。
    content_type: 可选 MIME 类型。

Returns:
    包含文件 id 等信息的 dict。

### upload_private_catalog_file

```python
upload_private_catalog_file(workspace_id: str, filename: str, content: Union[BinaryIO, bytes], *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

上传文件到当前用户的私有 Catalog Volume，与 Go UploadPrivateCatalogFile 等价。

### download

```python
download(workspace_id: str, file_id: str) -> bytes
```

下载文件内容。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

Returns:
    文件原始字节内容。

### preview

```python
preview(workspace_id: str, file_id: str) -> bytes
```

获取适合浏览器展示的文件预览内容。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

Returns:
    文件预览的原始字节内容。

### get

```python
get(workspace_id: str, file_id: str) -> Dict[str, Any]
```

获取文件元数据。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

Returns:
    文件信息 dict。

### delete

```python
delete(workspace_id: str, file_id: str) -> None
```

删除文件。

Args:
    workspace_id: 工作区 ID。
    file_id: 文件 ID。

### upload_bytes

```python
upload_bytes(workspace_id: str, filename: str, data: bytes, *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

使用 bytes 上传文件的便捷方法，与 Go UploadBytes 等价。

Args:
    workspace_id: 工作区 ID。
    filename: 文件名。
    data: 文件字节内容。
    content_type: 可选 MIME 类型。

Returns:
    包含文件 id 等信息的 dict。

### upload_file

```python
upload_file(workspace_id: str, file_path: str, *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

从本地路径上传文件，文件名使用路径的 basename，与 Go UploadFile 等价。

### upload_file_with_name

```python
upload_file_with_name(workspace_id: str, file_path: str, filename: str, *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

从本地路径上传文件并指定远程文件名，与 Go UploadFileWithName 等价。

### upload_private_catalog_file_from_path

```python
upload_private_catalog_file_from_path(workspace_id: str, file_path: str, *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

从本地路径上传私有 Catalog 文件，与 Go UploadPrivateCatalogFileFromPath 等价。

### upload_private_catalog_file_with_name

```python
upload_private_catalog_file_with_name(workspace_id: str, file_path: str, filename: str, *, content_type: Optional[str] = None, ) -> Dict[str, Any]
```

从本地路径上传私有 Catalog 文件并指定远程文件名。

### download_bytes

```python
download_bytes(workspace_id: str, file_id: str) -> bytes
```

下载文件并返回字节，与 Go DownloadBytes 等价（与 download 行为一致）。

### download_to_file

```python
download_to_file(workspace_id: str, file_id: str, dest_path: str, ) -> None
```

下载文件到本地路径，与 Go DownloadToFile 等价。

### download_to_writer

```python
download_to_writer(workspace_id: str, file_id: str, w: BinaryIO, ) -> int
```

下载文件并写入到 writer，与 Go DownloadToWriter 等价。返回写入字节数。

## FileQueryBuilder

文件查询构建器，支持链式调用。

    Example::

result = client.file_query().workspace(ws_id).in_volume(vol_id).with_file_name("report").get()

### __init__

```python
__init__(client: Client) -> None
```

初始化文件查询构建器。

### workspace

```python
workspace(workspace_id: str) -> FileQueryBuilder
```

设置查询的工作区 ID。

### in_volume

```python
in_volume(volume_id: int) -> FileQueryBuilder
```

限定在指定 Volume 内查询文件。

### with_file_name

```python
with_file_name(pattern: str) -> FileQueryBuilder
```

按文件名模式过滤。

### with_md5

```python
with_md5(md5: str) -> FileQueryBuilder
```

按 MD5 过滤。

### with_min_ref_count

```python
with_min_ref_count(min_count: int) -> FileQueryBuilder
```

设置最小引用计数过滤。

### with_max_ref_count

```python
with_max_ref_count(max_count: int) -> FileQueryBuilder
```

设置最大引用计数过滤。

### with_ref_count

```python
with_ref_count(count: int) -> FileQueryBuilder
```

精确匹配引用计数。

### with_orphan_files

```python
with_orphan_files() -> FileQueryBuilder
```

仅查询孤立文件（引用计数为 0）。

### page_size

```python
page_size(size: int) -> FileQueryBuilder
```

设置每页条数。

### page_token

```python
page_token(token: str) -> FileQueryBuilder
```

设置分页令牌。

### get

```python
get() -> Dict[str, Any]
```

执行查询并返回分页结果。

### count

```python
count() -> int
```

返回匹配文件总数。

### first

```python
first() -> Optional[Dict[str, Any]]
```

返回第一条匹配结果，无结果时返回 None。

### all

```python
all() -> List[Dict[str, Any]]
```

自动翻页获取所有匹配文件。

## WorkspaceService

工作区服务：工作区的创建、查询、更新、删除及成员与数据库连接管理。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### create

```python
create(name: str, *, description: Optional[str] = None, ) -> Dict[str, Any]
```

创建工作区。

Args:
    name: 工作区名称。
    description: 可选描述。

Returns:
    包含 id、name 等字段的工作区对象（dict）。

### get

```python
get(workspace_id: str) -> Dict[str, Any]
```

根据 ID 获取工作区详情。

Args:
    workspace_id: 工作区 ID。

Returns:
    工作区对象（dict）。

### list

```python
list(*, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Any
```

分页列出当前用户可访问的工作区。

Args:
    page_size: 每页条数。
    page_token: 分页令牌，来自上一页的 next_page_token。

Returns:
    包含 items、next_page_token 等字段的响应（dict）。

### update

```python
update(workspace_id: str, *, name: Optional[str] = None, description: Optional[str] = None, status: Optional[int] = None, ) -> Dict[str, Any]
```

更新工作区属性。

Args:
    workspace_id: 工作区 ID。
    name: 可选新名称。
    description: 可选新描述。
    status: 可选状态值。

Returns:
    更新后的工作区对象（dict）。

### delete

```python
delete(workspace_id: str) -> None
```

删除工作区。

Args:
    workspace_id: 工作区 ID。

### get_db_connection

```python
get_db_connection(workspace_id: str, role_id: Optional[str] = None ) -> Dict[str, Any]
```

获取工作区对应的数据库连接信息（如 host、port、account_name）。

Args:
    workspace_id: 工作区 ID。
    role_id: 可选，本次连接使用的 Explicit Role Override；缺省使用 Default Role。

Returns:
    包含连接信息的 dict（如 host、port、account_name）。

### get_user_db_connection

```python
get_user_db_connection(workspace_id: str, user_id: str, role_id: Optional[str] = None ) -> Dict[str, Any]
```

获取指定用户的数据库连接信息（仅系统用户可调用）。

Args:
    workspace_id: 工作区 ID。
    user_id: 目标用户 ID。
    role_id: 可选，目标用户本次连接使用的 Explicit Role Override；缺省使用 Default Role。

Returns:
    包含连接信息的 dict（如 host、port、account_name）。

### get_owner_credential_api_key

```python
get_owner_credential_api_key(workspace_id: str, user_id: str ) -> Dict[str, Any]
```

获取工作区 OWNER 独立 API Key 的元数据（仅系统用户可调用）。

### reveal_owner_credential_api_key

```python
reveal_owner_credential_api_key(workspace_id: str, user_id: str ) -> Dict[str, Any]
```

显示工作区 OWNER 当前独立 API Key（仅系统用户可调用）。

### rotate_owner_credential_api_key

```python
rotate_owner_credential_api_key(workspace_id: str, user_id: str ) -> Dict[str, Any]
```

轮换工作区 OWNER 独立 API Key（仅系统用户可调用）。

### get_owner_credential_db_connection

```python
get_owner_credential_db_connection(workspace_id: str, user_id: str ) -> Dict[str, Any]
```

获取工作区 OWNER 独立管理员数据库连接（仅系统用户可调用）。

### get_owner_db_connection

```python
get_owner_db_connection(workspace_id: str) -> Dict[str, Any]
```

获取工作区所有者数据库连接信息（仅系统用户可调用）。

Args:
    workspace_id: 工作区 ID。

Returns:
    包含连接信息的 dict（如 host、port、account_name）。

### get_system_roles

```python
get_system_roles(workspace_id: str) -> Dict[str, Any]
```

获取工作区系统角色引用（仅系统用户可调用）。

Args:
    workspace_id: 工作区 ID。

Returns:
    包含 superadmin_role、admin_role 的 dict。

## UserService

用户服务：用户的创建、查询、更新。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### create

```python
create(email: str, username: str, password: str, *, nickname: Optional[str] = None, phone: Optional[str] = None, ) -> Dict[str, Any]
```

创建用户。

Args:
    email: 邮箱。
    username: 用户名。
    password: 密码。
    nickname: 可选昵称。
    phone: 可选手机号。

Returns:
    包含 id、username 等字段的用户对象（dict）。

### get

```python
get(user_id: str) -> Dict[str, Any]
```

根据 ID 获取用户详情。

Args:
    user_id: 用户 ID。

Returns:
    用户对象（dict）。

### get_by_email

```python
get_by_email(email: str) -> Dict[str, Any]
```

根据邮箱获取用户。

Args:
    email: 用户邮箱。

Returns:
    用户对象（dict）。

### get_by_phone

```python
get_by_phone(phone: str) -> Dict[str, Any]
```

根据完整手机号获取用户。

### list

```python
list(*, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Any
```

分页列出用户。

Args:
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应。

### delete

```python
delete(user_id: str) -> None
```

删除用户，同时移除关联的 API Key 和角色绑定。

Args:
    user_id: 用户 ID。

Raises:
    Error: 用户不存在时返回 NOT_FOUND。

### update

```python
update(user_id: str, *, nickname: Optional[str] = None, phone: Optional[str] = None, status: Optional[int] = None, ) -> Dict[str, Any]
```

更新用户属性。

Args:
    user_id: 用户 ID。
    nickname: 可选新昵称。
    phone: 可选新手机号。
    status: 可选状态值。

Returns:
    更新后的用户对象（dict）。

## APIKeyService

API Key 服务：创建、列表、删除 API Key。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### create

```python
create(name: str, *, scopes: Optional[List[str]] = None, expires_in_days: Optional[int] = None, user_id: Optional[str] = None, ) -> Dict[str, Any]
```

创建 API Key。

Args:
    name: Key 名称。
    scopes: 可选权限范围列表。
    expires_in_days: 可选有效天数。
    user_id: 可选关联用户 ID。

Returns:
    包含 id、key（仅创建时返回）等字段的 dict。

### list

```python
list(*, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Any
```

分页列出 API Key（不含密钥明文）。

Args:
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token 等的响应。

### delete

```python
delete(api_key_id: str) -> None
```

删除 API Key。

Args:
    api_key_id: API Key ID。

## WorkflowService

工作流定义服务：工作流的创建、查询、更新、删除。

### __init__

```python
__init__(http: HTTPClient, workspace_id: str) -> None
```

使用给定的 HTTP 客户端与工作区 ID 初始化服务。

### create

```python
create(name: str, *, description: Optional[str] = None, ) -> Dict[str, Any]
```

创建工作流定义。

Args:
    name: 工作流名称。
    description: 可选描述。

Returns:
    包含 id、name 等的工作流对象（dict）。

### get

```python
get(workflow_id: str) -> Dict[str, Any]
```

根据 ID 获取工作流定义。

Args:
    workflow_id: 工作流 ID。

Returns:
    工作流对象（dict）。

### get_by_name

```python
get_by_name(name: str) -> Dict[str, Any]
```

根据名称获取工作流定义。

Args:
    name: 工作流名称。

Returns:
    工作流对象（dict）。

### list

```python
list(*, name: Optional[str] = None, ) -> Any
```

列出工作流，支持按名称过滤。

Args:
    name: 可选名称过滤。

Returns:
    包含工作流列表的响应。

### update

```python
update(workflow_id: str, *, name: Optional[str] = None, description: Optional[str] = None, ) -> Dict[str, Any]
```

更新工作流定义属性。

Args:
    workflow_id: 工作流 ID。
    name: 可选新名称。
    description: 可选新描述。

Returns:
    更新后的工作流对象（dict）。

### delete

```python
delete(workflow_id: str) -> None
```

删除工作流定义。

Args:
    workflow_id: 工作流 ID。

## WorkflowVersionService

工作流版本服务：版本的创建、查询、发布、弃用、删除。

### __init__

```python
__init__(http: HTTPClient, workspace_id: str) -> None
```

使用给定的 HTTP 客户端与工作区 ID 初始化服务。

### create

```python
create(workflow_id: str, workflow: Dict[str, Any], *, description: Optional[str] = None, workflow_type: Optional[int] = None, input_schema: Optional[str] = None, output_schema: Optional[str] = None, result_mode: Optional[int] = None, runtime_spec_json: Optional[str] = None, ) -> Dict[str, Any]
```

创建工作流版本。workflow 为 DSL 解析后的结构，与 Go CreateWorkflowVersionRequest 一致。

Args:
    workflow_id: 工作流 ID。
    workflow: 工作流定义（dict，与 proto Workflow 兼容）。
    description: 可选版本描述。
    workflow_type: 可选类型。
    input_schema: 可选输入 schema。
    output_schema: 可选输出 schema。
    result_mode: 可选结果模式。
    runtime_spec_json: 可选运行时规格 JSON。

Returns:
    版本对象（dict）。

### create_by_yaml

```python
create_by_yaml(workflow_id: str, yaml_content: Union[str, bytes], *, description: Optional[str] = None, **kwargs: Any, ) -> Dict[str, Any]
```

从 YAML 字符串或字节创建版本（内部转为 workflow 后调用 create）。

与 Go CreateByDSLBytes 完全对齐：客户端解析 YAML 为 workflow dict 后发送。

Args:
    workflow_id: 工作流 ID。
    yaml_content: YAML 字符串或字节。
    description: 可选版本描述。
    **kwargs: 其他参数透传给 create。

Returns:
    版本对象（dict）。

### get

```python
get(version_id: str) -> Dict[str, Any]
```

根据版本 ID 获取版本详情。

Args:
    version_id: 版本 ID。

Returns:
    版本对象（dict）。

### get_by_version

```python
get_by_version(workflow_id: str, version: int) -> Dict[str, Any]
```

根据工作流 ID 与版本号获取版本。

Args:
    workflow_id: 工作流 ID。
    version: 版本号。

Returns:
    版本对象（dict）。

### get_latest_published

```python
get_latest_published(workflow_id: str) -> Dict[str, Any]
```

获取工作流最新已发布版本。

Args:
    workflow_id: 工作流 ID。

Returns:
    版本对象（dict）。

### list

```python
list(workflow_id: str) -> Any
```

列出工作流的所有版本。

Args:
    workflow_id: 工作流 ID。

Returns:
    版本列表或包装响应。

### publish

```python
publish(version_id: str) -> Dict[str, Any]
```

发布版本。

Args:
    version_id: 版本 ID。

Returns:
    更新后的版本对象（dict）。

### deprecate

```python
deprecate(version_id: str) -> Dict[str, Any]
```

弃用版本。

Args:
    version_id: 版本 ID。

Returns:
    更新后的版本对象（dict）。

### delete

```python
delete(version_id: str) -> None
```

删除版本。

Args:
    version_id: 版本 ID。

## NotificationBuilder

通知配置构建器，与 Go NotificationBuilder 1:1 对齐。

    Example:
# HTTP 通知
cfg = NotificationBuilder.new_http("https://callback.example.com") \\
    .with_method("POST") \\
    .with_timeout(30) \\
    .build()

# Worker 通知
cfg = NotificationBuilder.new_worker("worker-123") \\
    .with_message("task_completed") \\
    .build()

### __init__

```python
__init__(config: Any) -> None
```

### new_http

```python
new_http(cls, url: str) -> "NotificationBuilder"
```

创建 HTTP 通知配置，与 Go NewHTTPNotification 一致。

### new_worker

```python
new_worker(cls, worker_id: str) -> "NotificationBuilder"
```

创建 Worker 通知配置，与 Go NewWorkerNotification 一致。

### with_method

```python
with_method(method: str) -> "NotificationBuilder"
```

设置 HTTP method。

### with_timeout

```python
with_timeout(timeout: int) -> "NotificationBuilder"
```

设置超时秒数。

### with_headers

```python
with_headers(headers: Dict[str, str]) -> "NotificationBuilder"
```

设置 HTTP 请求头。

### with_message

```python
with_message(message: str) -> "NotificationBuilder"
```

设置 Worker 回调消息类型。

### with_notify_node_states

```python
with_notify_node_states(*states: str) -> "NotificationBuilder"
```

设置要通知的节点状态列表。

### with_notify_node_ids

```python
with_notify_node_ids(*node_ids: str) -> "NotificationBuilder"
```

设置要通知的节点 ID 列表。

### with_notify_workflow_states

```python
with_notify_workflow_states(*states: str) -> "NotificationBuilder"
```

设置要通知的工作流状态列表。

### build

```python
build() -> Any
```

返回构建好的 mowl_pb2.NotificationConfig 对象。

## TaskService

任务服务：任务的创建、查询、取消及 Case 列表。

### __init__

```python
__init__(http: HTTPClient, workspace_id: str) -> None
```

使用给定的 HTTP 客户端与工作区 ID 初始化服务。

### create

```python
create(name: str, *, workflow_version_id: Optional[str] = None, workflow_json: Optional[str] = None, workflow: Optional[str] = None, cron_expression: Optional[str] = None, data: Optional[str] = None, vars: Optional[str] = None, transient: Optional[bool] = None, notification: Optional[Dict[str, Any]] = None, runtime_spec_json: Optional[str] = None, trace: Optional[Any] = None, ) -> Dict[str, Any]
```

创建任务。需指定 workflow_version_id 或 workflow_json/workflow（内联）其一。

Args:
    name: 任务名称。
    workflow_version_id: 工作流版本 ID（与 workflow 二选一）。
    workflow_json: 内联工作流定义（JSON 字符串）；与 data 互斥。
    workflow: workflow_json 的兼容别名（不建议新代码继续使用）。
    cron_expression: 可选 cron 表达式（周期任务）。
    data: 可选输入 data。
    vars: 可选变量 JSON。
    transient: 是否临时任务。
    notification: 可选通知配置。
    runtime_spec_json: 可选运行时规格 JSON。
    trace: TraceOptions（mowl_pb2.TraceOptions 或 dict）。

Returns:
    任务对象（dict）。

Raises:
    ValueError: 标识字段缺失，或 workflow_json/workflow 与 data 同时提供。

### get

```python
get(task_id: str) -> Dict[str, Any]
```

根据 ID 获取任务详情。

Args:
    task_id: 任务 ID。

Returns:
    任务对象（dict）。

### list

```python
list(*, status: Optional[int] = None, periodic_only: Optional[bool] = None, ) -> Any
```

列出任务，支持按状态与是否周期任务过滤。

Args:
    status: 可选状态过滤。
    periodic_only: 是否仅返回周期任务。

Returns:
    任务列表或包装响应。

### cancel

```python
cancel(task_id: str) -> None
```

取消任务。

Args:
    task_id: 任务 ID。

### get_cases

```python
get_cases(task_id: str) -> Any
```

获取任务下的 Case 列表。

Args:
    task_id: 任务 ID。

Returns:
    Case 列表或包装响应。

### trigger

```python
trigger(task_id: str, *, data: Optional[str] = None) -> Dict[str, Any]
```

触发任务执行。

Args:
    task_id: 任务 ID。
    data: 可选输入数据（JSON 字符串）。

Returns:
    触发结果（dict）。

### get_case_status

```python
get_case_status(task_id: str, case_id: str) -> Dict[str, Any]
```

获取任务 Case 的执行状态。

Args:
    task_id: 任务 ID。
    case_id: Case ID。

Returns:
    Case 状态信息（dict）。

## WorkItemService

WorkItem 元数据服务：列出工作区内的 WorkItem。

### __init__

```python
__init__(http: HTTPClient, workspace_id: str) -> None
```

使用给定的 HTTP 客户端与工作区 ID 初始化服务。

### list

```python
list() -> Any
```

列出当前工作区内的 WorkItem 元数据，与 Go WorkItemService.List 一致。

Returns:
    map[string]*WorkItemMetadataList — 以 node_id 为 key 的 WorkItem 元数据映射。

### list_catalog

```python
list_catalog() -> Any
```

List frontend-friendly workitem catalog items.

### get_catalog

```python
get_catalog(node_id: str) -> Any
```

Get a single workitem catalog entry by node_id.

### list_ui_metadata

```python
list_ui_metadata() -> Any
```

列出当前用户可见 WorkItem 的 UI 元信息（专用封装）。

基于 list() 返回结果，解析并校验每个 WorkItem 的：
  - semantic_profile
  - input_ui_schema
  - output_ui_schema

Returns:
    dict[node_id] -> {"items": [ ...parsed UI metadata... ]}

Raises:
    ValueError: 任一 WorkItem 的 UI 合约字段缺失或 JSON 非法。

## _GoStyleNotification

包装 protobuf 通知对象，同时支持 Go 风格（大写）和 Python protobuf 风格（小写）属性访问。

    Go SDK 返回的 WorkflowNotification 字段为 Status、CaseId 等（大写驼峰），
    Python protobuf 生成的字段为 status、case_id 等（小写下划线）。
    此包装类让测试代码可以用任一风格访问。

### __init__

```python
__init__(proto_obj: Any) -> None
```

## _StreamWriter

与 Go StreamWriter 一致：Emit 多次后 End 一次。

### __init__

```python
__init__(send_fn: Callable[[Any], None], case_id: str, work_item_id: str, producer: str, )
```

### emit

```python
emit(data: str, vars_str: str = "") -> None
```

发送一帧流式数据。

Args:
    data (str): 事件数据。
    vars_str (str): 变量字符串。

Returns:
    None: 无返回值。

### end

```python
end(status: str) -> None
```

结束流式写入。

Args:
    status (str): 结束状态。

Returns:
    None: 无返回值。

### end_with_result

```python
end_with_result(status: str, data: str, vars_str: str = "" ) -> None
```

结束流式写入并附带最终输出。

Args:
    status (str): 结束状态。
    data (str): 最终业务输出数据。
    vars_str (str): 最终变量字符串。

Returns:
    None: 无返回值。

## _WorkItemEntry

### __init__

```python
__init__(name: str, metadata: Any, handler: Optional[Callable] = None, stream_handler: Optional[Callable] = None, dispatch_prefix: str = "", )
```

## _WorkflowNotifyEntry

### __init__

```python
__init__(name: str, handler: Optional[Callable] = None, options: Optional[NotificationOptions] = None, chan: Optional[queue.Queue] = None, )
```

## _NodeNotifyEntry

### __init__

```python
__init__(name: str, handler: Optional[Callable] = None, options: Optional[NotificationOptions] = None, chan: Optional[queue.Queue] = None, )
```

## WorkerClient

Worker 端客户端，通过 Catalog 的 gRPC 代理连接 Mowl Engine。
    与 go-sdk WorkerClient 功能 1:1 对标。

### __init__

```python
__init__(endpoint: str, api_key: str, workspace_id: str, *, worker_id: Optional[str] = None, max_concurrent: Optional[int] = None, labels: Optional[Dict[str, str]] = None, **kwargs: Any, ) -> None
```

初始化 Worker 客户端。

Args:
    endpoint: 服务端地址。
    api_key: 认证用 API Key。
    workspace_id: 工作区 ID。
    worker_id: 可选 Worker ID（connect 时必须）。
    max_concurrent: 最大并发处理数，默认 cpu_count * 4。
    labels: 调度标签；注册 WorkItem 时必须包含 mowl.runtime/worker-type。

### connect

```python
connect() -> None
```

建立 gRPC 连接、注册 Worker/WorkItems、启动 WorkerSession 与心跳。

Raises:
    ValueError: 未设置 worker_id 时。
    RuntimeError: 已连接或 ATTACH 响应非 ATTACH_ACK 时。

### disconnect

```python
disconnect() -> None
```

关闭 WorkerSession、取消注册 Worker 并关闭 gRPC 连接。未连接时无操作。

### register_work_item

```python
register_work_item(name: str, metadata: Any, handler: Callable[..., Any], ) -> None
```

注册一次性 WorkItem 处理函数，与 Go RegisterWorkItem 一致。

Args:
    name: 工作项名称（与 DSL 中 WorkItem id 一致）。
    metadata: 节点元数据（含 isolation_level 等）。
    handler: 函数 (ctx, wctx, msg) -> result 或 (result, error)。

Raises:
    ValueError: name 已注册时。

### register_work_item_with_options

```python
register_work_item_with_options(name: str, metadata: Any, handler: Callable[..., Any], *opts: Callable[[Any], None], ) -> None
```

注册一次性 WorkItem 并应用选项（如 schema），与 Go RegisterWorkItemWithOptions 一致。

Args:
    name: 工作项名称。
    metadata: 节点元数据。
    handler: 处理函数。
    *opts: WorkItemOption 函数，如 with_input_schema / with_output_schema。

### register_work_item_dispatcher_with_options

```python
register_work_item_dispatcher_with_options(name: str, dispatch_prefix: str, metadata: Any, handler: Callable[..., Any], *opts: Callable[[Any], None], ) -> None
```

Register a WorkItem that handles a family of node IDs.

The worker advertises ``name`` to Mowl, while execution requests whose
resolved node ID starts with ``dispatch_prefix`` are handled by the same
local entry.

### register_stream_work_item

```python
register_stream_work_item(name: str, metadata: Any, stream_handler: Callable[..., Any], ) -> None
```

注册流式 WorkItem 处理函数，与 Go RegisterStreamWorkItem 一致。

Args:
    name: 工作项名称。
    metadata: 节点元数据。
    stream_handler: 函数 (ctx, wctx, msg, stream_writer) -> None，通过 stream_writer.emit/end 回写。

### register_stream_work_item_with_options

```python
register_stream_work_item_with_options(name: str, metadata: Any, stream_handler: Callable[..., Any], *opts: Callable[[Any], None], ) -> None
```

注册流式 WorkItem 并应用选项（如 schema），与 Go RegisterStreamWorkItemWithOptions 一致。

Args:
    name: 工作项名称。
    metadata: 节点元数据。
    stream_handler: 流式处理函数。
    *opts: WorkItemOption 函数。

### register_dual_mode_work_item

```python
register_dual_mode_work_item(name: str, metadata: Any, handler: Optional[Callable[..., Any]] = None, stream_handler: Optional[Callable[..., Any]] = None, ) -> None
```

注册同时支持一次性与流式的 WorkItem，与 Go RegisterDualModeWorkItem 一致。

Args:
    name: 工作项名称。
    metadata: 节点元数据。
    handler: 一次性处理函数（可选）。
    stream_handler: 流式处理函数（可选）。至少需提供其一。

Raises:
    ValueError: 两个 handler 均未提供或 name 已注册时。

### register_callback_handler

```python
register_callback_handler(message: str, handler: Callable[..., Any] ) -> None
```

与 Go RegisterCallbackHandler 一致。

### add_workflow_notify_handler

```python
add_workflow_notify_handler(name: str, handler: Optional[Callable[..., None]] = None, *opts: Any, ) -> None
```

与 Go AddWorkflowNotifyHandler 一致。handler 为 None 时仅投递到 chan（供 WaitFor 使用）。

### add_node_notify_handler

```python
add_node_notify_handler(name: str, handler: Optional[Callable[..., None]] = None, *opts: Any, ) -> None
```

与 Go AddNodeNotifyHandler 一致。

### remove_workflow_notify_handler

```python
remove_workflow_notify_handler(name: str) -> None
```

与 Go RemoveWorkflowNotifyHandler 一致。

### remove_node_notify_handler

```python
remove_node_notify_handler(name: str) -> None
```

与 Go RemoveNodeNotifyHandler 一致。

### wait_for_workflow_notification

```python
wait_for_workflow_notification(ctx: Optional[Any], *opts: Any, ) -> Any
```

与 Go WaitForWorkflowNotification 一致。阻塞直到收到匹配通知或 ctx 取消。返回 WorkflowNotification 或 None。

### wait_for_node_notification

```python
wait_for_node_notification(ctx: Optional[Any], *opts: Any, ) -> Any
```

与 Go WaitForNodeNotification 一致。

### share_to

```python
share_to(node_id: str, user_id: str) -> None
```

与 Go ShareTo 一致：将 workitem 共享给指定用户。

### shared_list

```python
shared_list(node_id: str) -> List[str]
```

与 Go SharedList 一致：返回该 workitem 已共享给的 user_id 列表。

### remove_shared

```python
remove_shared(node_id: str, user_id: str) -> None
```

与 Go RemoveShared 一致：移除 workitem 与用户的共享关系。

### execute_by_workflow_version

```python
execute_by_workflow_version(name: str, version_id: str, *opts: Any, ) -> Any
```

按工作流版本 ID 创建并执行任务，与 Go ExecuteByWorkflowVersion 一致。

Args:
    name: 任务名称。
    version_id: 工作流版本 ID。
    *opts: 可选 with_data、with_vars、with_cron_expression 等。

Returns:
    任务 proto 对象。

### execute_by_workflow_version_with_context

```python
execute_by_workflow_version_with_context(ctx: Optional[Any], name: str, version_id: str, *opts: Any, ) -> Any
```

带 context 按工作流版本 ID 创建并执行任务，与 Go ExecuteByWorkflowVersionWithContext 一致。

Args:
    ctx: 可选 context（可为 None）。
    name: 任务名称。
    version_id: 工作流版本 ID。
    *opts: 可选 with_data、with_vars、with_cron_expression 等。

Returns:
    任务 proto 对象。

### execute_by_workflow_name

```python
execute_by_workflow_name(name: str, workflow_name: str, *opts: Any, ) -> Any
```

与 Go ExecuteByWorkflowName 一致。

### execute_by_workflow_name_with_context

```python
execute_by_workflow_name_with_context(ctx: Optional[Any], name: str, workflow_name: str, *opts: Any, ) -> Any
```

带 context 按工作流名称创建并执行任务，与 Go ExecuteByWorkflowNameWithContext 一致。

Args:
    ctx: 可选 context（可为 None）。
    name: 任务名称。
    workflow_name: 工作流名称。
    *opts: 可选 with_data、with_vars 等。

Returns:
    任务 proto 对象。

### check_workflow

```python
check_workflow(case_id: str) -> tuple
```

与 Go CheckWorkflow 一致。返回 (data, error_text, status)。

### get_task_cases

```python
get_task_cases(task_id: str) -> list[str]
```

Return durable Case IDs created by one Task.

This is deliberately exposed on WorkerClient rather than making a
caller reach into ``_mowl_stub``: callers which submit a durable Task
need this owner API after a process restart to resume reconciliation.

### get_task

```python
get_task(task_id: str) -> Any
```

Read a Task by its caller-supplied durable ID.

### wait_workflow

```python
wait_workflow(case_id: str) -> tuple
```

与 Go WaitWorkflow 一致。

### cancel_workflow

```python
cancel_workflow(case_id: str) -> None
```

与 Go CancelWorkflow 一致。

### invoke_dynamic_service_sync

```python
invoke_dynamic_service_sync(workflow_name: str, input_json: str, *opts: Any, workspace_id: str = "", ) -> InvokeResult
```

同步调用动态服务（工作流），与 Go InvokeDynamicServiceSync 一致。

Args:
    workflow_name: 工作流名称。
    input_json: 输入 JSON 字符串。
    *opts: 可选 with_version_number 等。
    workspace_id: 可选，覆盖客户端默认的 workspace_id。

Returns:
    InvokeResult（CaseID、Status、Result、Error）。

### invoke_dynamic_service_stream

```python
invoke_dynamic_service_stream(workflow_name: str, input_json: str, *opts: Any, ) -> "_StreamResult"
```

与 Go InvokeDynamicServiceStream 一致。

## _StreamResult

与 Go StreamResult 一致：Recv() 返回 StreamEvent，Done 时返回 Done=True。

### __init__

```python
__init__(stream: Any) -> None
```

### recv

```python
recv() -> tuple
```

返回 (StreamEvent, error)。error 仅在协议/网络错误时非 None。

## CallbackMessage

回调消息，与 Go CallbackMessage 一致。

    Attributes:
CaseID: 用例 ID。
Message: 消息类型（如 workflow_completed）。
Data: JSON 数据。
Vars: 变量 JSON。

## CallbackResponse

回调响应，与 Go CallbackResponse 一致。

    Attributes:
Data: 返回给引擎的 JSON 字符串。
Error: 错误信息（非空表示失败）。

## WorkerTaskOptions

创建/执行任务时的选项，与 Go WorkerTaskOptions 一致。

    Attributes:
cron_expression: Cron 表达式（周期任务）。
data: 输入 data JSON。
vars: 变量 JSON。
transient: 是否临时任务。
notification: 通知配置（mowl_pb2.NotificationConfig）。
trace: TraceOptions（mowl_pb2.TraceOptions）。

## InvokeOptions

动态服务调用选项。

    Attributes:
version_number: 指定版本号，0 表示最新。
trace: TraceOptions（mowl_pb2.TraceOptions）。

## InvokeResult

动态服务同步调用结果，与 Go InvokeResult 一致。

    Attributes:
CaseID: 用例 ID。
Status: 状态（如 COMPLETED、FAILED）。
Result: 结果 JSON。
Error: 错误信息。

## StreamEvent

流式调用事件，与 Go StreamEvent 一致。

    Attributes:
CaseID: 用例 ID。
Data: 本帧数据。
Done: 是否结束。
Error: 错误信息。

## Status

工作流/节点状态常量，与 Go Status 一致（string）。

## NotificationOptions

通知过滤选项（工作流/节点通知）。

    Attributes:
states: 状态列表（workflow 或 node 状态）。
node_ids: 节点 ID 列表（仅节点通知有效）。
case_id: 过滤指定 case_id。
task_id: 过滤指定 task_id。

## WorkItemContext

WorkItem 执行上下文接口，与 Go WorkItemContext 一致。

    在 Worker 处理 WorkItem 时提供当前 workflow、节点、输入输出与变量读写。

### get_context

```python
get_context() -> Any
```

返回底层 context。

### get_workflow

```python
get_workflow() -> Optional[Any]
```

返回当前执行的 workflow 定义（proto Workflow 或 None）。

### get_node

```python
get_node() -> Optional[Any]
```

返回当前节点定义（proto Node 或 None）。

### get_input

```python
get_input() -> str
```

返回当前 work item 的 input data（字符串，通常为 JSON）。

### get_vars

```python
get_vars() -> str
```

返回 workflow 变量（JSON 字符串）。

### execution_context

```python
execution_context() -> Any
```

返回服务端注入的 ExecutionContext，包括唯一 Effective Role。

Returns:
    data_pb2.ExecutionContext 实例。

### workspace_id

```python
workspace_id() -> str
```

返回 workspace_id（从 workflow vars 解析）。

### user_id

```python
user_id() -> str
```

返回 user_id（从 workflow vars 解析）。

### user_api_key

```python
user_api_key() -> str
```

返回 user_api_key（从 workflow vars 解析）。

### effective_role_id

```python
effective_role_id() -> str
```

返回服务端持久化并注入的唯一 Effective Role。

### set_output

```python
set_output(data: str) -> None
```

设置当前 work item 的 output。

Args:
    data: 输出数据（字符串，通常为 JSON）。

### set_vars

```python
set_vars(vars_str: str) -> None
```

设置 workflow 变量。

Args:
    vars_str: 变量 JSON 字符串。

### write_stream_result

```python
write_stream_result(data: str) -> Any
```

动态服务 stream 模式下写入流式结果；其他模式下 no-op。

Args:
    data: 本帧要写入的数据。

## _WorkItemContextImpl

基于 MowlMessage 的 WorkItemContext 实现。

### __init__

```python
__init__(ctx: Any, msg: Any) -> None
```

### get_context

```python
get_context() -> Any
```

### get_workflow

```python
get_workflow() -> Optional[Any]
```

### get_node

```python
get_node() -> Optional[Any]
```

### get_input

```python
get_input() -> str
```

### get_vars

```python
get_vars() -> str
```

### execution_context

```python
execution_context() -> Any
```

### workspace_id

```python
workspace_id() -> str
```

### user_id

```python
user_id() -> str
```

### user_api_key

```python
user_api_key() -> str
```

### effective_role_id

```python
effective_role_id() -> str
```

### set_output

```python
set_output(data: str) -> None
```

### set_vars

```python
set_vars(vars_str: str) -> None
```

### write_stream_result

```python
write_stream_result(data: str) -> Any
```

## ProxyExtension

Proxy 扩展字段（请求体中的 "proxy" 对象，与 proto ProxyExtension 一致）。

    可选字段：
    - record_message: 是否记录消息
    - session_id: 会话 ID
    - source: 来源（如 "cli"）
    - role: 消息角色（int，如 0=USER, 1=SYSTEM, 2=ASSISTANT, 3=AGENT_TOOL）
    - original_content: 原始内容
    - config: 配置 JSON 字符串
    - mock_response: 仅 dev-llm 生效，用于测试时直接返回该内容

## LLMService

LLM operations scoped to a workspace. Use client.llm(workspace_id).

### __init__

```python
__init__(client: "object", workspace_id: str)
```

### base_url

```python
base_url() -> str
```

Return the absolute LLM base URL for OpenAI-compatible clients.

### create_session

```python
create_session(title: str, *, source: str = "", config: str = "", ) -> dict
```

Create a session in the workspace.

Parameters
----------
title : str
    Session title (required).
source : str, optional
    Session source (e.g. "cli").
config : str, optional
    Session config JSON.

Returns
-------
dict
    Created session (id, title, source, ...).

Example
-------
>>> session = client.llm(workspace_id).create_session("My Chat", source="cli")

### list_sessions

```python
list_sessions(*, source: str = "", page: int = 0, page_size: int = 0, keyword: str = "", tag: str = "", ) -> dict
```

List sessions with optional filters and pagination.

Parameters
----------
source : str, optional
    Filter by source.
page : int, optional
    Page (1-based).
page_size : int, optional
    Page size.
keyword : str, optional
    Filter by keyword (searches session title).
tag : str, optional
    Filter by tag name.

Returns
-------
dict
    {"sessions": [...], "total": N}.

Example
-------
>>> resp = client.llm(workspace_id).list_sessions(page_size=20)
>>> resp = client.llm(workspace_id).list_sessions(keyword="finance")
>>> resp = client.llm(workspace_id).list_sessions(tag="important")

### get_session

```python
get_session(session_id: int) -> dict
```

Get a session by ID.

Parameters
----------
session_id : int
    Session ID.

Returns
-------
dict
    Session (id, title, source, ...).

### update_session

```python
update_session(session_id: int, *, title: Optional[str] = None, config: Optional[str] = None, expected_title: Optional[str] = None, ) -> dict
```

Update a session. Only provided fields are updated.

Parameters
----------
session_id : int
    Session ID.
title : str, optional
    New title.
config : str, optional
    New config JSON.
expected_title : str, optional
    Update only if the stored title still matches this value. A mismatch
    returns the current unchanged session without an error.

Returns
-------
dict
    Resulting session. If expected_title does not match, this is the
    current unchanged session.

### delete_session

```python
delete_session(session_id: int) -> None
```

Delete a session by ID.

Parameters
----------
session_id : int
    Session ID.

### create_message

```python
create_message(session_id: int, msg: dict) -> dict
```

Create a message in a session. session_id is set on msg automatically.

Parameters
----------
session_id : int
    Session ID.
msg : dict
    Message (role, content, ...). Must not be None.

Returns
-------
dict
    Created message.

### list_messages

```python
list_messages(session_id: int, *, after: int = 0, limit: int = 0, role: Optional[int] = None, status: str = "", ) -> dict
```

List messages in a session.

Parameters
----------
session_id : int
    Session ID.
after : int, optional
    List messages after this message ID.
limit : int, optional
    Max number of messages.
role : int, optional
    Filter by role (e.g. MessageRole enum).
status : str, optional
    Filter by status.

Returns
-------
dict
    {"messages": [...], "total": N}.

### get_message

```python
get_message(message_id: int) -> dict
```

Get a message by ID.

### latest_message_id

```python
latest_message_id(session_id: int, *, completed_only: bool = False ) -> int
```

Return the latest message ID for a session.

Parameters
----------
session_id : int
    Session ID.
completed_only : bool, optional
    If True, only consider completed messages.

### modify_response

```python
modify_response(session_id: int, message_id: int, modified_response: str, ) -> None
```

Update the modified response of a message.

### append_modified_response

```python
append_modified_response(session_id: int, message_id: int, append_content: str, ) -> None
```

Append to the modified response of a message.

### list_tags

```python
list_tags(*, source: str = "", keyword: str = "", page: int = 0, page_size: int = 0, ) -> dict
```

List tags with optional filters and pagination.

Parameters
----------
source : str, optional
    Filter by source.
keyword : str, optional
    Filter by keyword.
page : int, optional
    Page (1-based).
page_size : int, optional
    Page size.

Returns
-------
dict
    {"tags": [...], "total": N}.

Example
-------
>>> resp = client.llm(workspace_id).list_tags(page_size=20)

### create_tag

```python
create_tag(source: str, name: str) -> dict
```

Create a tag (or get existing by source+name).

Parameters
----------
source : str
    Tag source (required).
name : str
    Tag name (required).

Returns
-------
dict
    Created or existing tag.

Example
-------
>>> tag = client.llm(workspace_id).create_tag("cli", "important")

### delete_tag

```python
delete_tag(source: str, name: str) -> None
```

Delete a tag by source and name.

### list_session_tags

```python
list_session_tags(session_id: int) -> List[dict]
```

List tags for a session.

### add_session_tag_relation

```python
add_session_tag_relation(session_id: int, tag_source: str, tag_name: str, ) -> None
```

Add a tag to a session.

### remove_session_tag_relation

```python
remove_session_tag_relation(session_id: int, tag_source: str, tag_name: str, ) -> None
```

Remove a tag from a session.

### list_message_tags

```python
list_message_tags(message_id: int) -> List[dict]
```

List tags for a message.

### add_message_tag_relation

```python
add_message_tag_relation(message_id: int, tag_source: str, tag_name: str, ) -> None
```

Add a tag to a message.

### remove_message_tag_relation

```python
remove_message_tag_relation(message_id: int, tag_source: str, tag_name: str, ) -> None
```

Remove a tag from a message.

### list_backends

```python
list_backends() -> dict
```

List backends with api_key redacted.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### list_models

```python
list_models(model_type: Optional[str] = None) -> dict
```

List all LLM models available on the workspace's backends.

Args:
    model_type: Optional model type filter, such as "chat", "vision",
        "ocr", or "reasoning".

Returns
-------
dict
    {"models": [{"model": "...", "backend_id": N, "backend_name": "..."}]}.

### resolve_route

```python
resolve_route(model: str, backend_id: int) -> dict
```

Resolve the workspace LLM route for a selected model and backend.

Requires a system API key.

### get_backend

```python
get_backend(backend_id: int) -> dict
```

Get a backend by ID with api_key redacted.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### create_backend

```python
create_backend(name: str, type_: int, *, api_key_encrypted: str = "", timeout_seconds: int = 0, models: Optional[List[str]] = None, ) -> dict
```

Create a backend. Requires PERM_MODEL_RESOURCE_CREATE or workspace admin.

### update_backend

```python
update_backend(backend_id: int, *, name: Optional[str] = None, api_key_encrypted: Optional[str] = None, timeout_seconds: Optional[int] = None, models: Optional[List[str]] = None, ) -> dict
```

Update a backend. Requires PERM_MODEL_RESOURCE_UPDATE or workspace admin.

### delete_backend

```python
delete_backend(backend_id: int) -> None
```

Delete a backend. Requires PERM_MODEL_RESOURCE_DELETE or workspace admin.

### create_endpoint

```python
create_endpoint(backend_id: int, address: str) -> dict
```

Add an endpoint. Requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE.

### list_endpoints

```python
list_endpoints(backend_id: int) -> list
```

List all endpoints for a backend.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### set_endpoint_status

```python
set_endpoint_status(backend_id: int, endpoint_id: int, status: int, ) -> None
```

Set endpoint status. Requires PERM_MODEL_RESOURCE_UPDATE or workspace admin.

### get_router_config

```python
get_router_config() -> dict
```

Get the router config. Requires PERM_MODEL_RESOURCE_READ or workspace admin.

Example
-------
>>> config = client.llm(workspace_id).get_router_config()

### put_router_config

```python
put_router_config(*, strategy: Optional[int] = None, health_check_interval_seconds: Optional[int] = None, max_retries: Optional[int] = None, enable_session_affinity: Optional[bool] = None, ) -> dict
```

Update the router config. Requires PERM_MODEL_RESOURCE_UPDATE or workspace admin.

### chat_completion

```python
chat_completion(question: str, model: str, *, temperature: Optional[float] = None, max_tokens: Optional[int] = None, messages: Optional[List[Dict[str, Any]]] = None, proxy_extension: Optional[ProxyExtension] = None, **extras: Any, ) -> Iterator[str]
```

Streaming chat completion (SSE). Yields content delta strings.

Server may send ': heartbeat' every ~10s; client ignores them.

Parameters
----------
question : str
    User question (required).
model : str
    Model name (required).
temperature : float, optional
    Sampling temperature (0..2).
max_tokens : int, optional
    Maximum tokens to generate.
messages : list of dict, optional
    Override default single user message.
proxy_extension : ProxyExtension, optional
    Proxy 扩展（见 ProxyExtension 类型，含 record_message, session_id, source,
    role, original_content, config, mock_response 等字段）.
**extras
    Other top-level body fields.

Yields
------
str
    Content delta (piece of assistant reply).

Example
-------
>>> for delta in client.llm(workspace_id).chat_completion("Explain Go", "gpt-4"):
...     print(delta, end="")
>>> # With proxy extension:
>>> ext = {"source": "cli", "session_id": 123, "record_message": True}
>>> for delta in client.llm(ws).chat_completion("hi", "gpt-4", proxy_extension=ext):
...     print(delta, end="")

### chat_completion_text

```python
chat_completion_text(question: str, model: str, *, temperature: Optional[float] = None, max_tokens: Optional[int] = None, messages: Optional[List[Dict[str, Any]]] = None, proxy_extension: Optional[ProxyExtension] = None, **extras: Any, ) -> str
```

Streaming chat completion, aggregated into one string.

Provider stream errors are raised instead of silently closing the stream.

## EmbeddingService

Embedding operations scoped to a workspace.

### __init__

```python
__init__(client: "object", workspace_id: str) -> None
```

### list_models

```python
list_models() -> Dict[str, Any]
```

List all embedding models available on the workspace's backends.

Returns:
    Dict with 'models' key containing a list of model info dicts.

### list_backends

```python
list_backends() -> Dict[str, Any]
```

List embedding backends with api_key redacted.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### get_backend

```python
get_backend(backend_id: int) -> Dict[str, Any]
```

Get an embedding backend by ID with api_key redacted.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### create_backend

```python
create_backend(name: str, type_: int, *, api_key_encrypted: str = "", timeout_seconds: int = 0, models: Optional[List[str]] = None, ) -> Dict[str, Any]
```

Create an embedding backend.

### update_backend

```python
update_backend(backend_id: int, *, name: Optional[str] = None, api_key_encrypted: Optional[str] = None, timeout_seconds: Optional[int] = None, models: Optional[List[str]] = None, ) -> Dict[str, Any]
```

Update an embedding backend.

### delete_backend

```python
delete_backend(backend_id: int) -> None
```

Delete an embedding backend.

### create_endpoint

```python
create_endpoint(backend_id: int, address: str) -> Dict[str, Any]
```

Add an endpoint to an embedding backend.

### list_endpoints

```python
list_endpoints(backend_id: int) -> List[Dict[str, Any]]
```

List all endpoints for an embedding backend.

Requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin.

### set_endpoint_status

```python
set_endpoint_status(backend_id: int, endpoint_id: int, status: int) -> None
```

Set an embedding backend endpoint status.

### get_router_config

```python
get_router_config() -> Dict[str, Any]
```

Get the embedding router config.

### put_router_config

```python
put_router_config(*, strategy: Optional[int] = None, health_check_interval_seconds: Optional[int] = None, max_retries: Optional[int] = None, enable_session_affinity: Optional[bool] = None, ) -> Dict[str, Any]
```

Update the embedding router config. Only provided fields are sent.

### create_embeddings

```python
create_embeddings(model: str, inputs: Union[str, Iterable[str]], *, encoding_format: Optional[str] = None, proxy: Optional[Dict[str, Any]] = None, mock_response: Optional[str] = None, extras: Optional[Dict[str, Any]] = None, ) -> Dict[str, Any]
```

Create embeddings via OpenAI-compatible endpoint.

Args:
    model: embedding model name.
    inputs: string or iterable of strings.
    encoding_format: optional encoding format ("float", etc.).
    proxy: optional proxy extension dict (e.g., {"mock_response": "..."}).
    mock_response: convenience field to set proxy.mock_response (dev-embedding).
    extras: additional top-level request fields.

Returns:
    Embedding response dict (OpenAI-compatible).

## DataAssetService

Data asset operations scoped to a workspace.

### __init__

```python
__init__(client: "object", workspace_id: str) -> None
```

### create_asset

```python
create_asset(raw_file_id: str, *, asset_id: Optional[str] = None, name: Optional[str] = None, volume_id: Optional[int] = None, source: Optional[str] = None, meta: Optional[Dict[str, Any]] = None, ) -> Dict[str, Any]
```

Register a data asset mapping to a raw file.

### create_derivation

```python
create_derivation(asset_id: str, kind: str, file_id: str, *, meta: Optional[Dict[str, Any]] = None, ) -> Dict[str, Any]
```

Link a derived artifact to a data asset.

### register_lineage

```python
register_lineage(source_file_id: str, parsed_file_id: str, *, source_file_name: Optional[str] = None, vector_table: Optional[str] = None, output_file_id: Optional[str] = None, volume_id: Optional[int] = None, source: Optional[str] = None, case_id: Optional[str] = None, recorded_by_workitem_id: Optional[str] = None, parallel_index: int = 0, runtime: Optional[Dict[str, Any]] = None, edge_provenance: Optional[Dict[str, Dict[str, Any]]] = None, ) -> Dict[str, Any]
```

Register typed assets, derivations, and parsed manifest atomically.

### upsert_parsed_manifest

```python
upsert_parsed_manifest(asset_id: str, raw_file_id: str, parsed_file_id: str, *, manifest: Optional[Dict[str, Any]] = None, ) -> Dict[str, Any]
```

Upsert parsed manifest mapping for a data asset.

### resolve

```python
resolve(*, asset_id: Optional[str] = None, raw_file_id: Optional[str] = None, ) -> Dict[str, Any]
```

Resolve asset by asset_id or raw_file_id.

### get_catalog_file_asset

```python
get_catalog_file_asset(file_id: str) -> Dict[str, Any]
```

Resolve a catalog file entry to its processed-artifact bridge.

### batch_resolve_catalog_files

```python
batch_resolve_catalog_files(file_ids: List[str]) -> Dict[str, Any]
```

Resolve multiple catalog file entries in one request.

## ParseResultService

解析结果服务（view/modify/export），与 Go ParseResultService 等价。

### __init__

```python
__init__(http: HTTPClient, workspace_id: str) -> None
```

初始化解析结果服务。

Args:
    http: HTTP 客户端。
    workspace_id: 工作区 ID。

### view

```python
view(parser: ParserInput, results: List[ParseResultInput]) -> List[Dict[str, Any]]
```

查看解析结果，与 Go View 等价。

Args:
    parser: 解析器类型（如 DOCUMENT_PARSER）。
    results: 解析结果列表。

Returns:
    处理后的解析结果列表。

### modify

```python
modify(parser: ParserInput, result: ParseResultInput, content: str, ) -> Dict[str, Any]
```

修改解析结果内容，与 Go Modify 等价。

Args:
    parser: 解析器类型。
    result: 待修改的解析结果。
    content: 新内容。

Returns:
    修改后的解析结果（dict）。

### export

```python
export(parser: ParserInput, file_name: str, results: List[ParseResultInput], ) -> List[Dict[str, str]]
```

导出解析结果为文件，与 Go Export 等价。

Args:
    parser: 解析器类型。
    file_name: 导出文件名。
    results: 解析结果列表。

Returns:
    导出文件信息列表。

## QueryBuilder

树形查询构建器，支持链式调用。

    Example::

tree = client.query().workspace(ws_id).catalogs().with_databases().with_volumes().get()

### __init__

```python
__init__(client: Client) -> None
```

初始化资源树查询构建器。

### workspace

```python
workspace(workspace_id: str) -> QueryBuilder
```

设置查询的工作区 ID。

### catalog

```python
catalog(workspace_id: str, catalog_id: int) -> QueryBuilder
```

设置查询单个 Catalog 的工作区 ID 与 Catalog ID。

### database

```python
database(workspace_id: str, database_id: int) -> QueryBuilder
```

设置查询单个数据库的工作区 ID 与数据库 ID。

### catalogs

```python
catalogs() -> QueryBuilder
```

包含 Catalog 层级。

### with_databases

```python
with_databases() -> QueryBuilder
```

包含数据库层级。

### with_volumes

```python
with_volumes() -> QueryBuilder
```

包含 Volume 层级。

### with_volume_children

```python
with_volume_children() -> QueryBuilder
```

包含 Volume 子节点。

### with_all

```python
with_all() -> QueryBuilder
```

包含所有层级（Catalog/数据库/Volume/Volume 子节点）。

### with_concurrency

```python
with_concurrency(n: int) -> QueryBuilder
```

设置并发请求数，默认 5。

### get

```python
get() -> Tree
```

执行查询并返回树。

## SchemaBuilder

JSON Schema 构建器，支持链式调用。

### __init__

```python
__init__(schema: Optional[Dict[str, Any]] = None) -> None
```

初始化 JSON Schema 构建器。

Args:
    schema: 可选初始 schema dict，默认为 ``{"type": "object"}``。

### type

```python
type(t: str) -> SchemaBuilder
```

设置 schema 类型（如 object、string、array）。

### title

```python
title(title: str) -> SchemaBuilder
```

设置 schema 标题。

### description

```python
description(desc: str) -> SchemaBuilder
```

设置 schema 描述。

### property

```python
property(name: str, prop_schema: SchemaBuilder) -> SchemaBuilder
```

添加对象属性。

### required

```python
required(*fields: str) -> SchemaBuilder
```

设置必填字段列表。

### items

```python
items(item_schema: SchemaBuilder) -> SchemaBuilder
```

设置数组元素 schema。

### enum

```python
enum(*values: Any) -> SchemaBuilder
```

设置枚举值列表。

### min_length

```python
min_length(min_val: int) -> SchemaBuilder
```

设置字符串最小长度。

### max_length

```python
max_length(max_val: int) -> SchemaBuilder
```

设置字符串最大长度。

### pattern

```python
pattern(pattern: str) -> SchemaBuilder
```

设置字符串正则模式。

### minimum

```python
minimum(min_val: float) -> SchemaBuilder
```

设置数值最小值。

### maximum

```python
maximum(max_val: float) -> SchemaBuilder
```

设置数值最大值。

### additional_properties

```python
additional_properties(allow: bool) -> SchemaBuilder
```

设置是否允许额外属性。

### build

```python
build() -> str
```

构建并返回 JSON Schema 字符串。

### must_build

```python
must_build() -> str
```

构建并返回 JSON Schema 字符串（与 build 等价）。

## TreeNode

树节点基类。

### __init__

```python
__init__(node_type: str, node_id: int, name: str, data: Optional[Dict[str, Any]] = None, ) -> None
```

初始化树节点。

Args:
    node_type: 节点类型（如 catalog、database、volume）。
    node_id: 节点 ID。
    name: 节点名称。
    data: 可选原始数据 dict。

### add_child

```python
add_child(child: TreeNode) -> None
```

添加子节点并设置其 parent 引用。

## Tree

资源树，与 Go Tree 等价。

### __init__

```python
__init__() -> None
```

初始化空资源树。

### walk

```python
walk(visitor: Callable[[TreeNode], bool]) -> None
```

深度优先遍历树，visitor 返回 False 时停止。

### walk_breadth_first

```python
walk_breadth_first(visitor: Callable[[TreeNode], bool]) -> None
```

广度优先遍历树，visitor 返回 False 时停止。

### find

```python
find(predicate: Callable[[TreeNode], bool]) -> Optional[TreeNode]
```

查找第一个满足条件的节点，未找到返回 None。

### find_all

```python
find_all(predicate: Callable[[TreeNode], bool]) -> List[TreeNode]
```

查找所有满足条件的节点。

### catalogs

```python
catalogs() -> List[TreeNode]
```

返回所有 Catalog 节点。

### databases

```python
databases() -> List[TreeNode]
```

返回所有数据库节点。

### volumes

```python
volumes() -> List[TreeNode]
```

返回所有 Volume 节点。

### catalog_count

```python
catalog_count() -> int
```

返回 Catalog 节点数量。

### database_count

```python
database_count() -> int
```

返回数据库节点数量。

### volume_count

```python
volume_count() -> int
```

返回 Volume 节点数量。

### stats

```python
stats() -> Dict[str, int]
```

返回树的统计信息。

### to_dict

```python
to_dict() -> Dict[str, Any]
```

将树序列化为 dict。

### to_json

```python
to_json() -> str
```

将树序列化为 JSON 字符串。

## _BaseIterator

通用分页迭代器基类。

### __init__

```python
__init__(fetch_page: Callable[[int, str], Dict[str, Any]], items_key: str = "items", page_size: int = 100, ) -> None
```

初始化分页迭代器。

Args:
    fetch_page: 分页请求函数，签名 (page_size, page_token) -> dict。
    items_key: 响应中数据列表的 key，默认 "items"。
    page_size: 每页条数，默认 100。

### has_next

```python
has_next() -> bool
```

是否还有下一条数据（自动触发翻页）。

### next

```python
next() -> Optional[Any]
```

返回下一条数据，无数据时返回 None。

### err

```python
err() -> Optional[Exception]
```

返回迭代过程中发生的错误，无错误时返回 None。

### stop

```python
stop() -> None
```

停止迭代，后续 has_next 将返回 False。

## CDHService

CDH 服务：CDH 配置管理和元数据同步。

    所有操作都限定在指定的 workspace 内。

### __init__

```python
__init__(client: Any, workspace_id: str) -> None
```

使用给定的客户端和工作区 ID 初始化服务。

Args:
    client: moi Client 实例。
    workspace_id: 工作区 ID。

### create_config

```python
create_config(name: str, metastore_address: str, hive_address: str, version: str, *, connect_timeout: int = 10, kerberos_principal: Optional[str] = None, kerberos_keytab: Optional[str] = None, ) -> Dict[str, Any]
```

创建 CDH 配置。

Args:
    name: 配置名称（workspace 内唯一）。
    metastore_address: Hive Metastore 地址（host:port）。
    hive_address: HiveServer2 地址（host:port）。
    version: CDH 版本（如 "6.3.2"）。
    connect_timeout: 连接超时（秒），默认 10。
    kerberos_principal: Kerberos 主体（可选）。
    kerberos_keytab: Kerberos keytab 路径（可选）。

Returns:
    创建的 CDH 配置对象（dict）。

### get_config

```python
get_config(config_id: int) -> Dict[str, Any]
```

根据 ID 获取 CDH 配置详情。

Args:
    config_id: CDH 配置 ID。

Returns:
    CDH 配置对象（dict）。

### list_configs

```python
list_configs(*, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出工作区内的 CDH 配置。

Args:
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### update_config

```python
update_config(config_id: int, *, name: Optional[str] = None, metastore_address: Optional[str] = None, hive_address: Optional[str] = None, version: Optional[str] = None, connect_timeout: Optional[int] = None, kerberos_principal: Optional[str] = None, kerberos_keytab: Optional[str] = None, ) -> Dict[str, Any]
```

更新 CDH 配置属性。

Args:
    config_id: CDH 配置 ID。
    name: 可选新名称。
    metastore_address: 可选新 Metastore 地址。
    hive_address: 可选新 Hive 地址。
    version: 可选新版本。
    connect_timeout: 可选新连接超时。
    kerberos_principal: 可选新 Kerberos 主体。
    kerberos_keytab: 可选新 Kerberos keytab 路径。

Returns:
    更新后的 CDH 配置对象（dict）。

### delete_config

```python
delete_config(config_id: int) -> None
```

删除 CDH 配置。

会级联删除所有关联的元数据（数据库、表、列）。

Args:
    config_id: CDH 配置 ID。

### sync_metadata

```python
sync_metadata(config_id: int, database_name: str, ) -> Dict[str, Any]
```

从 CDH 同步数据库元数据。

Args:
    config_id: CDH 配置 ID。
    database_name: 要同步的数据库名称。

Returns:
    包含 database、tables_synced、tables_updated、tables_deleted 的响应（dict）。

### list_databases

```python
list_databases(config_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 CDH 配置下已同步的数据库。

Args:
    config_id: CDH 配置 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### get_database

```python
get_database(config_id: int, database_id: int) -> Dict[str, Any]
```

根据 ID 获取 CDH 数据库详情。

Args:
    config_id: CDH 配置 ID。
    database_id: 数据库 ID。

Returns:
    CDH 数据库对象（dict）。

### list_tables

```python
list_tables(config_id: int, database_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 CDH 数据库下的表。

Args:
    config_id: CDH 配置 ID。
    database_id: 数据库 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### get_table

```python
get_table(config_id: int, database_id: int, table_id: int, ) -> Dict[str, Any]
```

根据 ID 获取 CDH 表详情（包含列信息）。

Args:
    config_id: CDH 配置 ID。
    database_id: 数据库 ID。
    table_id: 表 ID。

Returns:
    CDH 表对象（dict），包含 columns 字段。

### health_check

```python
health_check(config_id: int) -> Dict[str, Any]
```

检查 CDH 连接健康状态。

Args:
    config_id: CDH 配置 ID。

Returns:
    包含 status、message、cdh_version、version_mismatch 的响应（dict）。

### stop_sync

```python
stop_sync(config_id: int) -> Dict[str, Any]
```

停止 CDH 周期同步任务。

Args:
    config_id: CDH 配置 ID。

Returns:
    操作结果（dict）。

## MaxComputeService

MaxCompute 服务：MaxCompute 配置管理和元数据同步。

    所有操作都限定在指定的 workspace 内。

### __init__

```python
__init__(client: Any, workspace_id: str) -> None
```

使用给定的客户端和工作区 ID 初始化服务。

Args:
    client: moi Client 实例。
    workspace_id: 工作区 ID。

### create_config

```python
create_config(name: str, access_key_id: str, access_key_secret: str, endpoint: str, project_name: str, *, region: Optional[str] = None, ) -> Dict[str, Any]
```

创建 MaxCompute 配置。

Args:
    name: 配置名称（workspace 内唯一）。
    access_key_id: 阿里云 AccessKey ID。
    access_key_secret: 阿里云 AccessKey Secret。
    endpoint: MaxCompute Endpoint。
    project_name: MaxCompute 项目名称。
    region: 区域（可选）。

Returns:
    创建的 MaxCompute 配置对象（dict）。

### get_config

```python
get_config(config_id: int) -> Dict[str, Any]
```

根据 ID 获取 MaxCompute 配置详情。

Args:
    config_id: MaxCompute 配置 ID。

Returns:
    MaxCompute 配置对象（dict）。

### list_configs

```python
list_configs(*, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出工作区内的 MaxCompute 配置。

Args:
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### update_config

```python
update_config(config_id: int, *, name: Optional[str] = None, access_key_id: Optional[str] = None, access_key_secret: Optional[str] = None, endpoint: Optional[str] = None, region: Optional[str] = None, project_name: Optional[str] = None, ) -> Dict[str, Any]
```

更新 MaxCompute 配置属性。

Args:
    config_id: MaxCompute 配置 ID。
    name: 可选新名称。
    access_key_id: 可选新 AccessKey ID。
    access_key_secret: 可选新 AccessKey Secret。
    endpoint: 可选新 Endpoint。
    region: 可选新区域。
    project_name: 可选新项目名称。

Returns:
    更新后的 MaxCompute 配置对象（dict）。

### delete_config

```python
delete_config(config_id: int) -> None
```

删除 MaxCompute 配置。

会级联删除所有关联的元数据（数据库、表、列）。

Args:
    config_id: MaxCompute 配置 ID。

### sync_metadata

```python
sync_metadata(config_id: int, project_name: str, cron_expression: str, ) -> Dict[str, Any]
```

从 MaxCompute 同步项目元数据。

Args:
    config_id: MaxCompute 配置 ID。
    project_name: 要同步的 MaxCompute 项目名称。
    cron_expression: 周期性同步的 cron 表达式（如 "0 */6 * * *"）。

Returns:
    包含 database、tables_synced、tables_updated、tables_deleted 的响应（dict）。

### stop_sync

```python
stop_sync(config_id: int) -> None
```

停止 MaxCompute 配置的周期性同步工作流。

Args:
    config_id: MaxCompute 配置 ID。

### list_databases

```python
list_databases(config_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 MaxCompute 配置下已同步的数据库（项目）。

Args:
    config_id: MaxCompute 配置 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### get_database

```python
get_database(config_id: int, database_id: int) -> Dict[str, Any]
```

根据 ID 获取 MaxCompute 数据库（项目）详情。

Args:
    config_id: MaxCompute 配置 ID。
    database_id: 数据库 ID。

Returns:
    MaxCompute 数据库对象（dict）。

### list_tables

```python
list_tables(config_id: int, database_id: int, *, page_size: Optional[int] = None, page_token: Optional[str] = None, ) -> Dict[str, Any]
```

分页列出 MaxCompute 数据库下的表。

Args:
    config_id: MaxCompute 配置 ID。
    database_id: 数据库 ID。
    page_size: 每页条数。
    page_token: 分页令牌。

Returns:
    包含 items、next_page_token、total 的响应（dict）。

### get_table

```python
get_table(config_id: int, database_id: int, table_id: int, ) -> Dict[str, Any]
```

根据 ID 获取 MaxCompute 表详情（包含列信息）。

Args:
    config_id: MaxCompute 配置 ID。
    database_id: 数据库 ID。
    table_id: 表 ID。

Returns:
    MaxCompute 表对象（dict），包含 columns 字段。

### health_check

```python
health_check(config_id: int) -> Dict[str, Any]
```

检查 MaxCompute 连接健康状态。

Args:
    config_id: MaxCompute 配置 ID。

Returns:
    包含 status 和 message 的响应（dict）。

## GarbageService

垃圾回收服务：触发工作区级垃圾回收。

### __init__

```python
__init__(http: HTTPClient) -> None
```

使用给定的 HTTP 客户端初始化服务。

### trigger_garbage_collection

```python
trigger_garbage_collection(workspace_id: str, *, orphan_file_threshold_hours: Optional[int] = None, batch_size: Optional[int] = None, ) -> Dict[str, Any]
```

触发工作区垃圾回收，与 Go TriggerGarbageCollection 等价。

Args:
    workspace_id: 工作区 ID。
    orphan_file_threshold_hours: 孤儿文件最小存活时间（小时），
        与 Go WithOrphanFileThreshold 等价。
    batch_size: 单次处理最大条数，与 Go WithGarbageBatchSize 等价。

Returns:
    包含 orphan_files_cleaned、deleted_volumes_cleaned、message 的 dict。

## UpgradeService

System auto-upgrade diagnostics and tenant task retry API.

### __init__

```python
__init__(http: HTTPClient) -> None
```

Use the given HTTP client for system upgrade API calls.

### status

```python
status() -> Dict[str, Any]
```

Return global auto-upgrade status and tenant counters.

### list_tenant_tasks

```python
list_tenant_tasks(*, state: Optional[List[str]] = None, workspace_id: Optional[str] = None, upgrade_id: Optional[int] = None, limit: Optional[int] = None, offset: Optional[int] = None, ) -> Dict[str, Any]
```

List tenant upgrade tasks with optional state and workspace filters.

### get_tenant_task

```python
get_tenant_task(task_id: int) -> Dict[str, Any]
```

Return one tenant upgrade task by ID.

### list_tenant_task_events

```python
list_tenant_task_events(task_id: int, *, limit: Optional[int] = None, offset: Optional[int] = None, ) -> Dict[str, Any]
```

Return event history for one tenant upgrade task.

### retry_tenant_task

```python
retry_tenant_task(task_id: int, *, operator_id: Optional[str] = None, ) -> Dict[str, Any]
```

Request retry for a failed or blocked tenant upgrade task.
