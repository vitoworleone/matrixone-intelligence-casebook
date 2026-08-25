---
name: kb-product-matrix
description: |
  Knowledge Base 产品功能矩阵：用 Product SDK 对 live 本地栈做端到端验收。
  覆盖知识库生命周期、catalog 表入库、结构化本地上传建表、本地文档上传、
  文档知识库准备（standard_rag）与带图片索引准备（standard_rag_with_image_index）、
  catalog_file 血缘与内置 RAG 工作流执行、A2A 问答引用（含图片索引 KB）、
  追加/删除源、语义条目。
  知识库功能开发完成后、合并前或本地回归时运行本矩阵。

  Trigger: "知识库验收", "KB 矩阵", "kb product matrix", "知识库 SDK 验收",
  "跑知识库功能矩阵", "kb-product-matrix", "知识库回归".
  NOT for: 纯单测、mock、只改前端样式且无后端契约变化、非知识库子系统。
---

# Knowledge Base Product Matrix (SDK)

用 **Product SDK** 打 **真实本地服务**，不要 mock、不要只跑 go test。

## 何时使用

- 知识库相关开发完成后做产品验收
- 重启服务 / 换代码后回归
- 用户要求「用 SDK 验知识库」

## 前置

1. UC local-deploy 已启动（catalog / mowl / go-worker / moi-backend / frontend）。
2. 推荐 profile：`kb_catalog_lineage_acceptance`（或当前分支 profile）。
3. 样本文件存在：
   - `optools/matrixflow/moi-connector/sample_files/MatrixOne_Introduction.pdf`（标准文档准备 / M5）
   - `optools/matrixflow/moi-connector/sample_files/MatrixOne 简介（图文混合）.pdf`（图片索引文档准备 / M10）
   - `optools/matrixflow/moi-connector/agent_home_guide_sample_files/team_member_map.csv`
4. Python 可 `import requests`（PAT 签发）。可用 lineage runner venv 或 skill runner venv：
   - `skills/kb-product-matrix/runner/venv/bin/python`
   - `.runtime/kb-catalog-lineage-acceptance-runner/venv/bin/python`
5. seed 账号 UC **active control API key 必须有空位**（环境上限通常 `max=5`）。
   `run-matrix.sh` **默认在跑矩阵前自动回收临时 PAT**（见下节）；不要把 PAT 顶满当成产品失败。

重启服务（需要时）：

```bash
bash skills/local-deploy/scripts/restart-local.sh --profile kb_catalog_lineage_acceptance all
# 若 frontend 端口被占用：先释放端口再
bash skills/local-deploy/scripts/restart-local.sh --profile kb_catalog_lineage_acceptance frontend
```

## 一键跑矩阵

```bash
# 默认读 .runtime/local-deploy.uc.env 的端口与 profile
# 会先做 UC 临时 PAT 回收预检，再跑 M1–M11
bash skills/kb-product-matrix/scripts/run-matrix.sh
```

常用环境变量：

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `LOCAL_DEPLOY_PROFILE` / `UC_PROFILE` | 来自 env 或 `kb_catalog_lineage_acceptance` | seed 邮箱 `local-admin+<profile>@matrixflow.local` |
| `AISTUDIO_PORT` | `19050` | moi-backend |
| `AISTUDIO_PUBLIC_URL` | `http://localhost:18000` | FE；PAT OIDC 必须走 FE `/newmoi` |
| `UC_PORT` | `19080` | UC |
| `SEED_EMAIL` / `SEED_PASSWORD` | profile admin / `Admin@1234` | UC 种子账号 |
| `PYTHON_BIN` | 自动探测 venv 或 `python3` | 需 `requests` |
| `MATRIX_OUT` | `output/kb-product-matrix` | 报告目录（**gitignore，不入库**） |
| `MATRIX_CLEANUP` | `ask` | `yes` 跑完即删产物；`no` 保留；`ask` 交互询问 / 非交互则留给 Agent 问用户 |
| `MATRIX_PAT_RECLAIM` | `auto` | PAT 预检：`auto` 腾出空位；`force` 撤销全部临时 key；`list` 只列；`no` 跳过 |
| `MATRIX_PAT_RESERVE` | `1` | 至少保留几个空槽给本次 `issue_pat` |
| `MATRIX_PAT_TEMP_PREFIXES` | 见回收脚本默认 | 逗号分隔，可扩展临时 key 名前缀 |
| `MATRIX_PAT_KEEP_PREFIXES` | `AI Studio Runtime,...` | 永不自动撤销的前缀 |
| `SAMPLE_PDF` / `SAMPLE_PDF_IMAGE` / `SAMPLE_CSV` | 仓库内样本路径 | 可覆盖；`SAMPLE_PDF_IMAGE` 用于 M10 图文混合 PDF |

