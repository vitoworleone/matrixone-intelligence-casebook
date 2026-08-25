# 知识路由表（Knowledge Router）

本文件是 AI 的场景感知索引。当识别到用户意图匹配下表场景时，必须先加载对应知识源再开始工作。

## 行为规则

1. 先根据目标文件、调用点和用户目标匹配场景，再按“使用方式”加载知识源：直接读取 `.agents/skills/<name>/SKILL.md`；文档类知识源读取表中的精确路径。
2. 只加载当前任务实际命中的行；跨场景改动可以加载多个，但不要因为同属前端就批量读取所有 skill 或设计文档。
3. 无法确定场景时，先检查最近的 README、测试、调用点和 `pnpm agent:checks` 输出，再补充知识源；不要用一次加载全部文档代替调查。

## 开发场景路由

| 场景信号                  | 必须加载的知识源                                                                                | 使用方式              |
| ------------------------- | ----------------------------------------------------------------------------------------------- | --------------------- |
| 新建业务模块              | `flow-module-setup` → `flow-module-integration`                                                 | 读取 skill            |
| 新建共享模块              | `flow-module-setup` + 查阅 `modules/shared-moi-api/docs/design.md` 或对应 shared 模块 design.md | 读取 skill + 读取文档 |
| 新增 API 接口/类型        | 查阅 `modules/shared-moi-api/docs/design.md`（新增领域流程）                                    | 读取文档              |
| 新增共享 UI 组件          | 共同 `frontend-ui-standards` + 查阅 `modules/shared-moi-components/docs/design.md`（新增组件流程） | 读取根 skill + 读取文档 |
| 新增工具函数              | 查阅 `modules/shared-moi-utils/docs/design.md`（新增工具流程）                                  | 读取文档              |
| 写页面/组件代码           | 共同 `frontend-ui-standards` + `guide-ui-component`（页面模板 + data-testid 命名）               | 读取根 skill + 本地 skill |
| 写/改 Ant Design 组件代码 | 共同 `frontend-ui-standards` + `guide-antd-agent`（CLI/MCP 查询 + v6 API + token/semantic 检查） | 读取根 skill + 本地 skill |
| 写/改列表页               | 共同 `frontend-ui-standards` + `guide-ui-component` + `guide-list-page` + 按需 `guide-table-layout` | 读取根 skill + 本地 skill |
| 写表格页面                | 共同 `frontend-ui-standards` + `guide-ui-component` + `guide-table-layout`                       | 读取根 skill + 本地 skill |
| 写/改 CSS 样式            | 共同 `frontend-ui-standards` + `qa-css`（MUST/SHOULD 规则清单）                                  | 读取根 skill + 本地 skill |
| 写/改 i18n 实现代码       | `guide-i18n`（接入步骤 + 类型注册 + 反模式修复）                                                | 读取 skill            |
| 写/改 i18n locale         | `guide-i18n` + `qa-i18n`（先按指南接入，再做 key 命名与一致性审计）                             | 读取 skill            |
| 出现 `t()` 类型异常/兜底  | `guide-i18n`（如 `as never`、`unknown`、中文 fallback）                                         | 读取 skill            |
| 写/改测试文件             | `qa-test`（位置 + 测试意图说明 + 失败处理）                                                     | 读取 skill            |
| 添加 Mock 支持            | `guide-mock`（shared handler → 模块场景 → App 注册三层）                                        | 读取 skill            |
| 迁移 HTML/设计原型        | `flow-prototype-migration` → 按需 `flow-module-setup` / `flow-module-integration` → `qa-*` 验收 | 读取 skill            |
| 迁移旧模块                | `flow-migration-prep` → 实施 → `qa-css` + `qa-i18n` + `qa-test` 验收                            | 读取 skill            |
| 集成模块到 moi-new        | `flow-module-integration`（8 步接入 + 常见遗漏排行）                                            | 读取 skill            |
| 修改 moi-new core 子系统  | `ref-moi-new-design`（核心设计文档索引）                                                        | 读取 skill            |
| 排查认证/登录问题         | `guide-auth-local`                                                                              | 读取 skill            |
| 部署/Docker/环境变量注入  | 查阅 `apps/moi-app/docs/design/15-deployment.md`                                                | 读取文档              |
| 创建/更新模块文档         | `guide-docs`（README 模板 + docs 三件套 + design.md 编写流程）                                  | 读取 skill            |
| 可见代码完成后自检        | 共同 `frontend-ui-standards` + `qa-css` + `qa-i18n` + `qa-test` 按需检查                         | 读取根 skill + 本地 skill |
| 根据变更文件触发检查      | `pnpm agent:checks -- --files <changed-file...>`                                                | 运行 harness          |

## 仓库级流程

表中的共同 `frontend-ui-standards` 指向仓库根 `../skills/frontend-ui-standards/SKILL.md`。读取后按其路由只加载 `moi-frontend` 适配；它负责跨项目组件决策和视觉验收，本地 skill 继续负责 MOI 工程细则。

Commit message、PR 创建、PR review 和 PR comment 处理不在前端本地维护 skill。遇到这些任务时退出本路由表，回到仓库根 `AGENTS.md`，再按需使用根 `skills/pr-workflow/SKILL.md` 或仓库当前 review/comment 处理规范。

## 关键架构文档索引

当需要理解某个子系统的设计决策时，查阅对应文档：

| 主题                                         | 文档路径                                         |
| -------------------------------------------- | ------------------------------------------------ |
| 共享模块体系（API/组件/工具/协议的职责划分） | `apps/moi-app/docs/design/13-shared-modules.md`  |
| shared-moi-api 领域清单与新增流程            | `modules/shared-moi-api/docs/design.md`          |
| shared-moi-components 组件清单与新增流程     | `modules/shared-moi-components/docs/design.md`   |
| shared-moi-utils 工具清单与新增流程          | `modules/shared-moi-utils/docs/design.md`        |
| App-Module 协议设计                          | `modules/shared-moi-app-protocol/docs/design.md` |
| Mock 体系设计                                | `apps/moi-app/docs/design/14-mock.md`            |
| 路由系统                                     | `apps/moi-app/docs/design/02-routing.md`         |
| 认证系统                                     | `apps/moi-app/docs/design/03-auth.md`            |
| Context 系统                                 | `apps/moi-app/docs/design/04-context.md`         |
| HTTP/SSE                                     | `apps/moi-app/docs/design/05-request.md`         |
| i18n                                         | `apps/moi-app/docs/design/06-i18n.md`            |
| 菜单系统                                     | `apps/moi-app/docs/design/08-menu.md`            |
| 主题系统                                     | `apps/moi-app/docs/design/11-theme.md`           |
| 部署与环境变量体系                           | `apps/moi-app/docs/design/15-deployment.md`      |
