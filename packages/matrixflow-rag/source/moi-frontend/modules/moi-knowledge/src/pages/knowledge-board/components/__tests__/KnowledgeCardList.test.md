# KnowledgeCardList 知识库创建测试说明

## 业务目标

数据侧知识库页面创建空知识库时，只提交名称、备注和 `image_index_enabled`。
后端在 workspace 共享知识库 Catalog 下创建数据库、raw/processed Volume 和固定索引配置；请求不携带 `sources`、`source_selections` 或 `target_catalog_id`。

创建成功后仍需关闭创建弹窗，并在 1 秒后进入新知识库详情页。

## 验收边界

| 场景             | 验收标准                                                                              | 覆盖归属                                                                              |
| ---------------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| 基本信息弹窗     | 保留原大尺寸创建弹窗，仅展示名称、备注和高级图片索引选项，不展示第二步或来源选择      | `shows only basic information and advanced options, without source selection steps`   |
| 空知识库创建请求 | 调用 `create-empty`，请求不包含 `sources` 或 `source_selections`                      | `submits the empty knowledge-base request without sources or selections`              |
| 分页             | 列表 sentinel 可见时携带游标请求下一页并追加列表项                                    | `loads the next page when the list sentinel becomes visible`                          |
| 名称校验         | 名称为空时阻止提交并显示已有错误文案                                                  | `keeps name validation in the basic information form`                                 |
| 图片索引默认值   | 表单未提供 `image_index_enabled` 时请求传递 `false`                                   | `defaults image indexing to disabled when the form omits it`                          |
| 创建失败         | 同名冲突写入名称字段错误；其他失败显示通用创建错误                                    | 冲突和通用失败用例                                                                    |
| 页面流转         | 成功后重新打开创建弹窗会清除待执行的详情页跳转                                        | `cancels the pending redirect when creation is reopened`                              |
| data-domain 创建 | 后端创建共享 Catalog 数据库、双 Volume 和固定索引配置，不创建 source/job/RAG workflow | backend `TestSemanticModelServiceCreateEmptyModelInitializesDataDomainWithoutSources` |

## 状态矩阵

- `success`：创建请求不携带数据源字段；成功提示在 1 秒后跳转详情页；重新打开创建弹窗会取消该跳转。
- `error` / `permission-denied`：冲突显示名称字段错误，其他失败显示创建错误，页面不进入成功跳转。
- `loading`：提交期间继续使用现有创建按钮 loading 与防重复提交行为。
- `empty`：名称为空时由基本信息表单阻止提交。

## 回归命令

```bash
pnpm run --filter=@moi/moi-knowledge test -- KnowledgeCardList.test.tsx
pnpm --filter @moi/moi-knowledge exec vitest run src/pages/knowledge-board/components/__tests__/KnowledgeCardList.test.tsx
```