退出码：`0` = 全绿；非 0 = 有失败或前置不满足。

辅助子命令（不跑矩阵）：

```bash
# 只查看 seed 账号 active PAT
bash skills/kb-product-matrix/scripts/run-matrix.sh --list-pats-only

# 只回收临时 PAT（腾槽），不跑 M1–M11
bash skills/kb-product-matrix/scripts/run-matrix.sh --reclaim-pats-only

# 强制撤销所有匹配临时前缀的 active key
MATRIX_PAT_RECLAIM=force bash skills/kb-product-matrix/scripts/run-matrix.sh --reclaim-pats-only
```

### 故障：UC active PAT 已达到环境上限（会阻塞真实回归）

**这不是知识库产品回归失败。** 矩阵在签发 PAT 阶段就退出，**M1–M11 一个 cell 都不会跑**。

| 项 | 内容 |
| --- | --- |
| 典型报错 | `UCPATError: UC active PAT 已达到环境上限: active=5, max=5` 或 `MATRIX FAILED: issue PAT: ...` |
| 根因 | UC 对账号限制 `control_api_key_max_active`（通常 5）。手测 / 半截脚本 / 未走完 revoke 的临时 key 仍占 **active** 槽；**过期不会自动腾槽**，必须 DELETE revoke |
| 常见占槽名前缀 | `kb-matrix-acc-*`、`kb-accept-*`、`kb-lineage-acc-*`、`kb-dbchk-*`、`moi-tester-*` |
| 保留 | `AI Studio Runtime`（及 `MATRIX_PAT_KEEP_PREFIXES` 配置的前缀）— **不要**当垃圾删 |
| 默认防护 | `run-matrix.sh` 启动时 `MATRIX_PAT_RECLAIM=auto`：若 `free_slots < MATRIX_PAT_RESERVE`，只撤销临时前缀 key 直到腾出空位 |
| 一键修复 | `bash skills/kb-product-matrix/scripts/run-matrix.sh --reclaim-pats-only` 然后重跑矩阵 |
| 仍失败时 | `--list-pats-only` 看 active 列表；非临时前缀的 key 需在 UC UI/API 手撤，或把前缀加入 `MATRIX_PAT_TEMP_PREFIXES` 后再 `--reclaim-pats-only` |
| Agent 约定 | 见报错先跑 reclaim，**禁止**据此改 KB 业务代码“凑绿” |

实现：`skills/kb-product-matrix/runner/reclaim_temp_pats.py`（login → list → 按前缀 DELETE）。

### 其它常见前置失败（也不是产品 cell 失败）

| 症状 | 处理 |
| --- | --- |
| backend HTTP 非 401/200 | 先 `local-deploy` / `restart-local.sh` 起栈 |
| frontend 非 401/200/302 | PAT OIDC 必须走 FE `/newmoi`；检查 `AISTUDIO_PUBLIC_URL` 与 frontend 端口 |
| `ErrUserNotSynced` / 用户未同步 | 新 profile 需先浏览器登录或 OIDC 产品 session，再签发 PAT |
| 样本 PDF/CSV 缺失 | 确认 `optools/...` 样本路径存在或覆盖 `SAMPLE_PDF` / `SAMPLE_PDF_IMAGE` / `SAMPLE_CSV` |

### 产物与 Git

执行产物**不得**提交 git（根 `.gitignore` + skill `.gitignore`）：

