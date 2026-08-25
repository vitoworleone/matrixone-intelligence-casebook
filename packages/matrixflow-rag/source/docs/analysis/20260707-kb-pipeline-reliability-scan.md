# 解析/知识库管线可靠性扫描：三个缺陷家族（2026-07-07）

> 触发事件：生产 parse-v3 任务调 qwen3-vl-plus 报 ALB 400（HTML body）与 413 request_too_large（根因与修复设计见 `moi-core/docs/parser/vlm-image-size-guard.md`）。
> 本报告是在该根因基础上的全库同类模式扫描，覆盖 issue #12577 / #12764 的根因分析，以及与在报 issue 的映射。代码基线：origin/dev b7134c453。

## 0. 结论速览

扫描识别出**三个缺陷家族**，每个家族都有已爆发的生产 issue 和尚未上报的同类隐患：

| 家族 | 模式 | 已爆发 | 扫描新发现（未上报） |
|---|---|---|---|
| F1 无界/非法 payload 打上游 | 请求体大小/合法性无守卫，被网关/provider 拒绝，错误不可定位 | VLM 大图 400/413；#12802 空 content embedding 400 | structured-extract 全文拼 prompt、extract 批量无预算、embedding 批条数无上限、MinerU 直连无上限、go-sdk 413 可重试×3 且丢弃 HTML 诊断 |
| F2 存储层写成功但可发现性层未绑定 | 向量/lineage 落库成功，但 semantic model scope 元数据没人写，下游门控判空 | **#12764**（本报告完整根因）、#12577（同根因）、#12813（同症状） | 文本索引路径无任何 `updateSemanticModel*` 回写（仅图片有）；semantic model Update 全量替换 files 可静默解绑；图片 metadata 回写硬依赖文本先绑定 |
| F3 状态聚合掩盖真实进度 | 节点/接口显示成功，实际下游产物未就绪且无对账 | #12816（source-jobs 聚合隐藏 running）、#12675、#12777 | dataset 语言标注 best-effort 吞错；source job↔workflow 终态对账在 moi-core 内不存在（疑落 moi-backend） |

---

## 1. 家族 F1：无界/非法 payload 打上游接口

### 1.1 已定案部分

图片→VLM 的 9+1 个编码点、修复设计（`imageprep` 收口）见 `moi-core/docs/parser/vlm-image-size-guard.md`，此处不重复。#12802（空 content 进 embedding 批处理 → provider 400 `input[48] must be a non-empty string`）是同家族的「非法 payload」变体，issue 已含代码定位（`parser_split_documents_length.go` 空 content 直通、`embedding_input.go` 只截断不校验非空）。

### 1.2 新发现（按风险排序，均无对应 issue）

**F1-H1 structured-extract 全文拼 prompt（高）** — `moi-core/workers/go-worker/pkg/workitems/structured_extract_unified.go:455-458`：`mergeDocumentContents(docs)` 把**全部输入文档的完整 content 拼接**进单个 prompt，无截断/token 预算，直接 `ChatCompletion`。同文件 `:154`（整段 text 输入）、`:413`（整篇单文档）同病。用户丢一批大文档做结构化抽取即可触发与 VLM 故障同形态的 413/400。

**F1-H2 extract 分组批处理无字节预算（高）** — `moi-core/workers/go-worker/pkg/workitems/extract_group_extractor.go:44-60`（builder `:159-190`）：一个 group-batch 的所有 block 打进一条多模态消息。图片块单体已被 `ImageProcessor` 压缩（PDF 页为 pdftoppm 200DPI JPEG，未 resize 但尺寸可控），**但批内块数无上限**——百页 PDF 的页图 + 文本块可以全部进同一请求。文本块 `:179` 是全文无截断。性质：单元素可控、聚合无界。

**F1-M1 embedding 批条数无上限（中）** — 单条输入有守卫（`embedding_input.go:3-7` 截断 8KB），但一次 `CreateEmbeddings` 的**条数**无预算：`embedding_generate.go:42-56`、`data_retrieval_vector_write.go:322-338` 把所有缺 embedding 的 docs 一次性入批。几千 chunk → 数十 MB 请求体。注意与 #12802（同一代码带）修复时可一并处理：空 content 校验 + 批字节/条数预算。

