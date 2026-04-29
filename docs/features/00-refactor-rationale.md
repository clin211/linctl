# 00. lin 重构 RFC：从「项目生命周期平台」回归「项目骨架生成器」

> **状态**：Stable（决策已确定 · 详见 §10）
> **作者**：Architect Review
> **创建日期**：2026-04-29
> **关联文档**：[01-architecture-blueprint.md](./01-architecture-blueprint.md)、[02-command-set.md](./02-command-set.md)、[lin/docs/00-overview.md](../00-overview.md)（旧版总览）

---

## TL;DR（一页摘要）

`linctl` 当前的设计在对标 **osbuilder + Terraform + kubebuilder** 的合体，已积累约 **10,525 行**生产代码、**14 个内部模块**、**15+ 个子命令**。

但项目的**真实需求**只有两条：

1. 一键生成符合 [`miniblog-v4`](../../../miniblog-v4) 风格分层架构的 Go 服务骨架；
2. 在已有项目中快速生成一个**业务资源（Resource）**的全套分层文件，让开发者可以直接填充业务逻辑。

这两件事不需要：声明式 `plan/apply` 闭环、drift 检测、3-way merge、模板上游同步、复杂的 Feature 插件机制。

**重构核心动作**：以「**简单 over 全能**」为最高原则，砍掉 ~50-70% 的代码与子命令，把 `lin` 重新定位为**纯粹的、一次性的、约定优于配置的 Go 项目骨架生成器**。

---

## 1. 现状评估

### 1.1 当前代码量分布（`lin/internal/`）

```
internal/templatesync/    2,352 行   ← 模板上游同步、缓存、清单、锁文件
internal/cli/             1,776 行   ← 15+ 子命令
internal/component/       1,190 行   ← 三种 Component 抽象
internal/project/         1,057 行   ← 配置加载/状态管理
internal/template/          807 行   ← 模板渲染
internal/codegen/           565 行   ← plan/apply 流水线
internal/gitmerge/          550 行   ← 3-way merge
internal/validate/          435 行   ← schema 与项目校验
internal/ast/               411 行   ← Go AST(dst) 注入
internal/fs/                410 行   ← 文件操作 / hash
internal/feature/           350 行   ← Feature 插件系统
internal/orchestrator/      271 行   ← 命令编排
internal/linctlerr/         265 行   ← 错误码
internal/version/            86 行   ← 版本
─────────────────────────────────
总计：                  ~10,525 行
```

### 1.2 子命令分布（`lin/docs/03-cli-design.md`）

```
linctl
├── new                    ✅ 必需（项目骨架生成）
├── add ...                ✅ 必需（资源/组件追加）
├── plan                   ❌ 过度（演进闭环）
├── apply                  ❌ 过度（演进闭环）
├── lint                   ⚠️ 可保留（仅 yaml 校验）
├── doctor                 ⚠️ 可保留（环境检测）
├── import                 ❌ 过度（反向工程）
├── upgrade                ❌ 过度（schema 升级）
├── completion             ✅ 标准命令
├── plugin                 ❌ 过度（插件生态）
├── version                ✅ 标准命令
├── help                   ✅ cobra 自带
└── options                ✅ cobra 自带
```

### 1.3 设计哲学的偏离

`lin/docs/00-overview.md` 中宣称的核心价值主张是：

| 当前定位 | 实际需要 | 偏离度 |
| --- | --- | --- |
| 「声明式、可计划、可回滚、插件化」的脚手架 | 「一次性、约定式」的脚手架 | 高 |
| `plan/apply` 闭环（Terraform 风格） | 直接生成 | 高 |
| Drift 检测（hash 追踪每个文件） | 不需要 | 高 |
| 3-way merge（模板升级合并用户改动） | 不需要 | 高 |
| Feature 插件注册中心 | 内置几个常用变体即可 | 中 |
| AST 注入（自动改 `biz.go`/`store.go`） | 看注册策略而定 | 中 |
| 模板上游同步（manifest/lockfile） | 不需要 | 高 |

> **本质判断**：当前 lin 在做一个**项目生命周期管理平台**，但需求只是一个**项目骨架生成器**。差的不是能力，是**定位**。

---

## 2. 重构动机

### 2.1 复杂度的代价

1. **维护成本指数级上升**
   - 14 个模块，跨模块依赖复杂，新功能加进去要触动多处。
   - 仅 `templatesync` 一个模块就 2,352 行（占 22%），却完全不在用户的核心需求路径上。

