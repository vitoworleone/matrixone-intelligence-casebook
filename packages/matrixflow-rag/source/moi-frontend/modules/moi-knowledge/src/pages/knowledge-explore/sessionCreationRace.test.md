# KnowledgeExplorePage 懒创建会话竞态测试说明

## 覆盖范围

- 请求 B 触发固定知识库会话的懒创建并开始发送首条消息后，请求 D 到达并成为新的虚拟会话。
- B 的创建晚于 D 完成时，B 仍使用自己的知识库配置发送消息，但不得清除或覆盖 D 的虚拟会话状态。

## 验收标准

运行以下命令通过：

```bash
pnpm --dir modules/moi-knowledge exec vitest run src/pages/knowledge-explore/sessionCreationRace.test.tsx
```

该用例只覆盖前端页面内的异步状态隔离；真实后端创建、网络错误和权限拒绝由会话服务与其测试负责。
