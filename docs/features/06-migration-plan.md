# 06. 重构迁移计划

> **前置阅读**：[00-refactor-rationale.md](./00-refactor-rationale.md) ~ [05-registration-strategy.md](./05-registration-strategy.md)
>
> 本文档定义从**当前 lin** 到**重构版 lin** 的迁移步骤、阶段拆分、风险与回滚策略。

---

## 1. 迁移基本原则

| 原则 | 说明 |
| --- | --- |
| **分支隔离** | 重构在独立分支（如 `refactor/v2-skeleton-only`），主线代码不受影响 |
| **小步快跑** | 每个 Phase 独立可验证，避免一次性大爆炸 |
| **测试先行** | 先写好新版的 E2E 测试，再开始拆 / 改 |
| **旧版本保留** | 当前主分支作为 v1，重构版发布为 v2；老用户可继续用 v1 |
| **模板优先** | 模板内容（templates/project, templates/resource）是核心资产，保留并完善 |

---

## 2. Phase 拆分（5 个阶段）

```
                                       ┌──── 总周期：~3-4 周 ───┐
Phase 1  Phase 2     Phase 3      Phase 4         Phase 5
基础架构  new 命令    add 命令     lint/doctor      上线/文档
─────── ──────────  ──────────  ────────────  ────────────
  3d       4d           5d            2d             3d
```

### Phase 1：基础架构搭建（3 天）

**目标**：建立新版骨架，定义关键抽象。

**任务**：

