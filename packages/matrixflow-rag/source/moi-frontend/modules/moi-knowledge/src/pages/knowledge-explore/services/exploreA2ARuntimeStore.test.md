# ExploreA2ARuntimeStore 测试说明

## 覆盖范围

- 验证 Explore 会话的 A2A 请求、实时事件、取消和历史重放状态。
- 验证成功答案 artifact 与流式 assistant delta 的投影。
- 验证初始 SSE 在 terminal 前 EOF/报错时，从最后一个数据库 seq 调用 `tasks/resubscribe`，且不会在前端合成 completed/failed。
- 验证恢复流只有收到 completed/failed/canceled terminal event 后才结束并确定最终状态。
- 验证 `input-required + final=true` 正常结束当前流、保留待输入卡片，且不会追加内部错误或触发 `tasks/resubscribe`；刷新后从历史事件重放仍保持同一暂停状态。
- 验证 failed 终态（无论是否带 `reason_display`）只保留结构化失败状态，不将 provider、工具或 repair 的原始诊断写入 assistant 内容。
- 验证已取消任务仍展示取消响应的正常文本。

## 验收标准

运行以下命令通过：

```bash
pnpm --dir modules/moi-knowledge exec vitest run src/pages/knowledge-explore/services/exploreA2ARuntimeStore.test.ts
```
