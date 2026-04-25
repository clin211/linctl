# 02. linctl 代码组织结构

## 2.1 仓库总览

linctl 仓库**自身**的目录结构（不是它生成的项目结构）。

```
linctl/
├── cmd/
│   └── linctl/
│       └── main.go                # 唯一入口，<50 行
├── internal/                      # 全部业务代码
│   ├── cli/                       # L3: cobra 命令实现（每命令一个文件）
│   ├── orchestrator/              # L2: 业务编排（Planner/Applier/Reporter，**不含 Loader**：Loader 在 internal/project/）
│   ├── feature/                   # L1: Feature 注册中心 + 内置 Feature
│   ├── component/                 # L1: 组件抽象 + 内置组件（WebServer/Worker/CLI）
│   ├── template/                  # L1: 模板引擎 + FuncMap
│   ├── codegen/                   # L1: 代码生成调度（Plan/Pair/Apply）
│   ├── ast/                       # L1: AST 注入（Go via dst, Proto via protocompile）
│   ├── project/                   # L1: Project schema + Load/Save/Validate
│   ├── fs/                        # L0: afero + 原子写 + hash 追踪 + flock
│   ├── validate/                  # L0: validator.v10 + JSON Schema 导出
│   ├── log/                       # L0: log/slog 配置（彩色 handler、verbose）
│   ├── telemetry/                 # L0: opt-in 遥测
│   ├── security/                  # L0: SafeJoin + HookExecutor（执行策略 restricted/confirm/unrestricted）+ Redact
│   ├── diag/                      # L0: 内部 trace + Profiler
│   ├── ui/                        # L0: 终端 UI（颜色/表格/spinner/确认对话）
│   ├── version/                   # L0: 版本注入入口
│   ├── i18n/                      # L0: go-i18n 多语言（错误/输出本地化）
│   └── linctlerr/                 # L0: LinctlError 类型（详见 META 决策书 §1.1）
├── templates/                     # 模板源（go:embed 内嵌）
│   ├── common/                    # 全部组件共用
│   ├── framework/                 # 框架特定
│   │   ├── gin/
│   │   ├── grpc/
│   │   └── grpc-gateway/
│   ├── storage/                   # 存储特定
│   │   ├── memory/
│   │   ├── gorm-mysql/
│   │   ├── gorm-postgres/
│   │   ├── gorm-sqlite/
│   │   └── mongo/
│   ├── deploy/                    # 部署特定
│   │   ├── docker/
│   │   ├── kubernetes/
│   │   └── systemd/
│   ├── feature/                   # 特性特定
│   │   ├── healthz/
│   │   ├── opentelemetry/
│   │   ├── user/
│   │   ├── websocket/
│   │   └── preloader/
│   ├── component/                 # 组件级共用模板
│   │   ├── webserver/
│   │   ├── worker/
│   │   └── cli/
│   └── project/                   # 项目级文件
│       ├── go.mod.tpl
│       ├── README.md.tpl
│       ├── Makefile.tpl
│       ├── golangci.yaml          # 静态资产，不渲染
│       ├── gitignore.tpl
│       ├── air.toml.tpl
│       └── docs/                  # 文档骨架（zh-CN/en-US）
├── docs/                          # linctl 自身的文档
│   ├── README.md
│   ├── 00-overview.md             # → 来自当前 lin/docs
│   ├── 01-architecture.md
│   └── ...
├── examples/                      # 用户示例
│   ├── linctl.yaml                # 完整示例配置
│   ├── minimal.yaml               # 最小示例
│   └── README.md
├── tests/                         # 测试代码（与生产代码分离）
│   ├── e2e/                       # 端到端：跑完整 new + add api + go build
│   ├── snapshot/                  # 模板 snapshot 测试
│   ├── integration/               # 集成：MemMapFs 测多个命令组合
│   └── fixtures/                  # 测试 fixture（项目配置等）
├── scripts/                       # 工具脚本
│   ├── boilerplate.txt            # 版权头
│   ├── gen-jsonschema.sh          # 从 Go struct 生成 JSON Schema
│   ├── coverage.sh                # 覆盖率报告
│   └── release.sh                 # 发布脚本
├── tools/                         # 构建期工具（tools.go 模式）
│   └── tools.go
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   ├── release.yml
│   │   └── docs.yml
│   └── ISSUE_TEMPLATE/
├── .golangci.yaml                 # 自身的 lint 配置
├── .goreleaser.yaml               # 跨平台发布
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE                        # MIT
├── CHANGELOG.md
└── CONTRIBUTING.md
```