2. **学习曲线陡峭**
   - 用户要看懂当前 lin 需要理解：plan/apply 闭环、Pair 抽象、Feature 注册、AST 注入、3-way merge、Hook 安全策略……
   - 新人开发者上手成本远超 `goctl` / `kratos new`。

3. **"为以防万一"的依赖**
   - `dst`（AST 注入）、`protocompile`（Proto AST）、`validator/v10`（schema 校验）等。
   - 大量依赖只服务于 ~10% 的高级场景。

4. **测试覆盖压力**
   - 当前 `templatesync` 已有 5 个 `*_test.go` 文件，越复杂的功能测试维护成本越高。

### 2.2 简化的收益

| 维度 | 重构前 | 重构后（目标） | 收益 |
| --- | --- | --- | --- |
| 代码行数 | ~10,525 | ~3,000-4,000 | **-65%** |
| 内部模块数 | 14 | 5-7 | **-50%** |
| 子命令数 | 15+ | 3-5 | **-70%** |
| 二进制大小 | ~12 MB（目标） | ~5-6 MB | **-50%** |
| 直接依赖数 | ~13 | ~6-8 | **-40%** |
| 新人上手 | 3-5 天 | 0.5-1 天 | **大幅降低** |

---

## 3. 重新定位：lin 是什么 / 不是什么

### 3.1 一句话定位（重写）

> **`lin` 是一款**专注于 Go 后端服务的极简骨架生成器**，遵循 `miniblog-v4` 的分层架构（cmd / internal/{app}/{handler,biz,store,model,pkg} / internal/pkg / pkg / api）。**
>
> **它的使命是：让开发者在 30 秒内拿到一个可编译运行的项目骨架，在 5 秒内为既有项目加一个完整分层的业务资源。然后，工具退出舞台，把项目交还给开发者。**

### 3.2 核心价值主张（重写）

```
🚀 30 秒生成可编译的项目骨架
🧱 一行命令生成符合分层规范的业务资源
📐 强约定、零配置、零负担
🪶 极小依赖，单二进制 < 6 MB
```

### 3.3 是什么（Goals）

- ✅ 生成**单一 miniblog-v4 风格**的项目骨架。
- ✅ 在已有项目中追加**业务资源**（handler + biz + store + model + 可选 pkg/conversion + pkg/validation）。
- ✅ 通过 flag 支持几个**主流变体**（如 `--storage=postgres|mysql|mongo|memory`、`--with-otel`、`--with-grpc`）。
- ✅ 可读性强、错误信息清晰、生成完打印「下一步」清单。

### 3.4 不是什么（Non-Goals，明确边界）

| 不做 | 替代方案 |
| --- | --- |
| 声明式 `plan/apply` 闭环 | 直接生成；不可逆 |
| Drift 检测 / hash 追踪 | 生成完即结束，不再追踪 |
| 3-way merge | 模板升级要么覆盖、要么不动 |
| 模板上游同步 / 远程模板拉取 | 模板全部 `embed.FS` 内置 |
| 项目元数据状态文件（`PROJECT` / `linctl.yaml`） | 生成参数随命令传入，不持久化 |
| Feature 插件注册中心 | 通过 flag 选择变体（如 `--storage=postgres`） |
| 任意 Web/RPC 框架支持 | MVP 只支持 Gin（与 miniblog-v4 一致），未来再加 |
| 任意持久化方案支持 | MVP 支持 PostgreSQL+GORM（参考 miniblog-v4），未来再加 |
| 升级现有项目 | 不做 |
| 反向工程 / `import` | 不做 |
| Plugin 机制 | 不做 |

> 上述「不做」是**明确的设计取舍**，不是「以后再做」。如未来需求出现，应**重新评估**而非默认演进。

---

## 4. 重构后的形态预览

### 4.1 命令集（最小集）

```
lin
├── new <project-name>               # 生成项目骨架
├── add <resource-name>              # 添加业务资源
├── version                          # 版本信息
└── completion <shell>               # shell 补全
```

仅此而已。

### 4.2 模块裁剪表

| 模块 | 当前 | 处置 | 理由 |
| --- | --- | --- | --- |
| `templatesync` | 2,352 行 | **🗑️ 删除** | 不做模板同步 |
| `gitmerge` | 550 行 | **🗑️ 删除** | 不做 merge |
| `orchestrator` | 271 行 | **🗑️ 删除** | 直接命令调用即可 |
| `codegen` (plan/apply 部分) | ~400 行 | **🗑️ 删除** | 不做闭环 |
| `feature` | 350 行 | **⚠️ 大幅简化** | 改为 flag-based 变体 |
| `component` (三种抽象) | 1,190 行 | **⚠️ 简化** | MVP 只保留 webserver 风格 |
| `validate` | 435 行 | **⚠️ 简化** | 仅做基本输入校验 |
| `project` | 1,057 行 | **⚠️ 简化** | 生成参数随命令传入，不持久化 |
| `ast` | 411 行 | **🟨 待定** | 取决于「注册策略」决策（见 §5.2） |
| `template` | 807 行 | **✅ 保留** | 核心 |
| `cli` | 1,776 行 | **✅ 简化** | 命令数减少，cli 自然瘦身 |
| `fs` | 410 行 | **✅ 保留** | 核心 |
| `linctlerr` | 265 行 | **✅ 保留** | 核心 |

