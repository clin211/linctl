# 99. 术语表（Glossary）

> 本文档是 linctl 设计与实现中所有高频术语的**单一定义源**。任何后续文档/代码注释/PR 描述都应以此为准。如发现矛盾，**以术语表为准**，并提交 PR 修订其他文档。

## 阅读指南

- **A-Z 排序**：按英文术语字母序排列，便于查找。
- **格式**：每条目包含 **Term**（术语） / **Definition**（定义） / **Used in**（出现的文档/包） / **See also**（关联术语）。
- **缩写收录**：常用缩写（如 `AST`、`SSTI`）单列。
- **避免歧义**：同一概念在 osbuilder 与 linctl 中名称不同的，**明确标注对照**。

---

## A

### Action

- **Definition**：`Plan` 中描述"对单个文件做什么"的最小单元。包含 `Kind`（Create/Update/Skip/Conflict/Delete）、`Pair`、`Reason`、`HashOld/New` 等字段。
- **Used in**：`internal/codegen/plan.go`、`06-codegen-pipeline.md` 6.3.3
- **See also**：[Plan](#plan)、[Pair](#pair)、[ActionKind](#actionkind)

### ActionKind

- **Definition**：枚举类型，描述 `Action` 的种类。可选值：`Create` / `Update` / `Skip` / `Conflict` / `Delete`。
- **Used in**：`internal/codegen/plan.go`
- **See also**：[Action](#action)

### Apply (动词)

- **Definition**：把 `Plan` 中的 `Action` 实际写入磁盘（含模板渲染、AST 注入、备份、PROJECT 更新、PostApply Hook 执行）。
- **Used in**：`linctl apply` 命令、`orchestrator.Applier`
- **See also**：[Plan (动词)](#plan-动词)、[Reconcile](#reconcile)

### AST

- **Full**：Abstract Syntax Tree，抽象语法树。
- **Definition**：源代码经过 parser 后的树状结构表示。linctl 在两种语言上做 AST 操作：Go（基于 [`dave/dst`](https://github.com/dave/dst)）和 Protobuf（基于 [`bufbuild/protocompile`](https://github.com/bufbuild/protocompile)）。
- **Used in**：`internal/ast/`、`07-ast-injection.md`
- **See also**：[ASTMutator](#astmutator)、[dst](#dst-decorated-syntax-tree)

### ASTMutator

- **Definition**：描述"对一个文件做一次 AST 修改"的接口。内置实现包括 `AddInterfaceMethodMutator`、`AddStructMethodMutator`、`AddImportMutator`、`AddProtoRPCMutator` 等。
- **Used in**：`internal/ast/mutator.go`、`07-ast-injection.md` 7.4
- **See also**：[Mutator](#mutator)（同义）、[Feature](#feature)

---

## B

### Backup

- **Definition**：`apply` 之前对**所有将被 Update/Conflict/Delete 的文件**自动复制到 `.linctl/backups/<时间戳>/` 的快照。
- **Used in**：`internal/orchestrator/applier.go`、`06-codegen-pipeline.md` 6.9
- **See also**：[Restore](#restore)

### Builtin Feature

- **Definition**：linctl 二进制内嵌的 Feature（如 `healthz` / `opentelemetry` / `user` 等），通过 `init()` 自动注册到 `feature.Default` registry。区别于通过子进程调用的 [Plugin Feature](#plugin-feature)。
- **Used in**：`internal/feature/builtin/`、`08-feature-system.md`
- **See also**：[Feature](#feature)、[Plugin](#plugin)

---

## C

### Component

> ⚠️ "Component" 在 linctl 代码中**有两个同名但不同包**的类型，必须区分清楚：

#### Component (struct, 配置层) —— `project.Component`

- **包路径**：`internal/project/types.go`
- **本质**：YAML 反序列化的 struct，是 `linctl.yaml` 中 `spec.components[]` 的一项。
- **字段**：`Kind` / `Name` / `Framework` / `Storage` / `Features` / `Resources` / `Cron` / `Kafka` 等（见 [04-config-schema.md §4.4](./04-config-schema.md#44-完整-go-struct-定义)）。
- **用法**：作为参数传入 Feature/Component 接口的方法（如 `Apply(p *project.Project, c project.Component, ...)`）。

#### Component (interface, 行为层) —— `component.Component`

- **包路径**：`internal/component/component.go`
- **本质**：Go 接口，描述"组件的运行时行为"。
- **方法**：`Kind()` / `Name()` / `Validate(...)` / `BasePairs(...)` / `BaseMutators(...)` / `PostProcess(...)`（见 [09-component-design.md §9.2](./09-component-design.md#92-component-接口契约)）。
- **实现**：`WebServer` / `Worker` / `CLI` 三个内置类型 + 第三方插件。
- **关系**：每个 `project.Component`（struct）通过 `component.Registry.Build(spec)` 实例化为 `component.Component`（interface）。

#### 关键不变量

- 三种内置 Kind：`WebServer`（HTTP/gRPC 服务）、`Worker`（cron + MQ + 自定义 watcher）、`CLI`（命令行工具）。
- **osbuilder 对照**：osbuilder 的 `WebServer` / `JobServer` / `MQServer` / `CLITool` 四种 → linctl 合并为 `WebServer` / `Worker` / `CLI` 三种。
- **Used in**：`internal/component/`、`internal/project/`、`09-component-design.md`、`04-config-schema.md`
- **See also**：[Feature](#feature)、[Pair](#pair)、[Resource](#resource)

### Conflict

- **Definition**：linctl 检测到"用户修改过的文件"与"模板新版本"都需要写入同一文件时的状态。表现为 `ActionKind=Conflict`。需要通过冲突策略（[ConflictStrategy](#conflictstrategy)）解决。
- **Used in**：`06-codegen-pipeline.md` 6.7、`07-ast-injection.md`
- **See also**：[Drift](#drift)、[3-way merge](#3-way-merge)

### ConflictStrategy

- **Definition**：`linctl apply` 处理冲突的策略。可选 `skip`（保留用户改）/ `overwrite`（用模板覆盖）/ `merge`（3-way 合并）/ `ask`（交互询问）。默认 `ask`。
- **Used in**：`internal/orchestrator/applier.go`、`03-cli-design.md` 3.3.8
- **See also**：[Conflict](#conflict)

---

## D

### Defaults

- **Definition**：项目级或 Feature 级的默认值集合。位于 `linctl.yaml` 的 `spec.defaults` 字段。Component 未显式设置的字段会从 Defaults 继承。
- **Used in**：`internal/project/defaults.go`、`04-config-schema.md` 4.3.3

### Drift

- **Definition**：磁盘上文件的实际内容**与上次 linctl apply 时的快照（embedded hash）不一致**的状态，意味着用户在两次 apply 之间修改了文件。
- **Used in**：`06-codegen-pipeline.md` 6.7
- **See also**：[Conflict](#conflict)、[Hash Comment](#hash-comment)

### dst (Decorated Syntax Tree)

- **Definition**：[dave/dst](https://github.com/dave/dst) 库提供的语法树类型，是 `go/ast.Node` 的"装饰版"，能在修改后保留原文件的注释和空行。
- **Used in**：`internal/ast/go_inject.go`、`07-ast-injection.md` 7.3
- **See also**：[AST](#ast)

---

## E

### Embedded Hash (Hash Comment)

- **Definition**：linctl 在生成文件末尾追加的特殊注释 `// linctl: hash=<sha256>`，用于在下次 plan 时识别"该文件由 linctl 生成、且未被人改过"。
- **Used in**：`internal/fs/manager.go`、`06-codegen-pipeline.md` 6.7.1
- **See also**：[Drift](#drift)、[Hash Integrity](#hash-integrity)

### Engine (Template Engine)

- **Definition**：linctl 的模板渲染引擎。封装 `text/template` 的解析、FuncMap 注入、partial 加载、缓存、格式化等。
- **Used in**：`internal/template/engine.go`、`05-template-system.md` 5.5

---

## F

### Feature

- **Definition**：横切的可选能力（如 healthz、opentelemetry、user、websocket）。通过实现 `feature.Feature` 接口贡献模板对（Pair）、AST mutator、FuncMap、默认值。
- **Used in**：`internal/feature/`、`08-feature-system.md`
- **See also**：[Builtin Feature](#builtin-feature)、[Plugin Feature](#plugin-feature)、[Component](#component)

### FileManager

- **Definition**：linctl 的文件操作核心。封装 `afero.Fs`，提供原子写入（tmp+rename）、hash 计算、embedded hash 解析等能力。
- **Used in**：`internal/fs/manager.go`、`02-project-structure.md` 2.2

### FuncMap

- **Definition**：传入 `text/template` 的函数表（`template.FuncMap`）。linctl 默认提供 ~40 个函数（命名转换、字符串处理、集合判断、调试），Feature 可贡献额外函数。
- **Used in**：`internal/template/funcmap.go`、`05-template-system.md` 5.6

---

## G

### gofumpt

- **Definition**：[mvdan/gofumpt](https://github.com/mvdan/gofumpt) Go 源码格式化工具。比 `gofmt` 更严格。linctl 渲染 `.go` 文件后强制 gofumpt 格式化。
- **Used in**：`internal/template/engine.go` Format 方法

---

## H

### Hash Comment

- **Definition**：见 [Embedded Hash](#embedded-hash-hash-comment)。

### Hash Integrity

- **Definition**：linctl 的"诚信审查"机制：所有 generated 文件都应该有 hash comment；缺失视为"用户拥有的文件"，linctl 不会动它。`linctl lint` 会检查完整性。
- **Used in**：`03-cli-design.md` 3.3.9
- **See also**：[Embedded Hash](#embedded-hash-hash-comment)

### Hook

- **Definition**：项目级生命周期钩子，位于 `linctl.yaml` 的 `spec.hooks`，包含 `preApply`（apply 前执行的 shell 命令）和 `postApply`（apply 后执行）。
- **Used in**：`04-config-schema.md` 4.3.5、`15-security-model.md` Hook 执行策略
- **See also**：[Hook Execution Policy](#hook-execution-policy)（已替代旧 [Sandbox](#sandbox-已废弃术语) 概念）

---

## I

### Idempotent (幂等)

- **Definition**：linctl 的核心设计目标。同一份 `linctl.yaml` 重复执行 `linctl apply` 必须产生同样结果（基于 hash 比对）。
- **Used in**：核心设计原则、`README.md`

### Importer

- **Definition**：从已有 Go 项目反推 `linctl.yaml` 的工具。支持识别 `go.mod`、`cmd/` 子目录、`pkg/api/.../*.proto` 等结构。
- **Used in**：`internal/orchestrator/importer.go`、`03-cli-design.md` 3.3.11

---

## L

### Lock File

- **Full**：`.linctl/lock.yaml`（注意有 `.yaml` 后缀）
- **Definition**：linctl 维护的"文件级 hash 索引 + AST 修改记录"。是 drift 检测和 3-way merge base 的元数据来源。**提交到 git**。
- **Used in**：`06-codegen-pipeline.md` §6.10
- **See also**：[Project Lock](#project-lock)（**完全不同的另一个文件**）、[PROJECT File](#project-file)、[Drift](#drift)

### Project Lock

- **Full**：`.linctl/lock`（**无后缀**，不要与 [Lock File](#lock-file) 混淆）
- **Definition**：进程级互斥锁文件。任何写文件的命令（apply/add/new）开始时通过 [flock(2)](https://man7.org/linux/man-pages/man2/flock.2.html) 占用此文件；其他 linctl 进程再尝试时立即失败。Windows 上用 `LockFileEx` 实现等价语义。**不入 git**。
- **Used in**：`internal/fs/lock.go`、`06-codegen-pipeline.md` §6.14.2
- **See also**：[Lock File](#lock-file)（用途完全不同）

### Loader (ProjectLoader)

- **Definition**：负责加载、校验、补全默认值、转换 schema 版本的组件。
- **物理路径（唯一答案）**：`internal/project/loader.go`。
  - 与领域模型 `project.Project` 同包，符合 Clean Architecture 内聚原则。
  - **不在** `internal/orchestrator/`：orchestrator 仅做编排（loader → planner → applier → reporter），**不做 I/O**。
  - 任何文档/diagram/代码注释引用 ProjectLoader 时**必须**写 `internal/project/loader.go`，避免与 orchestrator 包混淆。
- **Used in**：`internal/project/loader.go`、`04-config-schema.md` 4.7、[META-fix-decisions-2026-04-25.md §1.2](./META-fix-decisions-2026-04-25.md)

---

## M

### MCP (Model Context Protocol)

- **Definition**：[Anthropic 提出的](https://modelcontextprotocol.io/) 大模型与工具通信协议。linctl 计划支持 `linctl mcp serve` 让 LLM Agent 直接调用其 plan/apply 等能力。
- **Used in**：`16-ai-integration.md`（Batch 3 文档）

### Mutator

- **Definition**：见 [ASTMutator](#astmutator)。

---

## O

### Order

- **Definition**：Feature 接口的方法 `Order() int`，返回排序权重，数字越小越先 Apply。用于解决 Feature 之间的隐式依赖（如 user 必须先于 opentelemetry）。
- **Used in**：`internal/feature/feature.go`、`08-feature-system.md` 8.6.1

### Owner

- **Definition**：`Pair` 与 `lock.yaml` 中的字段，标识"该文件由哪个 Feature/Component 贡献"。便于 plan 报告与 prune。
- **Used in**：`internal/codegen/pair.go`、`06-codegen-pipeline.md` 6.10

---

## P

### Pair

- **Definition**：linctl 中"待生成文件"的最小描述单元，包含 `Dst`（目标路径）、`TemplateID`（模板 ID）、`Mode`（写模式）、`Owner`（贡献者）。
- **Used in**：`internal/codegen/pair.go`、`06-codegen-pipeline.md` 6.3.1
- **osbuilder 对照**：osbuilder 用 `map[string]string`（dst → tpl）；linctl 升级为带元数据的结构体。

### PairBuilder

- **Definition**：`Pair` 的构建器，按 `Dst` 自动去重（后写覆盖前写）。每个 Component 在 Apply 时使用一个 PairBuilder 累积所有 Pair。
- **Used in**：`internal/codegen/pair.go`
- **See also**：[Pair](#pair)、[Feature](#feature)

### Phase

- **Definition**：linctl 的 5 阶段实施计划（**时间维度**）。`Phase 1`（MVP）→ `Phase 5`（插件生态）。每个 Phase 有明确的 DoD（Definition of Done）。
- **Used in**：`11-implementation-plan.md`、`README.md`
- **不要混淆**：Phase 是**时间分级**，[Tier](#tier) 是**能力分级**——同一段实现可以归属于某个 Phase（比如 Phase 4 实施 Tier 2/3 能力）。

### Tier

- **Definition**：能力分级（**功能维度**），由 [ADR-004 plan-apply-pattern](./adr/004-plan-apply-pattern.md) 引入。当前定义：
  - **Tier 1**：基础 Plan/Apply（Action: Create / Update / Skip）
  - **Tier 2**：增加 hash + Conflict + lock.yaml
  - **Tier 3**：增加 prune + drift detection
- **与 Phase 对应关系**：Tier 1 在 Phase 1 完成；Tier 2 在 Phase 4 完成；Tier 3 在 Phase 4-5 完成。详见 ADR-004 §5.1。
- **Used in**：`adr/004-plan-apply-pattern.md`、`META-fix-decisions §1.3`
- **See also**：[Phase](#phase)

### ResourceContributions

- **Definition**：`Feature` 接口的方法 `ResourceContributions(c Component) []Resource`，返回该 Feature 要为该 Component 注入的 Resource 集合。调度器在拓扑排序后聚合所有 Feature 的 ResourceContributions，统一注入到 Project 中。
- **设计动机**：消除 `Feature.Apply` 直接修改入参 `c.Resources` 的反模式，保证 Apply 是纯函数（详见 [META §1.16](./META-fix-decisions-2026-04-25.md#116-featureapply-不改入参) 与 §5.10）。
- **Used in**：`08-feature-system.md §8.3`
- **See also**：[Feature](#feature)、[Resource](#resource)

### Hook Execution Policy

- **Definition**：Hook 执行策略，三级：
  - `restricted`：仅允许 allowlist 中的命令前缀（gofumpt / go fmt / goimports / buf / make / protoc / wire 等）
  - `confirm`（**本地默认**）：每条非 allowlist 命令独立确认；`-y` 不能跳过
  - `unrestricted`：执行任意 shell 命令；CI 环境检测到时 `os.Exit(7)` 阻断
- **CI 强制规则**：`os.Getenv("CI") == "true"` 时强制 `restricted`，不可被 flag/配置覆盖
- **设计动机**：避免「沙箱」一词的误导（实际不是真隔离）；明确策略 + 确认 + CI 强制三层防御
- **Used in**：`15-security-model.md §15.2.1`、`04-config-schema.md §4.3.5 spec.hooks[].policy`、`META §1.10 / §5.2 / §5.3`

### Plan (名词)

> ⚠️ "Plan" 在 linctl 中**有两种语境**，必须区分清楚：

#### Plan (内部值对象) —— `codegen.Plan`

- **包路径**：`internal/codegen/plan.go`
- **本质**：Go struct，承载「目标 vs 现状」计算后的待执行 Action 列表 + `PlanStats`。
- **生命周期**：Phase 1（Story 1.7）即引入并被 internal Apply 使用，**不需要等待 `linctl plan` 子命令**。
- **字段**：`Actions []Action` / `Stats PlanStats` / `Digest string`（Tier 2+ 引入用于防撕裂，详见 [META-fix-decisions §1.12](./META-fix-decisions-2026-04-25.md)）。
- **Used in**：`internal/codegen/plan.go`、`06-codegen-pipeline.md`、所有命令的 Plan 阶段（即使命令未暴露子命令）。

#### Plan (CLI 输出) —— `linctl plan` 子命令的人类可读 / `--output json|yaml` 表达

- **本质**：`linctl plan` 命令将上述 `codegen.Plan` 序列化为彩色表格 / JSON / YAML 后输出给用户或 CI。
- **生命周期**：Phase 4（Story 4.1）才**对用户暴露**；Phase 1-3 用户**看不到** `linctl plan` 子命令。
- **format**：默认 text；可选 `--output json` / `--output yaml`（按 [META-fix-decisions §1.5](./META-fix-decisions-2026-04-25.md)，统一 flag）。
- **Used in**：`internal/cli/cmd_plan.go`（Phase 4）、`03-cli-design.md`、`04-config-schema.md`。

#### 关键不变量

- **数据结构 ≠ 子命令**：Phase 1 已有 `codegen.Plan` 结构体，但 Phase 1 用户不能调用 `linctl plan`；不要在 Phase 1 文档中暗示用户可直接 `linctl plan`。
- 详见 [11-implementation-plan.md §11.2.0](./11-implementation-plan.md#1120-phase-1-范围声明与-schema-的差异)。
- **See also**：[Action](#action)、[Plan (动词)](#plan-动词)

### Plan (动词)

- **Definition**：linctl 计算"目标 vs 现状"得出待执行变更清单的过程。**不写任何文件**。
- **Used in**：`linctl plan` 命令、`orchestrator.Planner`
- **See also**：[Apply](#apply-动词)

### Plugin

- **Definition**：第三方贡献的 linctl 扩展，以 `linctl-<name>` 可执行文件形式存在（kubectl 风格）。通过 stdio JSON-RPC 与主进程通信。
- **Used in**：`08-feature-system.md` 8.7、`linctl plugin` 子命令
- **See also**：[Plugin Feature](#plugin-feature)

### Plugin Feature

- **Definition**：通过 [Plugin](#plugin) 提供的 Feature，区别于 [Builtin Feature](#builtin-feature)。受序列化协议限制。

### PROJECT File

- **Definition**：linctl 在项目根维护的 YAML 文件（**注意全大写**），保存 Project 的最新有效配置 + status 元数据。**用户勿改**。
- **Used in**：`04-config-schema.md` 4.1
- **osbuilder 对照**：osbuilder 的 `PROJECT` 既作为输入又作为状态；linctl 拆为 `linctl.yaml`（用户输入）+ `PROJECT`（工具状态）两个文件。
- **See also**：[Project](#project)、[Lock File](#lock-file)

### Project

- **Definition**：linctl 的核心配置类型 `project.Project`，对应 `linctl.yaml` 的反序列化。包含 `apiVersion / kind / metadata / spec / status` 五段。
- **Used in**：`internal/project/types.go`、`04-config-schema.md` 4.4

### Prune

- **Definition**：`linctl apply --prune` 模式下，删除 lock.yaml 标记的"曾经由 linctl 生成、但当前 linctl.yaml 不再需要"的文件。
- **Used in**：`03-cli-design.md` 3.3.8、`06-codegen-pipeline.md`

---

## R

### Reconcile

- **Definition**：linctl 总能从"`linctl.yaml` 期望状态 + 当前磁盘状态"算出"下一步要做什么"的能力。借鉴 K8s controller 的 reconcile loop。
- **Used in**：`06-codegen-pipeline.md` 6.1
- **See also**：[Plan](#plan-动词)、[Apply](#apply-动词)

### Registry

- **Definition**：通用术语：注册中心。linctl 中有两类——
  - **Feature Registry**（`internal/feature/registry.go`）：注册所有可用 Feature。
  - **Plugin Registry**（`internal/plugin/registry.go`，Phase 5）：发现 PATH 中的 `linctl-*` 插件。

### Restore

- **Definition**：从 `.linctl/backups/<时间戳>/` 一键恢复，回滚一次失败的 apply。
- **Used in**：`06-codegen-pipeline.md` 6.12、`linctl restore` 子命令（Phase 4）
- **See also**：[Backup](#backup)

---

## S

### Sandbox（**已废弃术语**）

- **Definition**：旧术语，描述 linctl 对 [Hook](#hook) 命令的安全约束。
- **现状**：本术语已**废弃**，统一改用 [Hook Execution Policy](#hook-execution-policy)（三级策略 restricted/confirm/unrestricted）。文档中保留此条目仅用于历史交叉引用。
- **废弃原因**：本机制实际是「命令前缀白名单 + 用户确认 + CI 强制」而非真正的进程沙箱（如 nsjail/firejail），称「沙箱」会让用户对隔离强度产生错误期望。
- **See also**：[Hook Execution Policy](#hook-execution-policy)、[Hook](#hook)

### Schema

- **Definition**：本文档系列中"Schema"特指 [Project Schema](#project)（即 `linctl.yaml` 的结构定义），不指代数据库 schema。
- **Used in**：`04-config-schema.md`
- **See also**：[JSON Schema](#json-schema)

### JSON Schema

- **Definition**：linctl 通过 `gen-jsonschema` 工具自动从 Go struct 导出的 [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/schema)。供 IDE（VSCode/JetBrains）自动补全 `linctl.yaml`。
- **Used in**：`04-config-schema.md` 4.6

### slog

- **Definition**：Go 1.21+ 标准库的结构化日志（`log/slog`）。linctl 使用 slog 作为唯一日志框架，自定义彩色 handler。
- **Used in**：`internal/log/`、`02-project-structure.md`

### Snapshot Test

- **Definition**：linctl 测试模板渲染输出的方式：把模板渲染结果与 golden file 对比。通过 `UPDATE_GOLDEN=1` 环境变量更新 golden。
- **Used in**：`tests/snapshot/`、`05-template-system.md` 5.11、`12-testing-strategy.md`
- **See also**：[Golden File](#golden-file)

### Golden File

- **Definition**：[Snapshot Test](#snapshot-test) 的"标准答案"，存于 `tests/snapshot/golden/`，由人工审查后提交到 git。

### SSTI

- **Full**：Server-Side Template Injection
- **Definition**：模板注入攻击。linctl 通过 `Option("missingkey=error")` 严格模式 + 模板审查 CI 防护。
- **Used in**：`15-security-model.md`

### Status (字段)

- **Definition**：`Project.Status` 字段，由 linctl 维护，记录 generatedAt / cliVersion / schemaMigrations 等元数据。**用户勿改**。
- **Used in**：`04-config-schema.md` 4.3.6

---

## T

### Telemetry

- **Definition**：使用统计上报。linctl 默认**关闭**，需通过 `LINCTL_TELEMETRY=on` 环境变量显式开启（opt-in）。绝不上传项目内容。
- **Used in**：`internal/telemetry/`、`README.md`

### Template

- **Definition**：linctl 中"模板"特指 `templates/` 目录下的文件（`.tpl` / `.proto` / 静态资产），通过 `//go:embed` 嵌入二进制。
- **Used in**：`templates/`、`05-template-system.md`

### Template Data

- **Definition**：`template.TemplateData` 类型，是模板渲染时的上下文。包含 Project / Component / Feature / Resource / Helpers 字段。
- **Used in**：`internal/template/data.go`、`05-template-system.md` 5.8

---

## W

### WebServer / Worker / CLI

- **Definition**：linctl 三种内置 [Component](#component) 类型。
  - `WebServer`：HTTP/gRPC 服务（含 healthz / 鉴权 / WS / OTel 等可选 Feature）
  - `Worker`：合并 osbuilder 的 JobServer + MQServer + 自定义 watcher
  - `CLI`：类似 kubectl 的命令行工具
- **Used in**：`internal/component/`、`09-component-design.md`

---

## 数字

### 3-way merge

- **Definition**：合并算法，输入是 base（上次 apply 渲染结果，从 lock.yaml/cache 读）、current（用户当前磁盘内容）、new（最新模板渲染结果），输出合并后的内容或冲突 markers。Phase 4 落地。
- **Used in**：`internal/codegen/merger.go`、`06-codegen-pipeline.md` 6.8
- **See also**：[Conflict](#conflict)、[ConflictStrategy](#conflictstrategy)

### 5-段式（Command Lifecycle）

- **Definition**：每个 linctl 子命令的统一生命周期：`Complete → Validate → Plan → Apply → Report`。在 osbuilder 三段式（`Complete/Validate/Run`）基础上扩展。
- **Used in**：`01-architecture.md` 1.6

---

## 缩写表（Acronyms）

| 缩写 | 全称 | 解释 |
| --- | --- | --- |
| AST | Abstract Syntax Tree | 抽象语法树 |
| CLI | Command-Line Interface | 命令行工具 |
| CRD | Custom Resource Definition | K8s 自定义资源 |
| DCO | Developer Certificate of Origin | 开发者来源证书（开源贡献协议之一） |
| DoD | Definition of Done | 完成定义 |
| FuncMap | - | `text/template.FuncMap` |
| MCP | Model Context Protocol | LLM 工具协议 |
| MVP | Minimum Viable Product | 最小可用产品 |
| OTel | OpenTelemetry | 开源可观测性框架 |
| RPC | Remote Procedure Call | 远程过程调用 |
| SemVer | Semantic Versioning | 语义化版本 |
| SSTI | Server-Side Template Injection | 服务器端模板注入攻击 |

---

## osbuilder ↔ linctl 术语对照

| osbuilder | linctl | 备注 |
| --- | --- | --- |
| `WithUser` (bool 字段) | `features: [user]` (字符串列表) | 启用语法变更 |
| `JobServer` + `MQServer` | `Worker` (合一) | 合并相似抽象 |
| `Pairs() map[string]string` | `PairBuilder.Add(Pair{...})` | 升级为结构体 + 元数据 |
| `Run()` 三段式 | `Complete / Validate / Plan / Apply / Report` 五段式 | 引入 Plan 阶段 |
| `WithXxx` (硬编码) | `Feature` 接口（可插件） | 真正的可扩展性 |
| `PROJECT` 文件兼输入和状态 | `linctl.yaml`（用户输入） + `PROJECT`（工具状态） | 拆分职责 |
| `osbuilder create api` | `linctl add api` | 命令名简化 |
| 字符串行扫描改 .proto | `bufbuild/protocompile` AST | 升级精度 |
| `*ast.Ident{Name: "expr()"}` | `parser.ParseExpr` + dst | 修复反模式 |
| 主动遥测 | opt-in 遥测 | 隐私改进 |

---

## 修订记录

| 日期 | 版本 | 变更 |
| --- | --- | --- |
| 2026-04-25 | 0.1 | 初始版本，覆盖 00-15、META-roadmap、所有 ADR、所有 diagrams 的高频术语 |
| 2026-04-25 | 0.1.1 | 按 [META-fix-decisions](./META-fix-decisions-2026-04-25.md) 修订：拆分 `Plan (名词)` 为「内部值对象」+「CLI 输出」两条；明确 ProjectLoader 物理路径为 `internal/project/loader.go`（消除与 orchestrator 包的混淆） |
| 待续 | 0.2 | 增加 Batch 2 引入的术语（Sandbox、SSTI 详细等） |

---

> **维护原则**：每篇新文档引入新术语时，**必须**同步更新本表。CI 中加 `scripts/check-glossary.sh` 校验：所有 `_(.*)` 风格的高频名词是否在术语表中（启发式检查）。

---

_Last reviewed: 2026-04-25_
