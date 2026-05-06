# 13. 编码规范与 i18n 策略

> 本文档定义 linctl 自身代码（不是它生成的项目）的编码规范，包括命名、错误处理、日志、注释、PR 规范，以及 CLI 输出/错误信息的国际化策略。

## 13.1 通用原则

| 原则 | 解释 |
| --- | --- |
| **可读性 > 简洁性** | 一段精炼但需要思考 5 分钟的代码，不如一段 3 倍长但一眼明白的代码 |
| **显式 > 隐式** | 拒绝魔法；任何"自动化行为"必须在文档中明确 |
| **本地化处理 > 全局副作用** | 不用 `init()` 做重操作；不修改全局变量 |
| **小函数 > 大函数** | 单个函数 ≤ 80 行；超出必须拆分 |
| **面向接口 > 面向实现** | 测试边界都是接口；具体实现可替换 |
| **Fail fast** | 错误立即返回；不"先处理一半再说" |
| **零警告策略** | golangci-lint 必须 0 warning；任何 nolint 都要写注释解释 |

## 13.2 包组织规范

### 13.2.1 包命名

| 维度 | 规范 |
| --- | --- |
| 长度 | ≤ 12 字符；优先单单词（`fs` / `log` / `ast`） |
| 语言 | 全小写；无下划线；无连字符 |
| 名词 | 用名词，不用动词（`storage` 而不是 `store`，避免与方法名冲突） |
| 复数 | 不用复数（`utils` ❌，`util` 也 ❌；用 `helper` 或具体名） |
| 缩写 | 仅使用业界公认缩写：`http`/`url`/`api`/`json`/`db` |

**反例**：
- ❌ `utils` / `helpers` / `common` —— 没有语义
- ❌ `linctl_codegen` —— 含下划线
- ❌ `Project` —— 包名应小写

### 13.2.2 包职责

每个包必须满足：

1. **Single Responsibility**：能用一句话说清楚"这个包是干嘛的"。
2. **依赖方向单一**：按 [01-architecture.md §1.3](./01-architecture.md) 的层级依赖。
3. **无循环依赖**：CI 中 `go-cleanarch` 强制检查。
4. **public API 最小化**：未来需要的 API 才导出；其他用小写。

### 13.2.3 文件组织

```
internal/<pkg>/
├── doc.go              # 包注释（强制）
├── <type>.go           # 单一类型 + 其方法
├── <type>_test.go      # 同包白盒测试
├── <type>_external_test.go  # 黑盒测试（验证 public API）
└── testdata/           # 测试 fixture
```

**doc.go 模板**：

```go
// Package codegen implements the code generation pipeline:
// computing Plans from Project + Disk state, and applying them.
//
// The package is structured as:
//   - Pair / PairBuilder: 文件清单的最小单位
//   - Plan / Action: 待执行变更的清单
//   - Planner: 把 Project 翻译成 Plan
//   - Applier: 执行 Plan
//
// All operations on Plan are pure (no IO); Applier delegates IO to fs.FileManager.
package codegen
```

## 13.3 命名规范

### 13.3.1 通用规则

| 类型 | 规范 | 示例 |
| --- | --- | --- |
| 包 | 小写单单词 | `codegen` / `template` |
| 类型 | UpperCamelCase | `PairBuilder` / `LinctlError` |
| 接口 | 名词或 -er 后缀 | `Component` / `ASTMutator` / `Reader` |
| 公开方法 | UpperCamelCase | `Render(...)` |
| 私有方法 | lowerCamelCase | `renderInternal(...)` |
| 公开常量 | UpperCamelCase | `MaxPairCount` |
| 包级私有变量 | lowerCamelCase | `defaultLogger` |
| 局部变量 | lowerCamelCase / 短名 | `i, j, n` 在循环里 |
| 错误变量 | `Err` 前缀 | `ErrNotFound` |
| 错误类型 | `Error` 后缀 | `LinctlError` |

### 13.3.2 缩写处理

Go 官方约定：缩写**整体大小写一致**。

| ✅ 正确 | ❌ 错误 |
| --- | --- |
| `URL` / `urlField` | `Url` / `URLField`（不一致） |
| `ID` / `userID` | `Id` / `userId` |
| `HTTPServer` | `HttpServer` |
| `ASTNode` / `astNode` | `AstNode` |

### 13.3.3 接收者命名

