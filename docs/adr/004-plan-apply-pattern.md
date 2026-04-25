# ADR-004: 引入 Plan/Apply 两阶段流水线（Terraform 模式）

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态** | ✅ Accepted (2026-04-25) |
| **日期** | 2026-04-25 |
| **作者** | @clin211 |
| **审阅人** | TBD |
| **相关 ADR** | [ADR-005](./005-feature-as-first-class.md)（Feature 系统是 Plan 阶段计算的基础） |
| **影响范围** | `internal/codegen/`、`internal/orchestrator/`、CLI 命令体系 |
| **相关 issue/PR** | - |

---

## 1. 背景

osbuilder 的 `create project` / `create api` 等命令是 **"一遍走到底"** 模式：

```
Run() → 渲染 → 写文件
```

这导致以下问题：

1. **不可预览**：用户无法在不写文件的情况下知道"这次执行会动哪些文件"。
2. **冲突无策略**：遇到已存在的文件，要么 skip 要么 `--force` 全覆盖；没有"我看一下 diff 再决定"的中间态。
3. **失败难回滚**：写到一半出错，半成品文件留在磁盘上，用户要手动清理。
4. **Drift 不可见**：用户改过的文件 vs 模板版本 vs 期望版本之间的关系完全模糊。
5. **CI 场景痛**：CI 想"先 plan 看看是否需要 apply"做不到。

## 2. 决策

我们决定借鉴 **Terraform 的 Plan/Apply 两阶段** 设计：

```
declaration → plan → review → apply → reconcile
```

关键变更：

1. 引入 `*Plan` 数据结构，是"待执行变更的清单"，包含每个文件的 Action（Create/Update/Skip/Conflict/Delete）+ Reason + DiffPreview + Hash 信息。
2. 所有命令（含 `new` / `add` / `apply`）的执行流程都拆为 5 段式：`Complete → Validate → Plan → Apply → Report`。
3. 单独的 `linctl plan` 命令仅执行 Plan + Report，不写文件。
4. `linctl apply` 在 Apply 前再次 `Plan`（重新计算磁盘当前状态），输出与 `linctl plan` 一致的列表后开始执行。
5. 在 Apply 阶段引入 4 种冲突策略：`skip` / `overwrite` / `merge` / `ask`。
6. 引入 hash 注释 `// linctl: hash=xxx` 实现 drift 检测，让"用户改过 vs 没改过"可机器判定。

## 3. 备选方案

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | **Plan/Apply 两阶段**（chosen） | 可预览；可干涉；可回滚；CI 友好；与 K8s/Terraform 心智一致 | 实现复杂度高；引入 hash + lock.yaml 等基础设施 | - |
| 2 | 保留 osbuilder 一遍式，仅加 `--dry-run` | 改动最小 | dry-run 只能 print 命令，无法显示真实 diff；不能解决冲突问题 | 治标不治本 |
| 3 | 三阶段：Plan → Confirm → Apply（独立 Confirm 命令） | 更明确的人机交互 | 用户大多数时候不需要 Confirm；多一步命令 | 与 Apply 内的 ask strategy 重复 |
| 4 | Apply 时打印 plan 后立即执行（无独立 plan 命令） | 简单 | CI 场景不能"只看 plan 不执行" | 限制使用场景 |

## 4. 后果

### 4.1 正面

- **声明式心智**：用户改 `linctl.yaml` → `linctl plan` 看变化 → `linctl apply` 提交。完整的 GitOps 闭环。
- **可干预**：用户在 `--strategy=ask` 时对每个冲突文件单独决策。
- **CI 友好**：CI 可以 `linctl plan --output json`（或 `-o json`）解析后做条件执行（如"有 conflict 才人工介入"）。
- **可回滚**：每次 apply 自动 backup 到 `.linctl/backups/<ts>/`，失败可一键恢复。
- **生态对齐**：Terraform / Pulumi / Helm 用户立刻理解。

### 4.2 负面 / Trade-offs