| 路径 | 说明 |
| --- | --- |
| `output/kb-product-matrix/` | 报告目录（`matrix-report.json` / `matrix-summary.json` / `debug-lineage-topology.json` 等） |
| `skills/kb-product-matrix/runner/kb-product-matrix` | `run-matrix.sh` 本地 build 二进制 |
| `skills/kb-product-matrix/runner/kb-product-matrix-runner` | 手建/模块名二进制（与 go.mod module 同名）；**不得入库** |
| `skills/kb-product-matrix/runner/revoke_pat_once.py` | 运行时生成的临时 revoke 脚本 |
| `skills/kb-product-matrix/runner/reclaim_temp_pats.py` | **入库**；预检/手动腾出 UC PAT 槽位 |
| `skills/kb-product-matrix/runner/venv/` | 可选本地 venv |

金样（可入库）：`skills/kb-product-matrix/testdata/last-passed-matrix-report.json`（脱敏通过样例，不含 token）。

### 跑完后清理（必须问用户）

脚本结束时：

1. **TTY 交互**：直接提示 `cleanup artifacts? [y/N]`，用户确认才删。
2. **非交互（Agent/CI）**：打印 `CLEANUP_PROMPT` 块，**默认保留**；Agent **必须先问用户**，再按答案执行：
   - 清理：`bash skills/kb-product-matrix/scripts/run-matrix.sh --cleanup-only`
   - 或：`MATRIX_CLEANUP=yes bash skills/kb-product-matrix/scripts/run-matrix.sh`（下次跑完自动删）
   - 保留：无需操作。

产物内容：

- `matrix-report.json` — 全量 checks + artifacts
- `matrix-summary.json` — stdout 摘要
- stderr 行：`[matrix] PASS|FAIL cell/name — detail`，结束行 `MATRIX PASSED (n/n checks)`

## 产品功能矩阵（cells）

| Cell | 产品场景 | SDK 主路径 | 通过条件（摘要） |
| --- | --- | --- | --- |
| **M1** | 知识库生命周期 | `Create` / `Get` / `List` / `Update` / `Delete` | 空库 CRUD 成功且物理 database 位于系统默认 catalog；同名 Update 成功；改名 Update 返回 400/`ErrParamInvalid` 且名称不变 |
| **M2** | 添加 Catalog 表 | `CreateTable` + `CreateWithSources(catalog_table)` | 源列表含 table；data_domain catalog = 系统默认 catalog；**KB 名 = 物理库名**（`Name` 为物理标识；`display_name` 可为投影，不要求为空） |
| **M3** | 本地上传结构化文件建表 | `Connectors.LocalUpload` + `CreateWithSources(local_file, upload_kind=structured)` | data_domain catalog = 系统默认 catalog；job/源 settle；出现 table 源且 ingest/table_id 有效 |
| **M4** | 本地上传文档 | `UploadLocalFile` + `CreateWithSources(local_file)` | data_domain catalog = 系统默认 catalog；`ingest_status=succeeded` |
| **M5** | 文档知识库准备（`standard_rag`）+ 血缘 | Volume 上传 + `CreateWithSources(catalog_file, VolumeID)`（无 image index） | data_domain catalog = 系统默认 catalog；内置模板 `standard_rag` 存在；jobs settle；ingest succeeded；raw volume **不**挂源文件；lineage source→output；artifact `workflow_role=output`；恰好 1 条新 execution 且 `completed` |
| **M6** | 知识库对话（A2A，标准文档 KB） | explore `StreamText` + session（绑 M5 model） | final=`completed`；答案含 `80%`；`source_refs` 命中 M5 源文件 |
| **M7** | 追加 Catalog 源 | `AppendSources(catalog_file)` | 源列表出现新文件 row |
| **M8** | 语义条目 | `CreateEntry` / `ListEntries` / `DeleteEntry` | 创建→列表可见→删除 |
| **M9** | 删除源 | `DeleteSource` | 追加源删除后列表消失 |
| **M10** | 文档知识库准备（带图片索引，`standard_rag_with_image_index`） | 图文 PDF + `CreateWithSources(..., ImageIndexEnabled=true)` | data_domain catalog = 系统默认 catalog；内置模板 `standard_rag_with_image_index` 存在；model `files` 含 image index 配置；jobs settle；ingest succeeded；恰好 1 条新 execution 且 `completed` |
| **M11** | 知识库对话（A2A，图片索引 KB） | explore `StreamText` + session（绑 M10 model） | final=`completed`；答案含 `80%`；`source_refs` 命中 M10 源文件 |

实现：`skills/kb-product-matrix/runner/main.go`。

