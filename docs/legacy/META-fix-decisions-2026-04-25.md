# META · 修复决策书 (Single Source of Truth)

**日期**：2026-04-25
**目的**：锁定跨文档冲突的唯一选择，作为本轮文档修复的强制依据
**适用范围**：`lin/docs/` 全部主线文档（00-15、META、README、adr/、diagrams/、99）
**状态**：Active（修复完成后此文件转为 archived 状态，留作变更日志）

---

## 1. 命名 / 类型 / 路径决策（消除跨文档冲突）

### 1.1 错误类型

```go
// internal/linctlerr/error.go
package linctlerr

type Code string  // 错误码字符串类型

// 命名约定：所有错误码常量使用 Err<DescriptiveName> 形式（与 Go 标准库 errors 风格一致）
// 字符串值采用 snake_case（便于 JSON / YAML 输出与日志 grep）
const (
    ErrConfigInvalid     Code = "config_invalid"
    ErrComponentExists   Code = "component_exists"
    ErrComponentNotFound Code = "component_not_found"
    ErrTemplateRender    Code = "template_render"
    ErrFileConflict      Code = "file_conflict"
    ErrASTInjection      Code = "ast_injection"
    ErrNetwork           Code = "network"
    ErrEnvironment       Code = "environment"
    // ...
)

type LinctlError struct {
    Code    Code      // 错误码
    Message string    // 用户可读的错误信息
    Hint    string    // 可执行的修复提示
    Cause   error     // 底层 error
}
```

**统一规则**：

- 类型名：`LinctlError`
- 错误码字段名：`Code`
- 错误码常量类型名：`Code`（不是 `ErrCode`）
- main 函数中：`errors.As(err, &linctlerr.LinctlError{})` 后访问 `.Code`

### 1.2 ProjectLoader 物理路径

**唯一答案**：`internal/project/loader.go`

理由：

- ProjectLoader 解析 `linctl.yaml` 并返回 `*project.Project`
- 加载器与领域模型同包，符合 Clean Architecture 内聚原则
- `internal/orchestrator/` 仅做编排，不做 I/O

**所有引用必须改为**：`internal/project/loader.go`

### 1.3 Phase 命名（消除多义）

**实施计划（11-implementation-plan.md）**：使用 `Phase 1` ~ `Phase 5`，含义为 **时间/迭代阶段**

**ADR-004（plan-apply-pattern）**：改用 `Tier 1` / `Tier 2` / `Tier 3`，含义为 **能力分级**：

- Tier 1：基础 Plan（Create/Skip/Update Action）
- Tier 2：增加 hash + Conflict + lock.yaml
- Tier 3：增加 prune + drift detection

**所有出现 Phase / Tier 的位置必须明确语境**，不允许混用。

### 1.4 生成文件数

**`00-overview.md`**：明确写「全栈 Feature 启用时生成 250-350 个文件」
**`README.md` 示例**：标注「最小示例（gin-only + memory-store + healthz），约 32 个文件；启用全部 Feature 时 250-350」
**新增脚注**：在两份文档中添加文件数计算说明（以 component 数 × pair-per-component 估算）

### 1.5 CLI flag

**统一规则**：

- 全局机器可解析输出 flag：`--output / -o`，取值 `text`/`json`/`yaml`
- `--out` 在所有命令中删除（如已被引用，作为 `--output` 的 deprecated alias 1 个 minor 版本后移除）
- `--output` 默认值：`text`（人类可读）

### 1.6 CLI 生命周期

**唯一模型**：所有变更命令统一为 `Complete → Validate → Plan → Apply → Report`

```go
type Command interface {
    Complete(args []string) error
    Validate() error
    Plan(ctx context.Context) (*Plan, error)
    Apply(ctx context.Context, plan *Plan) (*Report, error)
}
```

**MVP（Phase 1）退化策略**：

- `Plan()` 内部仍然计算
- `Apply()` 内部直接执行 plan
- 命令默认 `--auto-approve`（因为 plan/apply CLI 子命令在 Phase 4 才暴露）
- 但接口契约保持稳定，不需要 Phase 4 时再重写

### 1.7 linctl.yaml 内容职责

**唯一规则**：

- `linctl.yaml`：仅包含 `apiVersion / kind / metadata / spec`（**用户编辑**，进 git）
- `PROJECT`：包含 `status`、`schemaMigrations`、`generatedAt`、`cliVersion`（**工具维护**，进 git，用户勿改）
- `.linctl/lock.yaml`：包含文件级 hash 索引（**工具维护**，进 git）
- `.linctl/lock`：进程互斥锁文件（**工具维护**，**不进 git**）
- `.linctl/cache/`：渲染缓存（**工具维护**，**不进 git**）

