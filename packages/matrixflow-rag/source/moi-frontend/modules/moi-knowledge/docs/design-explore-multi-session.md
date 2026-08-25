# Knowledge Explore 多会话保活

## 目标

用户在会话 A 发送问题后，可以切换到会话 B 继续工作。会话 A 的 A2A stream 在后台持续运行，切回时直接从 runtime cache 恢复最新进度。

## Runtime

`exploreA2ARuntimeStore` 按 `sessionId` 保存运行态：

- draft input
- selected model
- selected knowledge ids
- current task/context/connection id
- local user/assistant messages
- A2A projection
- stream start/end time

运行态最多缓存 10 个会话，淘汰时跳过当前访问会话和 active stream 会话。

## 切换流程

```
用户点击会话 B
  ├─ ExploreSessionStore.setSelectedSessionId(B)
  ├─ exploreA2ARuntimeStore.ensureSessionRuntime(B)
  └─ React useSyncExternalStore 订阅 B 的 runtime

会话 A
  ├─ active stream 不停止
  ├─ onMessage 继续 reduce A2A projection
  └─ 切回时读取 runtimeStateMap[A]
```

## 停止与删除

- 停止当前会话：发送 A2A `tasks/cancel`，然后 abort 本地 SSE controller。
- 删除会话：调用会话删除 API，并清理该 session 的 A2A runtime；如果存在 active stream，同步 abort。

## 不再保留的旧链路

以下旧页面机制已经删除：

- 旧 Explore message history store
- 旧 Explore SSE parser
- 旧双卡片预览
- 表格回答和表详情抽屉
- 引用文件预览抽屉
