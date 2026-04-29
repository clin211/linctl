# 07. 交互式终端 UX 设计

> **前置阅读**：[02-command-set.md](./02-command-set.md)、[03-resource-scaffold.md](./03-resource-scaffold.md)
>
> **设计灵感**：Vue CLI、Vite create、create-react-app、`npm create`、Cargo `cargo new` 风格
>
> 本文档定义 `lin new` 与 `lin add` 的**交互式默认体验**——用户无需记忆 flag，工具引导式收集参数。

---

## 1. 设计原则

### 1.1 三种使用模式（无缝切换）

```
┌────────────────────────────────────────────────────────────┐
│ Mode A: 交互式（默认）                                       │
│   $ lin new                                                 │
│   → 进入向导，逐步询问                                        │
│                                                             │
│ Mode B: 部分 flag                                            │
│   $ lin new myblog --module github.com/foo/myblog           │
│   → 已传的 flag 跳过对应 prompt，未传的继续问                │
│                                                             │
│ Mode C: 完全 flag (CI/脚本)                                  │
│   $ lin new myblog --module ... --storage ... --yes         │
│   → 所有参数都传齐 + --yes 跳过最终确认                       │
│                                                             │
│ Mode D: 强制非交互                                           │
│   $ lin new myblog --non-interactive --module ...           │
│   → 缺失参数直接报错（CI/脚本场景）                           │
└────────────────────────────────────────────────────────────┘
```

### 1.2 核心原则

| # | 原则 | 实现 |
| --- | --- | --- |
| 1 | **零认知负担** | 不要求用户记任何 flag；选项有合理默认 + 简短解释 |
| 2 | **flag 与交互无缝混合** | 已传 flag 跳过 prompt；未传则进入 prompt |
| 3 | **TTY 自适应** | 非 TTY（管道、CI）自动切到非交互；TTY 默认交互 |
| 4 | **可视化反馈** | 颜色 / 图标 / 进度条 / 摘要表 |
| 5 | **可中断 + 可回退** | `ESC` 返回上一题；`Ctrl+C` 干净退出 |
| 6 | **预览后再执行** | 摘要表 + 确认对话；`Ctrl+C` 取消 |
| 7 | **错误友好** | 每个 prompt 内置 validator；输入非法当场提示 |

---

## 2. 技术选型

| 候选 | 评估 | 决定 |
| --- | --- | --- |
| **`charmbracelet/huh`** | 现代声明式 form；表单/字段/选择/确认/文本一体；视觉效果优秀；活跃维护 | ✅ **采用** |
| `charmbracelet/bubbletea` | 底层 TUI 框架；更灵活但代码更重 | 备用（特殊场景） |
| `AlecAivazis/survey/v2` | 经典选择；stable 但维护偏弱 | 不采用 |
| `manifoldco/promptui` | 简单；但维护停滞 | 不采用 |
| `pterm` | 富 TUI 库；侧重渲染而非表单 | 仅用于结果展示 |

**关键依赖**：

```go
require (
    github.com/charmbracelet/huh v0.4.0     // 表单交互
    github.com/charmbracelet/lipgloss v0.10.0  // 样式渲染
    github.com/briandowns/spinner v1.23.0    // 进度 spinner（可选）
    golang.org/x/term v0.20.0                 // TTY 检测
)
```

---

## 3. `lin new` 完整交互流程（Mockup）

### 3.1 启动横幅

```
$ lin new
   ┌──────────────────────────────────────────┐
   │   ✨ lin - Go Project Scaffolder          │
   │   v2.0.0  ·  miniblog-v4 style            │
   └──────────────────────────────────────────┘

  Welcome! Let's craft your project.
```

### 3.2 项目身份（4 步）

```
  ? Project name › myblog
    Tip: 仅小写字母、数字、连字符。也将作为 cmd/<app>/ 目录名。

  ? Go module path › github.com/foo/myblog
    Tip: 默认尝试从 git remote / GOPATH 推导。

  ? Author name › Forest Lin               (默认: git config user.name)
  ? Author email › forest@example.com      (默认: git config user.email)
```

### 3.3 技术栈选择（多步表单）