**预期裁剪后结构**：

```
lin/
├── cmd/lin/main.go
├── internal/
│   ├── cli/                      # cobra 命令分发
│   │   ├── root.go
│   │   ├── new.go                # lin new 命令
│   │   ├── add.go                # lin add 命令
│   │   └── version.go
│   ├── scaffold/                 # 骨架生成核心
│   │   ├── project.go            # 项目骨架生成器
│   │   ├── resource.go           # 资源骨架生成器
│   │   └── render.go             # 模板渲染封装
│   ├── templates/                # embed.FS 模板源
│   │   ├── project/              # miniblog-v4 风格项目骨架
│   │   └── resource/             # 资源骨架（handler/biz/store/model/...）
│   ├── pkg/
│   │   ├── prompt/               # 交互式 UI（可选）
│   │   ├── fs/                   # 文件操作
│   │   └── errs/                 # 错误处理
│   └── version/
└── tools/
```

模块数：**5-7 个**（vs 当前 14 个）。

---

## 5. 关键决策点（必须用户确认）

> 以下决策**架构层面影响很大**，定下来后才能写后续详细设计。每条都给出**方案对比 + 我的建议**。

### 5.1 ⭐ 工具边界：是否完全砍掉演进能力？

**问题**：是否保留 `plan/apply`、drift 检测、template 同步等"项目演进"功能？

| 方案 | 含义 | 代码量 | 利 | 弊 |
| --- | --- | --- | --- | --- |
| **A. 极简骨架生成器**（推荐） | 只 `new`/`add`/`version`/`completion`，无任何演进能力 | ~3,500 行 | 极简，零负担，符合用户本意 | 模板升级时旧项目不会自动更新 |
| **B. 保留 `lint`/`doctor` 辅助命令** | 在 A 基础上加只读校验/环境检测 | ~4,000 行 | 略丰富，运维友好 | 增加边际复杂度 |
| **C. 保留 `plan`/`apply` 闭环** | 删除 sync/merge/drift，但保留 plan/apply | ~6,000 行 | 仍可声明式管理 | 复杂度仍偏高 |

**我的建议**：**A**。理由：
- 用户的本意非常明确，不需要演进闭环。
- `lint`/`doctor` 可以作为 Phase 2 增强，MVP 先不做。
- 一旦保留 plan/apply，drift/hash 追踪几乎不可避免地会被加回来。

### 5.2 ⭐⭐ 资源注册策略：`add resource` 后如何让骨架"被使用"？

**背景**：生成 `internal/apiserver/biz/post/post.go` 后，需要在某处把它注册到路由 / DI 容器 / 接口集合，否则只是孤立的文件。

**方案对比**：

| 方案 | 注册方式 | 利 | 弊 |
| --- | --- | --- | --- |
| **A. 约定优于配置** | 每个资源在 `init()` 自动注册到全局 registry；router 启动时遍历注册表 | 新文件不修改任何已有文件，零冲突；扩展性好 | "魔法"风格，不显式；调试链路稍复杂 |
| **B. AST 注入** | 自动修改 `biz.go`/`store.go`/`router.go` 添加方法/路由 | 显式、Go-style | 复杂、易碎、调试难；保留 `ast` 模块（411 行 + 测试） |
| **C. TODO 提示** | 生成完打印需要手工添加的代码片段，开发者复制粘贴 | 最简单、最透明、零侵入 | 用户多一步手工操作 |
| **D. wire/fx DI** | 生成 `wire_gen.go`，开发者跑 `make wire` | 标准的 Go DI | 引入 wire 依赖、首次配置成本 |

**我的建议**：**A + C 混合**。
- 核心层（handler/biz/store）使用 **A 约定式注册**，做到零冲突追加。
- 边缘文件（如 errno、validation 注册）使用 **C TODO 提示**，避免黑魔法过多。
- **不要 B（AST 注入）**：`miniblog-v4` 当前的注册风格已经是显式的 `biz.User = NewUserBiz()` 形式，AST 改这种代码很容易破坏作者风格。

