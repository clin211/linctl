# META · 模板上游同步设计书 (Template Upstream Sync)

**日期**：2026-04-28
**目的**：锁定 `miniblog-v4` → `lin/internal/template/templates/web-gin/` 的上游同步机制，与 `linctl` 主二进制集成
**适用范围**：`lin/internal/templatesync/`（新增包）、`lin/internal/cli/cmd_internal_templatesync.go`、根目录 CI
**状态**：Proposed（待 review）
**前置阅读**：[META-template-lifecycle-2026-04-28.md](./META-template-lifecycle-2026-04-28.md)（终端用户视角的项目升级；本文是其**正交补充**：lin 维护者视角的模板上游同步）

---

## 0. 阅读路径

| 你想…… | 看哪一节 |
| --- | --- |
| 看为什么要做这件事（与之前那份设计的区别） | [§1 真问题画像](#1-真问题画像) |
| 看选了什么方案、否决了哪些 | [§2 备选方案矩阵](#2-备选方案矩阵) |
| 看核心数据模型（sync.yaml） | [§3 数据模型](#3-数据模型) |
| 看 transform 怎么工作 | [§4 Transform 规则集](#4-transform-规则集) |
| 看 CLI 命令长什么样 | [§5 CLI 命令族](#5-cli-命令族) |
| 看 CI 怎么集成 | [§6 CI 集成](#6-ci-集成) |
| 看与生命周期文档的关系 | [§7 与已有文档的关系](#7-与已有文档的关系) |
| 看怎么落地、什么时候做 | [§8 实施路线（3 阶段）](#8-实施路线3-阶段) |
| 看风险与不做的事 | [§9 风险、回滚与非目标边界](#9-风险回滚与非目标边界) |
| 看怎么算"完成" | [§10 验收标准](#10-验收标准) |
| 看变更历史 | [§11 修订历史](#11-修订历史) |

---

## 1. 真问题画像

### 1.1 关键事实清单

| 实体 | 路径 | 角色 |
| --- | --- | --- |
| **miniblog-v4** | `osbuilder-demo/miniblog-v4/` | 真实可跑的"模板源"项目（demo / 沙盒 / 实验地） |
| **web-gin 模板** | `lin/internal/template/templates/web-gin/` | miniblog-v4 的**模板化镜像**（130 个 `.tpl` 文件） |
| **历史尝试** | 废弃的 `scripts/sync_onexstack_pkg.sh` | 见 `webserver.go:600-603` 注释，曾用于"运行时同步"，因引入版本不一致而废弃 |
| **存在分叉** | `miniblog-v4/internal/pkg/contextx/contextx.go`（英文注释）vs `lin/.../contextx.go.tpl`（中文注释） | 两边已分叉；不能简单 sed 覆盖 |

### 1.2 真问题陈述

> miniblog-v4 演进（修 bug / 加功能 / 升级依赖 / 重构）后，怎么把改动**结构化、半自动化、可追溯地**同步到 `lin/templates/web-gin/`，并且**集成到 linctl CLI 中**（而非孤立脚本）？

### 1.3 核心约束（用户提的硬约束）

| # | 约束 | 备注 |
| --- | --- | --- |
| **C1** | **不能用 git submodule** | 用户明确否定：分叉历史 + 子模块状态难管 + monorepo 内反而更乱 |
| **C2** | **不能简单 sed 覆盖** | lin/web-gin/ 已有人工模板化处理 + 中英文翻译，覆盖会丢 |
| **C3** | **必须支持双向 diff** | 每次同步前能看到 miniblog-v4 改了什么、对应 web-gin 该怎么变 |
| **C4** | **必须与 lin CLI 集成** | 做成 `linctl` 子命令而非独立 shell 脚本 |
| **C5** | **必须 CI 友好** | PR 改 miniblog-v4 但没同步 web-gin 时报警 |
| **C6** | **保留 lin-specific 改动** | lin 仓库可能给某些模板加了特定改动（如中文注释、`{{ }}` 占位），不能被覆盖 |

### 1.4 五大具体痛点

| 编号 | 痛点 | 后果 |
| --- | --- | --- |
| **U-P1** | 每次 miniblog-v4 升级要手工 diff + sed 改 | 易漏 / 易错 / 不可追溯 |
| **U-P2** | lin 维护者不知道哪些 web-gin 文件需要同步 | 模板逐渐落后于 miniblog-v4，分叉越来越严重 |
| **U-P3** | 无法表达"这个文件来自上游 / 这个是 lin 自创 / 这个是双方共维护" | owner 模糊导致冲突解决靠记忆 |
| **U-P4** | 没有 CI 闸门 | 上游改了但模板没改，PR 仍能合入 |
| **U-P5** | miniblog-v4 加了新文件（如新 `pkg/`），人工把它纳入 sync 流程低效 | 漏同步是常态 |

---

## 2. 备选方案矩阵

> 调研于 2026-04-28。所有方案以"是否解决 §1.3 六个硬约束"为评判标准。

| 方案 | 解决 C1 (no submodule) | 解决 C2 (preserve lin-side) | 解决 C3 (diff) | 解决 C4 (CLI 集成) | 解决 C5 (CI) | 解决 C6 (owner 区分) | 评级 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **A. Git submodule** | ❌ | ⚠️ | ⚠️ | ❌ | ⚠️ | ❌ | 否决（用户硬约束） |
| **B. Git subtree split** | ✅ | ⚠️ | ⚠️ | ❌ | ⚠️ | ❌ | 否决（不擅长跨目录映射 + transform） |
| **C. 手工 sed 脚本** | ✅ | ❌ | ❌ | ❌ | ⚠️ | ❌ | 否决（C2/C4/C6 全失败） |
| **D. Symlink** | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | 否决（go:embed 不跟随；无 transform） |
| **E. 通用 vendor 工具（vendir / glide）** | ✅ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ❌ | 否决（动态 transform 弱；无 owner 模型） |
| **F. 结构化同步工具 + Manifest 规则 + 3-way merge** ⭐ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **采纳** |
| **G. F + LLM 自动模板化新文件** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 远期增强 |

---

## 3. 数据模型

> 两个文件 + 一个目录约定。

### 3.1 `sync.yaml`（同步规则声明）

**位置**：`lin/internal/template/templates/web-gin/.sync.yaml`

**完整范例**（截取自真实 contextx 文件对子）：

```yaml
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: web-gin-upstream
  description: "Sync miniblog-v4 → lin/templates/web-gin/"

upstream:
  name: miniblog-v4
  rootPath: ../../../../miniblog-v4         # 相对本 sync.yaml
  moduleOld: github.com/clin211/miniblog-v4
  binaryNameOld: blog-apiserver
  componentNameOld: apiserver

# 全局变换：对所有 owner=upstream / shared 的文件默认应用
defaultTransforms:
  - kind: rewriteImports                     # AST 级 import 改写
    from: '{{ .Upstream.ModuleOld }}'
    to: '{{`{{ .Project.Module }}`}}'

  - kind: replaceLiteral                     # 字面量替换
    pairs:
      - from: blog-apiserver
        to: '{{`{{ .Component.Name }}`}}'
      - from: apiserver
        to: '{{`{{ .Component.Name }}`}}'

  - kind: stripCopyrightHeader               # 去掉上游版权头

  - kind: addExtension                       # 加 .tpl 后缀
    suffix: .tpl
    appliesTo: ['*.go', '*.proto', '*.yaml', '*.json']

  - kind: insertLinctlHeader                 # 加上 linctl 标记
    template: |
      // 此文件由 linctl 模板系统从 miniblog-v4@{{.UpstreamCommit}} 自动同步生成。
      // 上游路径：{{.UpstreamPath}}
      // 同步规则：lin/internal/template/templates/web-gin/.sync.yaml

# ============ 文件级映射 ============
files:
  # 例 1：完全跟随上游（owner=upstream）
  - src: pkg/log/log.go
    dst: pkg/log/log.go.tpl
    owner: upstream

  # 例 2：双方共维护（owner=shared）
  #   - 上游改了 → 走 3-way merge
  #   - lin 改了 → 保留
  - src: internal/pkg/contextx/contextx.go
    dst: internal/pkg/contextx/contextx.go.tpl
    owner: shared
    extraTransforms:
      - kind: translateComments              # 中英文注释翻译（半自动）
        triggeredBy: manual                  # 不在自动 sync 路径上跑
        cache: .linctl/translation-cache.yaml

  # 例 3：上游文件需要拆分到多个 dst
  - src: pkg/server/server.go
    splits:
      - dst: pkg/server/server.go.tpl
        sectionMatch: '// SECTION: core'
      - dst: pkg/server/http_server.go.tpl
        sectionMatch: '// SECTION: http'

# ============ lin-specific 文件（不来自上游）============
linSpecific:
  - dst: pkg/util/lint/hash/hash.go.tpl
    reason: "Linctl-specific lint analyzer; not in miniblog-v4 upstream"

# ============ 显式忽略（上游有但故意不模板化）============
ignored:
  - src: internal/apiserver/wire_gen.go
    reason: "Generated by wire; users run make wire after linctl new"
  - src: configs/blog-apiserver.docker.yaml
    reason: "Has its own template path in templates/component/webserver/configs/"

# ============ 新文件检测策略 ============
newFilePolicy:
  # 上游加了新文件、sync.yaml 没声明时怎么办？
  default: warn                              # warn / error / autoAdd
  
  # autoAdd 时的默认 owner
  autoAddOwner: upstream
```

### 3.2 `.linctl/upstream-sync.lock.json`（同步状态，机器写）

**位置**：`lin/.linctl/upstream-sync.lock.json`（在 lin 仓库内，提交进 git）

```json
{
  "schemaVersion": "1",
  "syncManifest": "internal/template/templates/web-gin/.sync.yaml",
  "lastSyncAt": "2026-04-28T13:45:00+08:00",
  "linctlVersion": "v0.3.0",
  "upstream": {
    "name": "miniblog-v4",
    "rootPath": "../../../../miniblog-v4",
    "lastSyncedCommit": "abc12345",
    "lastSyncedTreeHash": "sha256:..."
  },
  "files": {
    "internal/pkg/contextx/contextx.go.tpl": {
      "src": "internal/pkg/contextx/contextx.go",
      "owner": "shared",
      "srcHashAtSync": "sha256:...",
      "dstHashAtSync": "sha256:...",
      "transformsApplied": ["rewriteImports", "stripCopyrightHeader", "addExtension"],
      "lastSyncAt": "2026-04-28T13:45:00+08:00"
    },
    "pkg/log/log.go.tpl": {
      "src": "pkg/log/log.go",
      "owner": "upstream",
      "srcHashAtSync": "sha256:...",
      "dstHashAtSync": "sha256:...",
      "transformsApplied": ["rewriteImports", "stripCopyrightHeader", "addExtension"],
      "lastSyncAt": "2026-04-28T13:45:00+08:00"
    }
  }
}
```

**字段说明**：

| 字段 | 用途 |
| --- | --- |
| `lastSyncedCommit` | 上次同步时 miniblog-v4 的 git commit（用于 status 时算 upstream changed since） |
| `srcHashAtSync` | 上次同步时上游文件的 hash（**作为 3-way merge 的 base**） |
| `dstHashAtSync` | 上次同步后写盘的 dst hash（用于检测 lin-side 是否有 lin-specific 改动） |
| `owner` | 决定 sync 策略（upstream → 直接覆盖；shared → 3-way merge；linSpecific → 跳过） |

### 3.3 `.linctl/translation-cache.yaml`（翻译缓存，可选）

> 仅当 transform 含 `translateComments` 时使用。避免每次 sync 都重新翻译。

```yaml
# .linctl/translation-cache.yaml
"Define keys for the context.": "定义所有 context key 类型。"
"WithUserID stores the user ID into the context.": "WithUserID 把 user ID 存入 context 并返回新的 context。"
```

人工编辑此文件即可定制翻译。AI 辅助时只翻译缓存里没有的句子。

---

## 4. Transform 规则集

> 每个 transform 都是声明式 + 可组合的。新增 transform 只需在 `internal/templatesync/transforms/` 加文件并注册。

### 4.1 内置 transforms

| Transform | 作用 | 实现方式 |
| --- | --- | --- |
| `rewriteImports` | Go AST 级 import 改写（`github.com/clin211/miniblog-v4` → `{{ .Project.Module }}`） | `golang.org/x/tools/go/ast/astutil` |
| `replaceLiteral` | 字面量替换（项目名 / binary 名等） | 字符串处理（文件类型敏感） |
| `templatizeLiteral` | 把 hardcode 替换为 `{{ .X }}`（含转义处理） | 字符串 + 转义 |
| `stripCopyrightHeader` | 去掉文件顶部版权头（`// Copyright ...` 块） | 正则匹配头部连续注释块 |
| `insertLinctlHeader` | 加上 "auto-synced from miniblog-v4@\<sha\>" 标记 | 头部追加 |
| `addExtension` | 加 `.tpl` 后缀（按 appliesTo 过滤） | 字符串处理 |
| `stripExtension` | 反向（lin → miniblog-v4 路径推导用） | 字符串处理 |
| `regexReplace` | 通用正则替换（兜底） | regexp |
| `gofmt` | 渲染后跑 `go/format`（不破坏可编译性） | go/format |
| `translateComments` | 英→中注释翻译（半自动 + AI 辅助） | 离线工具，标 manual 触发 |

### 4.2 Transform 接口（Go 代码骨架）

```go
// internal/templatesync/transform/transform.go
package transform

import "context"

// Context 是 transform 执行时的运行时上下文。
type Context struct {
    UpstreamRoot string                       // 上游根目录绝对路径
    SrcPath      string                       // 当前 src 相对路径
    DstPath      string                       // 当前 dst 相对路径
    Owner        string                       // upstream / shared / linSpecific
    Manifest     *Manifest                    // sync.yaml 解析后的对象
}

// Transform 是单个变换规则。
type Transform interface {
    Kind() string                             // "rewriteImports" / "replaceLiteral" / ...
    Apply(ctx context.Context, tc *Context, content []byte) ([]byte, error)
}

// Registry 是 transform 注册中心。
type Registry struct {
    transforms map[string]TransformFactory
}

// TransformFactory 是从 sync.yaml 配置块构造 Transform 的工厂。
type TransformFactory func(rawConfig map[string]any) (Transform, error)
```

### 4.3 关键 transform 详解：`rewriteImports`

> 这是最重要的 transform：**不能用 sed**，否则会误改字符串字面量、注释里提到 module path 等情况。

**算法**：

```text
1. 用 go/parser 解析文件为 *ast.File
2. astutil.RewriteImport(file, oldPath, newPath)
3. 用 go/printer 输出
4. 检查输出包含 "{{" / "}}"（template 占位）→ 自动转义为 "{{`{{`}}" / "{{`}}`}}"
   （否则 lin 模板 engine 会把它当模板指令）
```

Go 代码骨架：

```go
// internal/templatesync/transform/rewrite_imports.go
package transform

import (
    "bytes"
    "go/parser"
    "go/printer"
    "go/token"
    "golang.org/x/tools/go/ast/astutil"
)

type RewriteImports struct {
    From string
    To   string
}

func (r *RewriteImports) Apply(_ context.Context, tc *Context, content []byte) ([]byte, error) {
    fset := token.NewFileSet()
    f, err := parser.ParseFile(fset, tc.SrcPath, content, parser.ParseComments)
    if err != nil {
        return nil, fmt.Errorf("parse %s: %w", tc.SrcPath, err)
    }

    if !astutil.RewriteImport(fset, f, r.From, r.To) {
        return content, nil  // 没有该 import，原样返回
    }

    var buf bytes.Buffer
    if err := printer.Fprint(&buf, fset, f); err != nil {
        return nil, err
    }
    return escapeTemplateLiterals(buf.Bytes()), nil
}

// escapeTemplateLiterals 把 Go 源码中可能的 "{{" / "}}" 转义，
// 防止后续模板渲染时被误解为 template 指令。
func escapeTemplateLiterals(content []byte) []byte {
    // ... see internal/template/funcmap.go for reference
}
```

---

## 5. CLI 命令族

> 所有命令通过 `linctl internal templatesync ...` 进入；`internal` 子命令族不出现在面向终端用户的 help 输出（隐藏）。

### 5.1 命令清单

| 命令 | 作用 | 退出码 |
| --- | --- | --- |
| `linctl internal templatesync status` | 显示当前同步状态：upstream 改动、lin-side 漂移、待 sync 文件数 | 0=in-sync / 1=needs sync / 2=conflicts |
| `linctl internal templatesync plan` | 干跑：算出每个文件的 action（不写盘） | 0=ok / >0=任何错误 |
| `linctl internal templatesync apply [--strategy=...]` | 应用 plan：含 git 3-way merge | 0=full sync / 1=conflicts deferred |
| `linctl internal templatesync check` | CI 用：上游有改动但 web-gin 没同步则失败 | 0=ok / 1=needs sync |
| `linctl internal templatesync add <upstream-path>` | 加新文件到 sync.yaml（自动推断 transform） | 0=ok |
| `linctl internal templatesync revert <dst>` | 把某 dst 还原到 lock 中的上次同步状态 | 0=ok |

### 5.2 典型工作流

```bash
# 维护者视角：刚拉了 miniblog-v4 的 PR 合入，想同步到模板

cd osbuilder-demo
git pull                                 # 假设 miniblog-v4/ 有 12 个文件改动

linctl internal templatesync status
# 输出：
# Upstream changes (since abc12345..def67890):
#   M  miniblog-v4/internal/pkg/contextx/contextx.go    (matches sync rule, owner=shared)
#   M  miniblog-v4/pkg/log/log.go                       (matches sync rule, owner=upstream)
#   A  miniblog-v4/pkg/cache/redis_cache.go             (NEW; no sync rule yet)
#   ...
# 
# lin-side state:
#   3 files have lin-specific edits (owner=shared, last sync at 2026-04-20)
# 
# Summary:
#   needs-sync: 9
#   needs-rule: 1 (run: linctl internal templatesync add)
#   in-sync:    120

linctl internal templatesync add miniblog-v4/pkg/cache/redis_cache.go
# 自动推断 dst = pkg/cache/redis_cache.go.tpl
# 自动应用 defaultTransforms
# 写入 sync.yaml + .lock.json

linctl internal templatesync plan
# 输出每个文件的 plan：create/update/merge/conflict

linctl internal templatesync apply --strategy=ask
# - upstream-only：直接覆盖
# - shared 无冲突：自动 merge
# - shared 有冲突：弹 $EDITOR 让维护者手动解决
```

### 5.3 包结构

```text
lin/internal/templatesync/
├── doc.go
├── manifest.go              # sync.yaml struct + Load + Validate
├── lockfile.go              # upstream-sync.lock.json
├── runner.go                # 端到端：load → diff → transform → merge → save
├── differ.go                # 上游 diff 检测（git or hash）
├── merger.go                # 调用 internal/gitmerge/（与 META-template-lifecycle 共用）
├── transform/
│   ├── transform.go         # interface + Registry
│   ├── rewrite_imports.go
│   ├── replace_literal.go
│   ├── strip_copyright.go
│   ├── insert_linctl_header.go
│   ├── add_extension.go
│   ├── translate_comments.go
│   └── *_test.go
└── *_test.go

lin/internal/cli/
└── cmd_internal_templatesync.go    # cobra 命令族（隐藏在 internal 父命令下）
```

---

## 6. CI 集成

### 6.1 GitHub Action 范例

```yaml
# .github/workflows/templatesync-check.yml
name: Template Upstream Sync Check

on:
  pull_request:
    paths:
      - 'miniblog-v4/**'
      - 'lin/internal/template/templates/web-gin/**'
      - 'lin/internal/template/templates/web-gin/.sync.yaml'

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build linctl
        run: cd lin && go build -o ../bin/linctl ./cmd/linctl

      - name: Check template sync status
        id: sync_check
        run: |
          ./bin/linctl internal templatesync check -o json > /tmp/sync.json
          cat /tmp/sync.json
        continue-on-error: true

      - name: Comment on PR if not in sync
        if: steps.sync_check.outcome == 'failure'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const data = JSON.parse(fs.readFileSync('/tmp/sync.json', 'utf8'));
            const body = `## ⚠️ Template Upstream Sync Required\n\n` +
              `${data.needsSync.length} files in miniblog-v4 changed but ` +
              `lin/templates/web-gin/ wasn't synced.\n\n` +
              `**Files needing sync:**\n` +
              data.needsSync.map(f => `- \`${f.src}\` → \`${f.dst}\``).join('\n') +
              `\n\nRun locally: \`linctl internal templatesync apply\``;
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
            core.setFailed('Template sync required');
```

### 6.2 本地 pre-commit hook（可选）

```bash
# .githooks/pre-commit
#!/bin/sh
linctl internal templatesync check --quiet || {
  echo "❌ Template upstream sync required."
  echo "   Run: linctl internal templatesync apply"
  exit 1
}
```

---

## 7. 与已有文档的关系

> **本文档与 [META-template-lifecycle-2026-04-28.md](./META-template-lifecycle-2026-04-28.md) 解决正交问题；不替代、不冲突，**复用同一个底层基础设施**。

### 7.1 两份文档的对比

| 维度 | META-template-lifecycle | **META-template-upstream-sync（本文档）** |
| --- | --- | --- |
| 解决的问题 | 终端用户用 lin 生成项目后怎么升级 | lin 维护者怎么把 miniblog-v4 演进同步到 web-gin 模板 |
| 使用者 | 终端开发者 | lin 仓库维护者 |
| 触发频率 | 用户跑 `linctl sync` | 维护者每次 miniblog-v4 升级后跑 |
| 命令前缀 | `linctl sync` / `linctl resolve` / `linctl status` | `linctl internal templatesync ...` |
| 关键算法 | git merge-file（用户改 vs 模板改） | 同样的 git merge-file（upstream 改 vs lin-specific 改） |
| CI 角色 | 不强制 | **强制** PR 检查 |
| 是否与终端用户可见 | ✅ 是 | ❌ 否（hidden in `internal`） |

### 7.2 共享基础设施

两份设计**共用** `internal/gitmerge/` 包：

```text
internal/gitmerge/
├── merge_file.go            # 包装 `git merge-file` 系统调用
├── fallback_diff3.go        # 用户机器无 git 时的 fallback
└── *_test.go
```

- `META-template-lifecycle` §4.2：用 gitmerge 处理用户 vs 模板的冲突
- `META-template-upstream-sync` §3-§5：用 gitmerge 处理 upstream vs lin-side 的冲突

### 7.3 实施依赖关系

```text
本设计的 U2 阶段依赖于 lifecycle 设计的 L2 阶段
（共享 internal/gitmerge/）

但 U1 阶段（基础同步无 merge）可以与 L1 并行启动
```

### 7.4 优先级建议

> 如果两份设计的所有 9-13 个阶段（lifecycle 6 + upstream 3）都要做，建议执行顺序：

| 优先级 | 阶段 | 理由 |
| --- | --- | --- |
| 1 | **upstream U1**（同步基础） | 解决你**当下最痛的**模板维护问题 |
| 2 | **lifecycle L1**（lockfile） | 为后续生命周期能力打地基 |
| 3 | **lifecycle L2** + **upstream U2** | 共享 gitmerge 基础设施，一起做 |
| 4-N | 后续阶段按业务优先级排 | 灵活 |

---

## 8. 实施路线（3 阶段）

> 本设计独立 3 阶段。与 lifecycle 设计的 6 阶段正交。

### 8.1 阶段总览

| 阶段 | 主题 | 关键交付 | 估时 | 依赖 |
| --- | --- | --- | --- | --- |
| **U1** | 同步基础（无 merge） | `internal/templatesync/`、`sync.yaml` 解析、5 个内置 transform、`status` / `plan` / `apply --strategy=force` 命令 | **2-3 天** | 无 |
| **U2** | 3-way merge + 完整 sync 命令族 | `add` / `revert` / `check`、`--strategy=ask\|abort\|ours\|theirs`、CI Action | **2-3 天** | lifecycle L2（共享 internal/gitmerge/） |
| **U3** | 高级能力 | `translateComments`（AI 辅助）、双向同步（lin → upstream patch）、自动新文件归类 | **3-5 天** | U1 + U2 |

### 8.2 阶段 U1 详细计划（2-3 天）

| Story | 内容 | 改动 | 验收 |
| --- | --- | --- | --- |
| U1.1 | 新增 `internal/templatesync/` 包：manifest.go + lockfile.go + runner.go 框架 | 新增约 600 行 | 单元测试覆盖 ≥80% |
| U1.2 | 实现 5 个核心 transform：rewriteImports / replaceLiteral / stripCopyrightHeader / addExtension / insertLinctlHeader | 5 文件 ~400 行 | 每个 transform 含单测；以 contextx.go → contextx.go.tpl 为黄金参照 |
| U1.3 | 写 `lin/internal/template/templates/web-gin/.sync.yaml` 初版（覆盖现有 130 个 .tpl 的 sync 规则） | 新增约 250 行 YAML | 跑 `linctl internal templatesync apply --strategy=force` 输出能与现状 100% 字节级一致 |
| U1.4 | 实现 `cmd_internal_templatesync.go` 父命令 + status / plan / apply 子命令 | 新增约 350 行 | E2E：跑 status → plan → apply 三连流程 |
| U1.5 | 写 `21-template-upstream-sync.md` §1-§5（开发者文档） | 新增文档 | 通过 §10 DoD |

**阶段 U1 退出条件**：

- 跑 `linctl internal templatesync apply --strategy=force` 能从 miniblog-v4 重新生成出当前 lin/templates/web-gin/ 下完全一致的内容（字节级 hash 比对全过）
- `sync.yaml` 完整覆盖 130 个 `.tpl` 文件的同步规则
- `.linctl/upstream-sync.lock.json` 落地

### 8.3 阶段 U2 详细计划（2-3 天）

| Story | 内容 | 验收 |
| --- | --- | --- |
| U2.1 | 引入 `internal/gitmerge/`（与 lifecycle L2 共享） | 单测覆盖 ≥85% |
| U2.2 | `runner.go` 增加 3-way merge 路径：owner=shared 时调用 gitmerge | E2E：modify miniblog-v4 + 改 lin/web-gin/ → apply 触发 merge |
| U2.3 | 实现 `add` / `revert` / `check` 三个子命令 | E2E：add 新文件流程跑通 |
| U2.4 | 实现 `--strategy=ask\|abort\|ours\|theirs\|force` 多策略 | 5 种策略各自单测 |
| U2.5 | 写 GitHub Action workflow + PR comment 模板 | 在 lin 仓库实际启用 |
| U2.6 | 文档 §6 + 新建 ADR-012: linctl-internal-templatesync-design | 通过 §10 DoD |

**阶段 U2 退出条件**：

- `check` 命令在 PR 上能阻断"miniblog-v4 改了但 web-gin 没同步"的 PR
- shared owner 文件的 3-way merge 流程跑通

### 8.4 阶段 U3 详细计划（远期，3-5 天）

| Story | 内容 |
| --- | --- |
| U3.1 | `translateComments` transform：调用 OpenAI / 本地 LLM 翻译注释，缓存到 `.linctl/translation-cache.yaml` |
| U3.2 | 双向同步：`linctl internal templatesync push <dst>` 把 lin/web-gin/ 改动反推 miniblog-v4（仅 owner=shared 时） |
| U3.3 | 自动新文件归类：检测 miniblog-v4 加新 `pkg/`，AI 推断 owner + transforms 写入 sync.yaml 草稿 |
| U3.4 | 多上游支持：sync.yaml 可声明多个 upstream（远期支持除 miniblog-v4 外的其他模板源） |

---

## 9. 风险、回滚与非目标边界

### 9.1 关键风险

| 风险 | 等级 | 缓解 |
| --- | --- | --- |
| `rewriteImports` AST 改写出错（上游代码语法非法时） | 中 | parse 失败 fallback 到 `regexReplace` + 输出 warning |
| Transform 顺序敏感 | 中 | sync.yaml 中显式声明顺序；`apply` 输出每步 hash 便于排查 |
| 上游加了新文件没人发现 | 高 | `newFilePolicy: warn` 默认；CI check 强制提示 |
| sync.yaml 与 lock.json 不一致 | 低 | `--rebuild-lock` 命令从磁盘 + sync.yaml 重建 lock |
| miniblog-v4 大版本 API 重构（不兼容） | 高 | 无自动方案；维护者手工编辑 sync.yaml + 跑 plan + 逐文件 review |

### 9.2 回滚策略

- sync.yaml + lock.json 是新增文件；可一键删除回滚到现状（手工维护模式）
- 每次 `apply` 前自动备份到 `.linctl/templatesync-backup/<ts>/`，retention 7 天

### 9.3 非目标边界

| # | 非目标 | 理由 |
| --- | --- | --- |
| **N1** | 不替代 lin 主二进制的 `linctl new` / `linctl sync` 等终端用户命令 | 这是维护者工具 |
| **N2** | 不做"自动 git commit / push" | 维护者自己控制 git workflow |
| **N3** | 不做远程 sync 服务（中央化） | 本地工具足够 |
| **N4** | 不引入 LLM 强制依赖（U3 是可选） | 基础能力必须能完全本地运行 |
| **N5** | 不支持模板树跨仓库挂载（如 web-gin 一部分来自 A 仓库一部分来自 B 仓库） | 复杂度爆炸；先支持单 upstream |
| **N6** | 不做 `pkg-only` 风格的细粒度 vendor（如 npm 依赖管理） | 那是包管理器的职责，不是脚手架职责 |

---

## 10. 验收标准

### 10.1 设计文档 DoD

- [x] §1.3 六大约束逐条对应解决方案
- [x] §2 备选方案矩阵含被否决项 + 理由
- [x] §3 数据模型含 sync.yaml + lock.json 完整范例
- [x] §4 transform 集合含 Go 代码骨架
- [x] §5 命令族每条含具体退出码 + 范例输出
- [x] §6 CI 集成含可直接落地的 GitHub Action workflow
- [x] §7 与 lifecycle 文档的关系明确（正交、共享 gitmerge）
- [x] §8 三阶段路线含 Story / 估时 / 退出条件
- [ ] **Review 通过、转 Accepted**

### 10.2 代码实施 DoD

#### 10.2.1 阶段 U1 DoD

- [ ] `linctl internal templatesync apply --strategy=force` 从 miniblog-v4 重生成 lin/templates/web-gin/ 与现状字节级一致
- [ ] sync.yaml 覆盖全部 130 个 .tpl 文件的同步规则
- [ ] 5 个核心 transform 单测覆盖 ≥85%
- [ ] 文档 §1-§5 落地

#### 10.2.2 阶段 U2 DoD

- [ ] `check` 命令可在 GitHub Action 中阻断不同步的 PR
- [ ] shared owner 文件的 3-way merge 流程跑通
- [ ] `add` / `revert` / `check` 三命令 E2E 测试全过
- [ ] 文档 §6 + ADR-012 落地

#### 10.2.3 阶段 U3 DoD（远期）

- [ ] `translateComments` 可在禁用 LLM 时降级到查 `translation-cache.yaml`
- [ ] `push` 命令能把 lin-side 的修复反推回 miniblog-v4 工作区
- [ ] 多 upstream 支持的 schema 兼容性测试通过

---

## 11. 修订历史

| 日期 | 变更 | 负责人 |
| --- | --- | --- |
| 2026-04-28 | 初稿（Proposed 状态） | @clin211 + assistant |
| TBD | Review 通过，转 Accepted | TBD |
| TBD | 阶段 U1 落地 | TBD |
| TBD | 阶段 U3 落地，本文档转 Archived | TBD |

---

_Last reviewed: 2026-04-28_
