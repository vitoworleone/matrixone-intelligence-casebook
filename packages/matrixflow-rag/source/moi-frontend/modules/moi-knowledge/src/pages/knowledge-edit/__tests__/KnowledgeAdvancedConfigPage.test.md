# KnowledgeAdvancedConfigPage 测试说明

## 数据源加载与作业推进

### 业务目标

- 数据源分页列表由 `/sources` 独立加载，不等待知识库级 `/source-jobs`。
- source job 驱动与资源页码无关，每个轮询周期最多推进一次 reconcile。
- job 查询或 reconcile 异常不得清空已成功加载的资源列表。

### 关键验收场景

- success：资源列表、分页、追加、删除、治理和详情操作保持可用。
- loading：job 请求延迟时，source 列表返回后立即结束主 loading。
- error：job 请求或最终 source 刷新失败不进入 source-list 失败状态，并按 5 秒恢复重试；append 成功后即使列表刷新失败也会启动 driver。
- permission：无 KB 更新权限时只读取全局信号，不自动 reconcile。
- convergence：legacy、pending 和 running 工作均按 5 秒轮询间隔推进；running 任务在后续轮询中消失时补一次最终 sources 刷新；在途 driver 不丢失追加后的补跑请求。
- isolation：后台最终刷新不抢占进行中的用户翻页；A→B→A 切换使用 generation 隔离同 ID 的旧 source/job 响应，旧 KB mutation 不能锁住当前 KB 轮询。

## 表格分段展示

### 业务目标

验证知识库高级配置页的来源详情、分段治理和编辑状态。Issue #13330 的回归范围是：后端返回
`segment_type=table` 时，页面应渲染表格预览，同时保留原始 HTML 作为编辑内容。

### 表格分段验收

- `segment_type=table` 且 `content` 非空时，页面显示“表格”类型并使用 iframe 预览，不把 HTML 当纯文本展示。
- iframe 保持 `sandbox=""`，不授予脚本、表单、弹窗、导航或同源权限。
- `srcdoc` 是完整 HTML 文档，并在内容之前声明
  `default-src 'none'; style-src 'unsafe-inline'` CSP。表格内即使包含外部 stylesheet 或 image URL，也不允许浏览器加载外部资源。
- `srcdoc` 保留后端返回的原始表格 HTML；进入编辑态后，textarea 的值仍与原始 `content` 完全一致。

### 状态范围

该回归只增加已加载文档详情中的展示分支，不新增请求、加载态、错误态或权限判断。页面其他
success、empty、loading、error 和 permission-denied 场景继续由同一测试文件中的既有用例负责。

## 解析产物预览权限契约

### 业务目标

分段图片可能是解析流程生成、尚未挂载到 Catalog volume 的内部产物。高级配置页必须通过知识库模型归属接口预览，
由后端先执行 `semantic_model.read`，再验证 `model_id` 与 `image_file_id` / `page_image_file_id` 的可信关联。

### 关键验收场景

- `image_file_id` 存在时优先预览该产物，调用模型归属接口且不回退通用 Catalog 文件预览。
- 只有 `page_image_file_id` 时仍可通过同一模型归属接口预览。
- 普通来源文档仅传 `source_file_id`，通过模型范围来源文件预览接口验证 workflow data lineage；不回退到通用 Catalog 文件预览或其他 source/KB resource ID。
- `source_file_id` 为空时，普通来源文档保持不可预览，不能用 `kb_file_id` 代替。
- loading、error 和 permission-denied 沿用页面现有加载提示、错误提示与后端 PEP 拒绝处理。
