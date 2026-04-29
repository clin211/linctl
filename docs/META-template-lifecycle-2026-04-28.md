# META · 模板生命周期管理设计书 (Template Lifecycle Management)

**日期**：2026-04-28
**目的**：锁定"生成项目的可持续演进"机制，解决模板升级 / 迭代 / 维护的复杂度
**适用范围**：`internal/codegen/`、`internal/template/`、`internal/project/`、`internal/cli/`、`internal/fs/`、新增 `internal/lockfile/` 与 `internal/template/catalog/`、`internal/template/manifest/`
**状态**：Proposed（待 review；review 通过后转 Accepted，并以本文件为后续 ADR-006/007/008/009 的索引）
**前置阅读**：[ADR-004 plan/apply pattern](./adr/004-plan-apply-pattern.md)、[06-codegen-pipeline.md](./06-codegen-pipeline.md)、[META-fix-decisions-2026-04-25.md](./META-fix-decisions-2026-04-25.md)

---

## 0. 阅读路径

| 你想…… | 看哪一节 |
| --- | --- |
| 看为什么要做这件事 | [§1 问题陈述](#1-问题陈述) |
| 看要解决什么 / 不解决什么 | [§2 设计目标与非目标](#2-设计目标与非目标) |
| 看与现有设计如何衔接 | [§3 与既有 ADR / 文档的关系](#3-与既有-adr--文档的关系) |
| 看核心抽象长什么样 | [§4 核心抽象（四件套）](#4-核心抽象四件套) |
| 看用户最终会怎么用 | [§5 端到端用户场景](#5-端到端用户场景) |
| 看怎么落地、什么时候做 | [§6 实施路线图（4 阶段）](#6-实施路线图4-阶段) |
| 看要新增 / 修改哪些文档 | [§7 文档对接清单](#7-文档对接清单) |
| 看风险与不做的事 | [§8 风险、回滚与非目标边界](#8-风险回滚与非目标边界) |
| 看怎么算"完成" | [§9 验收标准（DoD）](#9-验收标准dod) |
| 看变更历史 | [§10 修订历史](#10-修订历史) |

---

## 1. 问题陈述

### 1.1 现状速览

> 数据采集时间：2026-04-28，对应 `lin/` 仓库 HEAD。

| 维度 | 当前状态 | 证据位置 |
| --- | --- | --- |
| 模板总数 | 226 个 `.tpl` + 29 个 raw passthrough = **255 个文件** | `lin/internal/template/templates/` |
| 模板嵌入方式 | `go:embed all:templates` 编入二进制 | `lin/internal/template/embed.go` |
| Pair 清单维护方式 | 手写在 Component 实现中 | `lin/internal/component/webserver.go` 共 720 行 |
| Plan / Apply 实施层级 | **Tier 1**（仅 Create / Update / Skip） | `lin/internal/codegen/plan.go:17-25` |
| 写入语义 | "**Update 路径退化为「直接覆盖」**" | `lin/internal/codegen/applier.go:18-19` 注释 |
| 项目元数据 | 无 `linctl.yaml` / 无 `.linctl/` | `lin/internal/cli/cmd_new.go` 不写任何元数据 |
| sync / upgrade 命令 | 不存在 | `lin/internal/cli/` 仅 `cmd_new.go` + `cmd_add.go` |
| Catalog 抽象 | 不存在（模板与二进制 1:1 绑定） | `lin/internal/template/embed.go` |

### 1.2 七大具体痛点（编号便于后续追踪）

| 编号 | 痛点 | 具体危害 | 根因 |
| --- | --- | --- | --- |
| **L-P1** | 生成项目无身份标识 | 无法 replay、无法 diff、无法 upgrade | `linctl new` 不落 spec / lock |
| **L-P2** | Update 等于无脑覆盖 | 用户写在 `biz/v1/user/login.go` 的业务逻辑被毁 | `applier.go` 直接 AtomicWrite |
| **L-P3** | `WriteMode` 三态没生效 | 已定义 `WriteModeOnce` / `WriteModeAppend` 但 `planner.go` 完全没用 | `pair.go:17-28` 与 `planner.go:91-117` 脱节 |
| **L-P4** | 模板与二进制版本强耦合 | 改一个模板字符就要发新 lin | `go:embed` 单源 |
| **L-P5** | Pair 清单手写不可扩展 | `webserver.go` 每加 1 文件就要改 1 行 Go 代码并 `go build` | `BasePairs()` 是命令式手写函数 |
| **L-P6** | 无 sync / upgrade / diff 命令 | 老项目永远停留在生成时刻的版本 | CLI 命令集只覆盖 "create" |
| **L-P7** | 无迁移声明能力 | 文件改名 / 删除 / 拆分等 breaking change 无表达手段 | 缺 `migrations/` 概念 |

### 1.3 不做的代价

继续保持现状 6-12 个月后会出现的可观察后果：

1. 用 lin 生成的项目 **不敢二次跑** `linctl new`，因为会丢业务代码 → 用户停留在"用一次就扔"的脚手架心智，无法享受持续维护红利。
2. 模板修了 bug 但用户不能 sync → 大家挨个手抄 → fork 加速，社区分裂。
3. 想引入 gRPC + Gateway 的新模板 → `webserver.go` 已 720 行，再加 200 文件清单会失控。
4. 公司想用私有定制模板 → 只能 fork lin 仓库 → 永远跟不上上游。
5. 出现"我现在用的是 templates@哪个 commit?" 没有答案 → 无法做任何审计。

---

## 2. 设计目标与非目标

### 2.1 设计目标（按优先级排序）

> 每个目标都对应一组 [§1.2 痛点](#12-七大具体痛点编号便于后续追踪) 的解决，便于追溯。

| # | 目标 | 解决痛点 | 可验证 |
| --- | --- | --- | --- |
| **G1** | 每个 `linctl new` 出的项目都自带"身份证 + 体检表"（spec + lockfile） | L-P1 | 项目根存在 `linctl.yaml`（用户可读） + `.linctl/lock.json`（机器读） |
| **G2** | 三态写入语义落地，用户改过的代码绝不被静默覆盖 | L-P2 / L-P3 | 同一项目跑 2 次 `linctl new`：业务文件保持，骨架文件按需更新，冲突文件给出标准 git conflict markers |
| **G3** | `linctl sync` 命令把模板新版应用到老项目 | L-P6 | `cd myproj && linctl sync` 能在已有项目内复算 plan 并按 WriteMode 应用，使用 `git merge-file` 实现 3-way merge |
| **G4** | 模板版本与 lin 二进制可独立演进 + `~/.linctl/` 用户可定制 | L-P4 | `--catalog` 参数支持 embedded / local / git 三种来源；`linctl template init` 后用户可在 `~/.linctl/catalogs/` 改模板 |
| **G5** | Pair 清单从 720 行 Go 代码瘦身到 YAML manifest | L-P5 | `webserver.go` 减重至约 100 行；模板树根带 `MANIFEST.yaml` |
| **G6** | breaking change 可声明性表达（rename / delete / split） | L-P7 | 模板版本目录下含 `migrations/<from-ver>-<to-ver>.yaml` |
| **G7** | 全程 dry-run 友好、CI 可解析输出 | 横向支持 | `linctl status -o json` / `linctl sync --dry-run -o json` 可被 CI 消费 |
| **G8** | 细粒度 schematic 化生成器，单文件级到资源级到组件级到项目级分级 | L-P5 进一步深化 | `linctl g middleware <name>` / `linctl g resource <name>` / `linctl g webserver <name>` 三层分级可用 |
| **G9** | （远期）AutoUpdate 让 N 个项目"配置一次、自动跟上模板演进" | 长尾运营 | `linctl init-autoupdate` 后，模板新版发布会自动开 PR |

### 2.2 非目标（明确不做的事）

| # | 非目标 | 理由 |
| --- | --- | --- |
| **N1** | 不做项目级 git 集成（自动 commit / branch） | 用户的 vcs workflow 多样；linctl 只输出文件，git 是用户决定 |
| **N2** | 不做"AI 智能合并冲突" | 留给后续可选 Feature；先确保确定性算法（3-way merge）正确 |
| **N3** | 不做远程 lockfile 中央服务 | 增加运维成本；本地 lock 已足够；后续如有团队需求再做 |
| **N4** | 不引入新的 DSL 语言 | manifest 用 YAML、migration 用 YAML，避免学习成本 |
| **N5** | 本轮不重构 `internal/feature/` | Feature 系统已稳定（ADR-005），独立演进 |
| **N6** | 不做模板版本的"自动安全更新"（auto-merge） | 永远人工触发 sync，避免静默修改用户代码 |

---

## 3. 与既有 ADR / 文档的关系

> 这一节是关键：**本设计 90% 是在落地 ADR-004 已设计但 Tier 2/3 部分未实施的内容**，剩余 10% 是 Catalog + Manifest-Driven 两个新增点。

### 3.1 与 ADR-004 (Plan/Apply Pattern) 的对应关系

ADR-004 已经声明了 [§5 能力分级（Capability Tiers）](./adr/004-plan-apply-pattern.md#5-能力分级capability-tiers)：

| ADR-004 Tier | ADR-004 描述 | 本设计的处理 |
| --- | --- | --- |
| Tier 1 | Create / Update / Skip 三态，hash 直比 | ✅ 已实施（Phase 1） |
| Tier 2 | 引入 hash 注释 + lock.yaml 持久化 + Conflict Action + ask 策略 | 🎯 **本设计阶段 1+2 实施**（但用 lockfile 替代 inline hash 注释，详见 §4.1） |
| Tier 3 | --prune + 3-way merge + restore + backups retention + drift detection | 🎯 **本设计阶段 2 完整落地** |

**变更点**（相对 ADR-004 原文）：

- ADR-004 原方案是"在生成文件末尾加 `// linctl: hash=xxx` 注释" 实现 drift 检测。
- v0.3.x 已**废弃 inline hash 注释**（见 `applier.go:18-19`），原因是：① 噪音大；② 部分文件类型（json/二进制）无注释语法；③ 用户讨厌脏注释。
- 本设计**改用 `.linctl/lock.json` 集中存储 hash**：信息量等价，但代码文件保持纯净。
- 该决策需要写一份新的 **ADR-006: prefer-external-lockfile-over-inline-hash**（详见 §7.3）。

### 3.2 与 06-codegen-pipeline.md 的关系

`06-codegen-pipeline.md §6.2` 已绘制了 plan 流水线的 mermaid 图，其中 Decide 节点已包含：

```text
Create / Skip / Update / Conflict / WouldClaim / Delete
```

**结论**：流水线设计图早已 ready，本设计是把 Conflict / WouldClaim / Delete 三个未落地分支真正实现。

修改方式：在 §6 末尾**追加** `§6.16 模板生命周期与 Catalog`，不改图。

### 3.3 与 11-implementation-plan.md 的关系

11 文档原本规划的 Phase 4 Story 4.1-4.5（plan / apply / drift / merge / restore）正是本设计阶段 1-2。本设计把它们 **拆细 + 提前**：

- 把 Phase 4 的 Story 4.1-4.5 移到 **Phase 2.x**（提早交付，因为这是用户最痛的）
- Phase 4 改为 Catalog 抽象 + Manifest-Driven（更长期工作）

需要在 §11.4 重写 Phase 4 计划（详见 §7.2）。

### 3.4 与 META-roadmap.md 的关系

META-roadmap.md 现有的 Batch 4 计划：

- 18-release-governance.md
- 19-docs-site.md

本设计**不抢占** Batch 4 编号，使用：

- **新增 21-template-lifecycle.md**（生成项目侧的生命周期），承接 20-import-algorithm 之后
- **新增 ADR-006/007/008/009**（详见 §7.3）

### 3.5 业界标杆对比与借鉴

> 本节是本设计的"基准面"：先看清三个最优秀的 CLI 工具（goctl / Nest CLI / Kubebuilder）怎么解这个问题，再按"取长补短 + 适配我们的语境"原则裁剪。
> 调研日期：2026-04-28，三者均为 active maintenance、千万级用户、十年 + 演进。

#### 3.5.1 标杆能力矩阵

| 能力维度 | goctl (go-zero) | Nest CLI | Kubebuilder | linctl 现状 | linctl 目标 |
| --- | --- | --- | --- | --- | --- |
| **项目身份证** | 无（内嵌模板单源） | `nest-cli.json`（轻量） | `PROJECT` 文件（完整 SSOT） | ❌ 无 | ✅ `linctl.yaml` + `.linctl/lock.json`（学 Kubebuilder） |
| **模板可定制目录** | `~/.goctl/<version>/<category>/` | `node_modules/@nestjs/schematics`（包级） | 无（plugin 系统） | ❌ embed 单源 | ✅ `~/.linctl/catalogs/<id>@<ver>/`（学 goctl） |
| **远程模板源** | `--remote <git-repo> --branch <ref>` | `-c <package>` (npm) | 通过 plugin（Go module） | ❌ 无 | ✅ `--catalog git+https://...@<ref>`（融合三家） |
| **模板源优先级** | `--remote > --home > 内嵌` | `nest-cli.json.collection > -c flag` | plugin chain | ❌ 单源 | ✅ `--catalog flag > linctl.yaml.catalog > 内嵌` |
| **细粒度生成器** | `goctl api new` / `goctl model` / `goctl rpc` | **schematic**（细粒度，如 `nest g controller`） | plugin + scaffold action | ⚠️ 仅 `linctl new` / `linctl add api` | ✅ schematic 化（学 Nest）：见 §4.6 |
| **AST 自动注入** | 部分（model 字段同步） | ✅ 自动 update module.ts | ✅ 自动 update PROJECT + main.go | ⚠️ 仅 `linctl add api` 时改 biz.go/store.go | 已有，继续保持 |
| **升级机制** | `template update -c <category>`（覆盖式） | 无系统升级；用户重跑 `nest g` | ⭐ **`alpha update` 用 git 3-way merge** | ❌ 无 | ✅ **学 Kubebuilder：用 `git merge-file` 系统调用**（核心借鉴） |
| **冲突处理** | 用户自己 diff（手工） | 无（schematic 出错就报错） | ⭐ git conflict markers 提交（CI 友好） | ❌ 无 | ✅ git markers + `--strategy=ask\|abort\|force`（学 Kubebuilder） |
| **AutoUpdate** | 无 | 无 | ⭐ `autoupdate.kubebuilder.io/v1-alpha` plugin scaffolds GitHub Action | ❌ 无 | 🎯 远期：`linctl init-autoupdate`（学 Kubebuilder） |
| **dry-run** | ✅ | ✅ `-d`/`--dry-run` 一等公民 | ✅ | ✅（`--dry-run`） | 已有，继续保持 |
| **revert / 回滚** | ✅ `template revert <name>` | 无 | git 自身 | ❌ 无 | ✅ `linctl template revert <name>`（学 goctl） |
| **AI 集成** | 无 | 无 | ⭐ `--use-gh-models`（升级时 AI 总结 diff） | 无 | 🎯 远期 |

> ⭐ 表示该工具在该项上有"行业最优实践"，是 linctl 的重点借鉴对象。

#### 3.5.2 五个最重要的借鉴点

> 按"借鉴价值"降序排列。

##### 借鉴点 #1：Kubebuilder 用 git 自身做 3-way merge（颠覆性优化）

**Kubebuilder 的 `alpha update` 算法**：

```text
1. 创建三个临时分支：
     ancestor   = 旧版本模板 clean scaffold（用户当时生成的状态）
     upgrade    = 新版本模板 clean scaffold（升级后的目标）
     original   = 用户当前 main 分支（包含用户改动）

2. 让 git 自己做 3-way merge：
     git merge-tree ancestor upgrade original
     # 或者：先 checkout upgrade，然后 git merge --no-commit original
     #      失败时保留 <<<<<<< / ======= / >>>>>>> markers

3. 输出 reviewable branch：kubebuilder-update-from-v4.5.2-to-v4.6.0
   用户在该 branch 上 review、resolve conflicts、push PR。
```

**对 linctl 的指导意义**（替换原 §4.2 的"自写决策表 + BASE/OURS/THEIRS 三件"方案）：

- ❌ **原方案问题**：让用户读三个独立文件 + 手工 merge，UX 差、社区不熟悉
- ✅ **新方案**：直接用 `git merge-file` 子命令在工作区写入 conflict markers

  ```bash
  # linctl 内部调用，对每个 managed-mode 冲突文件：
  git merge-file -p --diff3 \
    <(echo "$ours")    \
    <(echo "$base")    \
    <(echo "$theirs")  \
    > .linctl/merged/server.go
  ```

- ✅ 用户看到的是熟悉的 `<<<<<<< ours / ======= / >>>>>>> theirs` 标准 conflict
- ✅ 任何 IDE（VSCode / Goland / Vim）都自带 conflict 解决 UI，零学习成本
- ✅ CI 模式下用 `--force`：把带 markers 的内容直接写盘提交，让人工 review

**取舍**：

| 方面 | 自写 merge | git merge-file |
| --- | --- | --- |
| 实现复杂度 | 高（要维护决策表 / 选择算法） | **低**（fork+exec git） |
| 用户认知 | 需学新概念（BASE/OURS/THEIRS 三件） | **零成本**（git markers 人人熟） |
| IDE 支持 | 无 | **完全有**（git conflict 解决 UI） |
| 跨平台 | OK | **OK**（git 在所有 dev 机器上都有） |
| 边界场景 | 容易漏 | **久经考验**（git 几十年磨炼） |

##### 借鉴点 #2：goctl 的 `~/.goctl/` 三段优先级模板源

**goctl 的优先级**：`--remote > --home > 内嵌`

```text
$ goctl api new myservice
   # 1. 看 --remote flag → git clone 临时目录
   # 2. 看 --home flag    → 用户指定路径
   # 3. 都没有 → 检查 ~/.goctl/<version>/api/ 是否存在
   #    存在 → 用它（用户已 init 过）
   #    不存在 → 用二进制内嵌 fallback
```

**对 linctl 的指导意义**：

- ✅ 让用户**渐进式定制**：先用默认 → 跑 `linctl template init` 复制到 `~/.linctl/templates/` → 改本地 → 单文件 `linctl template revert`
- ✅ 多版本并存：`~/.linctl/catalogs/embedded@v0.3.0/` 与 `embedded@v0.4.0/` 互不影响
- ✅ 公司私有模板：`linctl new --catalog git+ssh://github.com/myco/lin-templates@v2.1.0`
- ✅ 离线友好：~/.linctl 缓存命中即可工作

##### 借鉴点 #3：Nest CLI 的细粒度 Schematic（解放 `linctl new`）

**Nest CLI 的设计**：

```bash
nest new myapp                    # 整个项目（粗粒度）
nest g resource users --crud      # 5 个文件（中粒度）
nest g controller users           # 1 个文件（细粒度）
nest g service users              # 1 个文件（细粒度）
nest g middleware auth            # 1 个文件（最细粒度）
```

每个 schematic 由 `schema.json` 声明参数 + 对应模板 + 注入代码：

```json
// @nestjs/schematics/src/controller/schema.json
{
  "properties": {
    "name": {"type": "string"},
    "skipImport": {"type": "boolean"},
    "spec": {"type": "boolean"}
  }
}
```

**对 linctl 的指导意义**：

- ✅ **拆分 `linctl new`**：现在它是巨型函数，应拆为：
  - `linctl init` → 创建项目骨架（go.mod / Makefile / configs）
  - `linctl g webserver <name>` → 加一个 WebServer 组件
  - `linctl g resource <name>` → 加一个 CRUD 资源（已有 `linctl add api`，重命名）
  - `linctl g controller <name>` / `linctl g middleware <name>` → 单文件级生成
- ✅ 每个 schematic 自带 `schema.json` 描述参数（CLI flags 自动派生）
- ✅ schematic 之间可继承：`g resource` 内部组合 `g controller` + `g service`

> 这是 §4.6 新章节的核心思路。

##### 借鉴点 #4：Kubebuilder 的 PROJECT 文件 SSOT 模式

**Kubebuilder 的 PROJECT 文件**：

```yaml
# PROJECT
domain: example.com
layout:
  - go.kubebuilder.io/v4    # 当前 plugin
cliVersion: v4.6.0           # 生成时的 CLI 版本
projectName: myproject
repo: github.com/example/myproject
resources:
  - api:
      crdVersion: v1
    controller: true
    domain: example.com
    group: webapp
    kind: Guestbook
    version: v1
```

**关键洞察**：

- 一个文件 = 项目的全部决策
- `cliVersion` 字段是升级算法的"from-version"输入
- `resources` 数组让 `kubebuilder alpha generate` 能 replay 全部 `kubebuilder create api` 命令

**对 linctl 的指导意义**：

- ✅ `linctl.yaml` 必须含 `cliVersion` + `catalog` + `components[]`（已有 spec 设计，强化即可）
- ✅ 增加 `linctl.yaml.history`（操作历史）：

  ```yaml
  spec:
    components: [...]
  history:
    - at: 2026-04-28T13:45:00Z
      cmd: linctl init myblog --module github.com/foo/myblog
      cliVersion: v0.3.0
    - at: 2026-04-29T10:00:00Z
      cmd: linctl g resource Post
      cliVersion: v0.3.0
  ```

- ✅ replay 能力：`linctl init --from-yaml linctl.yaml` 可重建项目

##### 借鉴点 #5：Kubebuilder AutoUpdate Plugin（远期）

**Kubebuilder 的 AutoUpdate**：scaffold 一个 GitHub Action，定时检查上游新版本，自动开 PR。

**对 linctl 的指导意义**（远期 Phase 5+）：

- 🎯 提供 `linctl init-autoupdate` 命令，scaffold 一个 `.github/workflows/linctl-update.yml`：

  ```yaml
  on: { schedule: [{cron: '0 0 * * 0'}] }
  jobs:
    update:
      steps:
        - uses: clin211/linctl-update-action@v1
        # 内部跑 linctl sync --catalog-version=latest
        # 有更新就开 PR
  ```

- 🎯 让企业用户的 N 个项目"一次配置、永久跟上模板演进"

#### 3.5.3 不借鉴 / 反向取舍的点

| 对方做法 | 不借鉴的原因 |
| --- | --- |
| **Nest CLI 用 npm 分发 schematic** | linctl 是 Go 生态，应走 Go module 或 git URL，不绑定 npm |
| **goctl 全量覆盖式 `template update`** | 太粗暴，会丢用户改动；改用 sync + git merge |
| **Kubebuilder 的 plugin 体系（编译期注册）** | 太重；linctl 先用 catalog（数据型）做扩展点，未来若有 plugin 需求再叠加 |
| **goctl 的 `--style=gozero` 命名风格 flag** | linctl 强约定单一风格（snake_case 文件 / PascalCase 类型），不引入风格 flag |
| **Kubebuilder 的 `--use-gh-models` AI 集成** | 本轮非目标（§2.2 N2）；远期可加 |

---

## 4. 核心抽象（四件套）

### 4.1 抽象 1：Lockfile（`.linctl/lock.json` + `linctl.yaml`）

**目的**：让生成项目自带身份证，是后续一切操作（sync / diff / upgrade）的事实基准。

**两个文件分工**：

| 文件 | 用户能改吗 | 内容 | 序列化 |
| --- | --- | --- | --- |
| `linctl.yaml` | ✅ 是（手编辑） | Project spec（components / features / module / metadata） | YAML，与现有 `04-config-schema.md` 完全一致 |
| `.linctl/lock.json` | ❌ 否（机器写） | 每文件 owner / templateID / hash / mode / generatedAt / userTouched | JSON（紧凑、稳定 sort key） |

**`linctl.yaml` 范本**：

```yaml
apiVersion: linctl/v1
kind: Project
metadata:
  name: myblog
  module: github.com/foo/myblog
spec:
  defaults:
    framework: gin
    storage: gorm-postgres
  components:
    - kind: WebServer
      name: myblog
      framework: gin
      storage: gorm-postgres
      port: 8080
      features: [healthz]
```

**`.linctl/lock.json` 范本**（节选）：

```json
{
  "schemaVersion": "1",
  "linctlVersion": "v0.3.0",
  "catalog": {
    "id": "embedded",
    "version": "v0.3.0",
    "hash": "sha256:ab12cd34..."
  },
  "generatedAt": "2026-04-28T13:45:00+08:00",
  "lastSyncAt": "2026-04-28T13:45:00+08:00",
  "files": {
    "Makefile": {
      "owner": "WebServer:myblog",
      "template": "templates/project/Makefile.tpl",
      "hash": "sha256:0a1b2c3d...",
      "mode": "overwrite",
      "generatedAt": "2026-04-28T13:45:00+08:00"
    },
    "internal/myblog/biz/v1/user/login.go": {
      "owner": "WebServer:myblog",
      "template": "templates/component/webserver/internal/biz/v1/user/login.go.tpl",
      "hash": "sha256:f0e1d2c3...",
      "mode": "once",
      "generatedAt": "2026-04-28T13:45:00+08:00"
    }
  }
}
```

**关键字段语义**：

| 字段 | 含义 | 用途 |
| --- | --- | --- |
| `schemaVersion` | lockfile 自身格式版本 | 后续兼容性升级 |
| `catalog.id` | 模板源标识 | sync 时校验是否切换源 |
| `catalog.version` | 模板源版本 | sync 时算 from→to |
| `catalog.hash` | 模板树整体 sha256 | 检测模板是否被篡改 |
| `files[*].mode` | `once` / `overwrite` / `managed` | 决定 sync 时该文件的策略 |
| `files[*].hash` | 上次写盘时的内容 hash | 三向合并的 base |

**新增包**：`internal/lockfile/`

```text
internal/lockfile/
├── doc.go
├── lockfile.go           # struct + Load + Save + Update
├── lockfile_test.go
└── migrate.go            # schemaVersion 迁移（v1→v2 等）
```

### 4.2 抽象 2：WriteMode 三态化 + 三向合并

**WriteMode 重新定义**（替换 `pair.go:17-28` 现有定义）：

```go
type WriteMode string

const (
    // WriteModeOverwrite：每次重新生成都直接覆盖。
    // 用于：Makefile / go.mod / Dockerfile / pkg/util/*（基础设施类）。
    WriteModeOverwrite WriteMode = "overwrite"

    // WriteModeOnce：仅在文件不存在时创建；存在则永远 Skip。
    // 用于：biz/v1/<resource>/*.go / store/<resource>.go / handler/<resource>.go
    //      （用户写业务逻辑的文件，绝不能动）。
    WriteModeOnce WriteMode = "once"

    // WriteModeManaged：三向合并。
    //   base = lock.files[dst].hash  （上次 lin 写盘后的 hash）
    //   ours = sha256(read disk)
    //   theirs = sha256(render new template)
    // 用于：server.go / wire.go / configs/<app>.yaml（lin 与用户共同维护的文件）。
    WriteModeManaged WriteMode = "managed"
)
```

**三向合并决策表**：

| 用户改过 (ours ≠ base) | 模板变了 (theirs ≠ base) | 内容相同 (ours == theirs) | Action | Reason |
| --- | --- | --- | --- | --- |
| ❌ | ❌ | - | Skip | no change |
| ❌ | ✅ | - | Update | template upgraded |
| ✅ | ❌ | - | Skip | user customization preserved |
| ✅ | ✅ | ✅ | Skip | converged independently |
| ✅ | ✅ | ❌ | Conflict | 3-way merge needed |

**Conflict 落地策略**（**借鉴 Kubebuilder `alpha update`：调用系统 git 做 3-way merge**）：

> ⚠️ 重要决策变更：相比初稿"自写 BASE/OURS/THEIRS 三个独立文件让用户人工合并"，**改为调用系统 git merge-file 直接在工作区写入标准 git conflict markers**。理由见 [§3.5.2 借鉴点 #1](#352-五个最重要的借鉴点)。

**算法**（每个 managed-mode 检测到 Conflict 时）：

```text
1. 把三个版本写入临时文件：
     base   = lock.files[dst].lastRenderedContent  （从 .linctl/cache/<hash>/ 取，或重新渲染旧 catalog）
     ours   = 当前磁盘文件内容
     theirs = 新模板渲染结果

2. 调用 git merge-file（系统 git 自带）：
     git merge-file -p --diff3 \
       --marker-size=7 \
       <ours> <base> <theirs>

   退出码：
     0 = 自动合并成功（无 marker）
     >0 = 有冲突（输出含 markers）

3. 根据 --strategy flag 决定：
     strategy=ask    → 弹出 $EDITOR 让用户实时合并；保存后 commit 到 lockfile
     strategy=force  → 直接把含 markers 的内容写盘，让 user 后续 git 解决
     strategy=abort  → 不写盘，提示用户手工跑 linctl resolve
     strategy=ours   → 保留用户版本，跳过这次更新（lock 不变）
     strategy=theirs → 用模板版本完全覆盖（lock 更新）
```

**用户看到的标准 git markers**（以 `internal/myblog/server.go` 为例）：

```go
// 用户写的中间件
+ srv.Use(myCustomMiddleware())

func NewServer(opts *Options) (*Server, error) {
<<<<<<< ours (your edits)
    srv := &Server{
        addr: opts.Addr,
        log:  log.Default(),
    }
||||||| base (linctl v0.3.0)
    srv := &Server{
        addr: opts.Addr,
    }
=======
    srv := &Server{
        addr:   opts.Addr,
        log:    log.Default(),
        tracer: otel.Tracer("server"),    // ← 新模板加的 OTel
    }
>>>>>>> theirs (linctl v0.4.0 template)
    ...
}
```

**优势**：

- ✅ 任何 IDE（VSCode / GoLand / Vim）都自带 conflict 解决 UI，零学习成本
- ✅ 用户可以用熟悉的 `git mergetool` / `git checkout --ours/--theirs`
- ✅ CI 模式下用 `--strategy=force` + 后续 `git diff --check`，符合标准 git workflow
- ✅ 实现极简：fork+exec git，不需自维护 merge 算法

**缓存设计 `.linctl/cache/<contentHash>`**：

> base 内容必须在升级时可恢复。为此每次写盘后保留一份"最后渲染版"。

```text
.linctl/cache/
├── 0a1b2c3d.../        # contentHash = sha256(rendered template)
│   └── content         # 渲染后的字节流
└── prune.policy        # 保留策略（默认：保留最近 50 次 sync 的内容）
```

**新增命令 `linctl resolve [path...]`**：

```bash
$ linctl resolve internal/myblog/server.go
# 1. 检测文件是否还含 git conflict markers
# 2. 没有 markers → 把当前内容 hash 写入 lock；标记 file 已重新 sync
# 3. 还有 markers → 报错提示用户先解决
$ linctl resolve --all     # 一次性扫描所有曾冲突的文件
```

**包变更**：

- `internal/codegen/plan.go`：启用已存在的 `ActionConflict` 占位
- `internal/codegen/planner.go`：新增 `threeWayDecide()` 决策（5 case）
- `internal/codegen/applier.go`：处理 Conflict 时调用 `gitmerge.MergeFile()`
- **新增 `internal/gitmerge/`**：包装 `git merge-file` 系统调用（含 fallback：若用户机器没装 git，回退到内置 diff3 实现）
- 新增 `internal/cli/cmd_status.go`、`cmd_sync.go`、`cmd_resolve.go`

### 4.3 抽象 3：Catalog（模板源可插拔）

**接口**：

```go
// internal/template/catalog/catalog.go

type Catalog interface {
    // ID 是 catalog 的稳定标识，写入 lock.json。
    // 例："embedded" / "local:/abs/path" / "git:github.com/foo/lin-templates@v1.2.3"
    ID() string

    // Version 是 catalog 的版本号（语义化版本或 commit hash）。
    Version() string

    // Hash 是 catalog 内容的整体 sha256，用于检测篡改。
    Hash() string

    // FS 返回该 catalog 的模板文件系统（embed.FS / os.DirFS / git fetched dir）。
    FS() fs.FS

    // CatalogManifest 返回顶层清单（描述 catalog 内有哪些 component / feature templates）。
    Manifest() (*CatalogManifest, error)
}
```

**三种内置实现**：

| 实现 | 来源 | 用法 |
| --- | --- | --- |
| `EmbeddedCatalog` | `go:embed all:templates`（默认） | `linctl new` 不传 `--catalog` 时用 |
| `LocalCatalog` | 本地目录 | `linctl new --catalog file:///path/to/templates` |
| `GitCatalog` | git repo + ref | `linctl new --catalog git+https://github.com/foo/lin-templates@v1.2.3` |

**fetch / 缓存策略**（GitCatalog）：

- 缓存路径：`$XDG_CACHE_HOME/linctl/catalogs/<repo-hash>/<ref>/`
- 首次 fetch：`git clone --depth=1 --branch=<ref>`
- 后续：默认走缓存；`--catalog-refresh` 强制重新拉取
- 离线场景：缓存过期时降级到上次缓存版本（输出 warning）

**包结构**：

```text
internal/template/catalog/
├── doc.go
├── catalog.go              # interface + 公共类型
├── embedded.go             # EmbeddedCatalog
├── local.go                # LocalCatalog
├── git.go                  # GitCatalog
├── parse_url.go            # 解析 --catalog flag
├── manifest.go             # CatalogManifest 类型 + 解析
└── *_test.go
```

**Engine 集成**（`internal/template/engine.go` 改造）：

```go
// 新构造路径
type Engine struct {
    catalog catalog.Catalog       // ← 新增，替代直接持有 fs.FS
    funcMap texttemplate.FuncMap
    parsedCache sync.Map
}

func NewWithCatalog(c catalog.Catalog, opts ...Option) (*Engine, error)

// 旧构造路径保持兼容（内部默认包 EmbeddedCatalog）
func New(opts ...Option) (*Engine, error)
```

### 4.4 抽象 4：Manifest-Driven Templates

**目的**：把"哪个文件用哪个模板渲染到哪里、何时启用"这件事**从 Go 代码搬到 YAML**，实现声明化。

**模板树根 `MANIFEST.yaml` 范例**（webserver 组件）：

```yaml
# templates/component/webserver/MANIFEST.yaml
apiVersion: linctl-template/v1
kind: ComponentTemplate
metadata:
  kind: WebServer
  name: webserver
  version: v1.0.0
  description: "Gin / gRPC web server skeleton (miniblog-v4 aligned)"

# 模板继承（可选；引用其他 component 的 manifest）
extends:
  - templates/web-gin/MANIFEST.yaml

# 文件清单
files:
  - id: cmd-main-gin
    src: cmd_main.go.tpl
    dst: cmd/{{ .Component.Name }}/main.go
    mode: managed
    when: '{{ ne .Component.Framework "grpc" }}'
    owner: WebServer

  - id: cmd-main-grpc
    src: grpc/cmd_main.go.tpl
    dst: cmd/{{ .Component.Name }}/main.go
    mode: managed
    when: '{{ eq .Component.Framework "grpc" }}'
    owner: WebServer

  # 业务骨架：once 模式
  - id: biz-user-login
    src: internal/biz/v1/user/login.go.tpl
    dst: internal/{{ .Component.Name }}/biz/v1/user/login.go
    mode: once
    owner: WebServer

# 文件分组（便于条件批量启用）
groups:
  - id: storage-gorm-mysql
    when: '{{ eq .Component.Storage "gorm-mysql" }}'
    files:
      - id: db-mysql
        src: ../../web-gin/pkg/db/mysql.go.tpl
        dst: pkg/db/mysql.go
        mode: overwrite

  - id: third-party-protos
    when: '{{ or (eq .Component.Framework "gin") (eq .Component.Framework "grpc") }}'
    files: [...]
```

**对 `webserver.go` 的瘦身效果**：

| 文件 | 改造前 | 改造后 |
| --- | --- | --- |
| `internal/component/webserver.go` | 720 行（手写 9 个 `webGin*Pairs()` 函数） | 约 100 行（保留 Validate / Kind / Name / SpecComponent，BasePairs 改为调用 `manifest.Resolve(ctx, w.spec)`） |
| `templates/component/webserver/MANIFEST.yaml` | 不存在 | 约 250 行 YAML |

**实现包**：

```text
internal/template/manifest/
├── doc.go
├── manifest.go              # struct: ComponentTemplate / FileEntry / Group
├── parser.go                # YAML 解析 + extends 展开
├── resolver.go              # ResolveFiles(ctx, project, component) → []codegen.Pair
├── validate.go              # MANIFEST 自身校验
└── *_test.go
```

**Resolver 算法伪代码**：

```go
func Resolve(ctx, p, comp) ([]codegen.Pair, error) {
    m := loadManifest(componentTemplateRoot(comp.Kind))
    m = expandExtends(m)                        // 递归展开 extends
    pairs := []codegen.Pair{}
    for _, f := range allFlatFiles(m) {         // groups 也展平
        if !evalWhen(f.When, p, comp) {
            continue                            // when 条件不满足，跳过
        }
        dst := renderTemplateString(f.Dst, p, comp)  // 渲染 dst 路径
        pairs = append(pairs, codegen.Pair{
            Dst: dst,
            TemplateID: pathRel(m.Root, f.Src),
            Mode: parseWriteMode(f.Mode),
            Owner: f.Owner + ":" + comp.Name,
        })
    }
    return pairs, nil
}
```

### 4.5 抽象 5：`linctl template` 子命令族（goctl 风格）

> 借鉴 [§3.5.2 借鉴点 #2](#352-五个最重要的借鉴点)：goctl 的 `~/.goctl/` + `template init/clean/revert/update` 子命令家族。

**目录结构**（用户视角）：

```text
~/.linctl/
├── catalogs/
│   ├── embedded@v0.3.0/                          # init 后从内嵌复制
│   │   ├── component/webserver/...
│   │   └── ...
│   ├── git--github.com-myco-templates@v2.1.0/    # GitCatalog 缓存
│   │   ├── .meta.yaml                            # 含 origin/branch/ref/fetched-at
│   │   └── component/webserver/...
│   └── prune.policy
└── config.yaml                                    # 全局默认 catalog
```

**子命令清单**（与 goctl 命名对齐，含义略有调整）：

| 子命令 | 作用 | 类比 goctl |
| --- | --- | --- |
| `linctl template init` | 把内嵌模板复制到 `~/.linctl/catalogs/embedded@<ver>/` | `goctl template init` |
| `linctl template list` | 列出 `~/.linctl/catalogs/` 已有的所有 catalog 与版本 | （goctl 无） |
| `linctl template clean [--catalog <id>]` | 清空特定 catalog 缓存（默认全清） | `goctl template clean` |
| `linctl template revert <file>` | 把单个本地修改的模板文件恢复为该 catalog 的默认 | `goctl template revert` |
| `linctl template update [--catalog <id>] [--to <ver>]` | 拉取 catalog 新版本到 `~/.linctl/catalogs/<id>@<newver>/` | `goctl template update` |
| `linctl template diff <file>` | 对比本地修改 vs 该 catalog 默认 | （goctl 无；nest 无；新增） |
| `linctl template sync` | 把 `~/.linctl/catalogs/` 内的修改写回 origin（仅 LocalCatalog 支持） | （新增） |
| `linctl template publish` | 把当前 `~/.linctl/catalogs/<id>/` 推到远端（git push） | （新增；远期） |

**用户视角的工作流**（典型场景）：

```bash
# 场景 A：体验默认模板，不定制
$ linctl new myproj --module github.com/foo/myproj
✔ Used catalog: embedded@v0.3.0 (in-binary)

# 场景 B：想改模板但保留升级能力
$ linctl template init                          # 复制内嵌到 ~/.linctl/
✔ Initialized: ~/.linctl/catalogs/embedded@v0.3.0/ (255 files)

$ vim ~/.linctl/catalogs/embedded@v0.3.0/component/webserver/cmd_main.go.tpl
   # 加上你的公司 banner

$ linctl new myproj --catalog embedded@v0.3.0   # 自动用 ~/.linctl/ 而非内嵌

# 场景 C：lin 升到 v0.4.0，想 sync 你之前的本地改动
$ linctl template diff component/webserver/cmd_main.go.tpl
   --- ~/.linctl/catalogs/embedded@v0.3.0/...
   +++ embedded@v0.4.0 (new in this lin binary)
   ...

$ linctl template update --catalog embedded --to v0.4.0
✔ Pulled embedded@v0.4.0
✔ Detected 3 user-modified files; opened in $EDITOR for 3-way merge
```

**包结构**：

```text
internal/cli/
├── cmd_template.go            # template 父命令
├── cmd_template_init.go
├── cmd_template_list.go
├── cmd_template_clean.go
├── cmd_template_revert.go
├── cmd_template_update.go
├── cmd_template_diff.go
└── cmd_template_sync.go
```

### 4.6 抽象 6：Schematic 概念（Nest 风格的细粒度生成器）

> 借鉴 [§3.5.2 借鉴点 #3](#352-五个最重要的借鉴点)：拆解 `linctl new` 这个巨型函数为多个 schematic。

#### 4.6.1 现状的问题

| 当前命令 | 粒度 | 痛点 |
| --- | --- | --- |
| `linctl new <name>` | 巨型（255 文件） | 用户只想加一个 controller，必须接受全套 |
| `linctl add api <name>` | 中型（3 文件 + 2 处 AST） | 唯一的细粒度命令，但孤立 |

#### 4.6.2 目标：分级命令体系

```bash
# === 项目级 ===
linctl new <name>           # = init + g webserver <name>（兼容现有命令；保留作快捷方式）
linctl init <name>          # 仅创建项目骨架（go.mod / Makefile / configs / docs / scripts）

# === 组件级 ===
linctl g webserver <name>    # 加一个 WebServer 组件（gin / grpc）
linctl g worker <name>       # 加一个 Worker 组件
linctl g cli <name>          # 加一个 CLI 组件

# === 资源级 ===
linctl g resource <Name>     # 加 CRUD 资源（biz + store + handler + proto + AST 注入）
                             # 等价于现在的 `linctl add api <Name>`，重命名

# === 文件级 ===
linctl g middleware <name>   # 加单个 middleware
linctl g handler <name>      # 加单个 handler（不带 biz/store）
linctl g cron-job <name>     # 加单个 cron job（仅在 Worker 组件下有效）
linctl g kafka-topic <name>  # 加单个 kafka 消费者（仅在 Worker 组件下有效）
```

**别名**（仿 nest）：`linctl g r users` = `linctl generate resource users`

#### 4.6.3 Schematic 数据模型

每个 schematic 是模板树下一个独立目录，**自带 `schema.json` + 模板**：

```text
templates/schematics/
├── webserver/
│   ├── schema.json           # 参数定义（用于 CLI flags 派生）
│   ├── files/                # 该 schematic 包含的所有模板
│   │   └── cmd/{{.Name}}/main.go.tpl
│   ├── MANIFEST.yaml         # 文件清单
│   └── injectors.yaml        # AST 注入规则（声明式）
├── resource/
│   ├── schema.json
│   ├── files/
│   │   ├── biz/{{.Name}}.go.tpl
│   │   ├── store/{{.Name}}.go.tpl
│   │   └── handler/{{.Name}}.go.tpl
│   ├── MANIFEST.yaml
│   └── injectors.yaml
├── middleware/
│   └── ...
└── collection.json           # 总入口：声明本 catalog 提供哪些 schematic
```

**`schema.json` 范例**（resource schematic）：

```json
{
  "$id": "linctl/schematics/resource",
  "title": "Resource (CRUD)",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "description": "Resource name in PascalCase",
      "x-prompt": "Resource name?"
    },
    "component": {
      "type": "string",
      "description": "Owning WebServer component name (default: project root name)"
    },
    "crud": {
      "type": "boolean",
      "default": true,
      "description": "Generate full CRUD methods"
    },
    "framework": {
      "type": "string",
      "enum": ["gin", "grpc"],
      "description": "Framework override (default: read from linctl.yaml)"
    }
  },
  "required": ["name"]
}
```

**`injectors.yaml` 范例**（声明式 AST 注入，替代 `cmd_add.go` 中硬编码的 mutator 列表）：

```yaml
apiVersion: linctl-template/v1
kind: ASTInjectors
metadata:
  schematic: resource

injectors:
  - kind: AddInterfaceMethod
    file: internal/{{ .Component }}/biz/biz.go
    interface: IBiz
    method: '{{ .NamePascal }}V1'
    returns: '{{ .NamePascal }}Biz'
    doc: '{{ .NamePascal }}V1 返回 {{ .NamePascal }} 业务对象（v1 版本）。'

  - kind: AddInterfaceMethod
    file: internal/{{ .Component }}/store/store.go
    interface: IStore
    method: '{{ .NamePascal }}'
    returns: '{{ .NamePascal }}Store'
    doc: '{{ .NamePascal }} 返回 {{ .NamePascal }} 数据访问对象。'

  - kind: AddRouteRegistration   # Phase 3+ 新增的注入器
    file: internal/{{ .Component }}/server.go
    function: registerRoutes
    statement: '{{ .NameLower }}.RegisterRoutes(group, h)'
```

#### 4.6.4 命令派生

`linctl g <schematic> <name> [flags]` 的执行流程：

```text
1. 加载当前 catalog 的 collection.json
2. 找到 schematic="resource" 对应的目录
3. 读 schema.json，把 properties 派生为 cobra flags（含默认值、enum 校验）
4. 解析用户输入 → 构造 schematic context (含 name / namePascal / nameLower / project)
5. 加载 MANIFEST.yaml → 渲染所有 files
6. 加载 injectors.yaml → 调用 AST mutator 注入
7. 写 lockfile
```

#### 4.6.5 与现有 `linctl add api` 的兼容

- **保留** `linctl add api <Name>` 别名（向后兼容）
- 文档中标注："`add api` is now an alias for `g resource`，will be removed in v1.0"
- v0.5.0 起新文档全部用 `linctl g resource`

#### 4.6.6 包变更

```text
internal/cli/
├── cmd_g.go                  # g/generate 父命令
├── cmd_g_runner.go           # 通用 runner：load schema → derive flags → execute
└── (cmd_add.go 改为薄壳调用 cmd_g_runner.go)

internal/schematic/           # 新增包
├── doc.go
├── schema.go                 # schema.json 加载 + JSON Schema 校验
├── flag_deriver.go           # 把 schema → cobra flags
├── injector.go               # injectors.yaml 加载 + 转 ast.ASTMutator
└── runner.go                 # 端到端执行
```

---

## 5. 端到端用户场景

> 5 个场景从最简单到最复杂排列，覆盖本设计的全部新能力。

### 5.1 场景 A：首次创建项目（含 lockfile）

```bash
$ linctl new myblog --module github.com/foo/myblog --framework gin --storage gorm-postgres

✔ resolved catalog: embedded@v0.3.0 (sha256:ab12...)
✔ rendered 187 files (manifest: webserver@v1.0.0)
✔ wrote linctl.yaml
✔ wrote .linctl/lock.json (187 entries)

📋 Summary
   create: 187 (overwrite=145, once=32, managed=10)
   update: 0
   skip:   0

🚀 Next steps:
   cd myblog
   go mod tidy
   make wire
   make build
```

**与现状差异**：多了 `linctl.yaml` + `.linctl/lock.json` 两个文件落地、stats 按 mode 分类。

### 5.2 场景 B：用户改了业务代码后再跑一次（once 模式保护）

```bash
# 第一次：用户改了 internal/myblog/biz/v1/user/login.go，加入了实际业务逻辑

$ cd myblog && linctl new --replay   # 等价于读 linctl.yaml 重新计算

📋 Plan (vs lock @ 2026-04-28T13:45)
   skip:    187 (no change)
   preserve: 1
     # internal/myblog/biz/v1/user/login.go (mode=once, file exists)

✔ Apply complete. 0 files written.
```

**关键改动**：`once` 模式遇到已存在文件 → 不渲染、不读、不写、不报告 conflict。

### 5.3 场景 C：用户改了 server.go 后模板也升级了（三向合并）

```bash
# 用户在 internal/myblog/server.go 加了一行自定义中间件
# 模板新版本 v0.4.0 也修改了 server.go（加了 OTel）

$ linctl sync --catalog-version v0.4.0

📋 Plan (vs lock @ 2026-04-28T13:45 → catalog v0.4.0)
   skip:     185
   update:   1
     ~ Makefile (template upgraded; user not modified)
   conflict: 1
     ! internal/myblog/server.go
       base:   sha256:abc123 (lock @ v0.3.0)
       yours:  sha256:def456 (you modified)
       theirs: sha256:789abc (template v0.4.0)
       artifacts saved to .linctl/conflicts/internal/myblog/server.go/

❌ Apply aborted: 1 conflict requires manual resolution.
   Run: linctl resolve internal/myblog/server.go
        ↳ opens diff in $EDITOR; on save, lockfile is updated.
```

### 5.4 场景 D：把项目从内嵌模板切换到公司私有模板

```bash
$ linctl sync --catalog git+ssh://git@github.com/myco/lin-templates@v2.1.0

✔ fetched catalog: git:github.com/myco/lin-templates@v2.1.0 (sha256:cd34...)
✔ catalog migration: embedded@v0.3.0 → git:github.com/myco/lin-templates@v2.1.0

📋 Plan (vs lock @ 2026-04-28T13:45 → catalog v2.1.0)
   skip:     150
   update:   30  (basic infrastructure files matching new corp standard)
   conflict: 7   (server.go, configs/myblog.yaml, ...)

   1 migration step required:
     - migrations/v0.3.0-to-v2.1.0.yaml
       rename: pkg/util/strings/strings.go → pkg/strutil/strutil.go (1 file)
       delete: pkg/util/lint/* (5 files)

? Apply with --strategy=interactive: y
✔ Applied 30 updates, executed 1 migration, 7 conflicts deferred (see .linctl/conflicts/).
✔ updated .linctl/lock.json: catalog → git:github.com/myco/lin-templates@v2.1.0
```

### 5.5 场景 E：CI 中校验项目是否被偷偷修改

```bash
$ linctl status -o json
{
  "ok": true,
  "linctlYAMLValid": true,
  "lockfileValid": true,
  "drift": {
    "userTouched": [
      {"path": "internal/myblog/server.go", "mode": "managed"},
      {"path": "configs/myblog.yaml", "mode": "managed"}
    ],
    "missing": [],
    "obsolete": []
  },
  "summary": {
    "skip": 185,
    "wouldUpdate": 0,
    "userTouched": 2,
    "wouldConflict": 0
  }
}
$ echo $?
0
```

CI 用 `jq` 解析 `userTouched` 是否为空、是否符合白名单。

---

## 6. 实施路线图（4 阶段）

> 每阶段都是**独立可发布的小版本**，互不阻塞下一阶段。强烈建议从阶段 1 开始严格按顺序实施。

### 6.1 阶段总览

> 整体路线在借鉴业界标杆后从 4 阶段调整为 6 阶段，新增 L5（Schematic 拆分）+ L6（AutoUpdate 远期）。

| 阶段 | 主题 | 关键交付 | 估时 | 影响痛点 | 主要借鉴 |
| --- | --- | --- | --- | --- | --- |
| **L1** | Lockfile + linctl.yaml 持久化 | `internal/lockfile/`、`linctl new` 落地元数据、`linctl status` | **1.5-2 天** | L-P1 | Kubebuilder PROJECT |
| **L2** | WriteMode 三态 + **git 3-way merge** + sync | `linctl sync` / `linctl resolve`、Conflict Action 落地、`internal/gitmerge/` | **2.5-3 天** | L-P2 / L-P3 / L-P6 | Kubebuilder `alpha update` |
| **L3** | Catalog 抽象 + `~/.linctl/` 缓存 + `template` 子命令族 | `internal/template/catalog/`、`linctl template init/clean/revert/update/diff` | **4-6 天** | L-P4 | goctl `template` |
| **L4** | Manifest-Driven Templates | `internal/template/manifest/`、`webserver.go` 瘦身、`MANIFEST.yaml` | **2-3 天** | L-P5 / L-P7 | 自研 + nest collection |
| **L5** | Schematic 化（细粒度生成器） | `internal/schematic/`、`linctl g <schematic> <name>` 体系、`schema.json` + `injectors.yaml` | **3-4 天** | L-P5 进一步深化 | Nest CLI |
| **L6** | AutoUpdate Plugin（远期） | `linctl init-autoupdate`、`.github/workflows/linctl-update.yml` scaffold、`linctl-update-action` | **2-3 天** | 长尾运营 | Kubebuilder AutoUpdate |

### 6.2 阶段 L1 详细计划（1.5-2 天）

| Story | 内容 | 改动文件 | 验收 |
| --- | --- | --- | --- |
| L1.1 | 新增 `internal/lockfile/` 包，定义 `Lockfile` struct + Load/Save/Update | 新增 4 文件 (~300 行) | 单元测试覆盖 ≥85% |
| L1.2 | 新增 `internal/project/loader.go` 增强：从 `linctl.yaml` 反序列化为 `*project.Project` | `loader.go` 已存在，约 +120 行 | round-trip 测试：write→read 无损 |
| L1.3 | `cli/cmd_new.go` 在 Apply 后写 `linctl.yaml` + `.linctl/lock.json` | `cmd_new.go` 约 +50 行 | E2E：生成项目后两文件存在且内容对 |
| L1.4 | `cli/cmd_status.go` 实现"读 lock + 走 Plan + 输出报告"（仅打印不写盘） | 新增 1 文件 (~250 行) | dry-run 输出含 drift 列表 |
| L1.5 | 文档：新增 `21-template-lifecycle.md`（写到 §3 §4.1） | 新增文档 | 通过 §9 DoD |

**阶段 L1 退出条件**：

- 跑 `linctl new myproj` → 项目根有 `linctl.yaml` + `.linctl/lock.json`
- 跑 `linctl status` → 输出干净的"187 files in sync"
- 跑 `linctl status -o json` → 可被 `jq` 解析

### 6.3 阶段 L2 详细计划（2.5-3 天）

| Story | 内容 | 改动文件 | 验收 |
| --- | --- | --- | --- |
| L2.1 | 重写 `pair.go` `WriteMode` 类型为 `string`，明确 once/overwrite/managed 三态 | `pair.go` 约 +30 -20 行 | 单元测试覆盖 100% |
| L2.2 | `planner.go` 实施 `threeWayDecide()`，结合 lockfile.base / disk.ours / render.theirs | `planner.go` 约 +120 行 | 决策表 5 case 单元测试全过 |
| L2.3 | `applier.go` 处理 Conflict：写 `.linctl/conflicts/<dst>/{BASE,OURS,THEIRS}` | `applier.go` 约 +80 行 | 触发 conflict 后三件齐全 |
| L2.4 | 新增 `cli/cmd_sync.go`：复用 Plan→Apply 链路，支持 `--strategy=ask\|skip\|abort` | 新增 1 文件 (~350 行) | E2E：场景 B/C 全过 |
| L2.5 | 新增 `cli/cmd_resolve.go`：用户解决冲突后更新 lockfile | 新增 1 文件 (~150 行) | resolve 后 status 干净 |
| L2.6 | `webserver.go` 给业务文件标 `mode: once`、给 server.go/wire.go 标 `mode: managed` | `webserver.go` 微调（~30 处） | 场景 B/C 行为符合预期 |
| L2.7 | 文档：补 §4.2 + §5（场景 B/C/E） + 新建 ADR-007 | 文档 | 通过 §9 DoD |

**阶段 L2 退出条件**：

- 用户改了 `biz/v1/user/login.go`，跑 `linctl sync` 不会被覆盖
- 用户改了 `server.go` + 模板也变了，跑 `linctl sync` 输出 conflict + 三件
- `linctl resolve` 命令存在且能让冲突回归 sync 状态

### 6.4 阶段 L3 详细计划（4-6 天）

| Story | 内容 | 估行数 |
| --- | --- | --- |
| L3.1 | 新增 `internal/template/catalog/` 包：interface + `EmbeddedCatalog` | ~400 行 |
| L3.2 | `LocalCatalog` 实现 + URL 解析 (`file://` 前缀) | ~250 行 |
| L3.3 | `GitCatalog` 实现：clone / cache / refresh / 离线降级 | ~500 行 |
| L3.4 | Engine 接受 Catalog 注入；orchestrator 从 `lock.catalog` 还原 | ~150 行改造 |
| L3.5 | `--catalog` flag 全局接入 + `linctl new` / `linctl sync` 支持 | ~80 行 |
| L3.6 | **`linctl template` 子命令族**（init / list / clean / revert / update / diff），含 `~/.linctl/catalogs/` 目录管理 | ~600 行（8 个子命令文件） |
| L3.7 | 文档：§4.3 + §4.5 + 新建 ADR-008 | 文档 |

**阶段 L3 退出条件**：

- 场景 D 全过；E2E 测试覆盖 embedded/local/git 三种 catalog
- `linctl template init` 把内嵌模板复制到 `~/.linctl/catalogs/embedded@<ver>/` 验证通过
- `linctl template revert <file>` 单文件回滚验证通过
- `linctl template diff <file>` 输出 unified diff 验证通过

### 6.5 阶段 L4 详细计划（2-3 天）

| Story | 内容 |
| --- | --- |
| L4.1 | 新增 `internal/template/manifest/` 包：YAML 解析 + extends 展开 |
| L4.2 | `Resolver` 实现：根据 project + component 算出 `[]codegen.Pair` |
| L4.3 | 在 `templates/component/webserver/` 写 `MANIFEST.yaml`（包含全部 ~187 文件条目） |
| L4.4 | `webserver.go` 改造：`BasePairs()` 改为调用 `manifest.Resolve()` |
| L4.5 | 同步给 `worker.go` / `cli.go` 写 manifest |
| L4.6 | 引入 `migrations/v<from>-to-v<to>.yaml` 声明：rename/delete/split |
| L4.7 | 文档：§4.4 + 新建 ADR-009 |

**阶段 L4 退出条件**：`webserver.go` 减重至 ≤ 150 行；新增 1 文件改 manifest YAML 1 行而非 Go 代码。

### 6.6 阶段 L5 详细计划（3-4 天）

> 借鉴 Nest CLI Schematic 体系；详见 [§4.6](#46-抽象-6schematic-概念nest-风格的细粒度生成器)。

| Story | 内容 | 估行数 |
| --- | --- | --- |
| L5.1 | 新增 `internal/schematic/` 包：定义 `Schematic` / `Schema` / `Injector` 类型 | ~300 行 |
| L5.2 | `schema.json` 加载 + JSON Schema 校验（用 [`gojsonschema`](https://github.com/xeipuuv/gojsonschema)） | ~200 行 |
| L5.3 | `flag_deriver.go`：把 `schema.properties` 派生为 cobra flags（含默认值 / enum 校验 / required） | ~250 行 |
| L5.4 | `injectors.yaml` 加载 → 转 `ast.ASTMutator` 列表 | ~200 行 |
| L5.5 | `runner.go`：端到端执行（schema → derive flags → render manifest → run injectors） | ~300 行 |
| L5.6 | 新增 `cli/cmd_g.go`：通用 `linctl g <schematic> <name>` 入口 + 别名 `linctl generate` | ~150 行 |
| L5.7 | 把 `linctl add api` 改为 `linctl g resource` 别名（保留兼容） | `cmd_add.go` 改为 thin wrapper |
| L5.8 | 内置 schematic：`webserver` / `worker` / `cli` / `resource` / `middleware` / `handler` / `cron-job` / `kafka-topic`（8 个） | 新增 8 个 schema.json + 8 个 injectors.yaml |
| L5.9 | 文档：§4.6 + 新建 ADR-010（schematic 概念）+ 03-cli-design 增补 g 命令族 | 文档 |

**阶段 L5 退出条件**：

- 跑 `linctl g resource Post` 等价于现在的 `linctl add api Post`
- 跑 `linctl g middleware Auth` 单文件级生成 + 自动注入 server.go
- `schema.json` 改字段无须改 Go 代码即可改 CLI flag 集合
- 第三方 catalog 提供新 schematic 时立即可用（无须修改 lin 主仓库）

### 6.7 阶段 L6 详细计划（2-3 天，远期）

> 借鉴 Kubebuilder AutoUpdate Plugin。可在 L1-L5 全部完成后启动。

| Story | 内容 |
| --- | --- |
| L6.1 | 新增 `cmd_init_autoupdate.go`：scaffold `.github/workflows/linctl-update.yml` |
| L6.2 | 新增独立仓库 `clin211/linctl-update-action`（GitHub Action 实现） |
| L6.3 | Action 内部跑 `linctl sync --catalog-version=latest --strategy=force`，把含 markers 的提交开 PR |
| L6.4 | PR template：列出 conflict 摘要 + 模板 changelog 链接 |
| L6.5 | 文档：14-observability.md 增补 `linctl-update-action` 的 telemetry；新建 ADR-011 |

**阶段 L6 退出条件**：用户跑 `linctl init-autoupdate` 后，每周自动检查模板新版并开 PR。

---

## 7. 文档对接清单

### 7.1 需要新增的文档

| 文档 | 位置 | 状态 |
| --- | --- | --- |
| `21-template-lifecycle.md` | `lin/docs/` | 阶段 L1 起逐步写入；定稿于 L5 |
| `ADR-006: prefer-external-lockfile-over-inline-hash` | `lin/docs/adr/` | 阶段 L1 |
| `ADR-007: use-system-git-merge-for-three-way-conflict` | `lin/docs/adr/` | 阶段 L2（**关键决策**：调用 git merge-file 而非自实现） |
| `ADR-008: pluggable-template-catalog-with-home-cache` | `lin/docs/adr/` | 阶段 L3（融合 goctl `~/.goctl/` + linctl catalog） |
| `ADR-009: manifest-driven-component-templates` | `lin/docs/adr/` | 阶段 L4 |
| `ADR-010: schematic-as-fine-grained-generator` | `lin/docs/adr/` | 阶段 L5（借鉴 Nest schematic） |
| `ADR-011: autoupdate-as-github-action-plugin` | `lin/docs/adr/` | 阶段 L6（借鉴 Kubebuilder AutoUpdate） |
| `diagrams/seq-sync-with-git-merge.mmd` | `lin/docs/diagrams/` | 阶段 L2 |
| `diagrams/seq-template-init-revert.mmd` | `lin/docs/diagrams/` | 阶段 L3 |
| `diagrams/seq-schematic-execute.mmd` | `lin/docs/diagrams/` | 阶段 L5 |

### 7.2 需要修改的现有文档

| 文档 | 修改内容 | 阶段 |
| --- | --- | --- |
| `06-codegen-pipeline.md` | 追加 §6.16 模板生命周期；更新 §6.2 流水线图说明 lock 持久化时机；§6.17 git merge 集成 | L1+L2 |
| `05-template-system.md` | 追加 §5.6 Catalog 抽象 + §5.7 MANIFEST.yaml 规范 + §5.8 Schematic | L3+L4+L5 |
| `04-config-schema.md` | 追加 §4.10 lockfile schema + §4.11 schema.json schematic params | L1+L5 |
| `03-cli-design.md` | 新增 `linctl sync` / `linctl status` / `linctl resolve` / `linctl template *` / `linctl g <schematic>` / `linctl init-autoupdate` 子命令规范 | L2+L3+L5+L6 |
| `11-implementation-plan.md` | 重写 Phase 4：把原 Story 4.1-4.5 替换为本文 §6 路线图（L1-L6） | L1（破冰时） |
| `META-roadmap.md` | §1.2 文档清单加 21；修订历史加一行 | L1 |
| `99-glossary.md` | 加 Lockfile / Catalog / Manifest / WriteMode / Schematic / 3-way merge 术语 | L1-L5 渐进 |
| `07-ast-injection.md` | 追加 §7.x：injectors.yaml 声明式 mutator 配置 | L5 |

### 7.3 ADR 候选纲要

**ADR-006: prefer-external-lockfile-over-inline-hash**
v0.3.x 已废弃 inline hash 注释，本设计正式记录"用 `.linctl/lock.json` 集中存储 hash"的决策。备选方案对比：inline 注释 vs 外部 lockfile vs 双写。

**ADR-007: use-system-git-merge-for-three-way-conflict**（**关键决策**）
本设计**选择调用系统 git 的 `git merge-file` 命令**做 3-way merge，而非：

- 自写 diff3 算法（实现复杂、边界场景多）
- 引入 [`go-git`](https://github.com/go-git/go-git) 库（依赖膨胀 ~3MB）
- 引入 libgit2 cgo 绑定（编译复杂、跨平台差）

理由：① Kubebuilder `alpha update` 已验证有效；② 用户对 git markers 零学习成本；③ IDE 完整支持；④ 实现极简（fork+exec）。fallback 策略：用户机器无 git 时退化到内置 diff3。

**ADR-008: pluggable-template-catalog-with-home-cache**
Catalog 选择"接口 + 三种内置实现（embedded/local/git）+ `~/.linctl/catalogs/` 缓存目录"。融合 goctl `~/.goctl/` + Nest collection.json 的优点。备选：Go plugin .so 系统（编译复杂）/ npm-style 注册中心（生态错位）。

**ADR-009: manifest-driven-component-templates**
Manifest YAML 选择 vs 替代方案（HCL / Starlark / Cue）。理由：① YAML 是用户已熟悉的（linctl.yaml 也是 YAML）；② 解析库标准（gopkg.in/yaml.v3）；③ 不引入新 DSL 学习成本。

**ADR-010: schematic-as-fine-grained-generator**
拆分 `linctl new` 为多个 schematic（`g webserver` / `g resource` / `g middleware` 等），借鉴 Nest CLI。备选：保持现状（巨型 new 命令）/ goctl 风格按 category 分（粒度仍粗）。理由：① 用户实际场景多是小增量；② 第三方 catalog 可贡献新 schematic 而无须改 lin 主仓库。

**ADR-011: autoupdate-as-github-action-plugin**
AutoUpdate 选择"linctl scaffold GitHub Action workflow + 独立 action 仓库"vs 替代方案（lin 自带 daemon 监控 / IDE 插件）。理由：① 用户已熟悉 GitHub Actions；② 不绑定特定 IDE；③ Kubebuilder 验证过路径。

---

## 8. 风险、回滚与非目标边界

### 8.1 关键风险

| 风险 | 等级 | 缓解 |
| --- | --- | --- |
| Lockfile 损坏（用户误删 / 编辑器毁文件） | 高 | `linctl status --rebuild-lock` 从磁盘 + spec 重建 lock |
| 三向合并算法误判（hash 哈希碰撞） | 极低 | sha256 碰撞概率忽略不计；但保留 `linctl status --strict` 启用按行 diff |
| Catalog git fetch 网络失败 | 中 | 缓存降级 + 明确 warning；保留 embedded 作为 fallback |
| MANIFEST.yaml 写错（when 表达式错误） | 中 | parse 时做 `text/template` lint；resolver 提供 dry-run 模式 |
| 用户 lockfile 与 spec 不一致 | 中 | `linctl status` 检测到 spec 改动但 lock 未更新时提示 sync |
| 性能回归（每次都要算 manifest） | 低 | manifest 解析结果缓存；E2E 跑 187 文件不超过 1.2s |

### 8.2 回滚策略

- **本设计的每个阶段都是 additive**（L1 不影响 L2 之前的旧行为）：
  - L1 之前生成的项目 → 跑 `linctl status` 提示"未发现 lockfile, run linctl claim"
  - 新增 `linctl claim` 命令：扫描磁盘 + 反推 lock（best-effort）
- **若整个设计需要回滚**：
  - lockfile 是新增文件，删除目录无副作用
  - `cmd_new.go` 增量改动可 revert
  - WriteMode 改造前用 feature flag `LINCTL_LIFECYCLE=on/off` 守护

### 8.3 非目标边界（不要在本轮做）

| 边界 | 原因 |
| --- | --- |
| 不引入"AI 自动 conflict 解决" | 留给后续 Feature；先确保确定性 3-way merge |
| 不做团队级中央 lockfile | 本地 lockfile 已能解决；后续如有需求再做 |
| 不做"模板权限 / 签名" | 本轮不涉及安全升级；ADR-008 仅讨论功能正确性 |
| 不做 IDE 插件 | 单独 roadmap |
| 不做"模板市场 / 评分系统" | 远期 Batch 4+ |

---

## 9. 验收标准（DoD）

### 9.1 设计文档 DoD

- [x] 与 ADR-004 Tier 2/3 的对应关系明确（§3.1）
- [x] 与既有 06/05/11 文档的衔接点说明（§3.2 §3.3）
- [x] 4 个核心抽象都有可落地代码骨架（§4）
- [x] 5 个端到端用户场景全部含具体命令 + 预期输出（§5）
- [x] 4 个阶段的 Story / 估时 / 改动量 / 退出条件齐备（§6）
- [x] 文档对接清单含新增 + 修改 + ADR 候选（§7）
- [x] 风险与回滚策略明确（§8）
- [ ] **Review 通过、转 Accepted 状态**（由维护者完成）

### 9.2 代码实施 DoD（每阶段独立验证）

#### 9.2.1 阶段 L1 DoD

- [ ] `linctl new` 落地 `linctl.yaml` + `.linctl/lock.json`
- [ ] `linctl status` 命令存在，输出 5 类信息（ok/lockfileValid/drift/summary/exitCode）
- [ ] `internal/lockfile/` 单元测试覆盖率 ≥85%
- [ ] 文档 §3 §4.1 落地

#### 9.2.2 阶段 L2 DoD

- [ ] 三向合并 5 case 决策表全部覆盖
- [ ] `linctl sync` / `linctl resolve` 实现并通过 E2E 场景 B/C
- [ ] `webserver.go` 中所有业务文件标注为 `once`、所有 lin/用户共管文件标注为 `managed`
- [ ] 文档 §4.2 + §5 + ADR-007 落地

#### 9.2.3 阶段 L3 DoD

- [ ] embedded/local/git 三种 Catalog 都有 E2E 测试
- [ ] `linctl catalog clean` 可清理过期缓存
- [ ] 离线场景能降级到上次缓存
- [ ] 文档 §4.3 + ADR-008 落地

#### 9.2.4 阶段 L4 DoD

- [ ] `webserver.go` ≤ 150 行
- [ ] `MANIFEST.yaml` 完整覆盖 187 文件
- [ ] 新增模板文件只需改 YAML 不需改 Go 代码（验证：加 1 个 demo 文件）
- [ ] 文档 §4.4 + ADR-009 落地

#### 9.2.5 阶段 L5 DoD

- [ ] `linctl g resource Post` 等价于现有 `linctl add api Post`
- [ ] `linctl g middleware Auth` 单文件级生成 + 自动注入 server.go
- [ ] `schema.json` 改字段后 CLI flags 自动派生（无须改 Go 代码）
- [ ] 内置 8 个 schematic：webserver / worker / cli / resource / middleware / handler / cron-job / kafka-topic
- [ ] 第三方 catalog 提供新 schematic 时立即可用（E2E 验证：用 LocalCatalog 加自定义 schematic）
- [ ] 文档 §4.6 + ADR-010 落地

#### 9.2.6 阶段 L6 DoD（远期）

- [ ] `linctl init-autoupdate` 命令存在，scaffold 出可工作的 GitHub Action workflow
- [ ] 独立仓库 `clin211/linctl-update-action` 发布
- [ ] 端到端验证：模板新版发布后，使用了 autoupdate 的项目自动开 PR
- [ ] 文档 ADR-011 落地

---

## 10. 修订历史

| 日期 | 变更 | 负责人 |
| --- | --- | --- |
| 2026-04-28 | 初稿（Proposed 状态） | @clin211 + assistant |
| 2026-04-28 | **v0.2 增强版**：调研 goctl / Nest CLI / Kubebuilder 后大幅修订 — 新增 §3.5 业界对比、§4.5 template 子命令族、§4.6 Schematic、改 §4.2 conflict 策略为 git merge-file、增加阶段 L5/L6、新增 ADR-010/011 候选 | @clin211 + assistant |
| TBD | Review 通过，转 Accepted | TBD |
| TBD | 阶段 L1 落地完成 | TBD |
| TBD | 阶段 L6 落地完成，本文档转 Archived | TBD |

---

_Last reviewed: 2026-04-28_
