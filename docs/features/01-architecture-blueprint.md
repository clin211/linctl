# 01. 重构后架构蓝图

> **前置阅读**：[00-refactor-rationale.md](./00-refactor-rationale.md) §10「用户决策」
>
> 本文档定义重构后的整体架构，包括目录结构、模块职责、数据流、依赖关系。

---

## 1. 一图概览

```
┌──────────────────────────────────────────────────────────────────┐
│                         Entry Layer (cmd)                         │
│                       main.go (~50 行)                 │
└───────────────────────────────┬──────────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────────┐
│                       CLI Layer (internal/cli)                    │
│   root.go  new.go  add.go  lint.go  doctor.go  version.go         │
│                       (cobra 命令分发)                            │
└──────┬───────────────────┬───────────────────┬───────────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐   ┌──────────────────┐   ┌──────────────────┐
│  scaffold/  │   │      ast/        │   │     check/       │
│             │   │                  │   │                  │
│ project.go  │   │ injector.go      │   │ doctor.go        │
│ resource.go │   │ mutator_iface.go │   │ lint.go          │
│ context.go  │   │ mutator_proto.go │   │                  │
│ render.go   │   │ guard.go         │   │                  │
└──────┬──────┘   └─────────┬────────┘   └──────────────────┘
       │                    │
       ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Foundation Layer (internal/pkg)                 │
│   tpl/          fsx/         logx/        errs/                  │
│   (模板加载)    (安全文件IO)  (slog 日志)   (错误码)              │
└─────────────────────────────────────────────────────────────────┘
       ▲                    ▲
       │                    │
┌──────┴────────────────────┴──────────────────────────────────────┐
│                      Templates (embed.FS)                         │
│   internal/templates/project/    miniblog-v4 风格项目骨架        │
│   internal/templates/resource/   全栈资源骨架                    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. 目录结构（重构后）

```
lin/
├── cmd/
│   └── linctl/
│       └── main.go                 # 入口（≤50 行；signal/exit code）
│
├── internal/
│   ├── cli/                        # CLI 命令分发（cobra）
│   │   ├── root.go                 # 根命令、全局 flag
│   │   ├── new.go                  # linctl new
│   │   ├── add.go                  # linctl add
│   │   ├── lint.go                 # linctl lint
│   │   ├── doctor.go               # linctl doctor
│   │   ├── version.go              # linctl version
│   │   └── completion.go           # linctl completion <shell>
│   │
│   ├── scaffold/                   # 骨架生成核心
│   │   ├── context.go              # 项目上下文（module/appName 推断）
│   │   ├── project.go              # 项目骨架生成器（new 命令的核心）
│   │   ├── resource.go             # 资源骨架生成器（add 命令的核心）
│   │   ├── plan.go                 # 生成计划（文件列表 + 注入清单）
│   │   └── render.go               # 模板渲染封装
│   │
│   ├── ast/                        # AST 注入（保留并精简）
│   │   ├── injector.go             # 注入主入口
│   │   ├── mutator_interface.go    # 注入 biz.go/store.go 接口方法
│   │   ├── mutator_proto.go        # 注入 *.proto 中的 RPC 定义
│   │   ├── mutator_register.go     # 注入路由/errno 等注册点
│   │   └── guard.go                # 幂等性 / 防重复 / 回滚保障
│   │
│   ├── check/                      # lint / doctor 实现
│   │   ├── lint.go                 # 校验项目目录结构 + AST 完整性
│   │   └── doctor.go               # 校验环境（go/protoc/wire/...）
│   │
│   ├── templates/                  # embed.FS 模板源
│   │   ├── project/                # 项目骨架（参照 miniblog-v4）
│   │   │   ├── cmd/
│   │   │   ├── internal/
│   │   │   ├── pkg/
│   │   │   ├── api/
│   │   │   ├── configs/
│   │   │   ├── build/
│   │   │   ├── scripts/
│   │   │   ├── docs/
│   │   │   ├── go.mod.tpl
│   │   │   ├── Makefile.tpl
│   │   │   ├── README.md.tpl
│   │   │   └── ...
│   │   └── resource/               # 资源骨架（一组 8 个文件）
│   │       ├── handler.go.tpl
│   │       ├── biz.go.tpl
│   │       ├── store.go.tpl
│   │       ├── model.go.tpl
│   │       ├── conversion.go.tpl
│   │       ├── validation.go.tpl
│   │       ├── errno.go.tpl
│   │       └── proto.tpl
│   │
│   ├── pkg/                        # 内部基础库
│   │   ├── tpl/                    # 模板加载（embed + 外部覆盖）
│   │   │   └── loader.go
│   │   ├── fsx/                    # 安全文件 IO（SafeJoin、原子写）
│   │   │   ├── safe_path.go
│   │   │   └── writer.go
│   │   ├── logx/                   # slog 封装
│   │   │   └── logger.go
│   │   └── errs/                   # 错误码（保留 linctlerr 内核）
│   │       └── errs.go
│   │
│   └── version/
│       └── version.go              # ldflags 注入版本信息
│
├── tools/                          # 开发工具（保留）
│   └── tools.go
│
├── tests/                          # E2E 测试（保留）
│
├── docs/
│   ├── features/                   # 本次重构设计文档
│   │   ├── 00-refactor-rationale.md
│   │   ├── 01-architecture-blueprint.md
│   │   ├── 02-command-set.md
│   │   ├── 03-resource-scaffold.md
│   │   ├── 04-template-system.md
│   │   ├── 05-registration-strategy.md
│   │   ├── 06-migration-plan.md
│   │   ├── 07-interactive-ux.md
│   │   └── README.md
│   └── legacy/                     # 旧版设计归档
│
├── .editorconfig
├── .gitignore
├── .golangci.yaml
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
└── README.md
```

### 模块行数预算（目标）

| 模块 | 目标行数 | 说明 |
| --- | --- | --- |
| `cli/` | ~600 行 | 6 个子命令分发 |
| `scaffold/` | ~800 行 | 项目/资源生成器 + plan |
| `ast/` | ~500 行 | 接口/proto/注册三类 mutator |
| `check/` | ~400 行 | lint + doctor |
| `pkg/tpl/` | ~200 行 | 模板加载 |
| `pkg/fsx/` | ~250 行 | 安全 IO |
| `pkg/logx/` | ~80 行 | 日志封装 |
| `pkg/errs/` | ~200 行 | 错误码 |
| `version/` | ~80 行 | 版本 |
| `main.go` | ~50 行 | 入口 |
| **小计（生产代码）** | **~3,160 行** | vs 当前 ~10,525 行（**-70%**） |
| `templates/` | N/A | 模板内容不计入代码量 |

---

## 3. 模块职责矩阵

### L0：入口层

| 模块 | 职责 | 不做的事 |
| --- | --- | --- |
| `main.go` | signal 处理、调用 `cli.Execute`、错误转退出码 | 任何业务逻辑 |

### L1：CLI 层

| 模块 | 职责 | 输入 | 输出 |
| --- | --- | --- | --- |
| `cli/root.go` | 注册根命令、全局 flag、初始化 logger | argv | cobra.Command |
| `cli/new.go` | 解析 `linctl new` 参数，调用 `scaffold.NewProject` | argv | error |
| `cli/add.go` | 解析 `linctl add` 参数，调用 `scaffold.AddResource` | argv | error |
| `cli/lint.go` | 调用 `check.Lint` | argv | error |
| `cli/doctor.go` | 调用 `check.Doctor` | argv | error |
| `cli/version.go` | 打印版本 | argv | error |
| `cli/completion.go` | cobra 自带补全脚本生成 | argv | error |

### L2：核心生成层

| 模块 | 职责 | 关键 API |
| --- | --- | --- |
| `scaffold/context.go` | 推断项目上下文（module/appName/storage） | `LoadContext(rootDir, flags) (*Context, error)` |
| `scaffold/project.go` | 项目骨架生成 | `NewProject(ctx *Context) error` |
| `scaffold/resource.go` | 资源骨架生成（含 AST 注入触发） | `AddResource(ctx *Context, name string, opts) error` |
| `scaffold/plan.go` | 计算文件列表 + 注入清单 | `BuildPlan(ctx *Context, kind PlanKind) (*Plan, error)` |
| `scaffold/render.go` | 模板渲染 + 文件写入 | `Render(t *tpl.Template, vars any, dst string) error` |

### L3：能力层

| 模块 | 职责 | 关键 API |
| --- | --- | --- |
| `ast/injector.go` | 协调多个 mutator | `Inject(ctx, plan) error` |
| `ast/mutator_interface.go` | 向 `biz.go` / `store.go` 接口添加方法 | `AddInterfaceMethod(file, methods)` |
| `ast/mutator_proto.go` | 向 `*.proto` 添加 RPC 与 message | `AddRPCs(file, rpcs)` |
| `ast/mutator_register.go` | 向路由 / errno 注册点追加调用 | `AppendRegistration(file, code)` |
| `ast/guard.go` | 幂等检查（已存在不重复添加）+ 备份回滚 | `Guard(file, fn) error` |
| `check/lint.go` | 校验目录结构、import、AST 完整性 | `Lint(rootDir) (*Report, error)` |
| `check/doctor.go` | 校验工具链版本（go/protoc/wire/...） | `Doctor() (*Report, error)` |

### L4：基础层

| 模块 | 职责 | 依赖 |
| --- | --- | --- |
| `pkg/tpl/` | 加载 embed + 外部模板，解析 funcMap | `text/template` |
| `pkg/fsx/` | SafeJoin、原子写、目录创建 | `os` / `filepath` |
| `pkg/logx/` | slog 封装，区分 user-facing 和 debug | `log/slog` |
| `pkg/errs/` | 错误码、Wrap、Hint | - |

---

## 4. 依赖方向（强约束）

```
   main.go
      ↓
   cli/    ────────────────────────────────────┐
      ↓                                          │
   scaffold/  ──── ast/ ──┐                     │
      ↓          ↓         ↓                     ↓
                pkg/tpl/  pkg/fsx/  pkg/logx/  pkg/errs/
                  ↓         ↓
              embed.FS    os/filepath
