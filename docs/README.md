# linctl 设计文档

> 一款受 osbuilder/kubebuilder 启发、面向"**声明式、可计划、可回滚、插件化**"的下一代 Go 项目脚手架 CLI 工具。

## 文档导航

本文档集系统性地描述 `linctl` 从概念到落地的完整设计。**强烈建议按顺序阅读**。

### 主线文档（00-15）

| 编号 | 文档 | 主题 | 受众 |
| --- | --- | --- | --- |
| 00 | [overview.md](./00-overview.md) | 项目总览：定位、价值主张、目标用户、与 osbuilder 对比 | 所有人 |
| 01 | [architecture.md](./01-architecture.md) | 系统架构：分层设计、4 张架构图、关键抽象 | 架构师、核心开发 |
| 02 | [project-structure.md](./02-project-structure.md) | 代码组织：目录结构、模块职责、依赖方向 | 核心开发 |
| 03 | [cli-design.md](./03-cli-design.md) | CLI 命令体系：命令树、子命令规范、UX 细节 | 核心开发、UX |
| 04 | [config-schema.md](./04-config-schema.md) | Project Schema：完整 YAML 定义、JSON Schema | 核心开发、用户 |
| 05 | [template-system.md](./05-template-system.md) | 模板系统：embed.FS、模板分层、FuncMap | 核心开发 |
| 06 | [codegen-pipeline.md](./06-codegen-pipeline.md) | 代码生成流水线：Plan/Apply、Drift 检测、并发与一致性 | 核心开发 |
| 07 | [ast-injection.md](./07-ast-injection.md) | AST 注入：Go AST(`dst`)、Proto AST(`protocompile`) | 核心开发 |
| 08 | [feature-system.md](./08-feature-system.md) | 特性系统：插件化扩展、注册机制 | 核心开发、扩展开发 |
| 09 | [component-design.md](./09-component-design.md) | 组件抽象：WebServer/Worker/CLI 三种内置组件 | 核心开发 |
| 10 | [tech-stack.md](./10-tech-stack.md) | 技术栈与依赖选型 + 平台支持 + 体积预算 | 核心开发 |
| 11 | [implementation-plan.md](./11-implementation-plan.md) | 5 阶段详细实施计划与任务拆分 | 项目经理、核心开发 |
| 12 | [testing-strategy.md](./12-testing-strategy.md) | 测试策略：单测、集成、Snapshot、E2E + GitHub Actions | 核心开发、QA |
| 13 | [coding-standards.md](./13-coding-standards.md) | 编码规范：命名、错误、日志、注释、PR 规范 + i18n 策略 | 所有开发 |
| 14 | [observability.md](./14-observability.md) | 可观测性：日志/trace/profile + diagnostic + telemetry | 核心开发 |
| 15 | [security-model.md](./15-security-model.md) | 安全模型：威胁建模、Hook 执行策略、SSTI 防护、插件信任 | 核心开发、安全 |

### 治理文档（贯穿）

| 文档 | 主题 | 受众 |
| --- | --- | --- |
| [META-roadmap.md](./META-roadmap.md) | **规划演进追溯**：差距分析 + 4 个 Batch 路线图 + 文档质量标准 | 所有人 |
| [99-glossary.md](./99-glossary.md) | **术语表**：Pair / Mutator / Drift / Reconcile 等高频术语单一定义 | 所有人 |
| [adr/](./adr/) | **架构决策记录**：5 个关键决策（dst, embed, protocompile, plan/apply, feature） | 架构师、核心开发 |

### 远期规划（Batch 3-4）

