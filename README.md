<p align="center">
  <img src="assets/architecture/casebook-banner.png" alt="MatrixOne Intelligence Casebook" width="100%" />
</p>

<div align="center">

# MatrixOne Intelligence Casebook

### 企业级 AI 数据产品案例集

**多模态文档解析 · 可溯源 RAG · Agentic Workflow · 评测驱动迭代**

[案例总览](docs/00-portfolio/case-map.md) · [能力证据矩阵](docs/00-portfolio/jd-evidence-matrix.md) · [文档智能](docs/02-document-intelligence/) · [Agent 工作流](docs/03-agentic-data-workflow/) · [解决方案 PoC](docs/04-solution-poc/) · [English](README_EN.md)

</div>

<p align="center">
  <img src="assets/architecture/casebook-overview.png" alt="从企业数据到可信 AI 交付的产品闭环" width="900" />
</p>

> [!IMPORTANT]
> 这是一个待公开发布的个人作品集草案。所有内容须独立重建、完成脱敏并取得必要授权后才能发布；仓库绝不包含雇主代码、客户数据、内部文档、非公开指标或凭据。

## 从企业数据到可信 AI 交付

企业级 AI 产品的难点从来不只是接入模型，而是让数据、模型、工具、工作流和人的判断在一个可验证的闭环中协作。

本案例集围绕三个问题展开：

1. 如何把复杂企业文档转化为可追溯、可复核的 AI-Ready 数据？
2. 如何让 Agent 在调用工具、管理任务状态和处理失败时保持可控？
3. 如何把产品能力组织成客户可理解、可验收、可复制的 AI 解决方案？

## 三个案例

| 案例 | 要解决的问题 | 面试官可验证的产物 |
| --- | --- | --- |
| [01 · Document Intelligence](docs/02-document-intelligence/) | 复杂文档解析、结构化提取、检索问答与引用溯源。 | 架构、PRD、Prompt 版本、Golden Set、Rubric 与 Badcase。 |
| [02 · Agentic Data Workflow](docs/03-agentic-data-workflow/) | 多步骤任务中的规划、工具调用、状态追踪与人工接管。 | Agent 设计、工作流、工具契约、失败恢复与评测方案。 |
| [03 · Enterprise AI Solution PoC](docs/04-solution-poc/) | 将企业问题转化为可落地、可验收、可推广的 AI 方案。 | 需求澄清、方案设计、PoC、验收标准、推广与风险管理。 |

## AI Native 能力，不只是一份技术清单

| 能力 | 这里如何证明 |
| --- | --- |
| RAG | 文档接入、切片、检索、重排、引用溯源与无答案兜底。 |
| Prompt 工程 | 可版本化的提示词、示例、结构化约束，以及前后效果对比。 |
| 评测 | Golden Set、评分 Rubric、人工/自动复核、Badcase 与回归测试。 |
| Agent | 规划、工具调用、任务状态、重试、失败恢复和人工审批。 |
| Workflow | 输入输出 Schema、分支、重试、审计与可观测的节点状态。 |
| 技术选型 | 模型能力、上下文、延迟、成本、部署和安全之间的产品取舍。 |
| 解决方案 | 场景定义、PoC 范围、ROI、验收、培训和规模化推广。 |

完整映射见 [JD Evidence Matrix](docs/00-portfolio/jd-evidence-matrix.md)。

## 5 分钟阅读路径

```text
产品定位与能力地图
        ↓
Document Intelligence：从原始文档到可评测的 AI 输出
        ↓
Agentic Data Workflow：从单次回答到可控任务执行
        ↓
Solution PoC：从产品能力到客户可验收交付
```

1. 阅读 [产品定位](docs/00-portfolio/positioning.md) 和 [能力证据矩阵](docs/00-portfolio/jd-evidence-matrix.md)。
2. 查看文档智能案例中的架构、评测方案和 Badcase 复盘。
3. 查看 Agent 的工具契约、工作流状态和人工接管设计。
4. 最后进入解决方案 PoC，理解能力如何变成客户价值与验收结果。

## 仓库结构

```text
docs/       产品、方案、PoC 与交付文档
apps/       后续可运行的文档智能与 Agent Demo
packages/   Schema、Prompt Library 与 Evaluation Kit
evals/      Golden Set、Rubric、Badcase、回归结果
data/       仅公开授权或完全自制的样例数据
assets/     重新绘制的架构图、工作流图、截图与演示
scripts/    数据准备、评测、报告与公开发布检查
tests/      单元、集成、端到端和回归测试
```

## 当前进度

- [x] 搭建案例、代码、评测和公开发布检查的目录骨架。
- [x] 定义 JD 能力到公开证据的映射。
- [ ] 从源材料独立重写案例叙事与个人贡献边界。
- [ ] 使用公开或合成数据补全 Demo、Golden Set 与评测结果。
- [ ] 重新绘制架构图与工作流图。
- [ ] 通过公开发布检查后再创建 GitHub 仓库。

## Public Release Boundary

这是个人作品集，不是产品资料归档。提交任何内容前，请先阅读 [公开发布与保密边界](DISCLAIMER.md) 与 [数据来源及许可证清单](docs/05-references/data-license-manifest.md)。
