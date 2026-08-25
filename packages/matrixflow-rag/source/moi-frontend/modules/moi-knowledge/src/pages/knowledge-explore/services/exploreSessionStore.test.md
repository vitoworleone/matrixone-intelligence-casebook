# ExploreSessionStore 测试说明

## 覆盖范围

- 知识库卡片进入对话后的虚拟新会话不会被旧列表回包重新选成历史会话，历史列表仍可刷新。
- 普通新建和固定知识库会话在慢创建期间清空旧会话、拒绝重复创建；旧列表回包不会丢失刚创建的会话，新的虚拟会话不会被旧创建覆盖。
- 创建失败且选择未变化时恢复用户原先选择的会话。
- 用户手动切换已有会话时，晚到的置顶或普通列表仍会更新列表数据，但不会抢占当前选择。
- 虚拟会话所属页面离开后仅释放仍有效的空选择；用户已切换至已有会话时保持该选择。

## 验收标准

运行以下命令通过：

```bash
pnpm --dir modules/moi-knowledge exec vitest run src/pages/knowledge-explore/services/exploreSessionStore.test.ts
```
