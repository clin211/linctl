# 02. 命令集详细设计

> **前置阅读**：[00-refactor-rationale.md](./00-refactor-rationale.md) §10「用户决策」、[01-architecture-blueprint.md](./01-architecture-blueprint.md) §3「模块职责矩阵」
>
> **协同文档**：[03-resource-scaffold.md](./03-resource-scaffold.md)（资源文件清单）、[05-registration-strategy.md](./05-registration-strategy.md)（AST 注入语义）、[07-interactive-ux.md](./07-interactive-ux.md)（交互式 UX 流程）
>
> 本文档定义 **linctl** 的 6 个子命令：**`new` / `add` / `lint` / `doctor` / `version` / `completion`**，以及每个命令的 flag、行为、错误码、退出码与典型用法。

---

## 1. 命令一览

```
linctl
├── new <project-name>             # 生成新项目骨架（miniblog-v4 风格）
├── add <Resource>...              # 在已有项目中追加业务资源（含 AST 注入）
├── lint                           # 校验项目结构 + AST 完整性
├── doctor                         # 校验本地工具链与运行环境
├── version                        # 打印版本信息
└── completion <bash|zsh|fish|pwsh> # 生成 shell 补全脚本
```

**命令分类**：

| 类别 | 命令 | 默认交互 | 写入磁盘 | 网络访问 |
| --- | --- | --- | --- | --- |
| 生成类 | `new` / `add` | ✅（TTY 下） | ✅ | ❌ |
| 校验类 | `lint` / `doctor` | ❌ | 仅 `--fix` 时 | `doctor` 检查 `go env` 时可能间接访问 |
| 信息类 | `version` / `completion` | ❌ | ❌ | ❌ |

**设计原则**（贯穿所有命令）：