## 2.2 包详细说明（按层从下到上）

### L0：基础设施层

#### `internal/fs/`

```
internal/fs/
├── manager.go      # FileManager: 原子写入 + hash 追踪
├── manager_test.go
├── memfs.go        # afero.NewMemMapFs 工厂（仅测试用）
├── hash.go         # ExtractEmbeddedHash / AppendHashComment
└── walker.go       # 遍历项目目录，识别 generated 文件
```

关键 API：

```go
// internal/fs/manager.go
package fs

import "github.com/spf13/afero"

const HashCommentPrefix = "// linctl: hash="

type FileManager struct {
    fs       afero.Fs
    workDir  string
    dryRun   bool
    strategy ConflictStrategy
    hashes   map[string]string
}

func New(workDir string, dryRun bool, strategy ConflictStrategy) *FileManager {
    return &FileManager{
        fs:       afero.NewOsFs(),
        workDir:  workDir,
        dryRun:   dryRun,
        strategy: strategy,
        hashes:   make(map[string]string),
    }
}

// Read 读取文件
func (m *FileManager) Read(path string) ([]byte, error) {
    return afero.ReadFile(m.fs, m.abs(path))
}

// Write 原子写入：先写到 .tmp 再 rename
func (m *FileManager) Write(path string, content []byte) error {
    if m.dryRun {
        return nil
    }
    abs := m.abs(path)
    if err := m.fs.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
        return err
    }
    tmp := abs + ".tmp"
    if err := afero.WriteFile(m.fs, tmp, content, 0o644); err != nil {
        return err
    }
    return m.fs.Rename(tmp, abs)
}

// Hash 计算文件 sha256
func (m *FileManager) Hash(path string) (string, error) {
    if h, ok := m.hashes[path]; ok {
        return h, nil
    }
    data, err := m.Read(path)
    if err != nil {
        return "", err
    }
    h := sha256.Sum256(data)
    hex := hex.EncodeToString(h[:])
    m.hashes[path] = hex
    return hex, nil
}

// ExtractEmbeddedHash 解析文件尾部的 hash 注释
func ExtractEmbeddedHash(content []byte) (hash string, found bool) {
    // 寻找最后一行 "// linctl: hash=<hex>"
    lines := bytes.Split(content, []byte("\n"))
    for i := len(lines) - 1; i >= 0; i-- {
        line := bytes.TrimSpace(lines[i])
        if bytes.HasPrefix(line, []byte(HashCommentPrefix)) {
            return string(line[len(HashCommentPrefix):]), true
        }
    }
    return "", false
}
```

#### `internal/log/`

```
internal/log/
├── log.go          # NewLogger / SetGlobalLevel / WithVerbose
├── handler.go      # 自定义彩色 handler（基于 slog）
└── log_test.go
```

```go
// internal/log/log.go
package log

import (
    "io"
    "log/slog"
    "os"
)

var defaultLogger *slog.Logger

func Init(verbose bool, jsonOutput bool) {
    var handler slog.Handler
    if jsonOutput {
        handler = slog.NewJSONHandler(os.Stderr, nil)
    } else {
        handler = NewColorHandler(os.Stderr, levelFromVerbose(verbose))
    }
    defaultLogger = slog.New(handler)
    slog.SetDefault(defaultLogger)
}

func L() *slog.Logger { return defaultLogger }
```

#### `internal/validate/`

```
internal/validate/
├── validator.go         # 封装 validator.v10
├── jsonschema.go        # 从 struct tag 导出 JSON Schema
├── custom_rules.go      # 自定义校验：modulePath / kindName / ...
└── validator_test.go
```

#### `internal/ui/`

```
internal/ui/
├── color.go             # 彩色输出（fatih/color）
├── emoji.go             # emoji 别名（enescakir/emoji）
├── table.go             # 对齐表格（text/tabwriter）
├── confirm.go           # 交互式 [y/n] / 多选
├── spinner.go           # 长任务进度
└── plan_print.go        # Plan 的格式化输出
```

#### `internal/security/`

```
internal/security/
├── path.go              # SafeJoin / 路径越界防护
├── hook_policy.go       # HookExecutor: 三级执行策略（详见 15-security-model.md §15.2.1）
├── redact.go            # MaskEmail / MaskPath / MaskSecrets
├── plugin_trust.go      # 插件签名校验（Phase 5）
└── *_test.go
```