```go
// ✅ 正确
func (m *FileManager) Read(...) {...}
func (b *PairBuilder) Add(...) {...}

// ❌ 错误
func (this *FileManager) Read(...) {...}        // 不要 this/self
func (fileManager *FileManager) Read(...) {...} // 太长
```

接收者短名首字母（`f` for File, `m` for Manager），但同包内**统一**：要么都用 `m`，要么都用 `fm`。

### 13.3.4 错误命名

```go
// ✅ Sentinel 错误：ErrXxx
var ErrComponentNotFound = errors.New("component not found")

// ✅ 错误类型：XxxError
type LinctlError struct {...}

// ✅ 错误变量（局部）：err
if err := doSomething(); err != nil {...}

// ❌ e / e1 / err2
```

## 13.4 错误处理规范

### 13.4.1 三类错误

| 类别 | 示例 | 处理方式 |
| --- | --- | --- |
| **Sentinel** | `io.EOF` / `ErrNotFound` | `errors.Is(err, ErrXxx)` |
| **Type** | `*LinctlError` / `*os.PathError` | `errors.As(err, &target)` |
| **Wrapped** | `fmt.Errorf("read %s: %w", path, err)` | wrapping，保留链 |

### 13.4.2 LinctlError 强制使用场景

任何**到达用户层面**的错误必须包装为 `LinctlError`：

```go
import "github.com/<org>/linctl/internal/linctlerr"   // 包名定稿，详见 META §1.1 / §5.1

// ✅ 正确
return linctlerr.New(linctlerr.ErrConfigInvalid,
    fmt.Sprintf("invalid framework %q for component %q", c.Framework, c.Name),
    "Allowed: gin, grpc. To use kratos, install plugin: linctl plugin install kratos",
)

// ✅ 包装现有错误
data, err := os.ReadFile(path)
if err != nil {
    return linctlerr.Wrap(linctlerr.ErrEnvironment, err, fmt.Sprintf("read %s", path))
}

// ❌ 直接返回标准 errors
return errors.New("framework invalid")  // 用户看到没 hint
```

### 13.4.3 errcheck 严格

CI 强制 `errcheck`，所有返回 error 的函数必须显式处理：

```go
// ✅
if err := f.Close(); err != nil {
    log.L().Warn("close file", "err", err)
}

// ✅ 明确忽略
_ = f.Close()  // intentionally ignored, file already in error state

// ❌
f.Close()  // errcheck 失败
```

### 13.4.4 panic 仅用于"程序员错误"

```go
// ✅ 包初始化时配置错误
func init() {
    feature.MustRegister(&HealthzFeature{})  // 重复注册才会 panic
}

// ✅ 内部不变量违反
if pair.Dst == "" {
    panic("BUG: empty dst in PairBuilder.Add")
}

// ❌ 用户输入错误
if c.Framework == "" {
    panic("framework is empty")  // 应该返回 error
}
```

### 13.4.5 错误信息内容标准

```
[error_code] action: target: error_detail
```

**示例**：

```go
// ✅
return fmt.Errorf("[config_invalid] parse linctl.yaml: %s: %w", path, err)

// ❌
return fmt.Errorf("error parsing yaml: %v", err)  // 信息不够
return fmt.Errorf("Error: %s", err)              // 大写 + 多余 "Error:"
```

## 13.5 日志规范

### 13.5.1 唯一日志库：`log/slog`

```go
import "log/slog"

// 生产代码必须通过 internal/log/L() 获取
log.L().Info("apply complete",
    "files", plan.Stats.Create+plan.Stats.Update,
    "duration", elapsed,
)

// ❌ 禁止
fmt.Printf("apply complete: %d files\n", n)  // 不是结构化
log.Println("apply complete")                 // 标准库 log
zap.Info("apply complete")                    // 不是 slog
```

### 13.5.2 日志级别

| Level | 使用场景 | 示例 |
| --- | --- | --- |
| `Trace` | 极细粒度，仅 `--log-level=trace` 时输出 | "render template <name>" |
| `Debug` | 开发者调试 | "loaded 32 features", "pair override" |
| `Info` | 用户应知道的进展 | "apply complete", "backup created" |
| `Warn` | 不影响主流程的异常 | "buf not installed, skip format" |
| `Error` | 错误（但程序未崩溃） | "failed to write file" |

### 13.5.3 字段命名

