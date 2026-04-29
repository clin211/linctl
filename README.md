# linctl

[![CI](https://github.com/clin211/lin/actions/workflows/ci.yml/badge.svg)](https://github.com/clin211/lin/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/clin211/lin.svg)](https://pkg.go.dev/github.com/clin211/lin)

> **声明式、Plan/Apply 范式的 Go 微服务脚手架。** ✨
>
> 生成可幂等重跑、AST 友好的代码（不会覆盖你的修改）。设计灵感来自 Terraform 的 plan/apply 模型。

## 快速开始

```bash
# 通过 Go 安装（Phase 1 完成）
go install github.com/clin211/lin/cmd/linctl@latest

# 查看版本
linctl version --output json

# 1. 生成一个新项目（Phase 1 MVP 已就绪）
# Gin REST 项目：
linctl new myblog \
  --module github.com/foo/myblog \
  --framework gin \
  --storage memory \
  --features healthz

# gRPC 项目（Phase 3 Story 3.1 第一波）：
linctl new grpcdemo \
  --module github.com/foo/grpcdemo \
  --framework grpc \
  --storage memory \
  --features healthz

# Worker 项目（Phase 3 Story 3.2 第一波）— cron / kafka / customized 三种 variant 可共存：
linctl new reporter \
  --module github.com/foo/reporter \
  --kind Worker \
  --variants cron,kafka,customized

cd myblog

# 2. 增量添加 REST 资源（Phase 2 已就绪）
linctl add api Post       # 自动生成 3 个文件 + AST 注入 IBiz/IStore
linctl add api Comment    # 再次注入，多 resource 共存
linctl add api Post       # ✅ 幂等：0 文件改动

# 3. 编译运行
go mod tidy
go build ./...
./_output/bin/myblog          # 启动 HTTP 服务（默认 :8080）

# 4. 验证 healthz
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## 文档导航

完整文档位于 [`docs/`](./docs/) 目录（17 份文档，覆盖架构、CLI 设计、代码生成管道、AST 注入、安全模型等）。

推荐阅读顺序：

- [docs/00-overview.md](./docs/00-overview.md) — 5 分钟读懂全貌
- [docs/01-architecture.md](./docs/01-architecture.md) — L0–L4 分层架构
- [docs/03-cli-design.md](./docs/03-cli-design.md) — 全部 CLI 命令与 flag
- [docs/11-implementation-plan.md](./docs/11-implementation-plan.md) — Phase 1–5 路线图
- [docs/META-fix-decisions-2026-04-25.md](./docs/META-fix-decisions-2026-04-25.md) — SSOT 决策书（开发必看）

## 项目状态

| Phase | 状态 | 说明 |
|---|---|---|
| **Phase 1（MVP）** | ✅ **已完成核心闭环** | `linctl new` 可生成 gin + memory/postgres + healthz 项目；E2E 验证通过 |
| **Phase 2** | ✅ **全部 Story 完成** | Go AST + Proto AST 注入；冲突策略 skip/overwrite/ask（含 hash drift 检测）；LinctlError + RenderError + ConflictError 信息升级（Hint / Doc-link / line:col） |
| **Phase 3** | 🟡 **Story 3.1 + 3.2 第一波完成** | gin/gRPC/Worker 三种项目类型均可 `linctl new` 生成且 `go build` 通过；Worker 支持 cron/kafka/customized 三 variant 共存；后续：grpc-gateway / Deploy 模板 / `linctl add worker` |
| Phase 4 | ⏳ 计划中 | `plan` / `apply` / drift 检测 |
| Phase 5 | ⏳ 计划中 | 插件生态 |

详细 Story 进度见 [docs/11-implementation-plan.md §11.2](./docs/11-implementation-plan.md#112-phase-1mvp内核打通)。

## Phase 1 + Phase 2 已实现能力

```
✅ 15 个 Go 包，6500+ 行代码 + 测试
✅ 二进制大小 10MB（远低于 ≤15MB 预算）
✅ 测试覆盖率 75-93%（核心包）
✅ E2E 验证 1：linctl new → go build → curl /healthz 全链路通过
✅ E2E 验证 2：linctl add api Post → AST 注入 → go build 通过
✅ Mutator 幂等性已验证：第二次 add api Post 改动 0 文件
```

| 包 | 职责 |
|---|---|
| `cmd/linctl` | 主入口（< 50 行） |
| `internal/linctlerr` | 统一错误类型（LinctlError + Code） |
| `internal/cli` | 命令行解析（cobra + version + new） |
| `internal/version` | 版本元数据（含 ldflags 注入） |
| `internal/project` | linctl.yaml 数据模型 + Loader + Defaults + Saver |
| `internal/validate` | validator/v10 + 自定义规则（modulePath/projectName/...） |
| `internal/template` | 模板引擎（单一根 Template + ParseFS + Clone） |
| `internal/fs` | FileManager + 原子写 + flock + SafeJoin + hash 注释 |
| `internal/codegen` | Pair / PairBuilder / Plan / Planner / Applier |
| `internal/component` | Component 抽象 + WebServer 实现 |
| `internal/feature` | Feature 接口 + Registry + 拓扑排序（环检测） |
| `internal/feature/builtin` | healthz 内置 Feature |
| `internal/orchestrator` | Plan→Apply 编排 + Reporter |
| `internal/ast` | Go AST 注入（dst-based）：AddImportMutator + AddInterfaceMethodMutator + Batch + ConflictError（含 Hint）；Proto AST（protocompile）：AddProtoRPCMutator |
| `internal/ui` | 终端交互：Confirmer 接口 + IOConfirmer（非 tty 自动 fallback）+ AlwaysYes/AlwaysNo/QueueConfirmer 测试桩 |

## 开发指南

```bash
make tools          # 安装 pin 版本的工具（gofumpt v0.7.0、golangci-lint v1.59.1、mockgen v0.4.0 等）
make all            # lint + test + build
make build          # 输出 _output/bin/linctl
make build-otel     # 启用 OpenTelemetry 的构建（默认不带，遵循 ≤15MB 预算）
make test           # 运行测试（含 race + coverage）
make cover          # 生成 HTML 覆盖率报告
```

## 与 osbuilder 的对比

| 维度 | osbuilder | linctl |
|---|---|---|
| 模板嵌入 | `rakyll/statik`（已归档） | `embed.FS`（标准库） |
| Go AST 注入 | `go/ast`（注释丢失） | `dave/dst`（保留注释/空行）|
| 重入能力 | 一次性生成 + AST 追加 | `plan/apply` 完整闭环 + drift 检测 |
| 特性扩展 | 改 `Pairs()` 函数 + 加 switch | 注册 `Feature` 插件 |
| 错误处理 | 部分 `fmt.Printf` 吞错 | 统一 `LinctlError` + Code + Hint |
| 二进制大小 | ~20 MB | **10 MB**（实测） |
| 直接依赖 | ~45 | **9** |

## 贡献

参见 [CONTRIBUTING.md](./CONTRIBUTING.md)。

**重要**：所有设计变更必须先更新 [`docs/`](./docs/) 中的对应文档，并遵守 [SSOT 决策书](./docs/META-fix-decisions-2026-04-25.md) 中已锁定的 31 项决策。

## License

MIT — 详见 [LICENSE](./LICENSE)。