```

**关键规则**：

1. **`pkg/` 子包不允许依赖 `internal/cli`、`internal/scaffold`、`internal/ast`、`internal/check`**。
2. `scaffold/` 依赖 `ast/`、`pkg/`，但 `ast/` **不允许**依赖 `scaffold/`。
3. `cli/` 是唯一允许依赖业务模块的层，反向依赖禁止。
4. `version/` 是叶子模块，不依赖任何其他 internal 包。

---

## 5. 数据流：`linctl new` 命令

```
User: linctl new myblog --module github.com/foo/myblog --storage gorm-postgres --features otel,healthz
                                         │
                                         ▼
                ┌─────────────────────────────────────────────┐
                │ cli/new.go                                   │
                │  - 解析 flag                                 │
                │  - 校验目录不存在 / 可写                     │
                └────────────────┬────────────────────────────┘
                                 ▼
                ┌─────────────────────────────────────────────┐
                │ scaffold/context.go                          │
                │  ctx := &Context{                            │
                │    RootDir:  "myblog",                       │
                │    Module:   "github.com/foo/myblog",        │
                │    AppName:  "myblog",                       │
                │    Storage:  "gorm-postgres",                │
                │    Features: ["otel","healthz"],             │
                │    Frame:    "gin",                          │
                │  }                                            │
                └────────────────┬────────────────────────────┘
                                 ▼
                ┌─────────────────────────────────────────────┐
                │ scaffold/plan.go                             │
                │  Plan{                                       │
                │    Creates: [Makefile, go.mod, cmd/...,      │
                │              internal/<app>/handler/...]     │
                │    Injects: []  // 项目骨架阶段无注入        │
                │  }                                            │
                └────────────────┬────────────────────────────┘
                                 ▼
                ┌─────────────────────────────────────────────┐
                │ scaffold/render.go                           │
                │  for each file in Plan.Creates:              │
                │    - 加载模板（pkg/tpl）                     │
                │    - 渲染（含 if eq .Storage "postgres"）    │
                │    - 原子写入（pkg/fsx）                     │
                └────────────────┬────────────────────────────┘
                                 ▼
                ┌─────────────────────────────────────────────┐
                │ stdout                                        │
                │  ✔ ~40 files created                         │
                │  📦 Next steps:                              │
                │     cd myblog && make deps && make build     │
                └─────────────────────────────────────────────┘
