# moi-knowledge

知识库模块（`moi-new` 业务模块）。

## 模块定位

`moi-knowledge` 负责知识库能力的页面供应，包含知识库管理（列表/编辑/语义配置）和知识对话（基于 A2A 的多会话流式问答）两大功能域。

## 核心设计

- 以 `ModuleDefinition` 形式对外提供页面，不直接耦合宿主路由与菜单。
- 页面组件保持 zero-props，宿主通过页面 key 进行挂载。
- HTTP 请求统一通过宿主注入的 `useHttpClient()`。
- A2A 流式连接通过宿主注入的 `useSSEClient()`。
- 页面跳转统一通过宿主注入的 `useModuleNavigator()`。
- 时区通过宿主注入的 `useTimezone()` 对齐全局设置。

## 页面/功能清单

- `knowledge-board`：知识库列表页（创建入口、卡片列表、删除能力）。
- `knowledge-edit`：知识库编辑页（数据源管理、语义配置 Semantic Entries）。
- `knowledge-explore`：知识对话页（多会话管理、A2A 流式问答、推理过程和工具事件展示）。

## 知识对话核心能力

- 多会话保活：切换会话不中断 A2A stream，切回立即恢复。
- Session 与 A2A Runtime 分层：会话 CRUD 独立，运行态负责 A2A task/context、事件投影和停止。
- A2A 事件展示：推理过程、执行计划、状态时间线、过程产物、最终 Markdown 回答。
- 知识库和模型选择随会话配置持久化，发送时注入 A2A metadata。

## 宿主 App 需提供的能力

- AppContext：`httpClient`（必需）、`sseClient`（对话页必需）、`timezone`。
- BusinessContext：`workspaceId`、用户与权限信息。
- ModuleContext：模块导航能力（页面跳转、参数读取）。
- i18n 资源注册与渲染容器。

## ModuleDefinition

模块入口导出 `moiKnowledgeModule`，包含：

- `name: 'moi-knowledge'`
- `pages: knowledge-board / knowledge-edit`
- `locales: zh-CN / en-US`
- `requires: { httpClient: true, sseClient: true }`

定义文件：`src/index.ts`。

## 技术栈

- React 18 + TypeScript
- Ant Design 6 + CSS Modules
- Vitest + Testing Library

## 相关文档

- `docs/design.md` — 模块架构与集成约束（入口）
- `docs/design-knowledge.md` — 知识库管理设计（V2 整合、API 路由、数据模型）
- `docs/design-explore.md` — 知识对话 A2A 架构设计
- `docs/design-explore-multi-session.md` — 多会话保活设计
- `docs/changelog.md` — 变更记录