| # | 任务 | 涉及文件 |
| --- | --- | --- |
| 1.1 | 创建 `refactor/v2-skeleton-only` 分支 | git |
| 1.2 | 新建 `internal/scaffold/` 包 + `Context`/`Plan` 类型 | scaffold/context.go, plan.go |
| 1.3 | 新建 `internal/pkg/{tpl,fsx,logx,errs}` 基础库（沿用现有 `linctlerr`/`fs` 内核） | pkg/* |
| 1.4 | 新建 `internal/cli/root.go`（cobra 根命令 + 全局 flag） | cli/root.go |
| 1.5 | 新建 `cmd/lin/main.go`（替换 `cmd/linctl/main.go`） | cmd/lin/main.go |
| 1.6 | 引入 `dave/dst`、`bufbuild/protocompile`、`iancoleman/strcase`、`jinzhu/inflection` | go.mod |

**Phase 1 完成后**：`lin --help` 可以打印（即使啥都不能干）。

---

### Phase 2：`lin new` 命令（4 天）

**目标**：完成项目骨架生成端到端流程。

**任务**：

| # | 任务 | 涉及文件 |
| --- | --- | --- |
| 2.1 | 复用并精简 `internal/template/templates/project/` 下的模板，新增缺失的（按 [03 §10](./03-resource-scaffold.md) 清单） | templates/project/ |
| 2.2 | 实现 `scaffold/project.go`：`NewProject(ctx) error` | scaffold/project.go |
| 2.3 | 实现 `scaffold/render.go`：模板渲染封装 | scaffold/render.go |
| 2.4 | 实现 `pkg/tpl/loader.go`：embed + 外部目录覆盖 | pkg/tpl/loader.go |
| 2.5 | 实现 `cli/new.go`：解析 flag + 调用 scaffold | cli/new.go |
| 2.6 | 在 `internal/templates/project/` 中确保**初始 `biz.go`/`store.go`/`register.go` 包含锚点注释**（详见 [05 §4](./05-registration-strategy.md)） | templates/project/internal/app/biz/biz.go.tpl 等 |
| 2.7 | E2E 测试：`lin new` 生成项目可 `go build` | tests/e2e/new_test.sh |

**Phase 2 完成标准**：

```bash
lin new myblog --module github.com/test/myblog
cd myblog && go mod tidy && go build ./...
# 必须通过
```

---

### Phase 3：`lin add` 命令 + AST 注入（5 天）

**目标**：完成增量资源添加 + AST 注入。

**任务**：

| # | 任务 | 涉及文件 |
| --- | --- | --- |
| 3.1 | 完善 `internal/templates/resource/` 模板（13 个文件） | templates/resource/ |
| 3.2 | 实现 `scaffold/resource.go`：`AddResource(ctx, name, opts)` | scaffold/resource.go |
| 3.3 | 实现 `scaffold/context.go::LoadContext`：从 go.mod / cmd/* 推断元信息 | scaffold/context.go |
| 3.4 | 实现 `ast/mutator_interface.go` + 单测 | ast/mutator_interface.go |
| 3.5 | 实现 `ast/mutator_proto.go` + 单测 | ast/mutator_proto.go |
| 3.6 | 实现 `ast/mutator_register.go` + 单测 | ast/mutator_register.go |
| 3.7 | 实现 `ast/injector.go` + `ast/guard.go`：编排 + 幂等 + 回滚 | ast/* |
| 3.8 | 实现 `cli/add.go`：参数解析 + 调用 scaffold | cli/add.go |
| 3.9 | E2E 测试：`new` → `add Post Comment` → `go build` | tests/e2e/add_test.sh |
| 3.10 | E2E 测试：`add Post; add Post`（幂等） | tests/e2e/add_idempotent_test.sh |

**Phase 3 完成标准**：

```bash
lin new myblog --module github.com/test/myblog
cd myblog
lin add Post Comment
go mod tidy && go build ./...

# 检查注入完整性
grep -q "PostV1() postv1.PostBiz" internal/myblog/biz/biz.go
grep -q "Posts() PostStore" internal/myblog/store/store.go
grep -q 'import "post.proto";' pkg/api/myblog/v1/myblog.proto
grep -q "RegisterErrors(PostErrors()...)" internal/pkg/errno/register.go

# 幂等
lin add Post   # 期望：⊝ skipped 全部
```

---

### Phase 4：`lin lint` + `lin doctor`（2 天）

**目标**：完成辅助命令。

**任务**：

| # | 任务 | 涉及文件 |
| --- | --- | --- |
| 4.1 | 实现 `check/lint.go`：目录结构 + AST 完整性 | check/lint.go |
| 4.2 | 实现 `check/doctor.go`：环境工具检测 | check/doctor.go |
| 4.3 | 实现 `cli/lint.go` / `cli/doctor.go` | cli/* |
| 4.4 | 实现 `--fix` 模式：自动补全错误注册 | check/lint.go |
| 4.5 | 实现 `cli/version.go` + `cli/completion.go` | cli/* |
| 4.6 | E2E 测试 | tests/e2e/lint_test.sh, doctor_test.sh |

---

### Phase 5：上线 + 文档（3 天）

**目标**：旧版归档、新版发布、文档完善。

**任务**：

| # | 任务 | 涉及 |
| --- | --- | --- |
| 5.1 | 删除/归档旧模块（详见 [§3 删除清单](#3-删除清单)） | 大批文件 |
| 5.2 | 更新 `lin/README.md` | README.md |
| 5.3 | 更新 `lin/Makefile`（删除 `linctl`、改为 `lin`） | Makefile |
| 5.4 | 归档 `lin/docs/` 旧版主线文档到 `lin/docs/legacy/` | docs/ |
| 5.5 | 在 `lin/docs/features/README.md` 标记所有文档为 Stable | docs/features/README.md |
| 5.6 | 修订 `CONTRIBUTING.md` | CONTRIBUTING.md |
| 5.7 | 发布 v2.0.0-rc1 tag | git tag |
| 5.8 | CI 跑通：`go build` / `go test` / E2E | .github/workflows/ |
| 5.9 | 在 README 顶部添加 v1 → v2 迁移说明 | README.md |

---

## 3. 删除清单

### 3.1 完全删除的目录

| 目录 | 行数 | 删除理由 |
| --- | --- | --- |
| `internal/templatesync/` | 2,352 | 模板上游同步 — 不做 |
| `internal/gitmerge/` | 550 | 3-way merge — 不做 |
| `internal/orchestrator/` | 271 | 命令编排器 — 不必要 |
| `internal/feature/` | 350 | Feature 注册中心 — 删除 |
| `internal/component/` | 1,190 | 三种组件抽象 — 简化为单一 webserver 风格 |
| **小计** | **4,713 行** | |

### 3.2 大幅精简的目录

| 目录 | 当前 | 目标 | 处置 |
| --- | --- | --- | --- |
| `internal/cli/` | 1,776 | ~600 | 删除 plan/apply/import/upgrade/plugin/add-feature/add-webserver/add-worker/add-cli 等 |
| `internal/codegen/` | 565 | 0 | 删除（plan/apply 流水线不再需要） |
| `internal/project/` | 1,057 | 0 | 删除（不再有配置文件，元信息由 `scaffold/context.go` 推断） |
| `internal/validate/` | 435 | 0 | 删除（schema 校验不再需要） |
| `internal/template/` | 807 | ~200 | 仅保留 funcMap 与简化的渲染逻辑，移到 `internal/pkg/tpl/` |
| `internal/ast/` | 411 | ~500 | 重构：扩展为 3 个 mutator + injector + guard |
| `internal/fs/` | 410 | ~250 | 简化：保留 SafeJoin / 原子写，移到 `internal/pkg/fsx/` |

### 3.3 完全保留的目录

| 目录 | 处置 |
| --- | --- |
| `internal/version/` | 保留 |
| `internal/linctlerr/` | 重命名为 `internal/pkg/errs/`，逻辑不变 |
| `cmd/linctl/` | 重命名为 `cmd/lin/`，main.go 简化 |
| `tests/` | 保留并扩展 |
| `tools/` | 保留 |

### 3.4 docs 处置

| 路径 | 处置 |
| --- | --- |
| `lin/docs/00-overview.md` ~ `15-security-model.md` | 移到 `lin/docs/legacy/`（17 个文件） |
| `lin/docs/META-*.md` | 移到 `lin/docs/legacy/` |
| `lin/docs/adr/` | **保留**（架构决策记录，永远有价值） |
| `lin/docs/diagrams/` | 部分保留（可重用的图），其余移到 legacy |
| `lin/docs/plans/` | 移到 legacy（如有） |
| `lin/docs/features/` | **新主线文档** |

---

## 4. 关键风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
| --- | --- | --- | --- |
| 用户依赖被删除的功能（如 plan/apply） | 低 | 中 | 用户已确认不需要；v1 分支保留可用 |
| AST 注入实现不稳定 | 中 | 高 | 单测覆盖 ≥ 90%；E2E 含幂等 + 回滚场景 |
| 模板渲染产生不能编译的代码 | 中 | 高 | E2E 必跑 `go build`；CI 多平台验证 |
| 删除模块时遗留对其的 import | 低 | 低 | `golangci-lint` 自动捕获 |
| 锚点注释被用户误删导致重构后无法 add | 中 | 中 | `lint --fix` 提供恢复模板；文档显眼说明 |
| 模板与 miniblog-v4 风格漂移 | 中 | 中 | 模板提交前与 miniblog-v4 实际代码做 diff |
| 重构周期超出预期 | 中 | 低 | Phase 拆分；每 Phase 独立验证 |

---

## 5. 回滚策略

| 触发条件 | 回滚动作 |
| --- | --- |
| Phase 任意阶段发现关键设计错误 | 在分支内回退到 Phase 边界点 |
| 重构完成但发现重大缺陷 | v2 分支不合入 main，main 继续保留 v1 |
| 已发布 v2.0.0-rc1 后发现问题 | 标记 rc1 为 deprecated，发布 rc2 修正 |
| 完全放弃重构 | 保留 features/ 文档作为决策记录；废弃 refactor 分支 |

---

## 6. 沟通与里程碑

| 里程碑 | 交付物 | 验收 |
| --- | --- | --- |
| **M1：Phase 1 完成** | 分支建立 + 骨架代码 | `lin --help` 输出预期命令 |
| **M2：Phase 2 完成** | `lin new` 可用 | E2E 通过 |
| **M3：Phase 3 完成** | `lin add` + AST 注入可用 | E2E 通过（含幂等） |
| **M4：Phase 4 完成** | lint/doctor/version/completion 全套 | E2E 全绿 |
| **M5：v2.0.0-rc1 发布** | 删除清单全部生效 + 文档归档 | tag 可下载，新人按 README 30 分钟出 demo |

---

## 7. 与 miniblog-v4 的协同

模板的核心来源是 miniblog-v4，重构期间应：

1. **严格 mirror**：每次模板改动应能在 miniblog-v4 找到对应风格。
2. **diff 验证**：Phase 2 完成后，用 `lin new` 生成的项目应**能与 miniblog-v4 等价**（除业务逻辑外）。
3. **回流改动**：若发现 miniblog-v4 有需要改进的地方（如更清晰的注释），先在 miniblog-v4 改，再回流到 lin 模板。

---

## 8. 工作量估算汇总

| Phase | 估计工作量 | 关键里程碑 |
| --- | --- | --- |
| Phase 1 | 3d | 骨架可启动 |
| Phase 2 | 4d | `lin new` 可用 |
| Phase 3 | 5d | `lin add` 可用 |
| Phase 4 | 2d | lint/doctor 可用 |
| Phase 5 | 3d | 旧版归档 + v2 发布 |
| **小计** | **17d**（约 3-4 个工作周） | v2.0.0-rc1 |

> 估算前提：单人全职。如多人协作（如 1 人做模板 + 1 人做 AST），可缩短至 ~2 周。

---

## 9. 重构后的代码量验证（DoD）

迁移完成后，使用以下命令验证目标达成：

```bash
cd lin

# 1) 总代码量
find internal cmd -name "*.go" -not -name "*_test.go" | xargs wc -l | tail -1
# 期望: ≤ 4,000 行

# 2) 模块数
ls -d internal/*/ | wc -l
# 期望: ≤ 7 个

