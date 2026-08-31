# Storybook 运行、证据与报告规范

本规范定义一次 Storybook 运行应如何产生可复盘、可脱敏、可机器消费的结论。它适用于 UI、SDK/API 或混合执行方式；执行实现可以不同，但产物语义必须一致。

Case 的内容合同见 [Case 模板](case-template.md)，整体设计原则见 [测试方法论](testing-methodology.md)。

## 1. 运行结果语义

| 结果 | 含义 | 对 `required` 门禁的处理 |
|---|---|---|
| `pass` | 所有适用断言通过，且清理结果符合合同 | 通过 |
| `fail` | 产品行为、输出、Case 合同或清理不满足要求 | 阻断 |
| `blocked` | 已在前置阶段取得明确环境依赖不足的证据，未开始受控产品操作 | 阻断，需处理或正式豁免 |
| `not run` | 未取得本次运行证据 | 阻断 |

`blocked` 不能被用作“环境不稳定所以跳过”的通用标签；输入不合法、Case 配置错误、权限设计错误、产品返回失败或业务断言失败都应为 `fail`。

## 2. 统一阶段模型

所有执行器至少应报告以下阶段；没有执行的阶段必须带原因，而不是静默省略。

| 阶段 ID | 目的 | 最少证据 |
|---|---|---|
| `validate_preconditions` | 在副作用前验证能力、权限、依赖和配额 | 检查项、版本/能力摘要、失败原因 |
| `prepare_assets` | 校验夹具并准备隔离输入 | 夹具摘要、摘要校验、输入计数 |
| `create_resources` | 创建本次拥有的资源 | run ID、资源类型与匿名标识、所有权 |
| `run_workflow` | 执行产品 Workflow 或等价任务 | workflow/task ID、完整终态、节点/执行摘要 |
| `create_knowledge` | 创建并确认 Knowledge/索引（适用时） | knowledge ID、索引状态、选择来源摘要 |
| `create_agent` | 创建或绑定受控 Agent（适用时） | agent/version/binding 摘要 |
| `query_knowledge` | 进行检索/问答及事实验证（适用时） | 查询摘要、引用/工具证据、断言结果 |
| `run_agent_journey` | 执行多轮旅程及受控副作用（适用时） | 轮次、工具调用、回读确认 |
| `assert_contract` | 汇总生命周期、合同、事实、安全断言 | 逐条断言结果 |
| `cleanup_resources` | 逆依赖回收和确认 | 结构化清理记录 |

阶段记录使用统一字段：`id`、`title`、`status`、`started_at`、`ended_at`、`duration_seconds`；可附加经过脱敏的 `details`。耗时使用单调时钟计算，展示时间使用带时区 ISO-8601。

## 3. 报告最小合同

一次运行应输出机器可读 JSON；可选输出面向人的 Markdown 摘要。建议目录按 `<case-id>/<run-id>/` 隔离，报告和公共证据中均不得出现秘密。

```json
{
  "schema_version": "1.0",
  "case_id": "SB-EXAMPLE-001",
  "case_version": "1",
  "run_id": "opaque-run-id",
  "status": "pass",
  "started_at": "2026-01-01T00:00:00+08:00",
  "ended_at": "2026-01-01T00:02:00+08:00",
  "duration_seconds": 120.0,
  "environment": {
    "product_version": "<sanitized-version>",
    "runner_version": "<version>",
    "model_or_pool": "<approved-identifier>"
  },
  "fixture": [{"name": "<logical-name>", "sha256": "<digest>", "role": "business_input"}],
  "steps": [],
  "assertions": [],
  "artifacts": [],
  "cleanup_records": [],
  "failure": null
}
```

字段可以扩展，但不可改变既有字段语义。`run_id`、资源 ID 与请求 ID 如有敏感性，应使用不可逆的关联别名或在受控内部报告中保存原值。

### 3.1 断言记录

每条断言必须独立记录，避免只给一个笼统的“全部通过”。