**F1-M1b 图片 embedding 请求体无界（中，#14898 已收口）** — 原问题位于 `moi-core/workers/go-worker/pkg/workitems/document_visual_image.go`：`documentVisualImageEmbedding` 曾把用户原图整文件字节 base64 进 `Embeddings().CreateRaw` 请求体。现在 worker 与 catalog query-visual 共用 `agent-runtime-v2.LosslessImageDataURL`，JPEG、BMP 和静态 PNG 仅在 PNG 最佳压缩后更小时替换原始字节，并保留像素内容；达到上限、GIF、WebP、APNG、TIFF 或压缩无收益时原样透传。GIF、WebP 和 TIFF 不做单帧重编码，APNG 按 PNG 容器控制块原样透传，以避免通用单帧解码截断后续页面或帧。SVG、AVIF、HEIC 等不由 Go raster decoder 支持的 `image/*` 同样保留原字节，避免收窄 worker 已接受的图像输入类型；PNG/JPEG/BMP/GIF/WebP 等已知 raster MIME 的配置解析失败仍返回错误，参与重编码的图片在完整解码或 PNG 编码失败时返回错误。该修复不修改调用方提供的 `preprocess_version`；后续若改变像素级预处理，仍需 bump 版本并重建或双写索引。

**F1-M2 MinerU 直连 multipart 无上限（中）** — `moi-core/workers/go-worker/pkg/workitems/parser/clients/catalog_mineru.go:152-211`：整个 PDF 直接 multipart POST 到发现的 MinerU 端点，绕过 catalog `/convert` 的 100MB `MaxBytesReader` 守卫（`catalog/pkg/api/handlers/parser.go:1152,1175`）。中间层若吐 HTML 413 会被原样拼进错误（`:207`）。

**F1-M3 go-sdk 传输层三连（中，横切）** — `go-sdk/internal/http/client.go`：
- 请求体**零大小预检**（`buildChatCompletionBody` 只 marshal）；
- 非 JSON 错误体（ALB HTML 页）在 `parseErrorResponse:350-423` 解析失败后**被整个丢弃**，只留 `"unexpected status code: 413"`——诊断信息蒸发；
- 413 落 default 分支映射成 `ErrorCode_INTERNAL`，而 INTERNAL 可重试（`isRetryableError`）→ **一个注定失败的超大请求被重试 3 次**，放大网关压力和延迟（复核修正：400 本就映射 `INVALID_ARGUMENT` 不可重试，只有 413 有此问题）。
后两点已随 VLM 守卫 PR 修复（413 → 不可重试 + 非 JSON body 截断附进错误）；请求体预检暂缓，观察守卫落地后是否仍需要。
另登记（review-gate 顺手发现）：**429 同样落 default 分支映射 INTERNAL 可重试且无 backoff**，限流场景下重试放大压力——下次动该映射表时一并修。

**F1-L1 workflow 通用 LLM 节点（低）** — `user_visible_nodes.go:169-204`：prompt 来自工作流上游输出无界；需特定编排才触发。

**F1-L2 向量表 content 列无截断（低/理论）** — `data_retrieval_vector_write.go:62` 持久化的是原始 content（8KB 截断只作用于 embedding 输入），50 行/批的 multi-row INSERT 理论可撞 MO 包大小上限；常规 chunk ~800 字符，实际难触发。

**正面对照（应作为修复范式引用）**：音频/视频→ASR 走**引用传递**——go-worker 只发 `download_url` + 短时鉴权头（`converter_rich_media.go:298-330`），moi-audio-service 自己下载，大小契约在服务端（`catalog/pkg/parser/adapter/moi_audio_service.go:28`）。这正是 VLM 图片路径缺失的模式；长期看「大 payload 一律引用传递」比逐点加压缩守卫更根本，但需要网关/模型侧支持 URL 取图，短期仍以 imageprep 压缩收口。

---

## 2. 家族 F2：存储层写成功但可发现性层未绑定（#12764 根因）

### 2.1 #12764 完整根因链（代码级确证）

**判定链**：Explore 是否暴露 RAG 工具 → `platformKnowledgeScopeHasTextRAG`（`catalog/pkg/agents/platform_knowledge_tool_filter.go:111-121`）只看 scope 里有没有 vector_table → scope 由 `ResolveKnowledgeScope`（`platform_knowledge_tools.go:700-871`）从 `semantic_models.files` 解析。

