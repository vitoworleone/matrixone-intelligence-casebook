# 变更记录

## 2026-07-15

### 知识对话开发者视图

- 知识库对话页右上角新增“开发者视图”开关，默认使用普通摘要视图，刷新后恢复默认状态。
- 普通视图仅展示工具、状态、耗时和业务进度；开启后立即展示当前会话完整 Trace，并保留逐条折叠能力。

## 2026-04-17

### Explore A2A 页面重构

- `knowledge-explore` 页面切换为 A2A 交互，直接消费通用 `/agents/a2a` stream。
- 删除旧消息历史、表格回答和引用预览链路，页面展示 A2A thinking、plan、timeline、artifacts 和 Markdown answer。
- 统一 `knowledge-explore` / `KnowledgeExplore` / `useExplore*` 命名，移除旧拼写。

## 2026-04-14

### Knowledge V2 整合 — 前端适配

后端将 `knowledge_base` + `nl2sql_knowledge` 统一到 `semantic_models` + `semantic_entries`，前端同步完成一次性切换。

**shared-moi-api 层：**

- 删除 `knowledge.types.ts` 和 `knowledge.ts`（旧 KB 类型和 API），表查询 API 迁移到新建的 `table.ts`。
- 重写 `semantic.types.ts`：新增 `SemanticModelTable`、`SemanticModelFiles`、`SemanticModelUpsertRequest` 等 7 个类型；`SemanticModel` 移除 `knowledge_base_id`，新增 `semantic_model_id`、`files`，`tables` 改为结构化。
- 重写 `semantic.ts`：路由前缀从 `/knowledge_base` 改为 `/semantic-models`，参数名 `knowledgeBaseId` → `modelId`，新增 4 个 CRUD 函数（list 使用 cursor 分页）。
- Explore 类型：`ExploreKnowledgeBaseRef` → `ExploreSemanticModelRef`，`knowledge_bases` → `semantic_models`。

**moi-knowledge 服务层：**

- `knowledge.ts`：所有函数改调 semantic-models API，列表改为 cursor 分页（`page_size` + `page_token`）。
- `semanticModel.ts`：参数名 `knowledgeBaseId` → `modelId`，错误消息同步更新。
- `dialogSession.ts`：`CommonConfig.knowledge_base_ids` → `semantic_models: [{semantic_model_id}]`。

**Explore 链路：**

- `exploreA2ARuntimeStore.ts`：A2A metadata 写入 `semantic_model_ids` / `scope_metadata.semantic_model_ids`。
- `exploreSessionStore.ts`：`createFixedSession` config 从 `knowledge_base_ids` 改为 `semantic_models`。
- `useSessionConfigPersistence.ts`：persist 前显式清除遗留 `knowledge_base_ids` / `knowledge_bases` 字段。
- `sessionConfig.ts`：`normalizeKnowledgeIds` → `normalizeSemanticModelRefs`（输入 `unknown`，解析对象数组）。
- `index.tsx`：session config 解析从旧字段改为 `semantic_models`。

**页面适配：**

- `KnowledgeCardList.tsx`：字段映射（`usage_notes` → `description`、`files_id` → `file_ids`、`table_name` → `table_names`），分页改为 cursor + token map。
- `KnowledgeCard.tsx` / `KnowledgeMetadataEditModal.tsx`：`usage_notes` → `description`。
- `KnowledgeAdvancedConfigPage.tsx`：`sourceEntries` 构建从 `table_ids` 改为 `db_name::table_name` 稳定 key。
- `shared/knowledge/utils.ts`：映射函数重写，`table_ids` 替换为 `db_name::table_name`，输出改为 `SemanticModelFiles` + `SemanticModelTable[]`。
- 语义配置组件链（`KnowledgeAdvancedConfig` → `SemanticEntrySetting` → `SemanticModelToolbar` → `useSemanticEntries`）：`knowledgeBaseId` prop 统一改为 `modelId`，移除 `getKnowledgeDetail` fallback。
- `useExploreKnowledgeOptions.ts`：列表调用参数适配 cursor 分页。

## 2026-04-11

### 语义模型 V2 改造（Semantic Model / Semantic Entries）

- 后端 `nl2sql_knowledge` + `analysis_template` 接口下线，替换为 `semantic_model` + `semantic_entries` 体系。
- `shared-moi-api` 新增 `semantic.types.ts`（10 种 SemanticKind、Spec 结构、请求/响应体）和 `semantic.ts`（8 个 API 函数）。
- `moi-knowledge/service` 新增 `semanticModel.ts`，封装 V2 API 调用。
- 旧 nl2sql / analysis_template 共 11 个 API 函数标记 `@deprecated`。
- `dataExplore.ts` 整体标记 `@deprecated`。