### 核心路径对照（防回归）

| 用户可见能力 | Cell | 关键 SDK / 契约 |
| --- | --- | --- |
| 创建空知识库 / 改名拒绝 / 删除 | M1 | `Create` / `Update` / `Delete` |
| Catalog 表入库 | M2 | `CreateWithSources(catalog_table)` |
| 结构化本地上传建表 | M3 | `Connectors.LocalUpload` + structured source |
| 本地文档上传入库 | M4 | `UploadLocalFile` + `CreateWithSources(local_file)` |
| **文档知识库准备**（Prepare Document Knowledge Base） | M5 | `CreateWithSources(catalog_file)` → 模板 `standard_rag` → 工作流 execution completed |
| **文档知识库准备后问答** | M6 | explore A2A `StreamText` + `semantic_model_ids` |
| 追加 / 删源 / 语义条目 | M7–M9 | `AppendSources` / `DeleteSource` / entries |
| **文档知识库准备（带图片索引）** | M10 | `WithSemanticModelWithSourcesImageIndexEnabled(true)` → 模板 `standard_rag_with_image_index` |
| **带图片索引知识库问答** | M11 | 同 M6，绑定 M10 model |

## Agent 操作约定

1. **先 live 服务健康**，再跑矩阵；不要在服务未启动时改矩阵去“绿”。
2. **先区分前置失败 vs 产品失败**：
   - `issue PAT` / `active PAT 已达到环境上限` → 跑 `--reclaim-pats-only`（或依赖默认 `MATRIX_PAT_RECLAIM=auto`），**不要改 KB 代码**。
   - 服务 401/端口不通 → 先 local-deploy，**不要改矩阵断言**。
   - `pass=false` 的 M1–M11 cell → 才查 backend/catalog/go-worker 日志，修产品代码后 **重跑本脚本**。
3. 失败时：读 `matrix-report.json` 里 `pass=false` 的 cell；若报告几乎为空且 stderr 只有 `issue PAT`，按上一节 PAT 故障处理。
4. 新增知识库产品能力时：在 `runner/main.go` 增加 cell（`M12+`），更新本表，再跑通后才算验收完成。
5. 不要把 PAT、seed 密码写进 git 报告；`matrix-report.json` 可含 workspace/file id，不含 token。产物目录已 gitignore，**禁止** `git add output/kb-product-matrix`。
6. 跑完后若 stderr 出现 `CLEANUP_PROMPT`：**先询问用户是否清理本地产物**，得到明确 yes 再 `--cleanup-only`；未问清楚前不得擅自 `rm -rf`。
7. 本矩阵是 **API/SDK 产品验收**，不替代 Playwright UI 截图；UI 手测用 FE URL + 同 seed 账号。
8. 手测/半截脚本若自己 `issue_uc_pat`，**必须**在结束时 revoke；否则下次矩阵会被 max=5 卡住。
9. **M5/M6 与 M10/M11 成对**：文档准备工作流未 completed 就不要单独把问答算绿；图片索引路径不得只测创建不测 settle/execution/问答。

## 与其它验收的关系

| 工具 | 范围 |
| --- | --- |
| 本 skill | 知识库产品矩阵（多路径，含标准/图片索引文档准备 + 问答） |
| `.runtime/kb-catalog-lineage-acceptance-runner` | 历史 lineage 单路径验收（可被 M5+M6 覆盖） |
| `go test ./pkg/session` 等 | 单元/集成，不替代本矩阵 |

## 已知边界（未强制纳入矩阵）

以下若任务涉及，需在矩阵外单开验收或后续加 cell：

- 音频 / 视频专用 RAG 模板（`document_visual_rag` 等非知识库创建主路径）
- 既有知识库手动删旧 workflow 后自动重建模板
- 跨知识库向量复用、治理过期策略的全组合
- 分段编辑 / 重新 embed / 治理策略细粒度 API
- 浏览器 UI 交互与截图

## 手测入口（矩阵跑完后）

- FE：`AISTUDIO_PUBLIC_URL`（默认 `http://localhost:18000`）
- 账号：`local-admin+<profile>@matrixflow.local` / `Admin@1234`
- 报告 `artifacts` 中有 `m5_model_id`、`m10_model_id`、`workspace_id` 等可拼知识库 URL