### 1.8 测试金字塔

**唯一答案**：**Unit 70% / Integration 20% / E2E 10%**

- mermaid 图必须修改为 70/20/10
- snapshot/golden 测试归类到 Unit
- 删除「Snapshot 15%」分类

### 1.9 hash 接管策略

**修订规则**：

- 无 `// linctl: hash` 的已存在文件：plan 中标记为 `would_claim`（不再是 `Skip`）
- `apply` 时：如果文件内容与模板渲染结果一致 → 仅添加 hash 注释（claim）
- 如果不一致 → 走冲突策略（ask/skip/overwrite/merge）
- 提供 `linctl claim <component>` 命令，强制接管历史代码

### 1.10 Hook 执行策略（不再叫"沙箱"）

> ⚠️ **本节最终决策版本**（v0.2 已合并 §5.2 / §5.6）。任何与本节冲突的描述均以 §5.2 / §5.6 为准。

**全文删除"sandbox"一词**，改用"Hook 执行策略"（Hook Execution Policy）

**策略级别**（详见 §5.2）：

- `restricted`：仅允许 allowlist 中的命令前缀（gofumpt / go fmt / goimports / buf / make 等）
- `confirm`（**本地默认**）：对 allowlist 命令直接执行；对其他命令逐条 prompt 用户独立确认（`-y` 不能跳过）
- `unrestricted`：执行任意 shell 命令（仅本地，且必须二次确认）

**CI 强制规则**（详见 §5.6）：

- 检测到 `CI=true` 环境变量时，强制 `restricted`，无法用 flag 或配置覆盖
- 违例时使用 `os.Exit(7)`（**安全策略违规**退出码）+ 结构化错误日志，**不再用 `panic`**（避免与 panic 默认 exit code 2 与「码 2 = 配置错误」混淆）

**`-y` 与 hook 关系**：

- `-y` 仅跳过文件冲突的二次确认
- hook 如果是 `confirm` 或 `unrestricted` 级别，`-y` 不能跳过 hook 确认

### 1.11 flock 语义

**统一规则**：

- 默认行为：**阻塞 + 30 秒超时**
- 可配 flag：`--lock-timeout 60s`
- 实现：`syscall.Flock(LOCK_EX)` 在 goroutine 中执行 + select + time.After
- 文档伪代码必须与文字描述一致

### 1.12 plan-apply digest 防撕裂

**新规则**：

- `linctl plan` 输出包含 `digest: sha256(<canonical-json>)`
- 可选保存：`linctl plan -o json > plan.json`
- `linctl apply --plan plan.json` 时校验 digest，不匹配则报错并提示重新 plan
- `linctl apply` 不带 `--plan` 时仍会重新 compute 一次，但必须在 stderr 输出 `[WARN] re-computing plan; use --plan for guaranteed determinism`

### 1.13 3-way merge 必须可编译

**新规则**：

- merge 后必须运行 `go build -o /dev/null ./...`（如果是 Go 文件）
- build 失败 → 自动降级为 conflict markers + 报错
- 对于 proto 文件 → 运行 `buf lint`
- 对于其他文件 → 跳过 verify

### 1.14 PairBuilder 同文件覆盖

**新规则**：

- 默认行为：plan 中显示 `Override` 事件，apply 时打印 `[WARN]`
- `--strict` flag：plan 直接 fail
- 文档示例必须更新

### 1.15 Feature 依赖：DAG 而非总序

**新规则**：

- `Feature` 接口新增方法 `Requires() []string`（依赖的 Feature 名）
- 调度器使用拓扑排序
- 检测到环 → 启动失败 + 列出环路
- 保留 `Order() int` 仅作为同层 tie-break，默认 0

**示例修改**：

- `UserFeature.Order() = 100`
- `HealthzFeature.Order() = 200`
- `ObservabilityFeature.Requires() = []string{"healthz"}`

### 1.16 Feature.Apply 不改入参

**新签名**：

```go
type Feature interface {
    Name() string
    Requires() []string
    Order() int
    Apply(ctx context.Context, c Component) ([]Pair, error)  // 不再写 c.Resources
}
```

**Resource 注入由调度器统一聚合**，不在 Apply 中改 Component。

### 1.17 多文件配置

**新规则**：

- 主文件：`linctl.yaml`
- 自动 merge 目录：`linctl.d/*.yaml`（深合并，按文件名字典序）
- 优先级：`linctl.yaml > linctl.d/`（主文件覆盖 overlay）
- 显式 include：`includes: ["./shared/api.yaml"]`（相对路径）
- 环境覆盖：`linctl.yaml` + `linctl.dev.yaml`（通过 `--env dev` 启用）