```
  ? Storage backend › 
    > PostgreSQL (gorm)        ← 推荐
      MySQL (gorm)
      SQLite (gorm)
      MongoDB
      In-memory (仅开发用)
    
    ↓↑ 移动 · ↵ 选择 · ESC 返回

  ? Web framework › 
    > gin                      ← 当前 MVP 唯一支持
      grpc-gateway (Phase 2)   (灰显，不可选)
      grpc only (Phase 2)      (灰显，不可选)

  ? Features (空格切换, 回车确认) › 
    [✓] healthz       /healthz 健康检查端点
    [✓] otel          OpenTelemetry trace + metrics
    [ ] user          内置用户/认证/RBAC 模块
    [ ] swagger       Swagger UI 文档站点
    [ ] preloader     启动预热（缓存/连接池）
    
    ↓↑ 移动 · 空格 切换 · ↵ 确认 · A 全选 · N 全不选
```

### 3.4 部署模式

```
  ? Deployment › 
    > Docker (默认)
      Docker + Kubernetes manifests
      systemd
      None
    
  ? Image registry prefix › docker.io/foo
    Tip: 启用 Docker 时建议提供，否则使用 "lin/<project>"。
```

### 3.5 工程化选项

```
  ? 初始化 git 仓库? › Yes
  ? 创建初始提交? › Yes
  ? 在生成后运行 'go mod tidy'? › Yes
```

### 3.6 摘要确认

```
   ┌──────────────────────────────────────────────┐
   │   📦 即将生成项目                              │
   ├──────────────────────────────────────────────┤
   │   Project:    myblog                          │
   │   Module:     github.com/foo/myblog           │
   │   Author:     Forest Lin <forest@example.com> │
   │   Framework:  gin                             │
   │   Storage:    PostgreSQL (gorm)               │
   │   Features:   healthz, otel                   │
   │   Deployment: Docker                          │
   │   Registry:   docker.io/foo                   │
   │   git init:   ✓                               │
   │   go tidy:    ✓                               │
   ├──────────────────────────────────────────────┤
   │   预计创建:  ~43 个文件                        │
   │   预计大小:  ~120 KB                           │
   └──────────────────────────────────────────────┘

  ? 确认生成? (Y/n) › Y
```

### 3.7 执行阶段

```
  🎯 Generating project: myblog

  📦 Creating files...
     ✔ go.mod
     ✔ Makefile
     ✔ Dockerfile
     ✔ cmd/myblog/main.go
     ⠹ internal/myblog/handler/healthz.go      (spinner during render)
     ✔ internal/myblog/handler/healthz.go
     ...
     [████████████████████░░░░░░░░] 28/41

  🔧 Running post-actions...
     ✔ git init
     ✔ git add .
     ✔ git commit -m "chore: initial commit by lin"
     ⠹ go mod tidy                              (spinner)
     ✔ go mod tidy

  ✨ Project ready in 2.3s

   ┌──────────────────────────────────────────────┐
   │   📦 Next steps                                │
   ├──────────────────────────────────────────────┤
   │   $ cd myblog                                 │
   │   $ make build                                │
   │   $ ./_output/myblog server                   │
   │   $ curl http://127.0.0.1:8080/healthz        │
   ├──────────────────────────────────────────────┤
   │   📚 Docs:   docs/devel/zh-CN/development.md  │
   │   🐛 Issues: https://github.com/.../issues    │
   └──────────────────────────────────────────────┘
```

---

## 4. `lin add` 完整交互流程（Mockup）

### 4.1 上下文检测

```
$ lin add

  📂 Detected project context:
     Module:    github.com/foo/myblog
     App:       myblog
     Storage:   gorm-postgres
     Features:  healthz, otel
```

### 4.2 资源信息

```
  ? Resource name › Post
    Tip: PascalCase 单数；将派生 post / posts / PostBiz 等。

  ? 生成哪些层? (空格切换, 回车确认) › 
    [✓] handler       HTTP 路由处理         (必选)
    [✓] biz           业务逻辑（5 动词）    (必选)
    [✓] store         持久化层              (必选)
    [✓] model         数据模型              (必选)
    [✓] conversion    DTO 与 model 转换
    [✓] validation    入参校验
    [✓] errno         业务错误码
    [✓] proto         API 契约 (.proto)
    
    Tip: 必选项不可取消。

  ? CRUD 操作 (空格切换, 回车确认) › 
    [✓] Create
    [✓] Update
    [✓] Delete
    [✓] Get
    [✓] List
```