```json
{
  "id": "A03",
  "layer": "business_fact",
  "status": "passed",
  "summary": "合同编号与金标一致",
  "expected": {"equals": "MOI-2026-001"},
  "actual": {"value_digest": "<or-sanitized-value>"},
  "evidence_refs": ["artifact:output-json", "execution:opaque-id"]
}
```

允许的 `layer`：`lifecycle`、`product_contract`、`business_fact`、`security_side_effect`、`quality_metric`、`cleanup`、`manual`。`manual` 不能决定自动运行的 `pass`。

### 3.2 失败记录

```json
{
  "category": "product_output_contract_violation",
  "phase": "assert_contract",
  "message": "输出缺少必填字段",
  "product_code": "<optional-code>",
  "request_ref": "<opaque-request-ref>",
  "diagnostic_summary": "<redacted-and-bounded-text>"
}
```

失败分类应优先使用结构化错误码、HTTP 状态和明确的产品终态。异常文本只能提供辅助诊断，不能成为唯一的长期分类协议。

## 4. 证据最小化与脱敏

证据的目标是支持结论，不是复制整个执行环境。默认保存最小必要信息：

| 证据类型 | 公共 Casebook 可记录 | 禁止公开 |
|---|---|---|
| 输入 | 逻辑名、角色、格式、版本、SHA-256、合成说明 | 原始客户资料、个人信息、未脱敏内容 |
| 配置 | 模板/Schema 摘要、公开版本、能力标识 | URL、API Key、Token、内部开关细节 |
| 运行 | 匿名关联 ID、阶段状态、耗时、计数 | 原始请求体、完整响应、会话内容、内部日志 |
| 输出 | Schema 结果、摘要、字段级判定、文件元数据 | 受保护的下载 URL、完整敏感产物 |
| 失败 | 分类、脱敏消息、可行动建议 | 凭据、堆栈中的秘密、人员身份 |

执行器应在写盘前进行秘密扫描和字段白名单化。脱敏不是简单字符串替换：应尽量不采集不需要的数据，并对需要关联的敏感 ID 使用受控映射或不可逆摘要。

## 5. 异步、重试与超时

- 轮询异步任务时必须读取完整分页结果，验证总数、重复 ID、终态和超时；不得只读取首条任务或以“请求已提交”判定成功。
- 只对明确的瞬时基础设施错误进行有限重试和退避；每次尝试都要记录次数、原因和最终结论。
- 业务断言失败、Schema 违例、权限拒绝、输入错误及失败终态不得用重试掩盖。
- 超时应为确定的 `fail`，报告最后可见状态和取消/回收结果；不要留下悬挂任务。
- 多次独立采样仅用于 Case 已声明的质量指标；不能取最好的一次来决定通过。

## 6. 清理协议

每个由本次运行创建的资源都要记录 `resource_kind`、匿名 `resource_ref`、`action`、`status`、`message` 与 `confirmed_absent`。推荐状态为：`deleted`、`confirmed_absent`、`kept`、`deferred`、`skipped`、`blocked`、`failed`。

清理按依赖逆序进行。例如先解除会话、Agent 与外部绑定，再删除 Knowledge/导入任务，最后回收 Catalog、Database、Connector 或临时文件。若主业务断言已失败，清理失败作为 `secondary_failure` 保留；若主流程通过但清理失败，整体结果必须失败。

## 7. 发布与复盘消费

发布系统应从 JSON 报告读取 `required` Case 的实际结果，不得只计算平均指标。最小门禁规则是：

- 所有适用的 `required` Case 均为 `pass`；
- 无未审查的 `blocked` 或 `not run`；
- 无 `resource_cleanup_incomplete`；
- 关联的质量基准满足该版本已批准的阈值或差异规则。

事故或回归复盘应链接到 Case ID、版本、失败分类、受影响承诺、修复提交和后续运行结论。若事故不能沉淀为公开 Case，需写明数据、安全或环境原因，并在受控内部体系保留对应回归覆盖。