### 1.18 apiVersion 多义消除

**新规则**：

- 顶层 `apiVersion: linctl.dev/v1` —— 含义不变（linctl Schema API 版本）
- proto 字段 `spec.defaults.apiVersion: v1` → 重命名为 `spec.defaults.protoVersion: v1`
- 术语表必须列入此变更

---

## 2. 文档治理决策

### 2.1 图表阶段标注

每张 `.mmd` 文件第 1 行加阶段注释：

```
%% Stage: MVP | Target | Phase 4+
```

`README.md` 图表索引表新增「适用阶段」列。

### 2.2 术语表覆盖范围

`99-glossary.md` 修订记录中：

- 删除"覆盖 00-08"
- 改为"覆盖 00-15、META-roadmap、所有 ADR、所有 diagrams"

### 2.3 同文件 sqlite 矛盾

`11-implementation-plan.md`：

- §11.2.0 改为「Phase 1: gorm-postgres + memory（**不含 sqlite**）」
- §11.2.3 退出标准的 sqlite 删除（移到 §11.4 Phase 3 退出标准）

### 2.4 seq-new-project.mmd 并发→串行

`seq-new-project.mmd`：将「Pair 并发」改为「Pair 串行（Phase 1）」并加 `%% Phase 1` 注释；新增另一图 `seq-new-project-target.mmd` 表示终态并发。

### 2.5 README 路线图与卷首预览一致化

`README.md`：

- 卷首「快速预览」加上注释：`# Phase 4+ 功能预览`
- 或将卷首示例改为 Phase 1 实际能力（`linctl new` 直接生成）

---

## 3. 标记规范

修订过程中：

- ✅ 已修复
- 🚧 修复中
- ⚠️ 待人工决策（如果 agent 拿不准）

修订完成后所有文档应统一标记 `_Last reviewed: 2026-04-25_` 在文末。

---

## 4. 修复完成验证清单

修复后必须通过：

- [ ] `rg "ErrCode" lin/docs/` 仅在 archived 决策文件中出现
- [ ] `rg "internal/orchestrator/loader" lin/docs/` 无结果
- [ ] `rg "linctl plan.*--out " lin/docs/` 无结果
- [ ] `rg "sandbox" lin/docs/15-security-model.md` 仅出现在「为何不叫沙箱」的说明段
- [ ] `rg "Order.*100" lin/docs/08-feature-system.md lin/docs/diagrams/architecture-feature.mmd` 不再有同序碰撞
- [ ] `00-overview.md` 与 `README.md` 中的文件数描述一致
- [ ] `12-testing-strategy.md` 中 70/20/10 与 mermaid 图一致
- [ ] 每张 `.mmd` 第 1 行有 `%% Stage:` 注释
- [ ] `99-glossary.md` 修订记录写「覆盖 00-15」
- [ ] `11-implementation-plan.md` §11.2.0 与 §11.2.3 中 sqlite 出现位置一致

---

---

## 5. 二轮验证补充决策（2026-04-25 v0.2）

第一轮修复完成后做了一轮验证 review，发现 24 项跨组接合点 / 概览同步 / 边界条件类残留。本节作为 SSOT 增补：

### 5.1 错误包物理路径单点定稿

**唯一答案**：`internal/linctlerr`

- 删除所有 `internal/errors` 引用，删除别名 `linctlerr`（不再需要别名）
- 涉及文件：`01-architecture.md`、`02-project-structure.md`

### 5.2 默认 Hook 策略歧义消除

**最终决策**：
- **默认本地策略**：`confirm`（每条 hook 命令独立确认）
- **CI 环境强制策略**：`restricted`（仅允许 allowlist；检测到 `unrestricted` 时用 `os.Exit(7)` 阻断，详见 §1.10 / §5.6）
- **`restricted` 不是默认，是 CI 强制**

文档需统一表述：
- META §1.10 修正：默认 confirm，CI 强制 restricted
- `15-security-model.md:91` 表头修正：默认 confirm
- `15-security-model.md:230, 245` 已正确，作为基准
- `04-config-schema.md` 在 `spec.hooks[*].policy` 字段中加入策略字段（可选，覆盖默认）

### 5.3 Hook 策略在 yaml 中的定义

`04-config-schema.md` `spec.hooks[*]` 加入：
```yaml
spec:
  hooks:
    preApply:
      - name: format-go
        run: gofumpt -w .
        policy: restricted   # 可选：restricted | confirm | unrestricted；默认 confirm；CI 下被强制为 restricted
```

### 5.4 Phase 1 build 校验例外