```

---

## 6. 数据流：`linctl add` 命令

```
User: linctl add Post   (在 myblog/ 目录下执行)
                        │
                        ▼
        ┌─────────────────────────────────────────────┐
        │ cli/add.go                                   │
        │  - 解析 args: ["Post"]                       │
        │  - 解析 flag: --with conversion,validation   │
        └────────────────┬────────────────────────────┘
                         ▼
        ┌─────────────────────────────────────────────┐
        │ scaffold/context.go                          │
        │  - 读 ./go.mod → Module                      │
        │  - 列 ./cmd/* → AppName="myblog"             │
        │  - 探测 ./internal/myblog/store/ 中的 import │
        │    → Storage="gorm-postgres"                 │
        │  - Resource = "Post"                         │
        └────────────────┬────────────────────────────┘
                         ▼
        ┌─────────────────────────────────────────────┐
        │ scaffold/plan.go                             │
        │  Plan{                                       │
        │    Creates: [                                │
        │      internal/myblog/handler/post.go,        │
        │      internal/myblog/biz/v1/post/post.go,    │
        │      internal/myblog/store/post.go,          │
        │      internal/myblog/model/post.gen.go,      │
        │      internal/myblog/pkg/conversion/post.go, │
        │      internal/myblog/pkg/validation/post.go, │
        │      internal/pkg/errno/post.go,             │
        │      pkg/api/myblog/v1/post.proto,           │
        │    ],                                         │
        │    Injects: [                                │
        │      AppendInterface(biz/biz.go,             │
        │                      "PostBiz() PostBiz"),   │
        │      AppendInterface(store/store.go,         │
        │                      "Posts() PostStore"),   │
        │      AppendRPCs(api/.../myblog.proto, [...]),│
        │      AppendRegister(internal/pkg/errno/      │
        │                     register.go, "Post"),    │
        │    ],                                         │
        │  }                                            │
        └────────────────┬────────────────────────────┘
                         ▼
        ┌─────────────────────────────────────────────┐
        │ scaffold/render.go ──→ 8 file creations      │
        └────────────────┬────────────────────────────┘
                         ▼
        ┌─────────────────────────────────────────────┐
        │ ast/injector.go                              │
        │  for each Inject in Plan:                    │
        │    1. ast/guard.go：检查是否已存在           │
        │    2. ast/mutator_*.go：执行 AST 注入        │
        │    3. 失败时 git stash / 文件备份回滚        │
        └────────────────┬────────────────────────────┘
                         ▼
        ┌─────────────────────────────────────────────┐
        │ stdout                                        │
        │  ✔ 8 files created                           │
        │  ✏ 4 files updated via AST                   │
        │  📦 Next steps:                              │
        │     make protoc && go mod tidy && make build │
        └─────────────────────────────────────────────┘
```

---

## 7. 关键抽象（Type 定义草案）

### 7.1 `scaffold.Context`

```go
package scaffold

type Context struct {
    // 项目根目录（绝对路径）
    RootDir string

    // 从 go.mod 推断
    Module string

    // 从 cmd/<app>/ 推断
    AppName string

    // 从 import / flag 推断
    Storage   string  // memory | gorm-postgres | gorm-mysql | mongo
    Framework string  // gin | grpc
    Features  []string // otel | healthz | user

    // 仅 add 命令
    Resource string  // 资源名（PascalCase，如 Post）

    // 模板源
    Templates *tpl.Loader
}

func LoadContext(rootDir string, flags Flags) (*Context, error)
```

### 7.2 `scaffold.Plan`

```go
package scaffold

type Plan struct {
    Creates []FileSpec
    Injects []InjectSpec
}

type FileSpec struct {
    TemplatePath string  // 模板相对路径
    DestPath     string  // 输出相对路径（相对 RootDir）
    Permissions  os.FileMode
}

type InjectSpec struct {
    File    string       // 目标文件相对路径
    Mutator MutatorKind  // Interface | Proto | Register
    Payload any          // 具体注入数据
}
```

### 7.3 `ast.Injector`

```go
package ast

type Injector interface {
    Inject(ctx *scaffold.Context, plan *scaffold.Plan) error
}

type Mutator interface {
    Apply(file string, payload any) error
    IsAlreadyApplied(file string, payload any) (bool, error)
}
```

### 7.4 `tpl.Loader`

```go
package tpl

type Loader struct {
    embedFS    embed.FS
    overrideDir string  // 来自 --template-dir
}

func (l *Loader) Load(relPath string) (*template.Template, error)
// 优先级：overrideDir > embed
```

---

## 8. 与 miniblog-v4 的目录映射

| miniblog-v4 路径 | 由谁生成 | 备注 |
| --- | --- | --- |
| `cmd/<app>/main.go` | `linctl new` | 初次生成 |
| `cmd/<app>/app/` | `linctl new` | 初次生成 |
| `cmd/gen-gorm-model/gen_gorm_model.go` | `linctl new` | 数据库模型反推工具（详见 [03 §3.4.1](./03-resource-scaffold.md#341-推荐工作流搭配-cmdgen-gorm-model)） |
| `pkg/db/postgres.go` / `pkg/db/mysql.go` | `linctl new` | 数据库连接器（gen-gorm-model 与运行时复用） |
| `internal/<app>/handler/<resource>.go` | `linctl add` | 每个资源一份 |
| `internal/<app>/biz/biz.go` | `linctl new` 初始化 + `linctl add` AST 注入 | 接口集合 |
| `internal/<app>/biz/v1/<resource>/<resource>.go` | `linctl add` | 业务逻辑 |
| `internal/<app>/store/store.go` | `linctl new` 初始化 + `linctl add` AST 注入 | 接口集合 |
| `internal/<app>/store/<resource>.go` | `linctl add` | 持久化实现 |
| `internal/<app>/model/<resource>.gen.go` | `linctl add` | 数据模型（gorm gen） |
| `internal/<app>/pkg/conversion/<resource>.go` | `linctl add --with conversion` | DTO 转换 |
| `internal/<app>/pkg/validation/<resource>.go` | `linctl add --with validation` | 校验 |
| `internal/pkg/errno/<resource>.go` | `linctl add` | 业务错误 |
| `internal/pkg/errno/register.go` | `linctl new` 初始化 + `linctl add` AST 注入 | 错误注册 |
| `pkg/api/<app>/v1/<app>.proto` | `linctl new` 初始化 + `linctl add` AST 注入 | proto 定义 |
| `pkg/api/<app>/v1/<resource>.proto` | `linctl add --with proto` | 资源 proto（可选） |
| `internal/pkg/middleware/` | `linctl new` | 项目级中间件 |
| `internal/pkg/contextx/` | `linctl new` | 上下文工具 |
| `internal/pkg/known/` | `linctl new` | 常量定义 |
| `internal/pkg/rid/` | `linctl new` | ID 生成 |
| `configs/<app>.yaml` | `linctl new` | 单一应用配置（含 db section；gen-gorm-model 复用） |
| `Makefile` / `Dockerfile` / `.golangci.yaml` | `linctl new` | 工程化（Makefile 含 `gen-model` target，与 miniblog-v4 一致） |

---

## 9. 关键非功能需求

| 维度 | 目标 | 验证方式 |
| --- | --- | --- |
| 启动时间 | `linctl --help` ≤ 50ms | E2E |
| `linctl new` 性能 | ≤ 1.5s（30-50 文件） | E2E |
| `linctl add` 性能 | ≤ 0.5s（8 文件 + 4 注入） | E2E |
| 二进制大小 | ≤ 6 MB（macOS arm64） | `go build && du -h` |
| 直接依赖数 | ≤ 8 个 | `go list -m all \| wc -l` |
| 单测覆盖率 | `scaffold` ≥ 80%、`ast` ≥ 90% | `go test -cover` |
| 跨平台 | macOS arm64/amd64、Linux amd64、Windows amd64 | CI matrix |

---

## 10. 关键不变式

任何重构都必须保证：

1. **生成的项目可直接 `go build`**（E2E 验证）。
2. **`linctl add X && linctl add X` 是幂等的**（重复执行不破坏代码）。
3. **AST 注入失败时，所有创建的文件全部回滚**（事务性）。
4. **所有命令在 SIGINT 时干净退出**（不留半成品）。
5. **模板内不允许出现作者本机路径**（包括 `os.Getenv("USER")`）。

---

_Last reviewed: 2026-04-29_