> 以下文档已规划但尚未交付，详见 [META-roadmap.md §3](./META-roadmap.md#3-完善路线图4-个-batch)。

| 编号 | 文档 | 状态 |
| --- | --- | --- |
| 16 | `ai-integration.md` | 🚧 Batch 3 |
| 17 | `enterprise-platform.md` | 🚧 Batch 3 |
| 18 | `release-governance.md` | 🚧 Batch 4 |
| 19 | `docs-site.md` | 🚧 Batch 4 |
| 20 | `import-algorithm.md` | 🚧 Batch 3 |

## 图表索引（diagrams/ 目录）

所有架构图、时序图、状态图均使用 [Mermaid](https://mermaid.js.org/) 语法编写，可在 GitHub、VSCode（Mermaid Preview 插件）中直接渲染。

| 图 | 类型 | 适用阶段 | 说明 |
| --- | --- | --- | --- |
| [architecture-overall.mmd](./diagrams/architecture-overall.mmd) | 分层架构图 | Target | linctl 整体分层与模块依赖 |
| [architecture-layers.mmd](./diagrams/architecture-layers.mmd) | 包依赖图 | Target | L0~L4 各层包及其依赖方向 |
| [architecture-runtime.mmd](./diagrams/architecture-runtime.mmd) | 运行时数据流图 | Target | 单次命令执行的数据流向 |
| [architecture-feature.mmd](./diagrams/architecture-feature.mmd) | 特性系统类图 | Target | Feature 接口、Registry、内置 Feature |
| [seq-new-project.mmd](./diagrams/seq-new-project.mmd) | 时序图 | MVP / Phase 1 | `linctl new` 命令执行时序（Pair 串行版） |
| [seq-new-project-target.mmd](./diagrams/seq-new-project-target.mmd) | 时序图 | Target | `linctl new` 命令执行时序（Pair 并发版） |
| [seq-add-api.mmd](./diagrams/seq-add-api.mmd) | 时序图 | Phase 2+ | `linctl add api` 命令执行时序（含 AST 注入） |
| [seq-plan-apply.mmd](./diagrams/seq-plan-apply.mmd) | 时序图 | Phase 4+ | `linctl plan` + `linctl apply` 完整闭环 |
| [seq-drift-detection.mmd](./diagrams/seq-drift-detection.mmd) | 时序图 | Phase 4+ | Drift 检测与 3-way merge 流程 |
| [state-file-lifecycle.mmd](./diagrams/state-file-lifecycle.mmd) | 状态图 | Target | 单个生成文件的完整生命周期 |
| [seq-ast-injection.mmd](./diagrams/seq-ast-injection.mmd) | 时序图 | Phase 2+ | AST 注入（`dst` 库）的完整流程 |

## 核心设计原则

1. **声明式（Declarative）**：用户在 `linctl.yaml` 中描述"想要什么"，工具负责"怎么做"。
2. **可计划（Plannable）**：所有变更先 `plan` 后 `apply`，绝不偷偷修改用户代码。
3. **可回滚（Reversible）**：每次 `apply` 自动创建 git stash 或文件级备份，失败可一键回退。
4. **插件化（Pluggable）**：核心只暴露 `Feature/Component/Generator` 三个扩展点，新框架/新特性以插件方式注入。
5. **幂等（Idempotent）**：重复执行同样的命令产生同样的结果（基于 hash 比对）。
6. **零硬编码**：拒绝任何对作者本机路径、版本号、环境的依赖。
7. **零状态外发**：遥测 opt-in，且永不上传项目内容。
8. **测试先行**：核心生成路径 100% 单测覆盖，模板 snapshot 测试，端到端 `go build` 验证。
9. **安全默认**：Hook 执行策略（restricted/confirm/unrestricted 三级；本地默认 confirm，CI 强制 restricted），模板严格模式，路径强制 SafeJoin（详见 [15-security-model.md](./15-security-model.md)）。
10. **可观测性内建**：`--debug=*` 模块化 trace + `linctl profile` 性能分析（详见 [14-observability.md](./14-observability.md)）。

## 与 osbuilder 的关系

| 维度 | osbuilder | linctl | ADR |
| --- | --- | --- | --- |
| 模板嵌入 | `rakyll/statik`（已归档） | `embed.FS`（标准库） | [ADR-002](./adr/002-use-embed-not-statik.md) |
| 配置 Schema | 手写正则校验 | `validator/v10` + JSON Schema | - |
| Go AST 注入 | `go/ast`（注释丢失） | `dave/dst`（保留注释/空行） | [ADR-001](./adr/001-use-dst-not-goast.md) |
| Proto 修改 | 字符串行扫描 | `bufbuild/protocompile`（AST 级） | [ADR-003](./adr/003-use-protocompile-for-proto.md) |
| 重入能力 | 一次性生成 + AST 追加 | `plan/apply` 完整闭环 + drift 检测 | [ADR-004](./adr/004-plan-apply-pattern.md) |
| 特性扩展 | 改 `Pairs()` 函数 + 加 switch | 注册 `Feature` 插件 | [ADR-005](./adr/005-feature-as-first-class.md) |
| 错误处理 | 部分 `fmt.Printf` 吞错 | 统一 `error wrapping`，CheckErr 统一 exit | - |
| 日志 | klog + apex/log + fmt 混用 | 统一 `log/slog` | - |
| Hook 安全 | 任意 shell 执行 | Hook 执行策略：白名单 + 用户确认 + CI 强制 restricted | - |
| 测试 | shell 脚本 e2e | Go 表驱动 + MemMapFs + snapshot + 真实 `go build` | - |
| 遥测 | 无 opt-out | opt-in，可一键关闭 | - |
| 依赖体积 | ~45 直接 + ~160 间接 | ~13 直接 + ~50 间接（去除 k8s 大坨） | - |
| 二进制大小 | ~20 MB | 目标 ~12 MB | - |

## 路线图速览

| Phase | 周期 | 目标 |
| --- | --- | --- |
| Phase 1 | 3-4 周 | MVP：`new` + `add api` + gin + memory + gorm-postgres + 基础 Feature（sqlite/gorm-mysql/mongo 在 Phase 3 引入，详见 [§11.2.0](./11-implementation-plan.md#1120-phase-1-范围声明与-schema-的差异)） |
| Phase 2 | 2-3 周 | AST 注入强化 + Snapshot 测试 + 3-way merge base |
| Phase 3 | 2-3 周 | gRPC 全支持 + Worker（Job+MQ 合并）+ Docker/K8s/systemd |
| Phase 4 | 2-3 周 | `plan`/`apply` + drift 检测 + 交互式合并 |
| Phase 5 | 长尾 | 插件机制（kubectl 风格）+ 第三方 Feature 生态 |

详见 [11-implementation-plan.md](./11-implementation-plan.md)。

## 快速预览：MVP → 终态用户体验

> **示例阶段标注**：`linctl new` / `linctl add api` 为 **Phase 1 即可用**；`linctl plan` / `linctl apply` 为 **Phase 4+ 终态预览**（MVP 阶段命令默认 `--auto-approve`，生命周期接口契约已稳定，详见 [META 决策书 §1.6](./META-fix-decisions-2026-04-25.md#16-cli-生命周期) 与 [11-implementation-plan.md](./11-implementation-plan.md)）。

```bash
# === Phase 1（MVP 即可用） ===
$ linctl new myblog --module github.com/clin211/myblog --framework gin --storage memory --features healthz
✔ Project layout generated (32 files)   # 最小示例：gin-only + memory-store + healthz；启用全部 Feature 时 250–350
✔ go.mod initialized
✔ Makefile + Dockerfile + .golangci.yaml
✔ docs/ scaffolded

🚀 Next steps:
   cd myblog
   make deps && make build
   ./_output/myblog server

$ cd myblog
$ linctl add api --kinds post,comment --job-handler
✔ Generated 18 files for kinds: [post, comment]
✔ Updated internal/myblog/biz/biz.go (added 2 interface methods)
✔ Updated internal/myblog/store/store.go (added 2 interface methods)
✔ Updated pkg/api/myblog/v1/myblog.proto (added 10 RPC methods)
✔ Generated 6 job handler files

# === Phase 4+ 终态预览（MVP 阶段不暴露 plan/apply 子命令） ===
$ linctl plan
📋 Pending changes (compared with linctl.yaml):
   ~ Component "myblog" has new feature: opentelemetry
     → Will modify: internal/myblog/server.go
     → Will create: internal/myblog/pkg/observability/otel.go
     → Will create: configs/myblog.yaml (otel section)

$ linctl apply --strategy=ask
   ? Conflict: internal/myblog/server.go was modified by user (sha mismatch)
     [k]eep mine | [o]verwrite | [m]erge | [d]iff:  m
   ✔ 3-way merged successfully
```

> **文件数估算口径**（与 [00-overview.md §0.2](./00-overview.md#02-我们要解决的问题) 脚注一致；详见 [META 决策书 §1.4](./META-fix-decisions-2026-04-25.md#14-生成文件数)）：以 `Σ(component_i × pair_per_component_i) + project_level_files` 估算。
>
> - **最小示例**（gin-only + memory-store + healthz，1 个 WebServer）：约 **32** 个文件。
> - **全栈 Feature 启用**（gin + grpc-gateway + gorm-postgres + opentelemetry + user + websocket + preloader，含 docker/k8s/systemd 部署模板）：**250–350** 个文件。
> - 数字会随模板演进调整，但口径保持 `min ≈ 32 / full ≈ 250–350` 不变。

## 阅读路径建议

按你的角色选择阅读顺序：

| 角色 | 推荐路径 |
| --- | --- |
| **想理解整体** | 00 → 01 → 09 → 06 → 08 |
| **想看 CLI 怎么用** | 00 → 03 → 04 |
| **想参与开发** | 02 → 09 → 10 → 11 → 12 → 13 |
| **想理解决策原因** | [adr/README.md](./adr/README.md) → 各个 ADR |
| **想看 AST 怎么做** | 07 → [ADR-001](./adr/001-use-dst-not-goast.md) → [ADR-003](./adr/003-use-protocompile-for-proto.md) |
| **想做安全审查** | 15 → 13 §13.11 → 06 §6.14 |
| **想看可观测性** | 14 |
| **想理解项目演进** | [META-roadmap.md](./META-roadmap.md) |

## 文档治理

- **术语统一**：所有高频名词以 [99-glossary.md](./99-glossary.md) 为准。
- **决策记录**：架构级别的"为什么这么选"必须写入 [adr/](./adr/)。
- **演进追溯**：[META-roadmap.md](./META-roadmap.md) 记录所有规划层面的变更。
- **质量标准**：每篇文档应符合 [META-roadmap.md §4 Doc DoD](./META-roadmap.md#4-文档质量标准doc-dod)。

## 反馈与贡献

本文档持续演进。任何对设计的质疑、补充、改进建议，请在对应文档中以 `> TODO:` 标注，或提交 issue。重大变更请：

1. 在 [META-roadmap.md](./META-roadmap.md) 的修订历史中加一行
2. 如涉及关键决策，按 [adr/000-template.md](./adr/000-template.md) 撰写新的 ADR
3. 引用相关术语时，链接到 [99-glossary.md](./99-glossary.md)

---

下一步阅读：[00-overview.md](./00-overview.md)

_Last reviewed: 2026-04-25_