### 语义配置 UI 重构

- 删除 `Nl2sql/` 目录（11 个文件）和 `AnalysisTemplate/` 目录（3 个文件）。
- 新建 `Semantic/` 目录，包含：
  - `SemanticEntrySetting` — 统一列表 + 搜索 + 游标分页 + CRUD。
  - `SemanticEntryFormModal` — 按 kind 动态渲染 spec 表单。
  - `SpecFormRenderer` — 10 种 kind 的表单片段。
  - `SemanticModelToolbar` — 导入 / 导出 / 校验。
  - `useSemanticEntries` hook — 游标分页数据管理。
  - `useSemanticColumns` hook — 表格列定义。
- `KnowledgeAdvancedConfig` 左侧菜单从 5 项（旧 NL2SQL type）改为 10 项（SemanticKind）。
- 分页方式从 `page_number` 改为 `page_token` 游标分页，采用"加载更多"模式。

### i18n

- 新增 ~120 个 `knowledge.entry.*` key（中英文），覆盖 10 种 kind 的菜单、表单、spec 字段、工具栏、导入导出校验提示。

## 2026-04-07

### 知识对话 — 时间格式化统一

- `formatMessageTime`、导出报告时间、"最后更新"时间统一迁移至 `@moi/shared-moi-utils/datetime` 的 `formatDateTime`，移除裸 `toLocaleString` / `toLocaleTimeString` 调用。
- `exportMessageToWord` 新增 `timezone` 参数，调用方传入 `useTimezone()` 值。

### 知识对话 — 架构缺陷修复

- 预览卡片残留恢复：增加"重试刷新"和"关闭预览"按钮，streamError 30s 超时自动清理。
- 滚动定位降级：`locateLatestInsightCard` 找不到 DOM 时降级 `scrollToBottom`。
- 切回会话恢复：检测 `!isStreaming && pendingQuestion` 时主动触发刷新。
- 滚动定位改用 ref 注册表：`useInsightLocator` 维护 `Map<messageId, HTMLElement>`，不再依赖 `document.querySelector`。

### 知识对话 — 页面拆分

- `index.tsx` 从 ~1500 行拆分为 ~560 行。
- 旧版消息历史、引用预览、表格回答和消息转换链路已在 A2A 重构中删除归档。
- 当前实现以 `knowledge-explore` 的 A2A runtime、stream parser、session config 为准。

### 知识对话 — 功能恢复与增强

- 点赞/点踩并发保护（`pendingFeedbackRef`）+ optimistic update + rollback。
- 事件摘要 i18n：`eventSummaryI18n.ts` 翻译层，48 个中英文 key。
- 用户消息知识库摘要展示：解析 `config.knowledge_base_ids`，多知识库折叠。
- 旧版导出报告能力已随消息操作链路归档，当前 A2A 页面不再接入该链路。
- 消息解析异常可视化：检测 `modified_response` 解析失败，显示警告 + 复制原始数据按钮。
- 会话配置持久化：发送前从服务端 `getSession` 拉最新 config 做 merge 基线。
- 会话列表刷新：`visibilitychange` 监听，页面重新可见时自动刷新。
- 消息时间格式：接入 `useTimezone()` 对齐全局时区设置。
- 占位去重时钟容差：`streamStartedAt` 比较增加 2 秒容差。
- 错误日志脱敏：knowledge-explore 目录下所有 `console.error` 替换为 `console.warn` + `(error as Error).message`。
- 草稿会话机制决定不保留，直接创建正式会话。

## 2026-04-05

- 下线独立 `knowledge-create` 页面，创建流程统一为知识库列表页弹窗。
- 修复 NL2SQL 删除确认异步流：`onOk` 返回 Promise，补齐失败提示。
- 修复 NL2SQL 翻页时搜索词丢失问题，分页查询保留当前关键词。
- 新增模块内样式声明（`*.module.css` / `*.module.less`），保证模块级 `tsc --noEmit` 可通过。
- 完成模块内静态视觉 inline style 收敛，统一迁移至 CSS Modules/Less。
- 精简 `moi-knowledge` locale 文案，仅保留代码可达 key，并对齐 `zh-CN` / `en-US` key 集合。

## 2026-04-04

- 新建 `moi-knowledge` 模块。
- 迁移 `workflow/data-explore` 的知识库核心功能（列表/新建/编辑/NL2SQL/分析模板）。
- 去除旧数据探索左侧会话菜单依赖，仅保留知识库核心内容区。
- 完成 `moi-new` 路由与菜单集成（`knowledge`）。
