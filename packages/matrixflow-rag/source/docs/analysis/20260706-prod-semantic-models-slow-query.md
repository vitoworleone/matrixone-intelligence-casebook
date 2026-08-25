# Issue 12732: semantic-models List API N+1 source count

## 背景

`GET /newmoi/semantic-models?page_size=12` 在 prod Lenovo workspace
`ca7a6b00-ed19-41d1-e898-aa851bb06661` 出现 1.2s 到 13.82s 的响应时间。问题入口在
`moi-backend/pkg/session/semantic_model_service.go`:

- `ListModels` / `ListModelsByIDs` 先调用 moi-core catalog List/Get。
- 返回分页模型后调用 `enrichSemanticModelSourceCounts` 补充 `source_counts`。
- 旧实现对每个 model 单独执行 managed source、legacy job、raw volume、lineage 等查询。

当 `page_size=12` 时，最少会有多轮 `knowledge_base_sources` /
`knowledge_base_source_jobs` / raw volume 查询；存在 legacy 数据时还会继续放大到文件和
lineage 校验查询。

## 数据语义

`source_counts` 不能只等价为 `knowledge_base_sources` 的 active row 数。现有语义包含：

1. active `knowledge_base_sources`，其中 `catalog_table` 计入 `tables`，其他 source type 计入 `files`。
2. legacy explicit file/table：来自 semantic model `files` / `tables` JSON，仍需校验 catalog 中资源存在；缺失或歧义时忽略。
3. legacy source jobs：来自 `knowledge_base_source_jobs`，仍需校验 file metadata；已被现有 source row 覆盖时忽略。
4. raw volume legacy candidates：来自 model 绑定 raw volume 下的文件，按 file id 去重。
5. lineage legacy candidates：来自 vector table lineage，按 file id 去重，并继续忽略已存在 source row 覆盖的数据。

因此修复不能用单个 `COUNT(*) GROUP BY model_id, source_type` 替代，否则会丢 legacy candidate 计数。

## 修复方案

本次实现保留单 model 的旧路径，降低对 `GetModel` 和既有单项测试的影响；当一次
`enrichSemanticModelSourceCounts` 包含多个 model 时，切换到 batch path：

- 一次批量读取所有 model 的 `knowledge_base_sources` 轻量身份字段，active rows 直接计数，所有 rows 都进入 seen set。
- 一次批量读取所有 model 的 `knowledge_base_source_jobs`。
- 批量校验 explicit legacy file metadata，批量解析 explicit legacy table name。
- 批量读取 raw volume id，再批量读取 volume file ids。
- 批量读取 lineage vector table 对应 file ids。
- 仍使用既有 seen key 规则：`file:<id>` / `table:<id>`，避免重复计数，并保留 removed source row 覆盖 legacy candidate 的旧语义。

这样列表页多 model 的主要 DB round-trip 从按 model 线性增长，变为按资源类型固定增长。

## 实现位置

- `moi-backend/pkg/session/semantic_model_service.go`
  - `enrichSemanticModelSourceCounts`
  - `semanticModelSourceCountsBatch`
  - batch readers / aggregators for source rows, source jobs, explicit legacy, raw volume, lineage
- `moi-backend/pkg/session/semantic_model_service_test.go`
  - `TestIssue12732ListModelsBatchesSourceCountsPreservingLegacyCandidates`

## 验证

已运行：

```bash
cd moi-backend
go test ./pkg/session -run 'TestIssue12732|TestSemanticModelServiceListUsesCallerClientWhenSystemClientConfigured|TestSemanticModelServiceGetModelSourceCountsIncludesAllLegacyRawVolumeCandidates' -count=1 -v
go test ./pkg/session -count=1
```

结果：

- 新增 issue 回归测试通过。
- 既有 source-count 单 model 测试通过。
- `pkg/session` package 全量测试通过。

## i18n / API 影响

本次不新增、不修改用户可见文案，不改变 HTTP response schema。`source_counts` 字段语义保持原有行为，只优化列表页多 model 的查询形态。
