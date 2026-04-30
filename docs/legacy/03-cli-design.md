# 03. linctl CLI 命令体系设计

## 3.1 命令树总览

```
linctl
├── new <DIR>                  # 生成全新项目
├── add                        # 增量添加资源
│   ├── api <NAME...>            # 给 WebServer 加 REST 资源
│   ├── worker <NAME...>         # 给 Worker 加 cron/mq handler
│   ├── cli <NAME...>            # 给 CLI 加子命令
│   ├── webserver <NAME>         # 加新的 WebServer 组件
│   ├── feature <FEATURE>        # 给某 Component 启用 Feature
│   └── component <KIND> <NAME>  # 通用：加任意 Component
│
├── plan                       # 计算待执行变更（不写文件）
├── apply                      # 执行 plan 中的变更
├── lint                       # 校验 linctl.yaml 与项目结构
├── doctor                     # 检测环境（go/protoc/buf/git 等）
├── import                     # 从已有项目反推 linctl.yaml
├── upgrade                    # 跨版本升级 linctl.yaml schema
│
├── completion <SHELL>         # 输出 shell 补全脚本
├── plugin                     # 插件管理
│   ├── list
│   ├── install <NAME>
│   └── remove <NAME>
│
├── version                    # 版本信息
├── help                       # 帮助
└── options                    # 列出全局 flag
```

## 3.2 全局 Flag（所有命令通用）

### 3.2.1 通用控制

