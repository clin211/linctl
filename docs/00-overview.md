# 00. linctl 项目总览

## 0.1 一句话定位

> **linctl** 是一款**声明式、可计划、可回滚、插件化**的 Go 项目脚手架 CLI，让"生成项目"和"维护项目"在同一个工具下完整闭环。

它面向所有想"以 Go 最佳实践、零样板代码、可演进"方式构建后端服务的团队。

## 0.2 我们要解决的问题

### 现状的痛点

| # | 问题 | 现有方案的不足 |
| --- | --- | --- |
| 1 | 写一个能跑的 Go 服务样板代码巨多 | `cookiecutter` 等纯模板工具只能一次性生成，后续无法演进 |
| 2 | 想在 grpc/gin/go-zero 之间切换？ | `kratos new`、`go-zero goctl`、`osbuilder` 都绑死单一框架风格 |
| 3 | 项目跑了几个月想加个新资源（如 `Comment`） | 手写一遍 handler/biz/store/proto 巨痛苦，AI 生成又跟现有命名不一致 |
| 4 | 模板更新了，已生成的项目跟不上 | 无 `plan/apply` 闭环，要么人工 diff 要么放弃 |
| 5 | 团队约定的 Makefile/CI/Dockerfile 想规范化 | 散落在 wiki，没有"工具强制" |
| 6 | 不同业务想加 OTel/Sentry/AuthZ 等横切能力 | 主线代码越改越乱，没有"特性"概念 |
| 7 | 工具升级容易破坏既有项目 | 没有 hash 追踪、没有 3-way merge |
| 8 | 工具本身臃肿 | osbuilder 引入了 ~160 个间接依赖 |

### linctl 的解法（与本质对应）

| # | 解法 | 设计文档 |
| --- | --- | --- |
| 1 | 一键 `linctl new` 生成可运行项目（**全栈 Feature 启用时 250-350 个文件**[^1]；最小示例约 32 个） | [11-implementation-plan.md](./11-implementation-plan.md) |
| 2 | `framework` 字段 + Feature 插件，未来通过插件扩展任意框架 | [08-feature-system.md](./08-feature-system.md) |
| 3 | `linctl add api --kinds comment` 一键加资源，AST 注入到现有代码 | [07-ast-injection.md](./07-ast-injection.md) |
| 4 | `linctl plan / apply` 模仿 Terraform 的声明式闭环 | [06-codegen-pipeline.md](./06-codegen-pipeline.md) |
| 5 | 内置规范化的 Makefile / Dockerfile / golangci-lint / GitHub Actions 模板 | [05-template-system.md](./05-template-system.md) |
| 6 | Feature 系统：每个特性独立模板 + AST mutator + funcMap | [08-feature-system.md](./08-feature-system.md) |
| 7 | 文件级 `// linctl: hash=<sha256>` 注释 + 3-way merge | [06-codegen-pipeline.md](./06-codegen-pipeline.md) |
| 8 | 拒绝 k8s 全家桶，只保留 cobra/viper/afero 等核心 ~15 个依赖 | [10-tech-stack.md](./10-tech-stack.md) |

## 0.3 目标用户

### 主要用户：**Go 后端团队**

- **新项目场景**：希望 30 分钟内拿到一个生产级骨架（含 OTel / 鉴权 / Healthz / Docker / K8s manifests）。
- **存量项目场景**：希望以一致的方式增量添加资源、抽象、Feature。
- **平台团队**：希望把团队的 Go 项目规范沉淀为模板和 Feature，全公司统一。

### 次要用户：**第三方框架/特性贡献者**

- 通过 Feature 插件机制，给社区提供 `linctl-plugin-kratos`、`linctl-plugin-sentry` 等扩展。

### 不适合的场景

- 仅写一个临时小工具（用 `go init` 即可）。
- 已经使用 `kratos`/`go-zero` 等技术栈生态深度绑定，不打算迁移。
- 期望"图形界面"的场景（linctl 是纯 CLI，但可作为 IDE 插件后端）。

## 0.4 核心价值主张

### 对开发者

```
🚀 30 分钟从 0 到生产可用
💎 内置最佳实践（不需要靠 wiki）
♻️  改 schema 后能 reconcile，而不是手改文件
🔌 Feature 像积木：要哪个加哪个
🛡 安全的代码改动：plan 后才 apply，hash 追踪冲突
```

### 对团队

```
📐 统一规范：Makefile/CI/Dockerfile 一次定义全公司复用
🎯 提速新人 onboarding：一行命令出可跑项目
📉 减少代码评审噪音：模板生成的 boilerplate 不用 review
🔭 可观测性默认开启：OTel / metrics / log 标准化
```

### 对组织

```
💰 项目交付速度提升 50%+（基于 osbuilder 团队经验）
🛠 工具链标准化，跨团队协作成本骤降
🧬 业务知识可固化为 Feature 插件，新业务直接复用
🔒 安全/合规能力可作为强制 Feature 注入（如 audit log、PII 脱敏）
```

## 0.5 与同类工具的对比