```go
// ✅ snake_case + 短名
log.L().Info("plan computed",
    "create", n1,
    "update", n2,
    "conflict", n3,
    "duration_ms", elapsed.Milliseconds(),
)

// ❌
log.L().Info("plan",
    "createCount", n1,         // CamelCase
    "Conflicts", n3,           // 大写
    "Duration", elapsed.String(),  // 字符串而非数字
)
```

### 13.5.4 敏感信息脱敏

```go
// ✅ 脱敏
log.L().Info("loaded project",
    "module", proj.Metadata.Module,
    "author", maskEmail(proj.Metadata.Author.Email),  // 仅 ***@example.com
)

// ❌
log.L().Info("loaded", "email", proj.Metadata.Author.Email)  // 泄漏邮箱
```

## 13.6 注释规范

### 13.6.1 必须有注释

| 元素 | 规范 |
| --- | --- |
| 包（`doc.go`） | 一段说明包职责 |
| 公开类型 | `// TypeName 是...` |
| 公开方法 | `// MethodName 做...` |
| 公开常量 | `// ConstName 表示...` |

格式：**Go 风格**（首行用类型/函数名开头）。

```go
// FileManager 提供原子写入、hash 追踪、冲突策略的文件操作。
// 所有路径都相对于 workDir。
type FileManager struct { ... }

// Read 读取相对路径文件。如果不存在，返回 fs.ErrNotExist 类型错误。
func (m *FileManager) Read(path string) ([]byte, error) { ... }
```

### 13.6.2 禁止的注释（Code Comment Anti-patterns）

| 反模式 | 原因 |
| --- | --- |
| `// 增加 1` 在 `i++` 旁 | 翻译代码，无价值 |
| `// TODO 修复` 无说明 | 不知道修啥 |
| `// 这里有点 hacky` | 含糊；要么 fix 要么写清楚为什么 |
| 大段被注释的代码 | 用 git；删除 |
| `// 作者: xxx 时间: 2026-04-25` | 用 git blame |

### 13.6.3 推荐注释

```go
// EnsureDir 确保目录存在。如果中途任意一段是文件而非目录，返回 PathError。
// 使用了 os.MkdirAll，注意它在权限不足时**不会**自动 chmod。
func EnsureDir(path string) error { ... }

// HashCommentPrefix 是 linctl 在生成文件末尾追加的特殊注释前缀，
// 用于在下次 plan 时识别"由 linctl 生成且未被人改过"的文件。
//
// 示例：// linctl: hash=abc123def456...
//
// 完整定义见 06-codegen-pipeline.md §6.7。
const HashCommentPrefix = "// linctl: hash="
```

### 13.6.4 TODO 规范

```go
// TODO(@username, #NN): 描述
// FIXME(@username, #NN): 描述（更紧急）
```

CI 中 `grep -r "TODO\b" internal/` 限制最多 N 条；超出报警。

## 13.7 函数与类型设计

### 13.7.1 函数长度

| 类型 | 限制 | 拆分指导 |
| --- | --- | --- |
| 普通函数 | ≤ 80 行 | 提取成名 helper |
| 测试函数 | ≤ 200 行 | 拆 sub-test |
| `main` / `init` | ≤ 30 行 | 一切复杂逻辑都进 internal |

`golangci-lint cyclop` 设 `max-complexity: 15`。

### 13.7.2 参数

| 数量 | 处理 |
| --- | --- |
| ≤ 3 | 直接列出 |
| 4-6 | 考虑用 struct 包装（命名参数效果） |
| > 6 | 必须用 `XxxOptions` struct |

```go
// ✅ 合理
func Render(tplPath string, data any) ([]byte, error)

// ✅ 多参数用 Options
type ApplyOptions struct {
    Strategy   ConflictStrategy
    Prune      bool
    BackupDir  string
    NoBackup   bool
    GitStash   bool
    Verbose    bool
}

func Apply(ctx context.Context, plan *Plan, opts ApplyOptions) error
```

### 13.7.3 接口设计