**断链点 1（读取侧）**：`platform_knowledge_tools.go:830` 的门控
```go
hasFileRAGSource := len(ragSource.FileIDs) > 0 ||
    (!hasSourceGovernance && ... && knowledge.RAGSourceHasIndex(ragSource))
```
生产环境 governance store 恒被装配（`:670-673`）→ 第二子句恒 false → **`files.file_ids=[]` 时，带有效 vector_table 的 RAGSource 被整个丢弃** → `resolved.VectorTable=""` → `hasTextRAG=false` → `find_rag_files`/`search_rag_chunks` 等全部被 `FilterTools` 过滤 → issue 日志中的 `tool_count=0`。

**断链点 2（写入侧）**：`RegisterLineage`（`catalog/pkg/api/handlers/data_asset.go:501-713`）文本/音频分支只建 data asset + derivation + vector asset，**没有任何 `semantic_models` 回写**；只有图片分支（`:650-684`）末尾调 `updateSemanticModelImageIndexMetadata`（`:1024-1092`），且它也只写 `image_*` 字段——全仓不存在写 `files.file_ids` 或 `files.vector_table` 的文本对应物（grep 确证：`updateSemanticModel*` 仅 image 一处）。

**一致性假象的解释**：`source_counts.files=1` 由 moi-backend `addLineageLegacySourceCandidateCountsBatch`（`moi-backend/pkg/session/semantic_model_service.go:2991-3059`）按 vector_table **反查 lineage** 算出，与 `files.file_ids` 是两条独立路径——工作流喂饱了前者（lineage+vector 都写了），没碰后者，所以 `files=1` 与 `file_ids=[]` 并存，与 issue 日志证据 1/6 完全吻合。

**UI 路径为什么没事**：知识库页面上传走 moi-backend `appendSemanticModelFiles`（`semantic_model_service.go:7976-8021`），同时写 `files.file_ids` 和 `files.vector_table`。

**模板缺口**：`audio_kb_ingest` 模板（`moi-backend/pkg/workflowtemplate/seed.go:3576-3656`，workflow 名 `audio-kb-ingest-pipeline`）7 个节点里，`moi:knowledge.index.build` 只写向量表，`moi:data.lineage.register` 的 schema（`go-worker/pkg/worker/schemas.go:893-942`）根本没有 file_ids→semantic model 的绑定入参（唯一相关的 `semantic_model_ref_vector_table` 只驱动 image 字段回写，模板也没传）。**从头到尾没人把转录文件 append 进 `files.file_ids`。**

**#12577 / #12813 定性**：#12577（音频入库不可召回）与 #12764 同模板同链路，即同根因；#12813（选中 KB 但「没有知识库检索工具」）是同一读取侧断链的另一触发面（file_ids 为空的任何成因都会走到 tool_count=0）。

### 2.2 修复建议（按推荐序）

1. **主修**：`RegisterLineage` 文本/vector 分支（`data_asset.go:622-648`）新增 `updateSemanticModelTextFiles`——按 vector_table 匹配 semantic model（复用 `:1031-1032` 的匹配），把 `source_file_ids`/`parsed_file_id` **merge-append** 进 `files.file_ids`；模板 `seed.go:3644` 节点补传 `semantic_model_ref_vector_table`。与图片路径对称、权限语义不变（file_ids 真实回填）。需处理幂等与并发（多 source fan-out）。
2. **兜底（谨慎）**：放宽 `platform_knowledge_tools.go:830` 门控，允许「有完整 index 且 governance 未显式禁用」的 RAGSource 保留。单独上有权限绕过风险（分不清「未绑定」和「全禁用」），只能作为主修的容错叠加。
3. **不推荐**：让 `knowledge.index.build` 回写 semantic model——worker 跨服务写 scope 元数据，职责污染，改动面最大。

### 2.3 同家族扫描新发现（未上报）

**F2-H1 三处「写完不绑定」的向量写 workitem** — `data_retrieval_vector_write.go:22-153`、`retrieval_vector.go:100-137`、`document_visual.go:434-515` 都只写向量行+返回 written，绑定职责全部外移。任何自定义工作流直接用这些节点建库而不接绑定环节，就复刻 #12764。

**F2-M1 图片 metadata 回写硬依赖文本先绑定** — `data_asset.go:1031-1033` 的 UPDATE 用 `files.vector_table` 定位 semantic model，命中 0 行**整个 RegisterLineage 失败**（`:1087-1088` ErrSemanticModelNotFound）；`ref_vector_table` 传空则静默 no-op。纯图片 KB 或时序错位（文本 vector_table 未落库先跑图片索引）会硬失败或静默丢绑定。