### 4.3 摘要确认

```
   ┌──────────────────────────────────────────────┐
   │   📦 即将生成 Post 资源                         │
   ├──────────────────────────────────────────────┤
   │   将创建 13 个文件:                            │
   │     internal/myblog/handler/post.go           │
   │     internal/myblog/biz/v1/post/post.go       │
   │     internal/myblog/biz/v1/post/create.go     │
   │     internal/myblog/biz/v1/post/update.go     │
   │     internal/myblog/biz/v1/post/delete.go     │
   │     internal/myblog/biz/v1/post/get.go        │
   │     internal/myblog/biz/v1/post/list.go       │
   │     internal/myblog/store/post.go             │
   │     internal/myblog/model/post.gen.go         │
   │     internal/myblog/pkg/conversion/post.go    │
   │     internal/myblog/pkg/validation/post.go    │
   │     internal/pkg/errno/post.go                │
   │     pkg/api/myblog/v1/post.proto              │
   │                                               │
   │   将注入 4 个中央文件:                          │
   │     ✏ internal/myblog/biz/biz.go              │
   │     ✏ internal/myblog/store/store.go          │
   │     ✏ pkg/api/myblog/v1/myblog.proto          │
   │     ✏ internal/pkg/errno/register.go          │
   └──────────────────────────────────────────────┘

  ? 确认生成? (Y/n) › Y
```

### 4.4 执行阶段

```
  🎯 Adding resource: Post

  📦 Creating files...
     ✔ internal/myblog/handler/post.go
     ✔ internal/myblog/biz/v1/post/post.go
     ...
     [████████████████████] 13/13

  ✏ Injecting AST...
     ✔ biz.go      added: PostV1() postv1.PostBiz
     ✔ store.go    added: Posts() PostStore
     ✔ myblog.proto  added: import "post.proto";
     ✔ register.go  added: RegisterErrors(PostErrors()...)

  ✨ Done in 0.4s

   ┌──────────────────────────────────────────────┐
   │   📦 Next steps                                │
   ├──────────────────────────────────────────────┤
   │   $ make protoc                               │
   │   $ go mod tidy                               │
   │   $ go build ./...                            │
   ├──────────────────────────────────────────────┤
   │   💡 Tip: 实现业务逻辑请编辑                    │
   │      internal/myblog/biz/v1/post/create.go    │
   │      internal/myblog/biz/v1/post/update.go    │
   │      ...                                       │
   └──────────────────────────────────────────────┘
```

---

## 5. flag 与交互的合并规则

### 5.1 优先级

```
1. 命令行 flag （最高，跳过对应 prompt）
2. 交互式 prompt （TTY 默认）
3. 默认值       （prompt 的默认）
4. 错误         （非 TTY 且未传 flag 且无默认）
```

### 5.2 合并示例

```bash
# 部分 flag + 交互
$ lin new myblog --module github.com/foo/myblog --storage gorm-postgres
  
  → 跳过：项目名（已传 myblog）、module、storage
  → 仍然问：author/email（无 flag）、framework、features、deployment、git init...
```

```bash
# 完全 flag（CI 友好）
$ lin new myblog \
    --module github.com/foo/myblog \
    --storage gorm-postgres \
    --features healthz,otel \
    --with-docker \
    --author "Forest Lin" \
    --email "forest@example.com" \
    --yes  # 跳过最终确认

  → 不进入任何交互，直接执行
```

```bash
# CI 环境（自动判定非交互）
$ lin new myblog --module github.com/foo/myblog
  在 GitHub Actions 中（无 TTY）：
  → 缺失参数：error: --storage is required in non-interactive mode
  → 退出码 2
```

### 5.3 `--non-interactive` 行为

| 标志 | TTY | 非 TTY | 含义 |
| --- | --- | --- | --- |
| 无标志 | 交互 | 自动转非交互 | 智能默认 |
| `--non-interactive` | **强制非交互** | 非交互 | 缺参数立即报错 |
| `--yes` | 交互（保留），跳过最终确认 | 非交互+无确认 | "no questions asked" |

---

## 6. 视觉规范

### 6.1 配色（基于 lipgloss）

