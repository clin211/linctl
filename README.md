# linctl v2

[![CI](https://github.com/clin211/lin/actions/workflows/ci.yml/badge.svg)](https://github.com/clin211/lin/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/clin211/lin.svg)](https://pkg.go.dev/github.com/clin211/lin)

> **linctl** — 零摩擦、幂等友好的 Go 后端脚手架工具。两条命令，一次生成，重跑安全。（Go module：`github.com/clin211/lin`）

---

## 核心价值

| 特性 | 说明 |
|------|------|
| **零摩擦上手** | `linctl new` 生成完整 miniblog-v4 骨架，无需手写样板代码 |
| **幂等重跑** | `linctl add` 重复执行不会破坏既有代码（AST 注入基于锚点，跳过已有内容） |
| **自我诊断** | `linctl lint` 检查项目结构与 AST 完整性；`linctl doctor` 检查工具链 |
| **纯 Go** | 无外部依赖运行时；单二进制，嵌入所有模板 |

---

## 安装

```bash
go install github.com/clin211/lin/cmd/linctl@latest
```

或从源码构建：

```bash
git clone https://github.com/clin211/lin.git
cd lin
make build          # → _output/bin/linctl
```

---

## 快速上手

### 1. 生成项目骨架

**交互式（终端内逐步问答，类似 Vite）：** 直接执行 `linctl new`（需在真实 TTY 中；管道/CI 请用下面的非交互方式）。

**非交互 / CI：**

```bash
linctl new myblog \
  --module github.com/foo/myblog \
  --storage memory \
  --features healthz \
  --yes
```

```
✔ project plan computed (40 files)
✔ scaffold rendered into ./myblog
📦 Next steps:
   cd myblog
   make deps
   make protoc
   make build
```

### 2. 添加业务资源

```bash
cd myblog
linctl add Post          # 生成 14 个文件 + 4 处 AST 注入
linctl add Comment       # 再次添加；重跑幂等
```

```
✔ resource: Post
   + 14 files created
   ✏  4 files updated via AST
📦 Next steps:
   make protoc
   go mod tidy
   go build ./...
```

### 3. 校验项目健康

```bash
linctl lint              # 检查目录结构 + AST 锚点
linctl lint --fix        # 自动修复缺失的锚点注释
linctl doctor --offline  # 检查工具链（跳过网络检查）
```

---

## 命令清单

| 命令 | 说明 |
|------|------|
| `linctl new <project>` | 生成新项目骨架（miniblog-v4 风格） |
| `linctl add <Resource>...` | 在已有项目中添加全栈业务资源 |
| `linctl lint [--fix]` | 校验项目结构与 AST 锚点完整性 |
| `linctl doctor [--offline]` | 检查本地工具链与运行环境 |
| `linctl version` | 打印版本信息 |
| `linctl completion <shell>` | 生成 shell 补全脚本（bash/zsh/fish/powershell） |

全局 flag：`--log-level` / `--log-format` / `-C, --chdir` / `--no-color` / `--non-interactive` / `-y, --yes`

---

## 生成的项目结构

```
myblog/
├── cmd/myblog/                   # 主入口
├── internal/myblog/
│   ├── handler/                  # HTTP handlers (gin)
│   ├── biz/v1/<resource>/        # 业务逻辑层
│   ├── store/                    # 数据存储层
│   └── model/                    # 数据模型
├── internal/pkg/errno/           # 错误码统一管理
├── pkg/api/<app>/v1/             # Proto 定义 + 占位 Go 类型
└── ...
```

---

## v1 → v2 迁移

v2 重构了命令集，不再支持 v1 的 `plan`/`apply` 范式；**v2 的可执行文件名为 `linctl`**（入口 `cmd/linctl`）。

| v1 命令 | v2 对应 | 说明 |
|---------|---------|------|
| `linctl plan` | `linctl add --dry-run` | 预览生成计划 |
| `linctl apply` | `linctl add` | 执行生成 |
| `linctl new` | `linctl new` | 生成项目骨架 |
| — | `linctl lint` | v2：AST 完整性检查 |
| — | `linctl doctor` | v2：工具链检查 |

详细迁移指引见 [docs/features/06-migration-plan.md](./docs/features/06-migration-plan.md)。

---

## 设计文档

- [00 重构理由](./docs/features/00-refactor-rationale.md)
- [01 架构蓝图](./docs/features/01-architecture-blueprint.md)
- [02 命令集详细设计](./docs/features/02-command-set.md)
- [03 资源骨架生成](./docs/features/03-resource-scaffold.md)
- [04 模板系统](./docs/features/04-template-system.md)
- [05 注册策略](./docs/features/05-registration-strategy.md)
- [06 迁移计划](./docs/features/06-migration-plan.md)
- [07 交互式 UX](./docs/features/07-interactive-ux.md)

---

## 开发

```bash
make all        # lint + test + build
make test       # 单元测试（含 race detector）
make e2e        # 端到端测试（new / add / lint / doctor）
make vet        # go vet
```

需要 Go >= 1.22。