| 原则 | 实现 |
| --- | --- |
| 零配置开箱可用 | 所有 flag 都有合理默认值；TTY 下交互式向导 |
| flag 与交互无缝混合 | 已传 flag 跳过对应 prompt（详见 [07 §1.1](./07-interactive-ux.md#11-三种使用模式无缝切换)） |
| 非 TTY 自动切换 | 检测到 stdin 非 TTY 自动进入 `--non-interactive` 模式 |
| 失败可观测 | 出错打印「修复建议」与文档链接 |
| 写操作可预览 | 写入前打印 Plan 摘要；`--dry-run` 仅打印不写入 |
| 幂等友好 | `add` 重复执行不会破坏既有代码（详见 [05 §2.2](./05-registration-strategy.md)） |

---

## 2. 全局 flag

所有子命令均接受以下全局 flag（`cli/root.go` 注册）：

| flag | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--log-level` | enum | `info` | `debug` / `info` / `warn` / `error`；启用 slog |
| `--log-format` | enum | `text` | `text` / `json`；CI 推荐 `json` |
| `-C, --chdir` | path | `.` | 切换工作目录后再执行（与 `git -C` 同义） |
| `--no-color` | bool | `false` | 强制关闭终端颜色（`NO_COLOR` 环境变量同效） |
| `--non-interactive` | bool | auto | 强制非交互式；非 TTY 自动为 true |
| `-y, --yes` | bool | `false` | 跳过最终确认对话；不影响 prompt 收集 |
| `-h, --help` | bool | `false` | cobra 自带 |
| `-v, --version` | bool | `false` | 同 `linctl version`（兼容 cobra 习惯） |

> **环境变量**：`LIN_LOG_LEVEL` / `LIN_NO_COLOR` / `LIN_NON_INTERACTIVE` 可作为 fallback，优先级低于 flag。

### 2.1 工作目录解析

`-C` 与 `linctl add` 的"项目根"识别配合规则：

```
1. 若传 -C <dir> → cd <dir>
2. linctl add 在当前目录起向上回溯，第一个含 go.mod 的目录视为「项目根」
3. 找到根后，所有相对路径（创建文件、AST 注入）以根为基准
4. 若回溯到 / 仍未找到 go.mod → exit 20
```

**示例**：

```bash
$ pwd
/Users/foo/myblog/internal/myblog/handler
$ linctl add Comment            # ✅ 自动回溯到 /Users/foo/myblog/
$ cd /tmp
$ linctl -C /Users/foo/myblog add Comment   # ✅ 显式指定
```

> `linctl new` **不**回溯，因为新项目还没有 `go.mod`；目标目录由 `--output-dir` + `<project-name>` 计算。

### 2.2 结构化日志字段（slog）

`--log-format json` 输出 JSON 行；标准字段如下：

| 字段 | 类型 | 出现于 | 说明 |
| --- | --- | --- | --- |
| `time` | RFC3339 | 所有日志 | slog 标准 |
| `level` | string | 所有日志 | `debug`/`info`/`warn`/`error` |
| `msg` | string | 所有日志 | slog 标准 |
| `cmd` | string | 所有日志 | `new`/`add`/`lint`/`doctor`/`version`/`completion` |
| `phase` | string | 业务事件 | `parse`/`plan`/`render`/`inject`/`verify`/`rollback`/`done` |
| `file` | string | 文件操作 | 相对路径 |
| `mutator` | string | AST 注入 | `interface`/`proto`/`register` |
| `resource` | string | add | `Post`/`Comment`/... |
| `app` | string | add | `myblog`/`api`/`worker` |
| `duration_ms` | int64 | 操作完成 | 毫秒 |
| `exit_code` | int | 命令退出 | 见 §9 |
| `err` | string | error 级别 | 错误消息 |
| `err_code` | int | error 级别 | `errs.Code` 数值 |

**约定**：

- 用户可见输出（带 emoji 的 ✔/✗/⚠）走 stderr 的 `text` 格式（无论 `--log-format`）
- `--log-format json` 仅作用于 debug/info/warn/error 级别日志（适合 CI 解析）
- `--log-format text` 时这些日志带颜色 + 等级前缀

### 2.3 退出前最后一行 audit log

每条命令在退出前**强制**写一行 audit log（`info` 级别）：

```json
{"time":"2026-04-29T11:23:00Z","level":"info","msg":"command completed","cmd":"add","phase":"done","resource":"Post","app":"myblog","duration_ms":423,"exit_code":0}
```

CI 可以基于此行做指标采集。

---

## 3. `linctl new` — 生成项目骨架

### 3.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 在指定目录生成一个 miniblog-v4 风格的 Go 后端服务骨架 |
| **触发** | `linctl new <project-name>` 或 `linctl new`（交互式补全） |
| **副作用** | 创建新目录 `<project-name>/` 并写入 ~40 个文件（含 `cmd/<app>/`、`cmd/gen-gorm-model/`、`pkg/db/`、`configs/<app>.yaml`、`Makefile` 等） |
| **写入策略** | 目录不存在则创建；存在则报错（除非 `--force` 覆盖） |
| **不做** | `git init` / `go mod download` / `make build`（用户在「Next steps」按需执行） |

### 3.1.1 交互式与命令式（双模式，均需支持）

| 模式 | 适用场景 | 典型用法 |
| --- | --- | --- |
| **交互式** | 本机 TTY、逐步问答（见 [07](./07-interactive-ux.md)） | `linctl new`；或 `linctl new myblog` 仅补问缺少的 `--module` |
| **命令式 / CI** | 脚本、管道、无 TTY；须一次给齐参数 | `linctl new myblog --module github.com/foo/myblog --storage gorm-postgres --yes --non-interactive` |

**约束**：非 TTY 或 `--non-interactive` 时，**不能**走向导；必须提供 `project-name`（positional）与 `--module`，否则报错并提示用法。全局 `--yes` 用于跳过向导内的最终确认。

> **与 `add` 的区别**：`linctl add` 当前仅命令式（资源名须 positional / flag）；交互式向导若覆盖 `add`，见 [07 §12](./07-interactive-ux.md)。

### 3.2 完整 usage

```
linctl new <project-name> [flags]

Aliases: new, init

Flags:
      --module string         Go module path (e.g., github.com/foo/myblog)
      --app-name string       Application name; default = derived from project-name
      --framework string      Web framework: gin (default "gin")
      --storage string        Storage layer: memory|gorm-postgres|gorm-mysql|mongo
      --features strings      Optional features: otel, healthz, user
      --template-dir string   External template directory (overrides embed)
      --output-dir string     Parent directory to create project in (default ".")
      --force                 Overwrite if target dir exists
      --dry-run               Print plan, do not write files

Inherited from root:
      --log-level / --log-format / --no-color / --non-interactive / -y, --yes / -C, --chdir
```

### 3.3 参数详解

| flag | 必填 | 默认行为 | 校验 |
| --- | --- | --- | --- |
| `<project-name>` | 是 | TTY 下可省略，进入 prompt | 仅小写字母/数字/连字符；不与已存在目录冲突 |
| `--module` | 是 | TTY 下可省略，进入 prompt | 形如 `domain.tld/owner/name`；正则校验 |
| `--app-name` | 否 | `<project-name>` 同名 | 合法 Go 标识符；不能以数字开头 |
| `--framework` | 否 | `gin` | MVP 仅 `gin`；预留 `grpc` |
| `--storage` | 否 | `memory` | 枚举校验 |
| `--features` | 否 | `[]` | 仅接受白名单 token |
| `--template-dir` | 否 | embed.FS | 必须是绝对路径或 `~` 起始；目录需存在 |
| `--output-dir` | 否 | `.` | 父目录可写 |
| `--force` | 否 | `false` | 与 `--dry-run` 互斥 |
| `--dry-run` | 否 | `false` | 与 `--force` 互斥 |

> **关于 `--features`**：feature 标记在模板中表现为 `{{if has .Features "otel"}}` 等条件分支。决策见 [00 §10 §5.5](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29)。

### 3.4 行为流（高层）

```
1) cli/new.go：解析 flag
2) TTY 检测 → 进入交互式（详见 07 §3）或非交互
3) scaffold.LoadContext(flags) → Context
4) scaffold.BuildPlan(ctx, KindProject) → Plan{Creates: [...]}
5) 打印 Plan 摘要表（见 07 §3.6）
6) 用户确认（除 --yes）
7) scaffold.NewProject(ctx) 渲染并写入文件
8) 打印「Next steps」清单
```

详细数据流参见 [01 §5「数据流：linctl new」](./01-architecture-blueprint.md#5-数据流linctl-new-命令)。

### 3.5 输出示例

```
$ linctl new myblog --module github.com/foo/myblog --storage gorm-postgres --features otel,healthz --yes
✔ project plan computed (40 files)
✔ scaffold rendered into ./myblog
📦 Next steps:
   cd myblog
   make deps
   make protoc
   make build