```go
var (
    titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
    successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
    warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F1FA8C"))
    errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
    infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
    mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
    boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)
```

### 6.2 图标体系（emoji + nerd font fallback）

| 用途 | Emoji | ASCII Fallback |
| --- | --- | --- |
| 启动 / 标题 | ✨ | `*` |
| 创建文件 | ✔ | `[+]` |
| AST 注入 | ✏ | `[~]` |
| 跳过（已存在） | ⊝ | `[=]` |
| 删除 | ✗ | `[-]` |
| 信息 | 💡 | `[i]` |
| 警告 | ⚠ | `[!]` |
| 错误 | ❌ | `[X]` |
| 流程 | 🎯 / 📦 / 🚀 / 📚 | (留白) |

> `--no-emoji` 启用时全部降级 ASCII。

### 6.3 状态文本约定

```
  ✔ <action>            — 成功（绿色）
  ⊝ <action> (skipped)  — 跳过（灰色）
  ⚠ <action>            — 警告（黄色）
  ✗ <action>            — 失败（红色）
  ⠹ <action>            — 进行中（spinner，蓝色）
```

---

## 7. 可中断行为

| 触发 | 行为 |
| --- | --- |
| `Ctrl+C` 在交互期 | 立即退出，无副作用，退出码 130 |
| `Ctrl+C` 在执行期（创建文件中途） | 中止后续操作；已创建文件**全部回滚**；打印 "interrupted, rolled back" |
| `Ctrl+C` 在 AST 注入期 | 同上；中央文件从备份恢复 |
| `ESC` 在 prompt 中 | 返回上一题（如果是第一题则取消） |
| 网络/文件错误 | 立即停止，错误三段式输出，回滚 |

---

## 8. 实现要点（伪代码）

```go
package cli

import (
    "github.com/charmbracelet/huh"
    "golang.org/x/term"
    "os"
)

func runNewInteractive(o *newOptions) error {
    // 1. 检测 TTY
    isTTY := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
    if o.NonInteractive || !isTTY {
        return runNewFromFlags(o)
    }

    // 2. 构造 form
    form := huh.NewForm(
        // Group 1: Identity
        huh.NewGroup(
            huh.NewInput().
                Title("Project name").
                Description("仅小写字母、数字、连字符").
                Value(&o.ProjectName).
                Validate(validateProjectName),

            huh.NewInput().
                Title("Go module path").
                Description("默认尝试从 git remote 推导").
                Value(&o.Module).
                Suggestion(detectModuleFromGit()).
                Validate(validateModulePath),

            huh.NewInput().
                Title("Author name").
                Value(&o.Author).
                Suggestion(detectGitAuthor()),

            huh.NewInput().
                Title("Author email").
                Value(&o.Email).
                Suggestion(detectGitEmail()).
                Validate(validateEmail),
        ),

        // Group 2: Tech stack
        huh.NewGroup(
            huh.NewSelect[string]().
                Title("Storage backend").
                Options(
                    huh.NewOption("PostgreSQL (gorm) — 推荐", "gorm-postgres"),
                    huh.NewOption("MySQL (gorm)", "gorm-mysql"),
                    huh.NewOption("SQLite (gorm)", "gorm-sqlite"),
                    huh.NewOption("MongoDB", "mongo"),
                    huh.NewOption("In-memory (仅开发用)", "memory"),
                ).
                Value(&o.Storage),

            huh.NewMultiSelect[string]().
                Title("Features").
                Options(
                    huh.NewOption("healthz — /healthz 健康检查端点", "healthz").Selected(true),
                    huh.NewOption("otel — OpenTelemetry trace + metrics", "otel"),
                    huh.NewOption("user — 内置用户/认证/RBAC 模块", "user"),
                    huh.NewOption("swagger — Swagger UI 文档站点", "swagger"),
                    huh.NewOption("preloader — 启动预热", "preloader"),
                ).
                Value(&o.Features),
        ),

        // Group 3: Deployment
        huh.NewGroup(
            huh.NewSelect[string]().
                Title("Deployment").
                Options(
                    huh.NewOption("Docker (默认)", "docker"),
                    huh.NewOption("Docker + Kubernetes", "k8s"),
                    huh.NewOption("systemd", "systemd"),
                    huh.NewOption("None", "none"),
                ).
                Value(&o.Deployment),

            huh.NewInput().
                Title("Image registry prefix").
                Description("启用 Docker 时建议提供").
                Value(&o.RegistryPrefix).
                Suggestion("docker.io/" + currentUser()),
        ).WithHideFunc(func() bool {
            return o.Deployment == "none"  // 条件分支
        }),
    ).WithTheme(huh.ThemeDracula())

    if err := form.Run(); err != nil {
        return err
    }

    // 3. 摘要确认
    showSummary(o)
    var confirm bool
    if err := huh.NewConfirm().
        Title("确认生成?").
        Affirmative("Yes").
        Negative("No").
        Value(&confirm).Run(); err != nil {
        return err
    }
    if !confirm {
        return errs.UserCancelled
    }

    // 4. 执行
    return executeWithSpinner(o)
}
```