### 5.3 资源生成的"完整度"

**问题**：`lin add post` 应该生成多少个文件？

| 方案 | 文件层级 | 文件数 | 适用场景 |
| --- | --- | --- | --- |
| **A. 全栈** | handler + biz + store + model + conversion + validation + errno + proto | 7-9 | 完整复刻 miniblog-v4 |
| **B. 核心** | handler + biz + store + model | 4 | 最小可运行 |
| **C. flag 控制** | 默认 B，`--with conversion,validation,proto` 添加 | 4-9 | 灵活 |

**我的建议**：**C**。默认核心 4 个文件（最小成本上手），通过 flag 按需追加。

### 5.4 模板可定制性

**问题**：用户能否自定义模板？

| 方案 | 含义 | 适用 |
| --- | --- | --- |
| **A. 完全 embed** | 所有模板内置在二进制，无法外部覆盖 | MVP（推荐） |
| **B. 外部目录覆盖** | 支持 `--template-dir=./my-templates` 完全替换 | 团队定制场景 |
| **C. 二者兼有** | 默认 embed，可 override | 最灵活但复杂 |

**我的建议**：**A**。MVP 先 embed，未来确有团队定制需求时再加 B。

### 5.5 Feature 系统：是否完全删除？

**问题**：当前 `internal/feature/` 是 Feature 注册中心。是否保留？

| 方案 | 含义 |
| --- | --- |
| **A. 完全删除**，改为 flag 选择变体 | 如 `--storage=postgres`、`--with-otel`、`--with-grpc-gateway` |
| **B. 保留但简化**，仅作为模板分组的内部抽象 | 用户感知不到 feature 概念 |
| **C. 保留完整 feature 系统** | 用户可在 yaml 声明 features |

**我的建议**：**B**。在内部实现上保留模板分组（避免模板写得到处是 if/else），但**用户感知层只有 flag**，没有 yaml 配置文件。

### 5.6 是否保留 `linctl.yaml` 配置文件？

**问题**：当前用户可在项目根目录写 `linctl.yaml` 声明 components/features。重构后是否保留？

| 方案 | 含义 |
| --- | --- |
| **A. 完全删除** | 所有参数随 `lin new` / `lin add` 命令传入 |
| **B. 保留但作为「初始化记忆」** | `lin new` 时记录用户选择，`lin add` 时复用（如 module path、storage 类型） |
| **C. 保留完整 schema** | 类似当前 |

**我的建议**：**B**。生成项目时写一个**极简 `lin.yaml`**，只记录:

```yaml
module: github.com/foo/myblog
appName: myblog
framework: gin
storage: gorm-postgres
features: [otel, healthz]
```

这样 `lin add post` 不需要重复传一堆 flag，体验更好。**但这个 yaml 不是状态文件，只是参数记忆**。

---

## 6. 风险与权衡

| 风险 | 严重度 | 缓解 |
| --- | --- | --- |
| 用户后续真的需要演进能力（如 add feature） | 中 | 用户本意明确否定；如真出现，作为 Phase 2+ 重新设计，避免提前过度抽象 |
| 模板升级时已生成项目无法跟进 | 低 | 不是工具的责任；用户可自行 diff 模板和已有代码 |
| 删除模块带来的破坏性变更 | 高 | 通过 git 分支保留旧版本；新版本作为 v2 发布 |
| 约定式注册的"魔法感" | 中 | 文档清晰说明；通过测试 + 启动日志增加可观测性 |

---

## 7. 不在本 RFC 范围内的事项