| Flag | 默认 | 含义 |
| --- | --- | --- |
| `--config <file>` | `linctl.yaml` | 项目配置文件路径 |
| `--root-dir <dir>` | `.` | 项目根目录 |
| `--env <name>` | `""` | 启用环境覆盖：加载 `linctl.<env>.yaml` 与 `<env>` 子目录 overlay，详见 [04-config-schema.md §4.10](./04-config-schema.md#410-多文件配置) |
| `--log-level <level>` | `info` | trace/debug/info/warn/error |
| `--log-format <fmt>` | `text` | text/json |
| `--output / -o <fmt>` | `text` | **机器可解析输出格式**：`text`(人类可读) / `json` / `yaml`。适用于 `plan`/`version`/`lint`/`doctor` 等需要结构化结果的命令；其他命令忽略此 flag。 |
| `--no-color` | false | 禁用彩色输出 |
| `--no-emoji` | false | 禁用 emoji |
| `--dry-run` | false | 不写文件，仅打印将要发生的变更（详见 [§3.2.6 dry-run 行为矩阵](#326-dry-run-行为矩阵)） |
| `--yes / -y` | false | 自动同意所有文件冲突确认（**不跳过 hook 确认**，详见 [15-security-model.md](./15-security-model.md)） |
| `--verbose / -v` | false | 详细输出，等价于 `--log-level=debug`（与 `--debug` 正交，详见 [§3.2.2](#322-诊断与可观测详见-14-observabilitymd)） |
| `--help / -h` | - | 帮助 |

> **关于已删除的 `--out` 短名兼容性说明**（SSOT 锁定）：
>
> 旧版本（pre-v1.0）部分子命令使用过短名 flag（如 plan、import 等的 `--out`）。从 v1.0 起统一规则：
>
> - **plan / version**：原本表示「输出格式」的旧 flag 已删除，统一改为全局 `--output / -o`（取值 `text`/`json`/`yaml`）。
> - **import**：原本表示「输出文件路径」的旧 flag 已重命名为 `--output-file`（与全局 `--output` 是不同的 flag——前者是文件路径，后者是格式）。
>
> 全文档不再出现该已删除的短名 flag；如在历史脚本中遇到，请按上述规则迁移。

### 3.2.2 诊断与可观测（详见 [14-observability.md](./14-observability.md)）

| Flag | 默认 | 含义 |
| --- | --- | --- |
| `--debug <modules>` | `""` | **模块级 trace 开关**（正交于 `--log-level`）：逗号分隔的模块名（如 `template,ast,fs`），用 `*` 全部启用 |
| `--debug-out <fmt>` | `text` | trace 输出格式：`text` / `json` / `tree` |
| `--profile <kind>` | `""` | 启用 pprof；可选：`cpu` / `mem` / `goroutine` / `block` / `mutex`（多个用逗号） |
| `--profile-out <path>` | `_output/profile/{type}.pb.gz` | profile 输出路径，`{type}` 占位符 |

#### `--debug` / `--log-level` / `--verbose` 三者关系（正交矩阵）

| 维度 | `--log-level` | `--verbose / -v` | `--debug <modules>` |
| --- | --- | --- | --- |
| **作用** | 全局日志阈值 | `--log-level=debug` 的速记 | **模块级 trace**（细粒度调试事件） |
| **粒度** | 整个进程统一 | 整个进程统一 | 按模块（template/ast/fs/...）独立开关 |
| **取值** | trace/debug/info/warn/error | bool | 模块名 CSV 或 `*` |
| **是否互斥** | — | 与 `--log-level` 互斥（同时给出时 `--log-level` 优先） | **正交**：与 `--log-level` 不互斥 |

**关键规则**：

1. `--debug` 与 `--log-level`/`--verbose` **正交**——前者控制「哪些模块发出 trace 事件」，后者控制「日志阈值」。
2. **隐含提升**：当 `--debug <modules>` 非空且 `--log-level > debug` 时，linctl 自动将 `--log-level` 提升到 `debug`（否则 trace 事件会被阈值过滤掉）；提升时在 stderr 输出一行 `[INFO] --debug specified, raising --log-level to debug`。
3. `--verbose` 与 `--log-level=debug` 同义；显式给定 `--log-level` 时 `--verbose` 失效。
4. **不要把 `--debug` 当作 trace 开关**：trace 事件本身有性能开销，仅在排错时使用；生产 CI 默认不开启。

### 3.2.3 安全与 Hook（详见 [15-security-model.md §15.2.1](./15-security-model.md#1521-hook-执行策略-hook-execution-policy)）

| Flag | 默认 | 含义 |
| --- | --- | --- |
| `--hook-policy <p>` | `confirm`（本地）；CI 强制 `restricted` | Hook 执行策略：`restricted`（仅 allowlist）/ `confirm`（本地默认；非白名单逐条确认）/ `unrestricted`（⚠️ 任意命令；CI 拒绝并 `os.Exit(7)`） |
| `--hook-strategy <s>` | `confirm` | **执行流程**控制（不同于 policy；正交于安全策略）：`confirm`（询问每个 hook）/ `auto`（自动跑允许的）/ `skip`（跳过全部 hook） |
| `--lock-timeout <dur>` | `30s` | 项目锁等待超时（详见 [06 §6.13](./06-codegen-pipeline.md#613-并发与一致性-flock)） |
| `--strict` | false | 严格模式：PairBuilder 同文件覆盖时直接 fail（详见 [06 §6.4](./06-codegen-pipeline.md#64-pairbuilder)） |
| `--no-backup` | false | apply 前不创建 `.linctl/backups/<ts>/`（不推荐） |

### 3.2.4 国际化（详见 [13-coding-standards.md §13.9](./13-coding-standards.md#139-i18n-国际化策略)）

| Flag | 默认 | 含义 |
| --- | --- | --- |
| `--lang <code>` | 自动 | CLI 输出语言：`en` / `zh-CN`（自动检测 `LANG` env） |

### 3.2.5 环境变量映射（优先级低于 flag）

| 环境变量 | 等价 flag | 备注 |
| --- | --- | --- |
| `LINCTL_CONFIG` | `--config` | |
| `LINCTL_LOG_LEVEL` | `--log-level` | |
| `LINCTL_NO_COLOR` | `--no-color` | 也响应通用 `NO_COLOR` |
| `LINCTL_LANG` | `--lang` | 也响应通用 `LANG` |
| `LINCTL_TELEMETRY` | - | `on` 显式开启；其他值（含未设）= 关闭 |
| `LINCTL_DEBUG` | `--debug` | |
| `CI` | - | 为 `true` 时禁止 `--hook-policy=unrestricted`，违例时 `os.Exit(7)`（详见 15-security-model §15.2.1 与 META §5.6） |

### 3.2.6 dry-run 行为矩阵

`--dry-run` 是全局 flag，但不同子命令的语义有细微差异（与独立的 `plan` 子命令也存在交叠）。下表给出权威定义：

| 命令组合 | 等价于 | 是否写文件 | 是否计算 hash/diff | 是否运行 hook | 输出形态 |
| --- | --- | --- | --- | --- | --- |
| `linctl plan` | — | ❌ | ✅ 完整 | ❌ | 完整变更计划（结构化，可 `-o json`） |
| `linctl apply --dry-run` | **等价于 `linctl plan`**（实现共享同一 Planner） | ❌ | ✅ 完整 | ❌ | 完整变更计划 |
| `linctl new <DIR> --dry-run` | 仅打印预览 | ❌ | ❌（项目不存在，无 hash 比对） | ❌ | 文件树 + 大小预览（不渲染全部内容） |
| `linctl add <kind> --dry-run` | **等价于针对该子树的局部 plan** | ❌ | ✅（仅涉及 Pair 的范围） | ❌ | 局部变更计划（聚焦该 component / feature） |
| `linctl apply` | — | ✅ | ✅ | ✅ | 执行报告 |

**关键原则**：

1. **唯一 Planner**：`plan` 子命令、`apply --dry-run`、`add --dry-run` 都复用同一个 `Planner.Plan()` 实现；区别只在于「输入 Plan 范围」和「是否后续接 Apply」。
2. **`new --dry-run` 是特例**：因为目标项目尚不存在，没有现状可对比，故仅打印将创建的文件清单（不计算 hash、不渲染内容）；如果需要看完整渲染结果，请配合 `--detailed`。
3. **互斥说明**：`linctl plan --dry-run` 是冗余但允许的（等价于 `linctl plan`）；`linctl apply --dry-run --plan plan.json` 会校验 digest 后打印 plan，**不执行**。
4. **环境变量**：`LINCTL_DRY_RUN=1` 全局生效，但 CI 中**不应依赖**——CI 应显式使用 `linctl plan` 子命令。

## 3.3 详细命令规范

### 3.3.1 `linctl new <DIR>`

**用途**：从零生成一个完整 Go 项目骨架。

**Usage**：

```bash
linctl new <DIR> [flags]
```

**Flags**：

| Flag | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--module <path>` | 是* | 自动推导 | Go module 路径，如 `github.com/foo/bar` |
| `--project-name <name>` | 否 | DIR basename | 项目名 |
| `--framework <name>` | 否 | `gin` | Web 框架：`gin` / `grpc` |
| `--storage <name>` | 否 | `memory` | 存储：`memory` / `gorm-postgres` / `gorm-mysql` / `gorm-sqlite` / `mongo` |
| `--features <list>` | 否 | 空 | 启用 Feature 列表：`healthz,opentelemetry,user,websocket,preloader` |
| `--deploy <mode>` | 否 | `docker` | 部署模式：`none` / `docker` / `kubernetes` / `systemd` |
| `--registry-prefix <p>` | 仅 docker/k8s | - | 镜像仓库前缀，如 `docker.io/myorg` |
| `--makefile <mode>` | 否 | `unstructured` | `none` / `unstructured` / `structured` |
| `--author <name>` | 否 | git config | 项目作者 |
| `--email <email>` | 否 | git config | 项目作者邮箱 |
| `--components <yaml>` | 否 | 默认 1 个 webserver | 直接传 components 数组的 YAML |
| `--config-file <file>` | 否 | - | 直接读完整 `linctl.yaml`（与上述 flag 互斥） |
| `--force / -f` | 否 | false | 覆盖目标目录中已存在的文件 |
| `--show-tips <bool>` | 否 | true | 生成完成后打印下一步命令 |

> *注：`--module` 在不传时会尝试通过 `$GOPATH/src/...` 推导，推导失败则报错要求显式传入。

**示例**：

```bash
# 1) 最简：默认 gin + memory
linctl new myblog --module github.com/foo/myblog

# 2) 完整功能
linctl new myblog \
  --module github.com/foo/myblog \
  --framework gin \
  --storage gorm-postgres \
  --features healthz,opentelemetry,user,preloader \
  --deploy kubernetes \
  --registry-prefix docker.io/foo

# 3) 从配置文件
linctl new myblog --config-file ./examples/full.yaml

# 4) dry-run 预览
linctl new myblog --module github.com/foo/myblog --dry-run
```

**输出示例**：

```
🎯 Generating project: myblog
   module:    github.com/foo/myblog
   framework: gin
   storage:   gorm-postgres
   features:  [healthz, opentelemetry, user, preloader]
   deploy:    kubernetes

📋 Plan
   Create  259 files
   Skip      0 files
   Conflict  0 files

🚀 Applying...
   ✔ go.mod
   ✔ Makefile
   ✔ cmd/myblog/main.go
   ... (256 more)

✨ Project ready in 1.4s

📦 Next steps:
   cd myblog
   make deps
   make protoc
   go mod tidy
   make build
   ./_output/bin/myblog server
   curl http://127.0.0.1:5555/healthz

📚 Documentation:
   docs/zh-CN/quickstart/README.md
```

### 3.3.2 `linctl add api <NAME...>`

**用途**：给指定 WebServer 增量添加 REST 资源（每个资源生成 ~10 个文件 + AST 注入到 `biz.go`/`store.go`/`*.proto`）。

**Usage**：

```bash
linctl add api <NAME>[,<NAME>...] [flags]
```

**Flags**：

| Flag | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--component <name>` | 否 | 单一时自动选 | 目标 WebServer 名 |
| `--worker <bool>` | 否 | false | 同时为 Worker 生成异步 handler |
| `--strategy <s>` | 否 | `ask` | 冲突策略：`skip`/`overwrite`/`ask`/`merge` |
| `--force / -f` | 否 | false | 等价 `--strategy=overwrite` |

**示例**：

```bash
# 加单个资源
linctl add api Post

# 加多个
linctl add api Post Comment Tag

# 带分组（生成在 biz/v1/job/cron_job/cron_job.go）
linctl add api job/CronJob

# 同时生成异步 handler
linctl add api Post --worker
```

**输出示例**：

```
🎯 Adding API resources to: myblog
   kinds: [Post, Comment]

📋 Plan
   Create  18 files (handler/biz/store/model/proto/errno × 2 kinds)
   Update   3 files (biz.go, store.go, myblog.proto via AST)
   Skip     0 files
   Conflict 0 files

🚀 Applying...
   ✔ pkg/api/myblog/v1/post.proto
   ✔ pkg/api/myblog/v1/comment.proto
   ✔ internal/myblog/handler/gin/post.go
   ✔ internal/myblog/biz/v1/post/post.go
   ✔ internal/myblog/store/post.go
   ✔ internal/myblog/model/post.gen.go
   ✔ internal/myblog/pkg/validation/post.go
   ✔ internal/myblog/pkg/conversion/post.go
   ✔ internal/pkg/errno/post.go
   ... (Comment files)
   ✏ internal/myblog/biz/biz.go (added 2 interface methods)
   ✏ internal/myblog/store/store.go (added 2 interface methods)
   ✏ pkg/api/myblog/v1/myblog.proto (added 10 RPC methods)

✨ Done in 0.6s

📦 Next steps:
   make protoc
   go mod tidy
   go generate ./...
   make build
   curl -X POST http://127.0.0.1:5555/v1/posts -d '{"title":"hello"}'
```

### 3.3.3 `linctl add worker <NAME...>`

**用途**：给 Worker 组件添加 cron 任务或 MQ handler。

**Flags**：

| Flag | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--variant <name>` | 否 | `cron` | `cron` / `kafka` / `customized` |
| `--component <name>` | 否 | 单一时自动选 | 目标 Worker 名 |
| `--strategy <s>` | 否 | `ask` | 冲突策略 |

**示例**：

```bash
# 添加 cron 任务
linctl add worker DailyReport --variant cron

# 添加 Kafka 消费者
linctl add worker OrderEvent --variant kafka

# 添加自定义 watcher
linctl add worker LLMTrain --variant customized
```

### 3.3.4 `linctl add cli <NAME...>`

**用途**：给 CLI 工具添加子命令。

**Flags**：

| Flag | 必填 | 默认 | 说明 |
| --- | --- | --- | --- |
| `--component <name>` | 否 | 单一时自动选 | 目标 CLI 名 |
| `--parent <path>` | 否 | root | 子命令挂载位置（如 `get` 表示挂在 `mbctl get xxx` 下） |

**示例**：

```bash
linctl add cli post comment
# → 生成 internal/mbctl/cmd/post/post.go 等
# → 注入 _ "github.com/.../cmd/post" 到 all.go
```

### 3.3.5 `linctl add webserver <NAME>`

**用途**：在已有项目中新增一个 WebServer 组件（适用于多服务架构）。

**示例**：

```bash
linctl add webserver mb-adminserver \
  --framework grpc \
  --storage gorm-mysql \
  --features healthz,opentelemetry,user
```

### 3.3.6 `linctl add feature <FEATURE>`

**用途**：给已有 Component 启用某个 Feature（追加模板 + AST 注入）。

**示例**：

```bash
# 给 myblog 加 OTel
linctl add feature opentelemetry --component myblog

# 给 mb-adminserver 加 user 特性
linctl add feature user --component mb-adminserver
```

> 等价于：手动改 `linctl.yaml` 里 features 列表 + 跑 `linctl apply`。这个命令是更便捷的语法糖。

### 3.3.7 `linctl plan`

**用途**：对比当前项目代码与 `linctl.yaml` 的期望状态，输出可读的变更计划。**不写任何文件**。

**Flags**（输出格式使用全局 [`--output / -o`](#321-通用控制)）：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--detailed` | false | 显示完整 diff |
| `--filter <kind>` | 空 | 仅显示某类 action：`create`/`update`/`conflict` |
| `--save <path>` | 空 | 将 plan 序列化到文件，供后续 `apply --plan <path>` 使用（包含 digest 防撕裂，详见 [SSOT 1.12](./META-fix-decisions-2026-04-25.md#112-plan-apply-digest-防撕裂)） |

> 输出格式通过全局 `--output / -o`（`text`/`json`/`yaml`）控制；机器消费推荐 `linctl plan -o json --save plan.json`。

**输出示例**：

```
📋 Plan for myblog (vs linctl.yaml)

[+] Create        4 files
    + internal/myblog/pkg/observability/otel.go
    + configs/myblog/otel.yaml
    + docs/zh-CN/operations/otel.md
    + scripts/test-otel.sh

[~] Update        2 files
    ~ internal/myblog/server.go
        Reason: feature "opentelemetry" added (was: [healthz,user])
        Diff (preview):
          +import "github.com/foo/myblog/internal/myblog/pkg/observability"
          +    if err := observability.InitOTel(ctx); err != nil { ... }
        Hash: abcd1234... -> efgh5678...

[!] Conflict      1 file
    ! internal/myblog/biz/v1/post/post.go
        Reason: user-modified (sha mismatch)
        Linctl-generated hash: 1111...
        Current file hash:     2222...
        → Run 'linctl apply --strategy=ask' to resolve.

[=] Skip          0 files

Summary: 4 create, 2 update, 1 conflict
```

### 3.3.8 `linctl apply`

**用途**：执行 plan 中的变更。

**Flags**：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--strategy <s>` | `ask` | 冲突策略：`skip`/`overwrite`/`merge`/`ask` |
| `--prune` | false | 删除 plan 标记为 obsolete 的文件 |
| `--backup-dir <dir>` | `.linctl/backups/<ts>` | apply 前的备份目录 |
| `--no-backup` | false | 禁用备份 |
| `--git-stash` | false | apply 前自动 `git stash`（仅当 git repo 干净时） |

**冲突处理交互（--strategy=ask）**：

```
! Conflict: internal/myblog/server.go
  Linctl base hash:  abcd...
  Disk hash:         efgh...
  New template hash: ijkl...

  Choose:
    [k] keep mine (skip this file)
    [o] overwrite with new template
    [m] 3-way merge (suggested)
    [d] show diff
    [s] skip and continue
    [q] quit (abort apply)

? Choice: m
✔ 3-way merged successfully
```

### 3.3.9 `linctl lint`

**用途**：校验 `linctl.yaml` 的语法/语义合法性，以及项目结构是否符合规范。

**检查项**：

1. `linctl.yaml` schema 校验（validator/v10 + JSON Schema）
2. `PROJECT` 文件存在且与 `linctl.yaml` 一致
3. 每个 Component 的 BinaryName 在 `cmd/` 下都有对应目录
4. 所有 generated 文件的 hash 注释完整（缺失说明可能被人为删除）
5. Feature 列表中的所有项都已注册

**输出示例**：

```
🔍 Linting myblog...

Configuration ............ ✔ valid
PROJECT consistency ...... ✔ matches linctl.yaml
Component layout ......... ✔ all 3 components have cmd/ entries
Hash integrity ........... ✗ 2 files missing hash comment
   - internal/myblog/biz/v1/post/post.go
   - internal/myblog/store/post.go
Feature registration ..... ✔ all features available

Summary: 1 issue found

💡 Hint: Files missing hash comment will be treated as user-owned.
   Run 'linctl apply --rehash' to mark them as linctl-generated.
```

### 3.3.10 `linctl doctor`

**用途**：检测开发环境，给出修复建议。

**检查项**：

| 检查项 | 说明 |
| --- | --- |
| Go 版本 | ≥ 1.22 |
| Git | 已安装 |
| protoc | 安装且版本 ≥ 3.21 |
| protoc-gen-go | 已安装 |
| buf | 推荐安装（用于格式化 .proto） |
| wire | 推荐安装（依赖注入） |
| mockgen | 推荐安装（生成 mock） |
| docker | 当 deploy=docker/k8s 时检查 |
| kubectl | 当 deploy=k8s 时检查 |

**输出示例**：

```
🩺 Checking environment...

Required:
   ✔ go         1.22.5  
   ✔ git        2.42.0
   ✔ protoc     3.25.1

Recommended:
   ✔ buf        1.28.1
   ✔ wire       0.5.0
   ✗ mockgen    not found
     → Install: go install go.uber.org/mock/mockgen@v0.4.0   # pin per §10.7

Optional:
   ✔ docker     24.0.5
   ⚠ kubectl    1.27 (project requires 1.28+)

Summary: 1 missing, 1 outdated
```

### 3.3.11 `linctl import`

**用途**：从已有 Go 项目反推 `linctl.yaml`，便于将旧项目纳入 linctl 管理。

**Flags**：

| Flag | 默认 | 说明 |
| --- | --- | --- |
| `--output-file <file>` | `linctl.yaml` | 生成的配置文件路径（**注意**：与全局 `--output / -o` 是不同的 flag——前者是文件路径，后者是格式 `text/json/yaml`） |
| `--component-name <name>` | dir basename | 主组件名 |

**逻辑**：

- 读 `go.mod` → modulePath
- 检查 `cmd/` 下子目录 → Components
- 检查 `internal/<name>/handler/{gin,grpc}/` → Framework
- 检查 `pkg/api/.../*.proto` → 已有 REST 资源
- 检查 `Dockerfile` / `manifests/` → DeploymentMode
- 输出最佳猜测的 `linctl.yaml`，提示用户人工 review

### 3.3.12 `linctl upgrade`

**用途**：跨版本升级 `linctl.yaml`（schema 演进），不修改项目代码。

**示例**：

```bash
# v1alpha1 → v1
linctl upgrade

# 输出
🔄 Upgrading linctl.yaml: v1alpha1 → v1
   ↻ Renamed: spec.flags → spec.defaults
   ↻ Renamed: spec.workers[].topics → spec.workers[].mq.topics
   ↻ Added defaults: spec.defaults.observability=true

✔ Backup saved to linctl.yaml.bak
✔ Upgraded successfully

⚠ Please run 'linctl plan' to preview changes to your code.
```

### 3.3.13 `linctl completion <SHELL>`

**用途**：输出 shell 补全脚本。

**支持**：bash / zsh / fish / powershell

**示例**：

```bash
# zsh
linctl completion zsh > "${fpath[1]}/_linctl"

# bash
linctl completion bash | sudo tee /etc/bash_completion.d/linctl > /dev/null
```

### 3.3.14 `linctl plugin`

**用途**：管理插件（kubectl 风格：`linctl-<name>` 可执行文件放入 PATH）。

**子命令**：

- `linctl plugin list` —— 列出已发现的插件
- `linctl plugin install <name>` —— 通过 `go install` 安装（如 `linctl-plugin-kratos`）
- `linctl plugin remove <name>` —— 删除可执行文件
- `linctl plugin info <name>` —— 显示插件信息（version、author、homepage）

**插件协议**：

- 插件可执行文件名：`linctl-<plugin-name>`
- 调用方式：`linctl <plugin-name> <args>` 等价于 `linctl-<plugin-name> <args>`
- 元数据获取：插件支持 `--linctl-info` flag 返回 JSON 元数据

### 3.3.15 `linctl version`

```bash
$ linctl version
linctl version v1.0.0
git commit: abc1234
build date: 2026-04-25T10:00:00Z
go version: go1.22.5
platform: darwin/arm64
```

```bash
$ linctl version --short
v1.0.0

$ linctl version --output json
{"version":"v1.0.0","gitCommit":"abc1234","buildDate":"2026-04-25T10:00:00Z","goVersion":"go1.22.5","platform":"darwin/arm64"}
```

## 3.4 命令分组（cobra 帮助页面）

`linctl --help` 输出按以下分组展示：

```
Usage:
  linctl [command]

Project Commands:
  new          Generate a new project
  add          Add resources to existing project
  plan         Show pending changes
  apply        Execute pending changes
  
Validation:
  lint         Validate linctl.yaml and project structure
  doctor       Check development environment
  
Migration:
  import       Reverse-engineer linctl.yaml from existing project
  upgrade      Upgrade linctl.yaml schema to latest version

Plugin & Setup:
  plugin       Manage linctl plugins
  completion   Generate shell completion script

Other:
  version      Show version information
  help         Help about any command
  options      List global flags

Use "linctl [command] --help" for more information about a command.
```

## 3.5 命令命名一致性约定

| 动词 | 含义 | 例 | 备注 |
| --- | --- | --- | --- |
| `new` | 创建全新事物（项目级，未存在） | `linctl new` | 仅作用于不存在的目标目录 |
| `add` | **受控扩展**：在已存在的项目上追加资源（**会修改既有 PROJECT 配置**） | `linctl add api` / `linctl add feature` / `linctl add component` | 同时承担「追加新资源」与「在既有 component 上启用 feature」两类语义；后者**事实上是一种受控的 update** |
| ~~`update`~~ | 不作为独立子命令提供 | — | 受控更新通过 `add` 完成（如 `add feature` ≡ 在既有 component 上新增 feature 并触发 plan/apply）；任意字段级编辑请直接修改 `linctl.yaml` 后跑 `linctl plan && linctl apply` |
| `remove` | 删除（破坏性） | （暂不提供，避免误删） | 计划在 Phase 4+ 引入 `linctl prune`（依赖 lock.yaml drift 检测） |
| `plan` | 仅展示变更，不执行 | `linctl plan` | 与 `apply --dry-run` 等价，详见 [§3.2.6](#326-dry-run-行为矩阵) |
| `apply` | 执行变更 | `linctl apply` | 默认会先内部 plan 一次（除非传 `--plan plan.json`） |
| `list` | 只读列出 | `linctl plugin list` | |
| `info` | 只读详情 | `linctl plugin info` | |
| `lint` | 静态校验 | `linctl lint` | |
| `doctor` | 环境诊断 | `linctl doctor` | |

### 3.5.1 为什么没有 `update` 子命令

`add` 在 linctl 的语义里是**「向 PROJECT 受控添加新元素」**，包括：

1. 新增资源（`add api Post` → 增加 `spec.components[].resources[]`）
2. 新增组件（`add webserver mb-admin` → 增加 `spec.components[]`）
3. 在已有组件上启用新 Feature（`add feature opentelemetry --component mb-apiserver` → 在 `spec.components[].features[]` 追加）

第 3 类**事实上是更新已有组件**，但操作是「追加 feature 到列表」（幂等），所以仍归在 `add` 动词下。这个选择有两个理由：

- **避免动词碎片化**：`add` 与 `update` 在 CLI 表层难以严格区分（如 `add feature` 与 `update component --add-feature`），引入 `update` 反而让用户记忆负担更重；
- **强约束面**：`add` 只允许「单调追加」，从不删除/重命名既有字段；任何会丢字段的修改都必须通过手动改 `linctl.yaml` 完成。

如果未来 PROJECT schema 复杂到必须细分（如「修改 webserver 的端口」），将引入 `linctl set <path> <value>`，而不是 `update`。

## 3.6 退出码（Exit Codes）

| 码 | 含义 |
| --- | --- |
| 0 | 成功 |
| 1 | 一般错误（template render / IO / unexpected） |
| 2 | 配置错误（schema invalid / missing required） |
| 3 | 用户不存在的资源（component not found） |
| 4 | 文件冲突（apply 时无 strategy） |
| 5 | 环境错误（go/git 未安装等） |
| 6 | 网络错误（plugin install 等） |
| 7 | **安全策略违规**（hook policy violation；CI 检测到 `unrestricted` hook；模板/路径越界等。详见 [META 决策书 §5.6](./META-fix-decisions-2026-04-25.md#56-ci-panic-退出码消歧) 与 [15-security-model.md](./15-security-model.md)） |
| 130 | 用户中断（Ctrl-C / abort） |

## 3.7 与 osbuilder 命令的对照

| osbuilder | linctl | 备注 |
| --- | --- | --- |
| `create project` | `new` | 命名简化 |
| `create api` | `add api` | 子命令更直观 |
| `create job` + `create mq` | `add worker --variant=cron/kafka` | 合并 |
| `create cmd` | `add cli` | 命名一致 |
| `create quickstart` | `new --features=defaults`（或 `new` 不带 flag） | 删除独立命令 |
| `semver tag/bump/release` | （独立工具） | 不内置 |
| `addlicense` | （独立工具） | 不内置 |
| `upgrade`（自升级） | `linctl upgrade-self`（如保留） | 重命名避免歧义 |
| `sysload`/`cleanupzombies` | （删除） | 与脚手架无关 |
| `version` | `version` | 一致 |
| `options` | `options` | 一致 |
| `plugin` | `plugin` | 增强：list/install/remove |
| 无 | `plan` / `apply` / `lint` / `doctor` / `import` | **新增**核心能力 |

## 3.8 用户体验（UX）原则

1. **默认值合理**：所有 flag 都应有合理默认，最小命令 `linctl new myblog --module github.com/foo/myblog` 即可工作。
2. **错误清晰**：每个错误提示包含 `Reason + Hint + Doc-link` 三段。
3. **进度可视**：耗时 > 1s 的命令显示 spinner（`charmbracelet/bubbletea` 或 `briandowns/spinner`）。
4. **输出可解析**：所有结构化命令支持全局 `--output / -o`（取值 `text`/`json`/`yaml`），如 `plan` / `version` / `lint` / `doctor`。详见 [§3.2.1](#321-通用控制)。
5. **可中断**：所有命令响应 SIGINT，立即退出但保证状态一致（已写文件不删，未写文件不创建）。
6. **危险操作必须确认**：`apply` 在有 conflict 且无 `-y` 时必须交互确认。

## 3.9 命令实现规范

每个命令实现遵循以下骨架：

```go
// internal/cli/cmd_<name>.go
package cli

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "github.com/spf13/cobra/doc"
)

// <name>Options 命令选项
type <name>Options struct {
    // 1) 用户传入的参数
    Foo string
    Bar bool

    // 2) Complete 阶段补充的字段（不导出）
    project *project.Project
    workDir string
}

// new<Name>Cmd 构造命令
func newCmd<Name>() *cobra.Command {
    o := &<name>Options{}
    cmd := &cobra.Command{
        Use:   "<verb> <args>",
        Short: "<one-line description>",
        Long:  longDesc<Name>,
        Example: `
  # comment
  linctl <verb> <args> --foo bar

  # another example
  linctl <verb> <args> --foo bar --baz`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            if err := o.complete(args); err != nil {
                return err
            }
            if err := o.validate(); err != nil {
                return err
            }
            return o.run(ctx)
        },
    }

    cmd.Flags().StringVar(&o.Foo, "foo", "", "describe foo")
    cmd.Flags().BoolVar(&o.Bar, "bar", false, "describe bar")
    return cmd
}

func (o *<name>Options) complete(args []string) error {
    // 加载 project, 推导默认值
    return nil
}

func (o *<name>Options) validate() error {
    // 校验参数合法性
    return nil
}

func (o *<name>Options) run(ctx context.Context) error {
    // 实际执行
    return nil
}

const longDesc<Name> = `Long description here.

Multiple lines explaining what the command does, when to use it, and any important caveats.
`
```

---

下一步阅读：[04-config-schema.md](./04-config-schema.md)

_Last reviewed: 2026-04-25_
