# moi-knowledge 设计说明

## 模块定位

`moi-knowledge` 提供知识库核心页面能力，包含两大功能域：

1. **知识库管理**：列表、创建（弹窗）、编辑、数据源管理、语义配置（Semantic Entries）
2. **知识对话**：基于 A2A 的多会话流式问答、推理过程和工具事件展示

该模块为 `moi-new` 的业务模块，不感知 URL，仅通过 `ModuleNavigator` 进行页面导航。

## 页面定义

| 页面 key            | 说明                                                   |
| ------------------- | ------------------------------------------------------ |
| `knowledge-board`   | 知识库列表页（卡片列表 + 弹窗创建 + 对话入口）         |
| `knowledge-edit`    | 知识库编辑页（数据源管理 + 语义配置 Semantic Entries） |
| `knowledge-explore` | 知识对话页（侧边栏会话管理 + 主区域流式对话）          |

## 目录结构

```
src/
├── index.ts                          # ModuleDefinition 入口
├── locales/                          # i18n 资源（zh-CN / en-US）
├── service/                          # API 层
│   ├── semanticModel.ts              # V2 语义模型 API 封装（参数统一为 modelId）
│   ├── knowledge.ts                  # 知识库 CRUD（委托到 semantic-models API，cursor 分页）
│   ├── dialogSession.ts              # 会话管理（config 使用 semantic_models 格式）
│   └── http.ts                       # 错误处理工具
├── shared/knowledge/                 # 知识源映射工具（table_ids → db_name::table_name）
├── components/                       # 跨页面共享组件
│   ├── KnowledgeSourceEditModal/     # 数据源编辑弹窗
│   └── TreeOfFiles/                  # 文件/表树选择器
├── pages/
│   ├── knowledge-board/              # 知识库列表页
│   ├── knowledge-edit/               # 知识库编辑页
│   │   ├── components/
│   │   │   ├── KnowledgeAdvancedConfig.tsx
│   │   │   └── Semantic/             # 语义配置组件（10 种 kind）
│   │   └── ...
│   └── knowledge-explore/            # A2A 知识对话页
│       ├── index.tsx                 # 页面入口
│       ├── components/               # Composer / KnowledgeSummary
│       ├── hooks/                    # Session / Runtime / Knowledge / Model hooks
│       ├── services/                 # A2A Runtime Store + A2A 事件投影
│       └── utils/                    # 会话配置工具
└── mock/                             # Mock 注册
```

## 集成约束

- HTTP 通过 `useHttpClient()` 注入
- SSE 通过 `useSSEClient()` 注入
- 时区通过 `useTimezone()` 注入
- 工作空间 ID 通过 `useWorkspaceId()` 注入
- workspace/page 展示能力通过 `useHasCapability()` 消费 Current Principal bootstrap；对象级展示只消费 owner API 已明确发布的授权事实，未发布时不得在前端推导 allow/deny
- 国际化资源命名空间：`moi-knowledge`
- 样式使用 CSS Modules（`.module.css`），颜色通过 `var(--moi-*)` token

## 依赖

- `@moi/shared-moi-api` — 后端 API 类型与请求函数
- `@moi/shared-moi-app-protocol` — AppContext / BusinessContext / ModuleContext
- `@moi/shared-moi-components` — AiChatSessionManager / StreamMarkdown 等共享组件

## 详细设计文档

| 文档                                                                 | 内容                                                                                |
| -------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| [design-knowledge.md](./design-knowledge.md)                         | 知识库管理设计：V2 整合、API 路由映射、数据模型、table_ids 替代策略、文件级变更清单 |
| [design-explore.md](./design-explore.md)                             | 知识对话架构设计                                                                    |
| [design-explore-multi-session.md](./design-explore-multi-session.md) | 多会话保活设计                                                                      |