- **实现复杂度**：需要实现 `Plan` 计算、冲突检测、3-way merge、hash 注释、lock.yaml 等基础设施。
- **首次性能**：plan 阶段需要扫描全部期望生成的文件 + 渲染 + hash 计算。预算 < 2s（1000 文件项目）。
- **学习成本**：用户需要理解 `plan` 与 `apply` 的区别，相比"一行 generate" 多一步。
- **磁盘开销**：lock.yaml + backups 占用磁盘空间（可通过 retention 策略控制）。

### 4.3 中性

- 引入新概念：[Plan](../99-glossary.md#plan-名词) / [Action](../99-glossary.md#action) / [Drift](../99-glossary.md#drift) / [ConflictStrategy](../99-glossary.md#conflictstrategy)。
- CLI 命令数从 osbuilder 的 `create*` 一族扩展到 `new` / `add` / `plan` / `apply` 四个动词。

## 5. 能力分级（Capability Tiers）

> **术语澄清**：本节使用 **Tier**（能力分级）而非 Phase（时间阶段），以避免与 [11-implementation-plan.md](../11-implementation-plan.md) 中的"Phase 1~5（迭代时间）"混淆。Tier 描述 **Plan/Apply 流水线本身的能力增量**；Phase 描述 **整个 linctl 项目按时间推进的迭代节点**。两者关系见下方"5.1 与实施计划对应表"。

| Tier | 能力范围 |
| --- | --- |
| Tier 1 | Plan 计算、`Create` / `Update` / `Skip` 三种 Action；冲突策略仅 `skip` / `overwrite`（Update 通过模板渲染结果与磁盘内容直接 hash 对比判定，**不依赖** embedded hash） |
| Tier 2 | 引入 hash 注释（embedded hash + drift 区分用户改动 vs 模板更新）；新增 `Conflict` Action；冲突策略加 `ask`；`.linctl/lock.yaml` 持久化 |
| Tier 3 | `apply --prune` 删除 obsolete 文件；3-way merge；冲突策略加 `merge`；`linctl restore`；`.linctl/backups/<ts>/` retention；完整 drift detection |

### 5.1 Tier 与实施计划 Phase 对应表

> Tier 的能力增量并不与实施计划的 Phase 一一映射，因为 Phase 还包含 framework / component / deploy 等正交工作。

| Tier | 主要落地的实施计划 Story | 备注 |
| --- | --- | --- |
| Tier 1 | Phase 1 - Story 1.7（Codegen Pipeline 简版） | Plan 数据结构与 internal Apply；不暴露 `linctl plan` / `linctl apply` 子命令 |
| Tier 2 | Phase 2 - Story 2.5（冲突策略 ask + skip + overwrite）+ Phase 4 - Story 4.3（Drift 检测、`.linctl/lock.yaml`） | Story 2.5 引入 hash + ask；Story 4.3 把 lock.yaml 持久化与 drift 检测对齐 Tier 2 全集 |
| Tier 3 | Phase 4 - Story 4.1（`linctl plan` 命令）+ Story 4.2（`linctl apply` 含 `--prune` / `--backup-dir`）+ Story 4.4（3-way merge）+ Story 4.5（`linctl restore`） | Tier 3 是 plan/apply 流水线的"完全形态"；与 v0.9 RC 同步 |

> 详见 [11-implementation-plan.md](../11-implementation-plan.md)。

## 6. 风险缓解

- **不破坏初学者体验**：`linctl new` 默认情况下 plan + apply 一气呵成，初学者无感。只有 `linctl plan` 显式被调用时才停在 plan 阶段。
- **plan 与 apply 的一致性**：apply 内部再 plan 一次，确保用户看到的 plan 与实际执行一致；如果差异（说明磁盘有变），明确警告。
- **性能风险**：渲染阶段并发执行（errgroup + GOMAXPROCS 限速）；hash 计算用 sha256 硬件加速（CPU 上 ≥1 GB/s）。

## 7. 参考资料

- [Terraform Plan and Apply 文档](https://developer.hashicorp.com/terraform/cli/run)
- [Pulumi Preview vs Update](https://www.pulumi.com/docs/cli/commands/pulumi_preview/)
- [设计文档：06-codegen-pipeline.md](../06-codegen-pipeline.md)
- [设计文档：03-cli-design.md §3.3.7-3.3.8](../03-cli-design.md)

---

_Last reviewed: 2026-04-25_