- 模板的具体内容（例如 `Makefile.tpl` 应该长什么样）→ 详见 [03-resource-scaffold.md](./03-resource-scaffold.md) / [04-template-system.md](./04-template-system.md)
- 详细的迁移步骤 → 详见 [06-migration-plan.md](./06-migration-plan.md)
- 删除哪些文件、保留哪些文件的清单 → 详见 [06-migration-plan.md §3](./06-migration-plan.md#3-删除清单)
- 单测策略 → MVP 不强制；保留 `template`/`scaffold` 的关键单测

---

## 8. 与现有文档的关系

| 现有文档 | 处置 |
| --- | --- |
| `lin/docs/00-overview.md` ~ `15-security-model.md` | **归档**到 `lin/docs/legacy/`，作为旧版设计参考 |
| `lin/docs/META-*.md` | **归档** |
| `lin/docs/adr/` | **保留**（ADR 是历史决策记录，永远有价值） |
| `lin/docs/diagrams/` | 部分保留（`architecture-overall`、`seq-new-project`），其余归档 |
| `lin/docs/features/` | **新设计文档目录**，承载本次重构后的设计 |

---

## 9. 后续文档计划

```
lin/docs/features/
├── 00-refactor-rationale.md       ✅ 当前文档（RFC）
├── 01-architecture-blueprint.md   ✅ 重构后的整体架构
├── 02-command-set.md              ✅ 6 个命令详细设计（new/add/lint/doctor/version/completion）
├── 03-resource-scaffold.md        ✅ 资源分层（参照 miniblog-v4）
├── 04-template-system.md          ✅ 简化的模板系统
├── 05-registration-strategy.md    ✅ 资源注册策略（AST 注入）
├── 06-migration-plan.md           ✅ 从当前 lin → 新版的迁移步骤
├── 07-interactive-ux.md           ✅ 交互式终端 UX（lin new / lin add 向导）
└── README.md                       ✅ 索引
```

---

## 10. 用户决策（已确定 · 2026-04-29）

| 决策点 | 选择 | 含义 |
| --- | --- | --- |
| §5.1 工具边界 | **B. 极简 + lint/doctor** | 命令集：`new` / `add` / `lint` / `doctor` / `version` / `completion`；保留运维辅助命令；删除 plan/apply/templatesync/gitmerge |
| §5.2 资源注册策略 | **B. AST 注入** | 保留 `internal/ast/`（411 行）；自动改 `biz.go` / `store.go` / `*.proto` / `errno/` 等中央文件 |
| §5.3 资源完整度 | **A. 全栈** | 默认生成 7-9 个文件（handler+biz+store+model+conversion+validation+errno+proto） |
| §5.4 模板可定制性 | **B. 外部目录覆盖** | 支持 `--template-dir` 完全替换；定义清晰的目录约定 |
| §5.5 Feature 系统 | **A. 完全删除** | `internal/feature/` 整体移除；模板用 `{{if eq .Storage "postgres"}}` 处理变体 |
| §5.6 配置文件 | **A. 完全删除** | 不保留 yaml；元数据从 `go.mod` + `cmd/` 目录结构推断 |

### 10.1 决策的整体定位

```
                    ┌────────────────────────────────┐
                    │ "功能完备 · 但单一聚焦" 的脚手架 │
                    └────────────────────────────────┘
                                   ↓
              ┌────────────────────┴────────────────────┐
              ↓                                          ↓
     ✅ 保留的能力                              ❌ 砍掉的能力
     ─────────────────                         ─────────────────
     • 项目骨架生成                           • plan/apply 闭环
     • 全栈资源生成                           • drift / hash 追踪
     • AST 注入到中央文件                     • 3-way merge
     • lint / doctor 校验                     • 模板上游同步
     • 外部模板目录覆盖                       • Feature 注册中心
     • completion 补全                        • Plugin 机制
                                              • 配置 yaml 文件
```

类比定位：更像 **`goctl` + `kubebuilder` 的混合**，但**没有 reconcile 闭环**。

### 10.2 决策派生的设计约束（重要）

由「§5.6 删除配置文件 + §5.2 AST 注入」组合派生：

- AST 注入需要的元信息（module path / app name / resource name）必须从**项目现状**推断：
  - `module` ← 读 `go.mod`
  - `appName` ← `cmd/<app>/` 单一子目录推断；多个则要求 `--app` flag
  - `resourceName` ← 命令参数 `lin add <Name>`
- `lint` 命令在没有 yaml 的前提下，仅校验**目录结构 + AST 注入完整性 + import 正确性**，不做 schema 校验。

由「§5.4 外部模板 + §5.5 删除 feature」组合派生：

- 模板内通过 `{{if eq .Storage "postgres"}}` 等条件分支处理变体，**不再分组到 feature/ 目录**。
- 外部 `--template-dir` 必须遵守与 embed 同样的目录约定（在 [04-template-system.md](./04-template-system.md) 详述）。

由「§5.3 全栈 + §5.2 AST 注入」组合派生：

- `lin add post` 一次操作会**触动 ~12-15 个文件**：
  - 创建 7-9 个文件（handler/biz/store/model/conversion/validation/errno/proto/...）
  - AST 注入修改 4-5 个中央文件（`biz.go` / `store.go` / `<app>.proto` / `errno` / 路由注册）
- 这要求 `lin add` 必须有强幂等性（重复执行不破坏现有代码）。

> 决策已固化，后续详细文档（[01](./01-architecture-blueprint.md) ~ [06](./06-migration-plan.md)）基于此展开。

---

_Last reviewed: 2026-04-29_