| 维度 | osbuilder | kubebuilder | go-zero goctl | kratos new | cookiecutter | **linctl** |
| --- | --- | --- | --- | --- | --- | --- |
| 一次性生成 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 增量加资源 | ✅ (AST) | ✅ (controller) | ✅ (api) | ❌ | ❌ | ✅ (AST + 3-way) |
| 多框架 | 声称多/实际 2 | K8s 专用 | go-zero | kratos | N/A | gin/grpc + 插件扩展 |
| 声明式 plan/apply | ❌ | 部分 | ❌ | ❌ | ❌ | ✅ |
| Drift 检测 | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ (hash) |
| 3-way merge | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Feature 插件 | ❌ (硬编码) | CRD ext | ❌ | ❌ | hooks | ✅ (注册式) |
| 模板 snapshot 测试 | ❌ | 部分 | ❌ | ❌ | ❌ | ✅ |
| 二进制 size | ~20MB | ~35MB | ~25MB | ~30MB | N/A (Python) | 目标 ~10MB |
| 学习曲线 | 中 | 高 | 低 | 中 | 低 | 低 |

**关键差异化**：linctl 是唯一一个**同时**实现"声明式 + plan/apply + drift 检测 + 3-way merge"的 Go 脚手架。这套组合拳让"长期演进"成为现实。

## 0.6 项目命名说明

- **`linctl`**：取自项目目录 `lin/`（见 `/Users/forest/code/backend/Go/osbuilder-demo/lin`），后缀 `-ctl` 借鉴 `kubectl`/`osbuilder`/`mbctl` 的命名风格。
- 二进制名：`linctl`
- Go module 路径建议：`github.com/<org>/linctl`
- 配置文件名：`linctl.yaml`（项目根目录）
- 项目状态文件名：`PROJECT`（兼容 osbuilder 的命名约定，便于潜在用户迁移）

> **可选别名**：`forge`（锻造，象征"打造代码"）、`craft`（工艺）。最终命名以团队投票为准，但本文档统一使用 `linctl`。

## 0.7 设计哲学（北极星指标）

linctl 的每一个设计决策都围绕以下 5 个原则：

### 1. 声明式 over 命令式

- 用户描述"要什么"，工具决定"怎么做"。
- 配置即源（`linctl.yaml` 是单一真相源）。

### 2. 可观测 over 黑盒

- 任何变更都先 `plan` 给出可读 diff，再 `apply` 执行。
- 失败时打印**完整上下文**（哪个模板、哪行、什么数据）。

### 3. 可演进 over 一次性

- 工具升级不能破坏既有项目。
- 用户改过的文件优先级 > 模板更新。

### 4. 简单 over 全能

- 拒绝引入"为以防万一"的依赖（如 k8s 全家桶）。
- 拒绝在脚手架里塞 sysload/cleanupzombies 等无关功能。
- 用插件解决"长尾需求"。

### 5. UX 第一

- 错误信息必须是**人类可读 + 行动可执行**。
- 命令输出带颜色、emoji（可禁用）、对齐表格。
- 生成完成后打印"下一步"命令清单。

## 0.8 非目标（明确不做的事）

避免范围蔓延，明确以下不在 linctl 范围内：

| 非目标 | 替代方案 |
| --- | --- |
| 通用项目管理（任务/wiki/issue） | GitHub/Linear/Jira |
| CI/CD 编排 | GitHub Actions/GitLab CI/ArgoCD |
| 集群运维（部署/监控/日志聚合） | kubectl/ArgoCD/Datadog |
| 数据库迁移工具 | golang-migrate/atlas |
| Code review/lint runner | golangci-lint/reviewdog |
| 性能 profiling/调试工具 | pprof/dlv |
| 包/依赖管理 | go mod / dependabot |
| Semver/Changelog 生成 | 独立工具 release-please/svu（但 linctl 可生成模板让用户使用它们） |

> ⚠️ osbuilder 把 `semver / addlicense / sysload / cleanupzombies` 都塞在内部，linctl **不会**。这是有意识的取舍。

## 0.9 文档结构提示

读完本总览后，建议按以下路径深入：

- **想理解整体架构** → [01-architecture.md](./01-architecture.md)
- **想看完整的 CLI 命令列表** → [03-cli-design.md](./03-cli-design.md)
- **想看 Project Schema 怎么写** → [04-config-schema.md](./04-config-schema.md)
- **想参与开发/扩展** → [02-project-structure.md](./02-project-structure.md) → [08-feature-system.md](./08-feature-system.md)
- **想看实施时间表** → [11-implementation-plan.md](./11-implementation-plan.md)

---

下一步阅读：[01-architecture.md](./01-architecture.md)

[^1]: **文件数估算口径**（与 [README.md `linctl new` 输出示例](./README.md#快速预览mvp-用户体验) 一致；详见 [META 决策书 §1.4](./META-fix-decisions-2026-04-25.md#14-生成文件数)）：以 `Σ(component_i × pair_per_component_i) + project_level_files` 估算。

    - **最小示例**（gin-only + memory-store + healthz，1 个 WebServer）：约 **32** 个文件（cmd/*+ internal/server/* + healthz handler + Makefile/Dockerfile/.golangci/go.mod 等项目级文件）。
    - **全栈 Feature 启用**（gin + grpc-gateway + gorm-postgres + opentelemetry + user + websocket + preloader，含 docker/k8s/systemd 部署模板）：**250–350** 个文件，因 Feature 数 × 资源种数（kinds）线性增长。
    - 数字会随模板演进（增减 Feature / Component）调整，但口径保持 `min ≈ 32 / full ≈ 250–350` 不变。

_Last reviewed: 2026-04-25_