```

`--dry-run` 模式仅打印计划，不写入磁盘；输出包含相对路径与字节数估算。

### 3.6 退出码

| code | 含义 |
| --- | --- |
| `0` | 成功（含 `--dry-run`） |
| `2` | 参数非法（cobra 默认） |
| `10` | 目标目录已存在且未传 `--force` |
| `11` | `--module` 不合法 |
| `12` | 模板渲染失败（变量缺失等） |
| `13` | 文件写入失败（权限/磁盘） |
| `14` | 用户在确认阶段取消 |

> **退出码原则**：`0/1/2` 留给标准/常规错误；`10–19` 用于 `new`；`20–29` 用于 `add`；`30–39` 用于 `lint`/`doctor`；详见 §9。

### 3.7 与交互式 UX 的关系

- TTY 下不传 flag 时进入 [07 §3](./07-interactive-ux.md#3-lin-new-完整交互流程mockup) 的 7 步向导；
- 非 TTY 或 `--non-interactive` 时缺失必填参数报错（exit 2）；
- 已传的 flag 始终跳过对应 prompt，可与交互混合使用。

---

## 4. `linctl add` — 追加业务资源

### 4.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 在已有项目中追加业务资源（含全栈分层文件 + AST 注入） |
| **触发** | `linctl add <Resource>...` 或 `linctl add`（交互式） |
| **副作用** | 创建 13 个文件（默认全栈）+ AST 修改 4 个中央文件（参见 [03 §1](./03-resource-scaffold.md#1-资源分层总览)） |
| **执行位置** | 在项目根或其任意子目录下执行；自动向上回溯找含 `go.mod` 的目录视为项目根（详见 [§2.1](#21-工作目录解析)）；找不到则报错 |
| **不做** | `make protoc` / `go mod tidy`（用户在 Next steps 自行执行） |

### 4.2 完整 usage

```
linctl add <Resource>... [flags]

Aliases: add, generate

Args:
  <Resource>   One or more PascalCase resource names (e.g. Post Comment Tag)