**修订 §1.13**：
- **Phase 1**：merge 后只 build 受影响的文件 + 其 import closure（targeted build）
- **Phase 2+**：merge 后强制全量 `go build ./...`
- 文档需在 SSOT §1.13 与 06-codegen-pipeline §6.X 中同步说明阶段例外

### 5.5 flock 跨平台 timeout 语义

**承认现实**：
- `syscall.Flock(LOCK_EX)` 是阻塞且不可中断的系统调用
- goroutine + select 只能让**主调用返回 timeout**，但子 goroutine 仍卡在内核 flock
- 子 goroutine 持有的锁会在进程退出时由 OS 释放（除非进程不退出）

**文档要求**（在 06-codegen-pipeline.md flock 章节明确说明）：
1. timeout 含义：**用户态返回错误**，不保证立即解锁
2. 对应建议：超时后用户应该退出进程（让 OS 释放锁）
3. 跨平台：Windows 上用 `LockFileEx` 有 `LOCKFILE_FAIL_IMMEDIATELY`，但本项目 MVP 仅支持 Linux/macOS
4. 推荐替代：未来可考虑 `tryLock` 循环 + `LOCK_NB`（每 100ms 重试一次）

### 5.6 CI panic 退出码消歧

**新规则**：
- `03-cli-design.md §3.6` 新增退出码：
  - `7` = 安全策略违规（hook policy violation, CI restricted 强制失败）
- `15-security-model.md:101` 修正：CI 下违规 hook 不用 panic（panic exit code 为 2，会与配置错冲突），改用 `os.Exit(7)` + 错误日志
- 全文统一：safety / policy 类致命错误用退出码 7

### 5.7 跨组接合点 Checklist（强制）

为防止下次修复再次出现跨组遗漏，**任何修改下列字段时，必须同步**：

| 字段/概念 | 主文档 | 接合文档 |
|---|---|---|
| CLI flag（如 `--output`、`--env`） | `03-cli-design.md` | `04-config-schema.md`（当配置层引用），ADR 中所有命令示例 |
| spec.* 字段 | `04-config-schema.md` | `99-glossary.md`，`05-template-system.md`（数据模型），相关 mmd |
| Hook 相关 | `15-security-model.md` | `04-config-schema.md` `spec.hooks`，`03-cli-design.md` `--hook-policy` |
| 错误码常量 | `01-architecture.md` `LinctlError` | `13-coding-standards.md` 错误处理章节，`03-cli-design.md §3.6` |
| Feature 接口 | `08-feature-system.md` | `09-component-design.md`，`adr/005`，`architecture-feature.mmd` |
| 文件路径（如 `internal/project/loader.go`） | 此 SSOT 文件 | `02-project-structure.md` 概览 + 详情，所有 mmd 中的层节点 |
| Phase 1 storage 范围 | `11-implementation-plan.md` §11.2.0 | `README.md` 路线图，`00-overview.md`，所有相关 ADR |
| 工具链版本 | `10-tech-stack.md` §10.7 | `12-testing-strategy.md` CI 段，`15-security-model.md` 工具段 |

### 5.8 概览-详情双向同步规则

文档中存在「概览-详情」结构的位置：

| 概览处 | 详情处 | 修改原则 |
|---|---|---|
| `02-project-structure.md` 第 14 行树头 | `02-project-structure.md` `internal/orchestrator/` 与 `internal/project/` 详细说明 | 修改详情后必须同步树头注释 |
| `README.md` 路线图 | `11-implementation-plan.md` 各 Phase 退出标准 | 修改 Phase 范围后必须同步路线图 |
| `README.md` 图表索引（含「适用阶段」列） | `diagrams/*.mmd` 第 1 行 `%% Stage:` 注释 | 修改图表 stage 后必须同步索引列 |
| `adr/README.md` ADR 列表 + 状态 | 各 ADR 文件元数据 | 状态变更后必须同步索引 |

### 5.9 二轮验证发现需创建的新文件

- `lin/docs/diagrams/seq-new-project-target.mmd`：终态并发版本（与现有 seq-new-project.mmd 串行版本对照）

### 5.10 二轮验证待补的术语表条目

`99-glossary.md` 需新增：
- `### Tier`（与 `### Phase` 对读，明确两词的不同含义）
- `### ResourceContributions`（如果 Group C 把它加入正式 Feature 接口）
- `### Hook Execution Policy`（涵盖 restricted / confirm / unrestricted）

---

_v0.1 由首轮修复 review 生成；v0.2 由二轮验证补充。修复完成后归档为 `META-fix-decisions-2026-04-25-archived.md`_

_Last reviewed: 2026-04-25_