**F2-M2 semantic model Update 全量替换 files** — `catalog/pkg/service/storage/tenant/semantic_model_storage.go:258` 是整字段替换（Files 空时**置 NULL**），HTTP handler（`semantic_model.go:650-658`）直接 `Files: req.Files` 覆盖——调用方漏发 files 就把 file_ids+vector_table+image config 一次抹光、全模态工具静默消失。与 image 路径的 read-merge-write 语义不一致。另 `catalog/pkg/service/semantic/service.go:121-147` 的 `UpdateModel` 构造的 record 干脆不含 Files（当前无 caller，属地雷式死代码）。需验证 moi-backend 各编辑入口是否总是回传完整 files。

**F2-M3 门控读的字段和写路径保证的字段整体错位**（H1/H2 的抽象）——门控要 `files.vector_table`/`files.file_ids`，写路径保证的是「向量行 + VectorIndex asset + indexed_from 边」，两套账本靠 moi-backend UI 路径这根隐式链粘合。新增任何 ingest 入口（SDK、新模板、agent 工具）都会默认踩空。**建议把「写向量必须伴随 scope 绑定」固化为契约**（如 lineage.register 强制回写，或提供一个原子的 bind API）。

---

## 3. 家族 F3：状态聚合掩盖真实进度

- **#12816**（已上报，含代码线索）：`/source-jobs` 聚合把仍 running 的 load job 隐藏成 succeeded，segment 未生成时 `/sources` 已显示成功；reconcile 后 load job 仍残留 running。落点 moi-backend `ListSourceJobs`/`enrichKnowledgeBaseSourceJobRunsFromLinkedJobs`。
- **F3-L1 dataset 语言标注 best-effort 吞错** — `retrieval_vector.go:1279-1349` `updateDatasetLanguages` 全程 Warn 后继续、无返回值，节点照常成功；若语言过滤参与召回可致静默空结果。影响面小。
- **F3-L2 source job ↔ workflow execution 无对账（moi-core 侧确证缺失）** — moi-core 内没有把 KB source/文件状态随 workflow 终态推进的一致性代码；#12577 验收里「模板状态与知识库 source job 一致」的诉求需在 moi-backend 落，属跨仓缺口。

---

## 4. 与在报 issue 的映射 / 建议动作

| 发现 | 对应 issue | 建议 |
|---|---|---|
| VLM 图片无守卫 | （两起生产失败，未见 issue） | 按 vlm-image-size-guard.md 落地；**建议补开 issue 记录生产故障** |
| #12764 根因链 | #12764 / #12577 / #12813 | 本报告 §2.1-2.2 可直接作为根因评论贴到 #12764；三个 issue 同根因应关联合并处理 |
| F1-H1/H2 structured-extract 无预算 | 无 | 建议开 issue（kind/bug-moi），随 imageprep 家族一起修或单独修 |
| F1-M1 embedding 批无上限 | 相邻 #12802 | 建议在 #12802 修复 PR 里一并处理（同一代码带） |
| F1-M1b 图片 embedding 请求体无界 | #14898 | 已由共享无损图片 Data URL 能力收口；仍需单独关注多图片/批量请求的总字节预算 |
| F1-M2 MinerU 直连 | 无 | 建议开 issue，低优 |
| F1-M3 go-sdk 413 重试/HTML 丢弃 | 无 | 建议开 issue + 独立小 PR，横切收益大 |
| F2-H1/M1/M2/M3 | 无 | M2（全量替换 files）建议优先验证 moi-backend payload 后开 issue；其余随 #12764 主修带出 |
| F3-L1/L2 | #12816 部分覆盖 | L2 跨仓缺口在 #12816/#12577 讨论中点名 |

## 5. 需要运行时验证的项（本报告只做了代码级确证）

1. F2-M1 触发时序：moi-backend 是否保证建 KB 时总先写 `files.vector_table`。
2. F2-M2：前端「编辑语义模型」的 PUT payload 是否携带完整 files。
3. F2 兜底方案的权限语义：governance 记录「全禁用」与「未绑定」在数据上如何区分。
4. #12577 是否与 #12764 完全同根因（拿同模板在本地/QA 复跑一次验证 file_ids 是否为空即可定案）。