> 完整设计见 [15-security-model.md](./15-security-model.md)。每个安全防护机制都有对应的攻击向量编号（A1-A10）。

#### `internal/diag/`

```
internal/diag/
├── trace.go             # StartSpan / Span 树 / 输出
├── profile.go           # CPU / Mem / Goroutine 等 profile 启停
├── output_text.go       # 树形文本输出
├── output_json.go       # JSON 输出
└── *_test.go
```

> 完整设计见 [14-observability.md](./14-observability.md)。

#### `internal/i18n/`

```
internal/i18n/
├── i18n.go              # Init / T 函数
├── messages_en.yaml     # 英文资源
├── messages_zh-CN.yaml  # 中文资源
├── console_windows.go   # Windows UTF-8 console code page 设置
└── *_test.go
```

> 完整设计见 [13-coding-standards.md §13.9](./13-coding-standards.md#139-i18n-国际化策略)。

#### `internal/linctlerr/`

> **包名定稿**（详见 [META 决策书 §1.1 / §5.1](./META-fix-decisions-2026-04-25.md#11-错误类型)）：物理路径与包名均为 `internal/linctlerr`，**不再使用** `internal/errors` 与别名 `linctlerr`。

```go
// internal/linctlerr/error.go
package linctlerr

import "fmt"

type Code string

const (
    ErrConfigInvalid     Code = "config_invalid"
    ErrComponentNotFound Code = "component_not_found"
    ErrTemplateRender    Code = "template_render"
    ErrFileConflict      Code = "file_conflict"
    ErrASTInjection      Code = "ast_injection"
    ErrNetwork           Code = "network"
    ErrEnvironment       Code = "environment"
    ErrUserAbort         Code = "user_abort"
)

type LinctlError struct {
    Code    Code   // 错误码（类型即上面的 Code，不是 ErrCode）
    Message string // 用户可读的错误信息
    Hint    string // 可执行的修复提示
    Cause   error  // 底层 error
}

func (e *LinctlError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *LinctlError) Unwrap() error { return e.Cause }

func New(code Code, msg string, hint ...string) *LinctlError {
    h := ""
    if len(hint) > 0 { h = hint[0] }
    return &LinctlError{Code: code, Message: msg, Hint: h}
}

func Wrap(code Code, err error, msg string) *LinctlError {
    return &LinctlError{Code: code, Message: msg, Cause: err}
}
```

> 字段命名严格对齐 [META 决策书 §1.1](./META-fix-decisions-2026-04-25.md#11-错误类型)：`Code / Message / Hint / Cause`；错误码常量类型为 `Code`（不是 `ErrCode`）。

### L1：核心引擎层

#### `internal/project/`

```
internal/project/
├── types.go        # Project / ProjectMeta / ProjectSpec / Component（公共类型）
├── loader.go       # Load / LoadFromFile / LoadFromBytes
├── saver.go        # Save (写 PROJECT 文件，含禁止人工修改头注释)
├── defaults.go     # ApplyDefaults
├── version.go      # APIVersion 演进 (v1, v1alpha1, ...)
├── status.go       # Status 字段维护
└── *_test.go
```

#### `internal/template/`

```
internal/template/
├── engine.go       # NewEngine / Render
├── funcmap.go      # 全部模板函数
├── partial.go      # include/partial 支持
├── data.go         # TemplateData 结构
├── embed.go        # //go:embed templates/** 入口
└── *_test.go
```

#### `internal/component/`

```
internal/component/
├── component.go    # Component 接口
├── webserver.go    # WebServer
├── worker.go       # Worker（合并 osbuilder Job + MQ）
├── cli.go          # CLI
└── *_test.go
```

#### `internal/feature/`

```
internal/feature/
├── feature.go              # Feature 接口
├── registry.go             # Registry: 注册/查询/排序
├── builtin/                # 内置 Feature
│   ├── healthz.go
│   ├── opentelemetry.go
│   ├── user.go
│   ├── websocket.go
│   └── preloader.go
└── *_test.go
```

#### `internal/codegen/`

```
internal/codegen/
├── pair.go         # Pair / PairBuilder
├── plan.go         # Plan / Action / PlanStats
├── planner.go      # Planner: 计算 Plan
├── applier.go      # Applier: 执行 Plan
├── differ.go       # 文件 diff 计算（hexops/gotextdiff）
├── merger.go       # 3-way merge
└── *_test.go
```

#### `internal/ast/`

```
internal/ast/
├── go_inject.go    # Go AST 注入（基于 dave/dst）
├── go_helpers.go   # parseExpr / parseFuncDecl 助手
├── proto_inject.go # Proto 注入（基于 bufbuild/protocompile）
├── mutator.go      # ASTMutator 接口
└── *_test.go
```

### L2：业务编排层

#### `internal/orchestrator/`

```
internal/orchestrator/
├── orchestrator.go     # 主编排：Plan/Apply 全流程
├── importer.go         # 从已有项目反推 Project schema（linctl import）
├── reporter.go         # Reporter：彩色输出
├── conflict.go         # 冲突解决策略
└── *_test.go
```

> **`ProjectLoader` 不放在本包**：它属于 L1 领域模型，物理路径为 `internal/project/loader.go`（详见上文 §`internal/project/` 与 [META 决策书 §1.2](./META-fix-decisions-2026-04-25.md#12-projectloader-物理路径)）。L2 编排只消费已加载的 `*project.Project`，不做 I/O。

### L3：CLI 命令层

#### `internal/cli/`

```
internal/cli/
├── root.go              # NewRootCommand：组装命令树 + 全局 flag
├── globals.go           # 全局 Options（log level / no-color / dry-run / ...）
├── cmd_new.go           # linctl new
├── cmd_add.go           # linctl add (api/worker/cli/...)
├── cmd_plan.go          # linctl plan
├── cmd_apply.go         # linctl apply
├── cmd_lint.go          # linctl lint
├── cmd_doctor.go        # linctl doctor
├── cmd_import.go        # linctl import
├── cmd_version.go       # linctl version
├── cmd_completion.go    # linctl completion bash/zsh/fish
├── cmd_plugin.go        # linctl plugin (list/install/remove)
└── *_test.go
```

每个**变更类** `cmd_*.go` 文件实现 5 段式生命周期 `Complete → Validate → Plan → Apply → Report`（详见 [01-architecture.md §1.6](./01-architecture.md#16-命令生命周期统一模式) 与 [META 决策书 §1.6](./META-fix-decisions-2026-04-25.md#16-cli-生命周期)）：

```go
// internal/cli/cmd_new.go
package cli

import (
    "context"

    "github.com/spf13/cobra"

    "github.com/<org>/linctl/internal/codegen"
    "github.com/<org>/linctl/internal/orchestrator"
)

type newOptions struct {
    Dir       string
    Module    string
    Framework string
    Storage   string
    Features  []string
    Force     bool
}

func newCmdNew() *cobra.Command {
    o := &newOptions{}
    cmd := &cobra.Command{
        Use:   "new [DIR]",
        Short: "Generate a new project",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            if err := o.complete(ctx, args); err != nil { return err }
            if err := o.validate(ctx); err != nil { return err }
            plan, err := o.plan(ctx)
            if err != nil { return err }
            // Phase 1 退化：默认 --auto-approve（plan/apply 子命令在 Phase 4 才暴露）
            rep, err := o.apply(ctx, plan)
            if err != nil { return err }
            return o.report(ctx, rep)
        },
    }
    cmd.Flags().StringVar(&o.Module, "module", "", "Go module path")
    cmd.Flags().StringVar(&o.Framework, "framework", "gin", "Web framework")
    cmd.Flags().StringVar(&o.Storage, "storage", "memory", "Storage backend")
    cmd.Flags().StringSliceVar(&o.Features, "features", nil, "Features to enable")
    cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "Overwrite existing files")
    return cmd
}

// 5 段式生命周期：Complete → Validate → Plan → Apply → Report
func (o *newOptions) complete(ctx context.Context, args []string) error                       { /* 填充默认值、加载 Project */ }
func (o *newOptions) validate(ctx context.Context) error                                      { /* 校验入参 */ }
func (o *newOptions) plan(ctx context.Context) (*codegen.Plan, error)                         { /* 计算变更 */ }
func (o *newOptions) apply(ctx context.Context, plan *codegen.Plan) (*orchestrator.Report, error) { /* 执行 Plan */ }
func (o *newOptions) report(ctx context.Context, rep *orchestrator.Report) error              { /* 打印结果 */ }
```

### L4：CLI 入口层

#### `cmd/linctl/main.go`

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/<org>/linctl/internal/linctlerr"
    "github.com/<org>/linctl/internal/cli"
)

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    rootCmd := cli.NewRootCommand()
    rootCmd.SetContext(ctx)

    if err := rootCmd.Execute(); err != nil {
        var lerr *linctlerr.LinctlError
        if errors.As(err, &lerr) {
            // 已经在 cli 层友好打印过
            os.Exit(exitCodeFor(lerr.Code))
        }
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func exitCodeFor(code linctlerr.Code) int {
    switch code {
    case linctlerr.ErrUserAbort:
        return 130 // SIGINT 风格
    case linctlerr.ErrConfigInvalid, linctlerr.ErrComponentNotFound:
        return 2
    default:
        return 1
    }
}
```

## 2.3 模板目录组织（templates/）

模板目录设计为 **"分层 + 组合"** 风格，避免 osbuilder 那种"一个 mb-apiserver 目录里塞所有可能性"的做法。

```
templates/
├── common/
│   ├── go.mod.tpl
│   ├── README.md.tpl
│   ├── Makefile.tpl
│   ├── golangci.yaml             # 静态文件
│   ├── gitignore.tpl
│   ├── air.toml.tpl
│   ├── boilerplate.txt
│   └── docs/                     # 文档骨架
│       ├── zh-CN/
│       └── en-US/
├── component/
│   ├── webserver/
│   │   ├── cmd/main.go.tpl
│   │   ├── cmd/options.go.tpl
│   │   ├── cmd/server.go.tpl
│   │   ├── internal/server.go.tpl
│   │   ├── internal/wire.go.tpl
│   │   └── configs/server.yaml.tpl
│   ├── worker/
│   │   ├── cmd/main.go.tpl
│   │   ├── internal/worker.go.tpl
│   │   └── internal/scheduler.go.tpl
│   └── cli/
│       ├── cmd/main.go.tpl
│       └── internal/root.go.tpl
├── framework/
│   ├── gin/
│   │   ├── server.go.tpl
│   │   ├── middleware/
│   │   └── handler/
│   │       ├── handler.go.tpl
│   │       └── api/
│   │           └── resource.go.tpl   # 单 REST 资源模板
│   └── grpc/
│       ├── server.go.tpl
│       ├── interceptor/
│       └── handler/
│           ├── handler.go.tpl
│           └── api/
│               └── resource.go.tpl
├── storage/
│   ├── memory/
│   │   └── store/
│   │       ├── store.go.tpl
│   │       └── resource.go.tpl
│   ├── gorm-mysql/
│   │   ├── store/
│   │   ├── model/
│   │   └── migrations/
│   └── ...
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile.multi-stage.tpl
│   │   └── Dockerfile.runtime-only.tpl
│   ├── kubernetes/
│   │   ├── deployment.yaml.tpl
│   │   ├── service.yaml.tpl
│   │   └── configmap.yaml.tpl
│   └── systemd/
│       └── service.tpl
└── feature/
    ├── healthz/
    │   ├── healthz.proto
    │   ├── handler.go.tpl
    │   └── README.md.tpl
    ├── opentelemetry/
    │   ├── otel.go.tpl
    │   ├── metrics.go.tpl
    │   └── README.md.tpl
    ├── user/
    │   ├── user.proto
    │   ├── biz.go.tpl
    │   ├── store.go.tpl
    │   ├── handler.go.tpl
    │   └── middleware/{authn,authz}.go.tpl
    ├── websocket/
    │   ├── ws.proto
    │   └── handler.go.tpl
    └── preloader/
        ├── asyncstore.go.tpl
        └── README.md.tpl
```

**命名约定**：
- 模板文件后缀 `.tpl`（除 `.proto` / 静态资产文件外）
- 静态资产（如 `golangci.yaml`）不加 `.tpl`，直接拷贝
- 同一个目标资源（如 `gin/handler/api/resource.go.tpl`）由不同 Feature 通过 Apply 决定是否引入

## 2.4 测试目录组织

```
tests/
├── e2e/                            # 端到端：黑盒测试
│   ├── new_project_test.go         # 创建项目 → go build → 真正跑起来
│   ├── add_api_test.go
│   └── upgrade_test.go             # 升级 linctl 后旧项目能否 reconcile
├── snapshot/                       # 模板快照
│   ├── webserver_gin_test.go
│   ├── webserver_grpc_test.go
│   ├── worker_test.go
│   └── golden/                     # golden files
│       ├── webserver_gin/
│       │   ├── cmd_main.go.golden
│       │   └── internal_server.go.golden
│       └── ...
├── integration/                    # 集成：MemFS 测多命令组合
│   ├── plan_apply_test.go
│   ├── conflict_resolution_test.go
│   └── feature_combinations_test.go
└── fixtures/                       # 测试 fixture
    ├── projects/
    │   ├── minimal.yaml
    │   ├── full.yaml
    │   └── invalid.yaml
    └── templates/                  # 用于单测的迷你模板集
```

## 2.5 构建工件目录

```
_output/                            # gitignored
├── bin/                            # 本地构建产物
│   └── linctl
├── coverage.html
├── coverage.out
└── dist/                           # goreleaser 跨平台产物
    ├── linctl_v1.0.0_darwin_arm64.tar.gz
    ├── linctl_v1.0.0_linux_amd64.tar.gz
    └── ...
```

## 2.6 Go module 路径与命名

- Module 路径：`github.com/<org>/linctl`（待选定）
- Major version：`v1.0.0` 起步
- 内部包前缀：`internal/`，禁止外部直接 import（Go 编译器强制）
- 公开 API：暂不暴露（CLI 工具不需要 SDK，未来如果有需求再开 `pkg/`）

## 2.7 文件命名规范

| 类型 | 规范 | 示例 |
| --- | --- | --- |
| Go 源文件 | snake_case | `feature_registry.go` |
| Go 测试文件 | `_test.go` 后缀 | `feature_registry_test.go` |
| 模板文件 | `.tpl` 后缀（特殊扩展名除外） | `server.go.tpl` / `handler.proto` |
| Markdown | `kebab-case.md` | `coding-standards.md` |
| Mermaid 图 | `kebab-case.mmd` | `seq-add-api.mmd` |
| YAML 文件 | `kebab-case.yaml` | `linctl.yaml` |
| Shell 脚本 | `kebab-case.sh` | `gen-jsonschema.sh` |

## 2.8 与 osbuilder 仓库结构的差异

| 维度 | osbuilder | linctl | 原因 |
| --- | --- | --- | --- |
| 模板存储 | `internal/osbuilder/tpl/` + `statik/statik.go` | `templates/` + `//go:embed` | 标准库替代归档项目 |
| 命令组织 | `internal/osbuilder/cmd/cmd.go` 顶层 + `cmd/create/*.go` 子目录 | 全部在 `internal/cli/` 下，按功能命名 | 扁平化，减少嵌套 |
| 类型定义 | `internal/osbuilder/types/` 与 `cmd/create/` 强耦合 | `internal/project/` 独立，纯数据类型 | 分离模型与命令 |
| 文件操作 | `internal/osbuilder/file/` + `helper/` 重复 | 单一 `internal/fs/` | DRY |
| 测试 | `test/test_all.sh` | `tests/{e2e,integration,snapshot}/` 分层 | Go 表驱动 |
| 工具命令 | `cmd/{semver, addlicense, sysload}/` 散落 | **不内置**，外部独立工具 | 职责聚焦 |
| 模板组织 | 平铺式 `tpl/project/internal/apiserver/...` | 分层 `templates/{common,framework,storage,deploy,feature}/` | 可组合 |

## 2.9 IDE / 开发工具配置

仓库根目录提供以下配置（提交到 git）：

```
.editorconfig             # 跨编辑器的基础格式（缩进、换行）
.vscode/
├── settings.json         # gopls 配置、保存自动 gofumpt
├── extensions.json       # 推荐扩展列表
└── launch.json           # debug 配置
.cursor/
└── rules/                # Cursor 的规则文件（让 AI 理解项目）
.git/hooks/               # pre-commit 钩子（用 lefthook 或 pre-commit 框架）
```

## 2.10 lint 与 format 配置

`.golangci.yaml`（推荐 ≥ 100 行的严格配置）：

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gofumpt
    - goimports
    - revive
    - bodyclose
    - prealloc
    - errorlint
    - misspell
    - nilerr
    - exhaustive
    - dupl
    - cyclop

linters-settings:
  cyclop:
    max-complexity: 15
  revive:
    rules:
      - name: exported
      - name: var-naming
      - name: package-comments

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - dupl
        - cyclop
```

---

下一步阅读：[03-cli-design.md](./03-cli-design.md)

_Last reviewed: 2026-04-25_
