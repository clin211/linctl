# lin 仓库 · linctl 重构设计文档

> **目标**：把本仓库发布的 **linctl**（Go module 仍为 `github.com/clin211/linctl`）从「项目生命周期管理平台」回归到「项目骨架生成器」的本质定位。
>
> **核心矛盾**：当前实现 ~10,525 行 / 14 模块 / 15+ 子命令，远超用户实际需求（仅生成项目骨架 + 添加业务资源两件事）。
>
> **决策结论**（2026-04-29）：定位为「**功能完备 · 但单一聚焦**」的脚手架——保留 lint/doctor、AST 注入、全栈资源、外部模板覆盖；删除 plan/apply 闭环、Feature 系统、配置文件。

---

## 文档索引

| 序号 | 文档 | 状态 | 摘要 |
| --- | --- | --- | --- |
| 00 | [refactor-rationale.md](./00-refactor-rationale.md) | ✅ Stable | 重构 RFC：动机、定位、6 项关键决策（已确定） |
| 01 | [architecture-blueprint.md](./01-architecture-blueprint.md) | ✅ Stable | 重构后整体架构：目录、模块、数据流、依赖 |
| 02 | [command-set.md](./02-command-set.md) | ✅ Stable | 6 个命令详细设计（new/add/lint/doctor/version/completion） |
| 03 | [resource-scaffold.md](./03-resource-scaffold.md) | ✅ Stable | 业务资源全栈分层规范（参照 miniblog-v4） |
| 04 | [template-system.md](./04-template-system.md) | ✅ Stable | 简化的模板系统：embed + 外部目录覆盖 |
| 05 | [registration-strategy.md](./05-registration-strategy.md) | ✅ Stable | 资源注册策略（AST 注入 + 锚点 + 幂等 + 事务） |
| 06 | [migration-plan.md](./06-migration-plan.md) | ✅ Stable | 5 阶段迁移计划（17 工作日） |
| 07 | [interactive-ux.md](./07-interactive-ux.md) | ✅ Stable | `linctl new` / `linctl add` 交互式终端 UX 设计 |

---

## 阅读路径

| 角色 | 推荐路径 |
| --- | --- |
| **想理解为什么要重构** | [00 RFC](./00-refactor-rationale.md) |
| **想看重构后的整体架构** | [00](./00-refactor-rationale.md) → [01](./01-architecture-blueprint.md) |
| **想看 CLI 怎么用** | [00](./00-refactor-rationale.md) → [02](./02-command-set.md) → [03](./03-resource-scaffold.md) |
| **想看 CLI 交互体验** | [02](./02-command-set.md) → [07](./07-interactive-ux.md) |
| **想参与开发** | [01](./01-architecture-blueprint.md) → [04](./04-template-system.md) → [05](./05-registration-strategy.md) |
| **想看怎么从当前迁移** | [00](./00-refactor-rationale.md) → [06](./06-migration-plan.md) |
| **想看资源会生成哪些文件** | [03](./03-resource-scaffold.md) |

---

## 关键数据对比（重构前 vs 重构后）

| 维度 | 重构前 | 重构后 | 变化 |
| --- | --- | --- | --- |
| 代码行数（仅 internal） | ~10,525 | ~3,160 | **-70%** |
| 内部模块数 | 14 | 5-7 | **-50%** |
| 子命令数 | 15+ | 6 | **-60%** |
| 二进制大小 | ~12 MB（目标） | ≤ 6 MB | **-50%** |
| 直接依赖数 | ~13 | ≤ 8 | **-40%** |
| 新人上手时间 | 3-5 天 | 0.5-1 天 | **大幅降低** |

---

## 6 项关键决策（已确定）

| 决策点 | 选择 | 含义 |
| --- | --- | --- |
| §5.1 工具边界 | B. 极简 + lint/doctor | 命令集：new / add / lint / doctor / version / completion |
| §5.2 资源注册策略 | B. AST 注入 | 保留 ast 模块；自动改 biz.go / store.go / proto / errno |
| §5.3 资源完整度 | A. 全栈 | 默认生成 13 文件 + 4 注入 |
| §5.4 模板可定制性 | B. 外部目录覆盖 | `--template-dir` + `.lin/templates/` + `~/.lin/templates/` |
| §5.5 Feature 系统 | A. 完全删除 | 模板用 `{{if eq .Storage}}` 处理变体 |
| §5.6 配置文件 | A. 完全删除 | 元信息从 go.mod / cmd/* 推断 |

详见 [00-refactor-rationale.md §10](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29)。

---

## 历史与归档

- 旧版主线文档（00-overview ~ 15-security-model）将归档到 `lin/docs/legacy/`（在 [Phase 5](./06-migration-plan.md#phase-5上线--文档3-天) 执行）。
- ADR 记录全部保留在 `lin/docs/adr/`，作为历史决策依据。
- 本次重构本身可能在未来产出新的 ADR（如「为什么删除 plan/apply」、「为什么用锚点注释」）。

---

## 反馈

任何对本设计的质疑、补充、改进建议，请在对应文档中以 `> TODO:` 标注或在 chat 中提出。重大变更请：

1. 在 `00-refactor-rationale.md` 的修订历史中加一行；
2. 如涉及关键决策，写新的 ADR。

---

_Last reviewed: 2026-04-29_