Flags:
      --app string            Override app name (default = inferred from cmd/*)
      --with strings          Optional layers: conversion,validation,proto,errno
                              (default "conversion,validation,proto,errno")
      --without strings       Inverse of --with; mutually exclusive with --with
      --ops strings           CRUD verbs to generate: create,update,delete,get,list
                              (default "create,update,delete,get,list")
      --version string        API version segment (default "v1")
      --plural string         Override plural form for store (e.g. Octopuses)
      --template-dir string   External template directory
      --dry-run               Print plan, do not write files
      --no-inject             Skip AST injection (only create new files)
      --skip-imports          Skip auto-add of import statements
      --force                 Overwrite existing resource files (backed up to user cache dir, not the project tree)

Inherited from root:
      --log-level / --log-format / --no-color / --non-interactive / -y, --yes / -C, --chdir
```

### 4.3 参数详解

| flag | 默认行为 | 校验 |
| --- | --- | --- |
| `<Resource>...` | 至少 1 个；可批量 `linctl add Post Comment` | PascalCase；首字符大写；不与既有资源冲突 |
| `--app` | 自动推断 `cmd/<app>/` 目录 | 多个 cmd 子目录时必填；详见 §4.4 |
| `--with` | `conversion,validation,proto,errno` | 白名单子集；详见 [03 §5](./03-resource-scaffold.md#5---with----without-flag-控制矩阵) |
| `--without` | `[]` | 与 `--with` 互斥（同传则 exit 23） |
| `--ops` | `create,update,delete,get,list` | 子集白名单；详见 [03 §6](./03-resource-scaffold.md#6-crud-动词裁剪) |
| `--version` | `v1` | 形如 `v\d+` |
| `--plural` | 英语规则推断（`jinzhu/inflection`） | 仅当推断错误时使用（如 `Octopus → Octopuses`） |
| `--no-inject` | `false` | 调试用；跳过 AST 注入但仍创建文件 |
| `--skip-imports` | `false` | 仅在用户已手动管理 import 时使用 |
| `--force` | `false` | 覆盖既有资源文件（备份到用户缓存目录）；详见 §4.6 文件冲突表 |

### 4.4 上下文推断

`linctl add` 不依赖配置文件，运行时从项目现状推断元信息（详见 [01 §6 数据流](./01-architecture-blueprint.md#6-数据流linctl-add-命令)）：

| 元信息 | 推断来源 | 缺失时 |
| --- | --- | --- |
| `Module` | `./go.mod` 第一行 `module ...` | 报错（exit 21） |
| `AppName` | `./cmd/<app>/` 唯一子目录 | 多个时要求 `--app`（exit 22） |
| `Storage` | `./internal/<app>/store/` 中的 import | 默认 `memory`；可被 flag 覆盖 |
| `Framework` | `./internal/<app>/handler/` 中的 import | 默认 `gin` |
| `Resource` | 命令参数 | 必须传入 |

#### 4.4.1 单 app 项目（默认）

```
myblog/
├── cmd/
│   └── myblog/         ← 唯一子目录，自动推断 AppName=myblog
└── internal/
    └── myblog/
        ├── handler/
        ├── biz/
        └── store/

$ linctl add Post              # ✅ 无需 --app
```

#### 4.4.2 monorepo（多 cmd 子目录）

```
my-monorepo/
├── cmd/
│   ├── api/             ← 多个子目录
│   ├── worker/
│   └── cron/
└── internal/
    ├── api/
    ├── worker/
    └── cron/

$ linctl add Post              # ❌ exit 22 "multiple apps detected, use --app"
$ linctl add Post --app=api    # ✅ 仅向 internal/api/ 注入
$ linctl add Post --app=api,worker   # ❌ MVP 不支持多 app 同时注入
```

#### 4.4.3 monorepo 行为约束

| 维度 | 行为 |
| --- | --- |
| `--app` 取值 | 必须是 `cmd/<x>/` 的精确子目录名 |
| 多 app 同时注入 | **MVP 不支持**；需多次 `linctl add Post --app=X` |
| 共享 `internal/pkg/errno/` | 多次执行时**幂等**（`register.go` 的 `RegisterErrors(PostErrors()...)` 仅注入一次） |
| 共享 `pkg/api/<app>/v1/` | 每个 app 独立目录，互不干扰 |
| 资源命名冲突 | `linctl add Post --app=api` 与 `linctl add Post --app=worker` 各自注入，互相独立 |

#### 4.4.4 当 `cmd/` 不存在或非标准

| 场景 | 行为 |
| --- | --- |
| 无 `cmd/` 目录 | exit 20（不识别为项目根） |
| `cmd/` 为空 | exit 22（无 app 候选） |
| `cmd/<x>/main.go` 不存在 | 视为非 app 子目录，跳过；只统计含 `main.go` 的子目录 |
| 用户结构不是 `cmd/<app>/` 而是 `cmd/main.go` | exit 20，提示用户「lin v2 仅支持 `cmd/<app>/main.go` 布局」 |

### 4.5 行为流

```
1) cli/add.go：解析 args + flags
2) scaffold.LoadContext(rootDir, flags) → 推断 Module/AppName/Storage
3) 校验项目结构（有 go.mod / cmd/<app>/ / internal/<app>/）
4) for each Resource:
   a) scaffold.BuildPlan(ctx, KindResource, name) → Plan
   b) 打印 Plan 摘要（Creates + Injects）
5) 用户确认（除 --yes）
6) for each Resource：
   a) render：scaffold/render.go 写入 12 个新文件
   b) inject：ast/injector.go 编排 4 个 mutator
      - guard 检查幂等
      - 失败时回滚（从用户缓存目录还原）
7) 打印「Next steps」（make protoc / go mod tidy / go build）
```

详细 AST 注入语义见 [05 §3 注入流程](./05-registration-strategy.md)。

### 4.6 幂等性与文件冲突策略

#### 4.6.1 幂等检查（参见 [05 §2.2](./05-registration-strategy.md#22-幂等性idempotency)）

| 检查点 | 默认行为 |
| --- | --- |
| AST 已存在的注入点 | `⊝ skip`（不重复添加） |
| AST 部分注入失败 | 回滚已成功的注入；exit 25 |
| 锚点注释完全缺失 | `error`；提示用 `linctl lint --fix` 恢复；exit 24 |

#### 4.6.2 文件冲突策略表

`linctl add` 创建文件时按以下决策树处理冲突：

```
              Resource 文件已存在?
                    │
         ┌──────────┴──────────┐
         │ No                   │ Yes
         ▼                      ▼
      创建写入             与模板渲染产物 byte-equal?
                                │
                     ┌──────────┴──────────┐
                     │ Yes                  │ No (用户已修改)
                     ▼                      ▼
                ⊝ silent skip          ┌────┴─────────┐
                                       │              │
                                  --force?         默认?
                                       │              │
                                       ▼              ▼
                  用户缓存目录后覆盖    ⚠ warn skip
                                                  （文件保留，
                                                   提示用户检查）
```

| 场景 | 默认行为 | `--force` 行为 |
| --- | --- | --- |
| 文件不存在 | 创建 | 同 |
| 文件存在且与模板等价 | `⊝ silent skip` | 同 |
| 文件存在但内容已修改 | `⚠ warn skip`，列出文件 | 备份到用户缓存目录 后覆盖 |
| 中央文件锚点存在但用户在锚点外手写了 | 仍在锚点内注入；`⚠ warn` | 同 |
| 中央文件锚点完全缺失 | `error` exit 24 | 同（`--force` 不绕过锚点） |

> **设计原则**：`add` 默认**永不覆盖用户改动**。`--force` 是显式逃生舱，**所有覆盖均备份到用户缓存目录**（详见 [05 §5.2](./05-registration-strategy.md#52-备份生命周期)）。

### 4.7 输出示例

```
$ linctl add Post Comment
✔ context loaded   module=github.com/foo/myblog appName=myblog storage=gorm-postgres
✔ resource: Post
   + 13 files created
   ✏ 4 files updated via AST
✔ resource: Comment
   + 13 files created
   ✏ 4 files updated via AST
📦 Next steps:
   make protoc
   go mod tidy
   go build ./...
```

### 4.8 退出码（add）

| code | 含义 |
| --- | --- |
| `0` | 成功（含 `--dry-run` / 全部 skip） |
| `20` | 不在项目根目录（找不到 `go.mod`） |
| `21` | `go.mod` 解析失败 |
| `22` | 多 app 但未传 `--app` |
| `23` | 资源名不合法（非 PascalCase 等） |
| `24` | 锚点注释缺失，无法注入 |
| `25` | AST 注入失败且回滚成功 |
| `26` | AST 注入失败且回滚失败（人工介入） |
| `27` | 用户在确认阶段取消 |
| `28` | flag 互斥冲突（如 `--with` 与 `--without` 同传） |

---

## 5. `linctl lint` — 校验项目结构与 AST 完整性

### 5.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 静态检查项目骨架是否符合规范，并校验注册一致性 |
| **触发** | 在项目根目录执行 `linctl lint` |
| **副作用** | 默认只读；`--fix` 预留接口，当前为 no-op（注册补齐建议直接运行 `linctl add`） |
| **不做** | `go vet` / `golangci-lint` 这类语义级检查（请用专用工具） |

### 5.2 完整 usage

```
linctl lint [flags]

Flags:
      --fix                  Auto-fix issues (reserved; currently no-op)
      --report-format string text|json (default "text")
      --rules strings        Subset of rule IDs to enable (default: all)
      --skip strings         Rule IDs to skip
```

### 5.3 检查项

注：linctl v2 已移除"锚点注释"这一概念。所有 AST 注入完全基于 Go 语法结构（接口名、receiver 名、函数名）定位插入点，因此不再需要 `anchor/*` 类规则。

| 类别 | 检查 ID | 说明 | `--fix` 行为 |
| --- | --- | --- | --- |
| 目录结构 | `dir/cmd-app` | `cmd/<app>/main.go` 存在 | ❌ 仅报错（创建空 main.go 风险大） |
| 目录结构 | `dir/internal-app` | `internal/<app>/{handler,biz,store,model}` 存在 | ❌ 仅报错 |
| 注册一致性 | `register/biz-impl` | 每个 `biz/v1/<resource>/` 都在 `biz.go` 出现（warning） | ⚠️ MVP 仅报告，提示运行 `linctl add` |
| 注册一致性 | `register/store-impl` | 每个 `store/<resource>.go` 都在 `store.go` 出现（warning） | ⚠️ 同上 |
| 占位文件 | `lin/post-protoc-placeholder` | 提示 `_lin.go` 占位文件需 `make protoc` 后清理（info） | ❌ 不报错 |
| 路径安全 | `safety/path-traversal` | 资源路径未跳出项目根（`fsx.SafeJoin` 校验） | ❌ 报错 |

#### 5.3.1 关于 `--fix`

当前 `--fix` 仅作为占位 flag 保留，未真正执行修复操作。如发现 `register/*` 报告资源未注册，请直接运行：

```
linctl add <Resource>
```

由于 AST 注入完全是幂等的（已存在的方法/语句会被跳过），重新运行 `linctl add` 即可恢复一致性。

### 5.4 输出示例

```
$ linctl lint
✔ dir/cmd-app          cmd/myblog/main.go exists
✔ dir/internal-app     internal/myblog/{handler,biz,store,model} all exist
✔ register/biz-impl    biz.go: registration consistent
✔ register/store-impl  store.go: registration consistent
ℹ lin/post-protoc-placeholder found 1 _lin.go placeholder(s): post_lin.go
✔ safety/path-traversal no path traversal detected

5 ok / 0 warning / 0 error
```

`--report-format json` 输出结构化结果，便于 CI 集成：

```json
{
  "items": [
    {"category": "dir", "name": "dir/cmd-app", "status": "ok", "message": "cmd/myblog/main.go exists"}
  ],
  "summary": {"ok": 5, "errors": 0, "warnings": 0}
}
```

### 5.5 退出码（lint）

| code | 含义 |
| --- | --- |
| `0` | 无问题 |
| `30` | 至少一项 error |
| `31` | `--fix` 部分失败（保留语义，当前 `--fix` 为 no-op） |
| `32` | 项目根识别失败 |

---

## 6. `linctl doctor` — 校验本地工具链与运行环境

### 6.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 检测开发环境中 lin 所依赖的工具是否可用、版本是否兼容 |
| **触发** | 任意目录执行 `linctl doctor` |
| **副作用** | 只读 |
| **网络** | 默认不访问网络；`go env` 子调用可能间接触发 module proxy |

### 6.2 完整 usage

```
linctl doctor [flags]

Flags:
      --report-format string text|json (default "text")
      --strict               Treat warnings as errors (exit 30)
      --offline              Skip network-dependent checks (CI/airgap friendly)
      --check strings        Subset of check IDs to run (default: all)
      --skip strings         Check IDs to skip
```

#### 6.2.1 `--offline` 行为

| 检查类别 | `--offline` 时 |
| --- | --- |
| `proxy.golang.org` 可达性 | ⊝ skipped |
| `go env GOPROXY` 校验（仅 sanity check） | 仍执行（无网络调用） |
| 工具链版本检测（`go`/`protoc`/`wire`） | 仍执行 |
| terminal sanity（TTY/colors） | 仍执行 |

> CI / 内网 / airgap 环境推荐默认带 `--offline`。

### 6.3 检查项

| 类别 | 项 | 期望 | 等级 |
| --- | --- | --- | --- |
| Go 工具链 | `go` | ≥ 1.22 | error |
| Go 工具链 | `GOFLAGS` | 不含 `-mod=vendor`（与生成场景冲突） | warning |
| 代码生成 | `protoc` | ≥ 3.20 | warning（仅 `add --with proto` 需要） |
| 代码生成 | `protoc-gen-go` | 在 `$PATH` | warning |
| 代码生成 | `wire` | 在 `$PATH` | warning |
| 系统 | `git` | ≥ 2.30 | warning |
| 系统 | `terminal` | TTY？尺寸？颜色？ | info |
| 网络 | `proxy.golang.org` 可达性 | 仅在 `--strict` 检查 | info |

### 6.4 输出示例

```
$ linctl doctor
✔ go            1.22.3
✔ git           2.42.0
✔ protoc        3.21.12
⚠ protoc-gen-go missing, install via: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
ℹ terminal      tty=true colors=true width=120

3 ok / 1 warning / 0 error
```

### 6.5 退出码（doctor）

| code | 含义 |
| --- | --- |
| `0` | 所有 error 项通过 |
| `35` | 至少一项 error |
| `36` | `--strict` 下有 warning |

---

## 7. `linctl version` — 版本信息

### 7.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 打印 lin 二进制的版本、commit、构建时间 |
| **触发** | `linctl version` 或 `linctl -v` |
| **数据来源** | `internal/version/version.go` 由 ldflags 注入 |

### 7.2 完整 usage

```
linctl version [flags]

Flags:
      --short                 Print only semver (e.g. "0.1.0-alpha")
      --format string         text|json|yaml (default "text")
```

### 7.3 输出示例

```
$ linctl version
linctl version 0.1.0-alpha
  commit:   3f2e0b1
  built:    2026-04-29T11:22:33Z
  go:       go1.22.3
  os/arch:  darwin/arm64
```

```
$ linctl version --short
0.1.0-alpha
```

```
$ linctl version --format json
{"version":"0.1.0-alpha","commit":"3f2e0b1","built":"2026-04-29T11:22:33Z","go":"go1.22.3","os":"darwin","arch":"arm64"}
```

### 7.4 退出码

| code | 含义 |
| --- | --- |
| `0` | 总是 |

---

## 8. `linctl completion` — Shell 补全

### 8.1 概述

| 项目 | 说明 |
| --- | --- |
| **作用** | 生成 bash/zsh/fish/powershell 的命令补全脚本 |
| **触发** | `linctl completion <shell>`（重定向到 shell 配置文件） |
| **来源** | 由 `cobra` 自动生成；lin 仅注册子命令 |

### 8.2 完整 usage

```
linctl completion <shell>

Args:
  <shell>   bash | zsh | fish | powershell

Flags:
      --no-descriptions       Disable completion descriptions (smaller scripts)
```

### 8.3 用法示例

```bash
# bash (per-user)
$ linctl completion bash > ~/.local/share/bash-completion/completions/linctl

# zsh
$ linctl completion zsh > "${fpath[1]}/_linctl"

# fish
$ linctl completion fish > ~/.config/fish/completions/linctl.fish

# PowerShell
PS> linctl completion powershell | Out-String | Invoke-Expression
```

### 8.4 退出码

| code | 含义 |
| --- | --- |
| `0` | 成功生成脚本 |
| `2` | 不支持的 shell（cobra 默认） |

---

## 9. 退出码总表

### 9.1 退出码分段

> 设计原则：`0` 成功；`1` 通用失败；`2` cobra 默认参数错误；`10+` 按命令分段。

| 范围 | 用途 |
| --- | --- |
| `0` | 成功 |
| `1` | 通用未知错误 |
| `2` | 参数错误（cobra） |
| `10–19` | `linctl new` 错误（详见 §3.6） |
| `20–29` | `linctl add` 错误（详见 §4.8） |
| `30–34` | `linctl lint` 错误（详见 §5.5） |
| `35–39` | `linctl doctor` 错误（详见 §6.5） |
| `40–49` | 预留：模板加载/渲染层（被 §3/§4 透传） |
| `50–59` | 预留：AST 注入层（被 `add` 透传） |
| `130` | 用户 Ctrl+C（标准 SIGINT 约定） |

> **CI 友好**：`--report-format json` 输出始终独立于退出码；脚本可同时基于 exit code 与 JSON 字段做决策。

### 9.2 错误类型 → 退出码映射（`internal/pkg/errs/codes.go`）

CLI 层根据 `errs.Code(err)` 转换为退出码。错误类型集中定义如下：

```go
package errs

type Code int

// 通用
const (
    CodeOK              Code = 0
    CodeUnknown         Code = 1
    CodeInvalidArg      Code = 2
)

// new (10-19)
const (
    CodeTargetExists    Code = 10  // 目标目录已存在且未传 --force
    CodeBadModule       Code = 11  // module path 不合法
    CodeRenderFailed    Code = 12  // 模板渲染失败（透传 40-49）
    CodeWriteFailed     Code = 13  // 文件写入失败（权限/磁盘）
    CodeUserCancelled   Code = 14  // 用户取消
)

// add (20-29)
const (
    CodeNotProjectRoot  Code = 20  // 找不到 go.mod
    CodeBadGoMod        Code = 21  // go.mod 解析失败
    CodeMultiAppNoFlag  Code = 22  // 多 app 但未传 --app
    CodeBadResourceName Code = 23  // 资源名不合法（非 PascalCase 等）
    CodeSymbolMissing   Code = 24  // AST 注入目标符号缺失（接口/函数未定义）
    CodeInjectFailed    Code = 25  // AST 注入失败但回滚成功
    CodeRollbackFailed  Code = 26  // AST 注入失败且回滚失败（人工介入）
    CodeAddCancelled    Code = 27  // 用户取消
    CodeFlagConflict    Code = 28  // flag 互斥冲突（如 --with 与 --without 同传）
)

// lint (30-34)
const (
    CodeLintIssues      Code = 30  // 至少一项 error
    CodeFixPartial      Code = 31  // --fix 部分失败
    CodeLintNoProject   Code = 32  // 项目根识别失败
)

// doctor (35-39)
const (
    CodeDoctorErrors    Code = 35  // 至少一项 error
    CodeDoctorWarnStrict Code = 36 // --strict 下有 warning
)

// 模板层（40-49，透传给 new/add）
const (
    CodeTplNotFound     Code = 40
    CodeTplParseError   Code = 41
    CodeTplExecError    Code = 42
    CodeTplPathTraversal Code = 43 // 安全：渲染路径跳出 RootDir
)

// AST 层（50-59，透传给 add）
const (
    CodeASTParseError    Code = 50
    CodeASTSymbolMissing Code = 51  // 内部：AST 找不到目标符号
    CodeASTApplyError    Code = 52
    CodeASTBackupFailed  Code = 53
)

// 标准约定
const (
    CodeSIGINT Code = 130
)

// 错误类型支持包装
type Error struct {
    Code    Code
    Message string
    Hint    string  // 用户可读的修复建议
    Cause   error   // 链式根因
}

func (e *Error) Error() string  { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

// CLI 层转换：内部码 → 当前命令的用户可见退出码。
// cmd 是当前执行的命令名（"new" / "add" / ...），用于将共享内部码映射到该命令的范围。
func CodeOf(cmd string, err error) Code {
    var e *Error
    if errors.As(err, &e) {
        return mapToUserCode(cmd, e.Code)
    }
    return CodeUnknown
}

// mapToUserCode 将内部码（如模板/AST 层 40+ / 50+）按当前 cmd 映射到用户可见范围。
//
// 设计原则：
//   - 模板层错误（40-49）由触发命令吸收：new → 12；add → 25
//   - AST 层错误（50-59）仅可能由 add 触发，按错误细分映射到 24/25/26
//   - 命令自身的码（10-39）原样透传
func mapToUserCode(cmd string, internal Code) Code {
    // 命令自身范围内的码原样返回
    switch cmd {
    case "new":
        if internal >= 10 && internal <= 19 { return internal }
    case "add":
        if internal >= 20 && internal <= 29 { return internal }
    case "lint":
        if internal >= 30 && internal <= 34 { return internal }
    case "doctor":
        if internal >= 35 && internal <= 39 { return internal }
    }

    // 模板层 (40-49) → 命令对应码
    if internal >= CodeTplNotFound && internal <= CodeTplPathTraversal {
        switch cmd {
        case "new":  return CodeRenderFailed   // 12
        case "add":  return CodeInjectFailed   // 25
        case "lint": return CodeFixPartial     // 31
        default:     return CodeUnknown
        }
    }

    // AST 层 (50-59) → 仅 add 关心
    switch internal {
    case CodeASTSymbolMissing:                    // 51
        return CodeSymbolMissing                  // 24
    case CodeASTBackupFailed:                     // 53
        return CodeRollbackFailed                 // 26
    case CodeASTParseError, CodeASTApplyError:    // 50, 52
        return CodeInjectFailed                   // 25
    }

    // 标准约定 / 通用
    return internal
}
```

> **关键约束**：用户**永远只看到** `§9.1` 的退出码范围（0/1/2/10–39/130）；40+/50+ 的内部码仅出现在 debug 日志，绝不作为退出码。

> **实现位置**：`internal/pkg/errs/codes.go`（参见 [01 §3 L4 基础层](./01-architecture-blueprint.md#l4基础层)）。

#### 9.2.1 调用端约定

CLI 层各命令的 `RunE` 函数返回 error 时，root 命令的 `OnPostRun` 拦截：

```go
// internal/cli/root.go
func main() {
    ctx := context.Background()
    rootCmd := newRootCmd()
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        cmd, _, _ := rootCmd.Find(os.Args[1:])
        os.Exit(int(errs.CodeOf(cmd.Name(), err)))
    }
    os.Exit(0)
}
```

---

## 10. 命令矩阵速查

### 10.1 命令 vs 全局 flag

| flag | new | add | lint | doctor | version | completion |
| --- | --- | --- | --- | --- | --- | --- |
| `--log-level` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `--log-format` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `-C, --chdir` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `--no-color` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `--non-interactive` | ✅ | ✅ | n/a | n/a | n/a | n/a |
| `-y, --yes` | ✅ | ✅ | n/a | n/a | n/a | n/a |
| `--dry-run` | ✅ | ✅ | n/a | n/a | n/a | n/a |
| `--template-dir` | ✅ | ✅ | n/a | n/a | n/a | n/a |

### 10.2 命令 vs 写入策略

| 命令 | 创建文件 | 修改既有文件 | AST 注入 | 备份机制 |
| --- | --- | --- | --- | --- |
| `new` | ✅ | ❌（除非 `--force`） | ❌ | n/a（新目录无需备份） |
| `add` | ✅ | ✅（中央文件） | ✅ | 用户缓存目录（详见 [05 §5.2](./05-registration-strategy.md#52-备份生命周期)） |
| `add --force` | ✅ | ✅（强制覆盖资源文件 + 中央文件） | ✅ | 同上 |
| `lint --fix` | ❌ | ✅（锚点恢复 / 注册补全） | ⚠️ 调用 add 内部 mutator | 同上 |
| `doctor` | ❌ | ❌ | ❌ | n/a |
| `version` / `completion` | ❌ | ❌ | ❌ | n/a |

> **统一**：所有写操作的备份都集中在**用户级缓存目录**（`<UserCacheDir>/linctl/backups/<project>/<ts>/`），**不**在用户项目内留下任何 `.bak` 后缀或 `.linctl/` 子目录。理由：集中备份易于审计、不污染项目目录、支持未来 `--restore-backup`。

---

## 11. 与决策记录、其他文档的关系

| 决策点（[00 §10](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29)） | 在本文档中的体现 |
| --- | --- |
| §5.1 工具边界（B：极简 + lint/doctor） | §1 命令一览仅 6 个 |
| §5.2 资源注册策略（B：AST 注入） | §4.5 行为流；详见 [05](./05-registration-strategy.md) |
| §5.3 资源完整度（A：全栈） | §4.2 `--with` 默认 `conversion,validation,proto,errno`；13 文件（详见 [03 §5](./03-resource-scaffold.md#5---with----without-flag-控制矩阵)） |
| §5.4 模板可定制性（B：外部目录覆盖） | §3.2 / §4.2 `--template-dir`；详见 [04](./04-template-system.md) |
| §5.5 Feature 系统（A：完全删除） | §3.2 `--features` 仅触发模板条件分支；无 yaml |
| §5.6 配置文件（A：完全删除） | §4.4 上下文推断；无 `lin.yaml` |

| 关联文档 | 关系 |
| --- | --- |
| [03-resource-scaffold.md](./03-resource-scaffold.md) | `add` 创建的 13 个文件清单与命名约定 |
| [04-template-system.md](./04-template-system.md) | `--template-dir` 与 embed.FS 的查找优先级 |
| [05-registration-strategy.md](./05-registration-strategy.md) | `add` 的 AST 注入语义、锚点、幂等、回滚 |
| [06-migration-plan.md](./06-migration-plan.md) | 命令落地的 5 阶段排期与 DoD |
| [07-interactive-ux.md](./07-interactive-ux.md) | `new` / `add` 在 TTY 下的向导流程 |

---

## 12. 设计取舍（FAQ）

### Q1：为什么不提供 `linctl remove <Resource>` 删除资源？

A：MVP 范围内不做，原因：
- 删除业务代码涉及大量人工判断（数据迁移、引用清理），工具难以替用户兜底；
- 与「[00](./00-refactor-rationale.md) 一次性约定式脚手架」定位不符；
- 用户可用 `git rm` + 手工清理替代。未来如需要，作为 Phase 6+ 重新设计。

### Q2：为什么 `add` 不自动跑 `make protoc / go mod tidy`？

A：副作用最小化原则。
- 跑工具会触发网络/磁盘操作，扩大失败面；
- 不同项目可能有不同的 Makefile 目标；
- 工具退出舞台、把项目交还给开发者（[00 §3.1](./00-refactor-rationale.md)）。

### Q3：`linctl lint` 与 `golangci-lint` 是什么关系？

A：完全不重叠。
- `linctl lint` 仅校验 lin 自己关心的规范（目录布局、AST 锚点、注册一致性）；
- `golangci-lint` 校验 Go 语法/语义；
- 两者建议都跑，互补。

### Q4：为什么不引入 `linctl update-templates` 或 `linctl upgrade`？

A：见 [00 §3.4「不是什么」](./00-refactor-rationale.md#34-不是什么non-goals明确边界)。
- 模板升级时已有项目不会被自动改动；
- 用户用 `git diff` + 手工合入；
- 真有强需求时作为 v2 发布后再评估（[06 §10](./06-migration-plan.md)）。

---

## 13. 实现 checklist（落地用）

> 本节供 [06-migration-plan.md](./06-migration-plan.md) Phase 1–4 实施时核对。

- [ ] `internal/cli/root.go`：注册全局 flag、PersistentPreRun 初始化 logger
- [ ] `internal/cli/new.go`：参数 + interactive bridge → `scaffold.NewProject`
- [ ] `internal/cli/add.go`：参数 + interactive bridge → `scaffold.AddResource`
- [ ] `internal/cli/lint.go`：调用 `check.Lint`，支持 `--fix` `--report-format`
- [ ] `internal/cli/doctor.go`：调用 `check.Doctor`
- [ ] `internal/cli/version.go`：读 `internal/version/version.go`
- [ ] `internal/cli/completion.go`：cobra 自带的 `GenCompletion`
- [ ] 退出码集中在 `internal/pkg/errs/codes.go`，CLI 层根据 error 类型映射
- [ ] `--non-interactive` 在 `cli/root.go` 的 `PersistentPreRun` 中按 TTY 自动设置
- [ ] 所有命令都有至少 1 条 `tests/e2e/<cmd>_test.sh`

---

_Last reviewed: 2026-04-29_