# 3) 子命令数
grep -l "cobra.Command" internal/cli/*.go | wc -l
# 期望: ≤ 6 个文件（new/add/lint/doctor/version/completion）

# 4) 直接依赖数
go list -m all | tail -n +2 | head -20 | wc -l
# 期望: ≤ 8 个

# 5) 二进制大小
go build -ldflags "-s -w" -o /tmp/lin ./cmd/lin
ls -lh /tmp/lin
# 期望: ≤ 6 MB
```

---

## 10. 后续演进（v2 发布后）

| 时间 | 任务 |
| --- | --- |
| v2.0 发布后 1 个月 | 收集用户反馈，调整模板细节 |
| v2.0 发布后 3 个月 | 评估是否需要支持 grpc 框架（当前仅 gin） |
| v2.0 发布后 6 个月 | 评估是否需要 `lin upgrade-templates`（让旧项目跟进模板修复） |
| v2.0 发布后 1 年 | 评估是否需要回流"演进闭环"（v3 可能） |

> **核心原则**：演进基于真实使用反馈，**不为了"全能"而提前抽象**。

---

## 11. SemVer 版本承诺

`lin` 严格遵循 [SemVer 2.0](https://semver.org)，且赋予 lin 特定语义：

### 11.1 版本号 → 变更范围映射

| 字段 | 含义 | 影响 | 示例 |
| --- | --- | --- | --- |
| `MAJOR`（v2 → v3） | **不兼容**变更 | 命令名/flag/退出码/锚点格式/模板布局 | 删除 `lin add`；锚点格式改为 `// lin>>` |
| `MINOR`（v2.0 → v2.1） | **向后兼容的功能增强** | 新命令、新 flag、新可选层级 | 增加 `lin add --with grpc`；新 mutator |
| `PATCH`（v2.0.0 → v2.0.1） | **向后兼容的 bug 修复** | 修 bug、改优化、不影响生成结果 | 修 AST 边界 bug；改善错误信息 |

### 11.2 模板内容变化的 SemVer 语义

由于「模板内容」是 lin 的核心资产，模板变化按以下规则映射版本号：

| 模板变化类型 | 影响范围 | 版本字段 |
| --- | --- | --- |
| 修 bug（错误注释、typo） | 已生成项目不影响 | PATCH |
| 给现有变量增加 funcMap 函数 | 已生成项目不影响（新模板才用） | PATCH |
| 给模板增加新可选 feature 分支 | 已生成项目不影响 | MINOR |
| **改 biz.go.tpl 默认结构**（如 `biz` 改为 `Biz`） | 老项目重跑 `lin add` 仍可用 | MINOR |
| **删除某个 feature**（如废弃 `--features=user`） | 既有项目用 `--features=user` 会失败 | MAJOR |
| **锚点注释格式变化** | 既有项目无法 `lin add` 直到迁移 | MAJOR |

### 11.3 已生成项目的兼容性承诺

| 维度 | 承诺 |
| --- | --- |
| **PATCH 升级** | 既有项目 `lin add` **完全无感**（仅修 bug） |
| **MINOR 升级** | 既有项目 `lin add` **不破坏既有代码**；可能在新生成的资源中体现新风格（不强制替换老资源） |
| **MAJOR 升级** | 不保证向后兼容；提供独立 [迁移指南](#114-跨-major-迁移指南)；旧 MAJOR 分支按 §11.5 定义获得维护 |

### 11.4 跨 MAJOR 迁移指南

每个 MAJOR 发布伴随：

1. **`docs/migration-vN-to-vN+1.md`** 文档
2. **`lin lint --migrate-from vN`** 子命令（Phase 2+ 评估）
3. **CHANGELOG 中的破坏性变更清单**

### 11.5 v1.x → v2.0 的关系

| v1.x（当前主线 lin） | v2.0（本次重构） |
| --- | --- |
| 10,525 行 / 14 模块 / 15+ 命令 | 3,160 行 / 5–7 模块 / 6 命令 |
| `linctl` 二进制名 | `lin` 二进制名 |
| 有 plan/apply/templatesync/feature | 全部删除 |

**v1 用户的迁移路径**：

| 场景 | 推荐 |
| --- | --- |
| 已用 v1 生成项目，不想升级 | 保留使用 v1 二进制；详见下方维护承诺 |
| 已用 v1 生成项目，想用 v2 | 用 v2 重新 `lin new` 一个空项目 → 手工迁移业务逻辑 |
| 新项目 | 直接用 v2 |

> **关键**：**不提供** v1 → v2 的自动迁移工具。两者哲学差异过大，自动迁移成本远超手工。

#### 11.5.1 旧 MAJOR 分支维护承诺

每个 MAJOR 版本（如 v1.x）发布后享受以下维护：

| 时段（自上一个 MAJOR 发布起算） | 维护内容 |
| --- | --- |
| 0 – 6 个月 | **全维护**：bug fix（PATCH 发布）+ 安全补丁 + 二进制可下载 |
| 6 – 12 个月 | **仅安全补丁**：critical CVE 修复 + 二进制可下载（不再修一般 bug） |
| 12 个月 之后 | **冻结**：仅二进制可下载（GitHub Releases 永久保留），不再发布新版本 |

例：v2.0.0 于 2026-04 发布 → v1.x 分支：
- 2026-04 ~ 2026-10：bug fix + 安全
- 2026-10 ~ 2027-04：仅安全
- 2027-04 之后：冻结，仅可下载

### 11.6 RC / Pre-release 阶段

`v2.0.0-rc1` ~ `v2.0.0-rcN`：

- 命令、flag、退出码、锚点格式**已冻结**
- 模板内容可能小幅迭代（仅 PATCH 类）
- rc 期间收集真实反馈，正式 v2.0.0 发布前不再做 MAJOR/MINOR 变更

---

## 12. CI 集成示例

### 12.1 lin 自身仓库的 CI（GitHub Actions）

```yaml
# lin/.github/workflows/ci.yml
name: CI
on:
  push: { branches: [main] }
  pull_request:

jobs:
  build-test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ matrix.go }}' }
      - run: go build -ldflags "-s -w" -o lin ./cmd/lin
      - run: go test ./...
      - run: go vet ./...

  e2e:
    runs-on: ubuntu-latest
    needs: build-test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go build -o lin ./cmd/lin
      - run: bash tests/e2e/new_test.sh
      - run: bash tests/e2e/add_test.sh
      - run: bash tests/e2e/add_idempotent_test.sh

  size-budget:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - name: Build & check size
        run: |
          go build -ldflags "-s -w" -o lin ./cmd/lin
          size=$(stat -c %s lin)
          max=$((6 * 1024 * 1024))   # 6 MB
          if [ "$size" -gt "$max" ]; then
            echo "::error::lin binary $size bytes exceeds 6MB budget"
            exit 1
          fi
          echo "::notice::lin binary size: $((size / 1024)) KB"
```

### 12.2 用户项目 CI 示例（用 `lin lint`）

```yaml
# <user-project>/.github/workflows/lin-check.yml
name: lin scaffold check
on: [pull_request]

jobs:
  lin-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install lin
        run: |
          curl -sSL https://github.com/<org>/lin/releases/latest/download/lin-linux-amd64 \
            -o /usr/local/bin/lin
          chmod +x /usr/local/bin/lin
          lin version
      - name: lin doctor (offline)
        run: lin doctor --offline --report-format json > doctor.json
      - name: lin lint
        run: lin lint --report-format json > lint.json
      - name: Annotate PR
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const lint = JSON.parse(fs.readFileSync('lint.json', 'utf8'));
            for (const issue of lint.issues || []) {
              core.error(`${issue.id}: ${issue.message}`, {
                file: issue.file,
                startLine: issue.line || 1,
              });
            }
```

### 12.3 GitLab CI 示例（用户项目）

```yaml
# .gitlab-ci.yml
stages: [check, build]

lin-check:
  stage: check
  image: golang:1.22-alpine
  script:
    - wget -O /usr/local/bin/lin https://gitlab.com/<org>/lin/-/releases/permalink/latest/downloads/lin-linux-amd64
    - chmod +x /usr/local/bin/lin
    - lin doctor --offline
    - lin lint --report-format json | tee lint.json
  artifacts:
    when: always
    paths: [lint.json]
    reports:
      codequality: lint.json
```

### 12.4 关键约束

| 规则 | 实现 |
| --- | --- |
| CI 中 `lin` 命令必须**非交互** | 自动检测非 TTY；或显式 `--non-interactive` |
| `lin doctor` 在内网 / airgap | 用 `--offline` 跳过网络检查（详见 [02 §6.2.1](./02-command-set.md#621---offline-行为)） |
| 退出码为信号源 | CI 基于 `exit_code` 判断；JSON 报告作为补充 |
| 锁定 lin 版本 | 项目 `Makefile` 中固化 `LIN_VERSION := 2.0.0`，下载时校验 |

---

_Last reviewed: 2026-04-29_