---

## 9. flag 推断与默认（减少 prompt）

| 字段 | 优先级（高 → 低） |
| --- | --- |
| project name | argv[0] → prompt |
| module | `--module` → `git remote get-url origin` 推断 → `${GOPATH}/src/<inferred>` → prompt |
| author | `--author` → `git config user.name` → prompt |
| email | `--email` → `git config user.email` → prompt |
| storage | `--storage` → 默认 `gorm-postgres` → prompt |
| framework | `--framework` → 默认 `gin` → 跳过（仅一个选项） |
| features | `--features` → 默认 `healthz` → prompt |
| deployment | `--with-docker/--with-k8s` → 默认 `docker` → prompt |
| image registry | `--registry-prefix` → `docker.io/${USER}` → prompt |

---

## 10. 测试策略

### 10.1 交互式难以单测，但可以：

| 层级 | 测试方式 |
| --- | --- |
| **prompt validator** | 单测：输入 → 期望（valid/invalid + message） |
| **flag merge 逻辑** | 单测：模拟 `Options{Module: "x"}` → form 跳过 module |
| **TTY 检测** | 单测：mock `term.IsTerminal` |
| **执行阶段** | E2E：用 `--yes --non-interactive` 走 flag 模式，覆盖端到端 |
| **TUI 渲染** | 手动 + 截图 review；CI 跑 `--non-interactive` |

### 10.2 E2E 跑非交互即可

```bash
lin new myblog \
  --module github.com/test/myblog \
  --storage gorm-postgres \
  --features healthz \
  --with-docker \
  --author "Test" \
  --email "test@test.com" \
  --yes \
  --non-interactive
```

---

## 11. 退出行为

| 场景 | 退出码 | 说明 |
| --- | --- | --- |
| 用户在 prompt 取消（ESC 多次或 Ctrl+C） | 130 | 干净退出，无副作用 |
| 用户在确认对话拒绝 | 0 | 正常退出，无操作 |
| 非交互模式缺失必要参数 | 2 | "missing required: --module" |
| 不支持的选项值 | 2 | "unknown storage 'foo'" |
| 执行期错误 | 1 | 回滚 + 错误信息 |

---

## 12. 移植到其他命令

| 命令 | 是否交互式 | 备注 |
| --- | --- | --- |
| `lin new` | ✅ 完整交互 | 见 §3 |
| `lin add` | ✅ 完整交互 | 见 §4 |
| `lin lint` | ❌ 无交互 | 直接输出报告 |
| `lin doctor` | ❌ 无交互 | 直接输出环境表 |
| `lin version` | ❌ 无交互 | 直接输出 |
| `lin completion` | ❌ 无交互 | 输出脚本 |

---

## 13. 与现有命令文档的协调

[02-command-set.md](./02-command-set.md) 各命令的 Usage 段需补充：

> **默认行为**：`lin new` / `lin add` 不传 flag 时进入交互式向导（见 [07-interactive-ux.md](./07-interactive-ux.md)）。已传的 flag 跳过对应 prompt。在非 TTY 环境（管道、CI）自动切换为非交互模式，缺参数报错。

---

## 14. 与决策记录的关系

本设计**不改变** [00 §10](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29) 的 6 项决策，仅强化 §5.6（删除 yaml）的 UX 实现路径：

> "删除配置文件" + "默认交互式 prompt" = 用户既不需要写 yaml 也不需要记 flag，但仍能在 CI 用 flag 自动化。

---

_Last reviewed: 2026-04-29_