- **Small interfaces**：理想情况 1-3 个方法（[Rob Pike: "The bigger the interface, the weaker the abstraction."](https://go-proverbs.github.io/)）。
- **Accept interfaces, return concrete types**：函数参数用接口；返回具体类型。
- **Define interfaces close to consumer**：在使用方定义，不在实现方导出。

```go
// ✅ consumer 定义接口
package codegen

type FileWriter interface {
    Write(path string, content []byte) error
}

func Apply(plan *Plan, fw FileWriter) error { ... }

// ❌ implementer 定义巨型接口
package fs

type FileSystem interface {
    Read(...) ...
    Write(...) ...
    Mkdir(...) ...
    // ... 20 个方法
}
```

### 13.7.4 不变量约束

构造函数返回前 + 公共方法入口处校验前置条件。无效输入立即报错。

```go
func New(workDir string, dryRun bool, strategy ConflictStrategy) (*FileManager, error) {
    if workDir == "" {
        return nil, errors.New("workDir is empty")
    }
    if !strategy.IsValid() {
        return nil, fmt.Errorf("invalid strategy %q", strategy)
    }
    return &FileManager{...}, nil
}
```

## 13.8 并发规范

### 13.8.1 默认串行；显式并发

```go
// ✅ 显式
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(runtime.NumCPU())
for _, action := range plan.Actions {
    action := action  // shadow before goroutine
    g.Go(func() error {
        return executeAction(ctx, action)
    })
}
return g.Wait()

// ❌ 隐式
go applyAction(action)  // 谁等？错误怎么传？
```

### 13.8.2 共享状态

| 场景 | 选择 |
| --- | --- |
| 仅读 | 不需要锁 |
| 单 goroutine 写、多读 | `atomic` / `sync.Map` |
| 多 goroutine 读写 | `sync.RWMutex` |
| 一次性初始化 | `sync.Once` |
| Channel pipeline | `chan T` |

### 13.8.3 context 传递

- 所有可能阻塞 / 长耗时的函数必须接收 `ctx context.Context`，且为**第一个参数**。
- 不存 `ctx` 到 struct（特殊情况除外）。
- 不传 `ctx = nil`；用 `context.TODO()` 占位。

```go
// ✅
func Apply(ctx context.Context, plan *Plan) error { ... }

// ❌
func Apply(plan *Plan) error {
    ctx := context.Background()  // 谁取消？
    ...
}
```

## 13.9 i18n 国际化策略

### 13.9.1 设计目标

linctl 是面向**全球开发者**的工具。CLI 输出 + 错误消息 + 文档骨架必须支持多语言：

| 范围 | i18n 优先级 |
| --- | --- |
| 错误消息 | ⭐⭐⭐⭐⭐ |
| CLI 输出（plan/apply 报告） | ⭐⭐⭐⭐ |
| 帮助文本（`--help`） | ⭐⭐⭐ |
| 文档骨架（生成项目的 README） | ⭐⭐⭐ |
| 内部日志 | ⭐（仅英文） |
| ADR / 设计文档 | ⭐ 中文优先 + 英文摘要 |

### 13.9.2 默认语言策略

| 元素 | 默认 | 切换方式 |
| --- | --- | --- |
| CLI 输出 | 英文 | `LINCTL_LANG=zh-CN` 环境变量 |
| 错误消息 | 英文 | 同上 |
| 文档骨架 | `defaults.docs.languages` 配置 | YAML |

### 13.9.3 实现选型

| 方案 | 评估 | 决策 |
| --- | --- | --- |
| `nicksnyder/go-i18n` | 主流；JSON/YAML 资源；CLI 工具齐 | ✅ 选用 |
| `golang.org/x/text/message` | 标准库系；功能少 | 备选 |
| 自研 map[string]string | 简单 | 早期用，后期换 |

### 13.9.4 错误码 → 多语言映射

```go
// internal/i18n/messages_en.yaml
config_invalid:
  message: "[config_invalid] {{.detail}}"
  hint: "{{.hint}}"

component_not_found:
  message: "[component_not_found] component {{.name}} not found"
  hint: "Available components: {{.available}}"

// internal/i18n/messages_zh-CN.yaml
config_invalid:
  message: "[配置错误] {{.detail}}"
  hint: "{{.hint}}"

component_not_found:
  message: "[组件未找到] 组件 {{.name}} 不存在"
  hint: "可用组件：{{.available}}"
```

```go
// internal/i18n/i18n.go
package i18n

import (
    "embed"

    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
    "gopkg.in/yaml.v3"
)

//go:embed messages_*.yaml
var messagesFS embed.FS

var bundle *i18n.Bundle

func Init(lang string) error {
    bundle = i18n.NewBundle(language.English)
    bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

    files := []string{"messages_en.yaml", "messages_zh-CN.yaml"}
    for _, f := range files {
        data, err := messagesFS.ReadFile(f)
        if err != nil {
            return err
        }
        bundle.MustParseMessageFileBytes(data, f)
    }
    return nil
}

func T(messageID string, data map[string]any) string {
    loc := i18n.NewLocalizer(bundle, currentLang())
    msg, _ := loc.Localize(&i18n.LocalizeConfig{
        MessageID:    messageID,
        TemplateData: data,
    })
    return msg
}

func currentLang() string {
    if l := os.Getenv("LINCTL_LANG"); l != "" {
        return l
    }
    if l := os.Getenv("LANG"); strings.HasPrefix(l, "zh") {
        return "zh-CN"
    }
    return "en"
}
```

### 13.9.5 在错误中使用

```go
// ✅
return linctlerr.New(linctlerr.CodeComponentNotFound,
    i18n.T("component_not_found", map[string]any{
        "name":      name,
        "available": strings.Join(available, ", "),
    }),
    i18n.T("component_not_found.hint", map[string]any{...}),
)
```

### 13.9.6 终端编码

- Windows 必须显式 `chcp 65001` 或用 `golang.org/x/sys/windows` 设置 console code page 为 UTF-8。
- linctl 启动时自动检测 + 设置（仅 Windows）。

```go
// internal/i18n/console_windows.go
//go:build windows

package i18n

import "golang.org/x/sys/windows"

func init() {
    _ = windows.SetConsoleOutputCP(windows.CP_UTF8)
}
```

## 13.10 PR 规范

### 13.10.1 PR 标题（Conventional Commits）

```
<type>(<scope>): <subject>

<type>: feat / fix / docs / refactor / test / chore / perf / build / ci
<scope>: codegen / template / ast / feature / cli / fs / log / ...
<subject>: 一句话描述（小写开头，无句号）
```

**示例**：

```
feat(codegen): introduce Plan/Apply two-phase pipeline
fix(ast): preserve comments when adding interface methods
docs(09): add Worker variant validation rules
test(template): cover funcmap.kebab edge cases
```

### 13.10.2 PR 描述模板

```markdown
## Why

为什么需要这个改动？解决什么 issue？

## What

具体改了什么？关键决策点是什么？

## How (Implementation Detail)

技术实现细节（diff 难看出的），如算法选择 / 数据结构。

## Test Plan

- [ ] 单测（含覆盖率）
- [ ] 集成测试
- [ ] E2E（如涉及）
- [ ] Manual test 步骤

## Breaking Changes

如有，列出 + 提供迁移指南。

## Related

- Issue #NN
- ADR #XXX (如新增决策)
```

### 13.10.3 PR Reviewer Checklist

```markdown
- [ ] PR 标题符合 Conventional Commits
- [ ] commit messages 清晰
- [ ] 代码符合本文档规范（命名/错误/日志/注释）
- [ ] 单测覆盖率达标（核心包 ≥ 80%）
- [ ] CI 全绿（lint + unit + integration + e2e）
- [ ] 如新增依赖，更新了 10-tech-stack.md
- [ ] 如关键决策，写了 ADR
- [ ] 如改 schema，考虑了向后兼容
- [ ] 如改公开 API，提到 SemVer 影响
- [ ] 文档已更新（含本 PR 涉及的设计文档）
- [ ] 用户面错误信息支持 i18n（如适用）
```

## 13.11 安全编码（防御式）

| 主题 | 规范 |
| --- | --- |
| 路径处理 | 用 `filepath.Clean`；拒绝 `..` 越界（详见 [15-security-model.md](./15-security-model.md)） |
| 子进程 | 用 `exec.CommandContext`；明确 PATH；详见 [15-security-model.md §15.2.1](./15-security-model.md#1521-hook-执行策略-hook-execution-policy) |
| 模板渲染 | `Option("missingkey=error")` 严格模式 |
| YAML 加载 | `KnownFields(true)` 严格模式 |
| 网络 | 默认禁用；遥测必须 opt-in；详见 [15-security-model.md](./15-security-model.md) |
| 敏感信息 | 不打印到日志；不上报到遥测 |
| 输入校验 | 所有外部输入（CLI / YAML / file content）必须 validator 校验 |

## 13.12 性能规范

| 操作 | 指南 |
| --- | --- |
| 字符串拼接 | > 5 段用 `strings.Builder` |
| Slice 预分配 | `make([]T, 0, n)` 而不是 `nil` 然后 append |
| 大对象传递 | 用指针 |
| `defer` | 在循环里慎用（每次 defer 有开销） |
| 反射 | 仅在必要时（如 funcmap）；其余用类型断言 |

## 13.13 工具链强制

CI 必须执行下列检查，任何一项失败 = PR 不能 merge：

```yaml
checks:
  - gofumpt --diff (no diff allowed)
  - golangci-lint run --timeout=5m
  - go vet ./...
  - go test -race -short ./...
  - coverage >= 65% (overall)
  - binary size <= 15 MB
  - dependency count <= 65 (direct + indirect)
  - layer dependency check (scripts/check-layer-deps.sh)
  - template hardcode check (scripts/check-templates.sh)
  - glossary check (scripts/check-glossary.sh)
```

## 13.13.5 OpenTelemetry 集成约定（**单一规范**，与 §10.6 / §14.4 同源）

> 详见 [14-observability.md §14.4](./14-observability.md#144-l2-内部-trace-体系)（**主规范**）+ [10-tech-stack.md §10.6](./10-tech-stack.md#106-体积约束)（体积约束）+ [META 决策书 §1.10 / §5.7](./META-fix-decisions-2026-04-25.md)。本节仅给出**编码风格层面**的强制约定，文件路径与包名以 §14.4 为准。

### 文件路径与包名（与 §14.4 一致）

OTel 相关代码必须放在 **`internal/diag/`** 包下，按下列双文件模式拆分：

```go
// internal/diag/otel_export.go
//go:build otel
// +build otel

package diag

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
)

func init() {
    // OTel exporter 注册逻辑
}
```

并提供「无 OTel」时的 stub 文件：

```go
// internal/diag/otel_export_stub.go
//go:build !otel
// +build !otel

package diag

func init() {
    // no-op
}
```

### 构建命令

| 目标 | 命令 | 产物 |
| --- | --- | --- |
| 默认（无 OTel） | `go build ./...` 或 `make build` | ≤15MB，零 OTel 依赖 |
| OTel 启用 | `go build -tags otel ./...` 或 `make build-otel` | ≤25MB（详见 §10.6 体积预算），含 OTel SDK |

### CI 双构建

```yaml
build:
  matrix:
    include:
      - target: default
        cmd: go build -o linctl .
      - target: otel
        cmd: go build -tags otel -o linctl-otel .
  steps:
    - uses: actions/upload-artifact@v4
      with:
        name: linctl-${{ matrix.target }}
        path: linctl${{ matrix.target == 'otel' && '-otel' || '' }}
```

## 13.14 与 osbuilder 的对比

| 维度 | osbuilder | linctl |
| --- | --- | --- |
| 日志 | klog + apex/log + fmt 混用 | 仅 slog |
| 错误处理 | `fmt.Printf` 吞错 | LinctlError + Code + Hint |
| 注释 | 函数级稀疏 | 公开 API 必须有 |
| 包组织 | 各包职责模糊 | 严格分层 + go-cleanarch 检查 |
| i18n | 仅中文 + 部分英文混杂 | go-i18n + 资源文件 |
| PR 规范 | 无 | Conventional Commits + 模板 |

## 13.15 Open Questions

| 问题 | 待决议 |
| --- | --- |
| 是否引入 `golines` 自动折行？ | 评估中，目前依靠 reviewer |
| ADR / 设计文档是否最终也英文化？ | 远期；目前中文 + 关键英文摘要 |
| 是否引入 generic-typed FuncMap？ | Go 1.22+ 可考虑；评估收益 |

---

## 修订记录

| 日期 | 版本 | 变更 |
| --- | --- | --- |
| 2026-04-25 | 0.1 | 初始版本，含 i18n 策略 |
| 2026-04-25 | 0.2 | 按 [META-fix-decisions-2026-04-25](./META-fix-decisions-2026-04-25.md) 修订：i18n 选用的 `nicksnyder/go-i18n` 已回写到 [10-tech-stack.md §10.3.9](./10-tech-stack.md) 直接依赖表 |

---

下一步阅读：[14-observability.md](./14-observability.md)

---

_Last reviewed: 2026-04-25_
