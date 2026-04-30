# Phase 1: Detempling — 把"伪模板"从 `.tpl` 中解放

> **状态**: Proposed (2026-04-28) · **影响范围**: `internal/template/`、`internal/component/*.go`、`internal/codegen/`、`internal/cli/cmd_add.go`
>
> 本文档是方案 ⑥ "Patch-friendly Vendoring" 的第一阶段执行计划。完整背景与方案讨论参见对话记录与 README 战略章节。

## 0. TL;DR

`lin/internal/template/templates/` 下共 **226 个 `.tpl` 文件**，其中 **64% 没有任何 `{{ }}` 模板语法**——它们是从上游 `onexstack` / `miniblog-v4` vendored 过来的纯 Go 源码，挂 `.tpl` 后缀只是为了走 codegen 渲染管道。

Phase 1 目标：**让这些"伪模板"回归 `.go` 形态**，IDE / `gopls` 能正确高亮、跳转、查阅；同时不影响生成产物的字节级行为。

完成后：

| 维度 | 改造前 | 改造后 |
|---|---|---|
| 模板根目录 | `lin/internal/template/templates/` | `lin/internal/template/_templates/` |
| 文件后缀分布 | 226 个 `.tpl` | ≈143 个 `.go` (raw copy) + ≈83 个 `.tpl` (真模板) |
| IDE 体验 | 无高亮、无跳转 | `.go` 文件全功能 |
| `go build` 影响 | 零（被 .tpl 屏蔽） | 零（被 `_templates/` 屏蔽） |
| Engine 渲染逻辑 | 全部走 `text/template` | `.tpl` → 渲染；其他 → raw copy |

## 1. 背景：为什么会有这么多"伪模板"

### 上下游链路

```
github.com/onexstack/onexstack/pkg/...         (上游, 仍在迭代)
        │
        │  ① miniblog-v4 团队 fork & 改造（go.mod 不再 require 上游）
        ▼
miniblog-v4/pkg/...                            (vendored, 离线)
        │
        │  ② linctl 团队再 fork、加 .tpl 后缀
        ▼
lin/internal/template/templates/web-gin/pkg/...
```

### 数字摸底（2026-04-28）

| 指标 | 数值 |
|---|---|
| `templates/` 总文件 | **226 个** |
| 后缀全部是 `.tpl` | 226 / 226 = **100%** |
| `web-gin/` 子目录文件数 | 130 |
| `web-gin/` 中**含 `{{ }}` 的真模板** | **47 个（36%）** |
| `web-gin/` 中**不含 `{{ }}` 的伪模板** | **83 个（64%）** |

### 真模板里实际用到的表达式（去重）

```
58 次  {{ .Project.Metadata.Module }}      ← module path 替换
 4 次  {{ .Component.Storage }} 分支         ← gorm vs mongo 切换 (~3 个文件)
11 次  {{ "{{" }} / {{ "}}"                  ← 转义字面量 {{ }}（生成 yaml 等内嵌时用）
```

也就是说"真正需要 `text/template` 表达力"的文件只有 ≈3 个含 `if/range` 的。其余真模板里 95%+ 也只是简单的字符串替换。

### 痛点

1. **升级链路断裂**：上游 onexstack 修了 bug → 需要两次手动同步（先到 miniblog-v4，再到 linctl 的 templates）。
2. **可测性退化**：`.tpl` 文件不会被 `gopls` / `go vet` / `golangci-lint` 扫到，IDE 也没语法高亮。
3. **诊断成本高**：模板渲染失败时只能拿到行号，不像普通 Go 文件那样直接 `go build`。
4. **隐性碎片化**：同一份 `pkg/ptr/ptr.go` 在仓库里至少存在 3 份副本（miniblog-v4 / 模板 / 历史生成产物）。
5. **无序 drift**：64% 的文件其实根本不需要模板化，但因为统一加了 `.tpl` 后缀，无法直观区分"需要渲染"vs"纯复制"。

## 2. 目标与非目标

### 目标

1. 把 64% 的"伪模板"从 `.tpl` 后缀解放回 `.go`，IDE / `gopls` 立即能识别。
2. 保证生成产物**字节级一致**（重跑 `linctl new` 输出与改造前完全相同）。
3. 不引入额外的运行时依赖、不破坏现有 Pair / Engine / Applier 抽象。
4. 为 Phase 2-6（溯源标头 / `linctl upgrade` / runtime+app 物理分层）打好基础。

### 非目标

- ❌ 不解决"双层 fork"问题（这是 Phase 6 的"内部 sync-templates"工具链做的）。
- ❌ 不引入 `linctl upgrade` 子命令（Phase 5）。
- ❌ 不改变生成项目的目录结构、不重命名 import 路径。
- ❌ 不动 47 个真模板（保留 `.tpl` 后缀和 `text/template` 渲染语义）。

## 3. 决策

### 3.1 模板根目录改名

**`lin/internal/template/templates/` → `lin/internal/template/_templates/`**

理由：

- Go 编译器自动忽略以 `_` 或 `.` 开头的目录（[Go spec - Source file organization](https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns)），未来重命名 `.tpl → .go` 时不会被纳入主 build。
- `//go:embed all:_templates` 可以正常嵌入（已用 `/tmp/embedtest` 实测验证）。
- 一次性大替换，但纯字符串改名，机械可控。

### 3.2 Engine 增加 raw copy 分支

**`Engine.Render` 按 `tplPath` 后缀分流：**

- 以 `.tpl` 结尾 → 走 `text/template` 渲染（原有逻辑不变）
- 不以 `.tpl` 结尾 → 走 `fs.ReadFile` 直接返回字节（新增分支）

理由：

- 64% 的文件不需要 parse + execute 的开销；
- 调用方（Applier）零感知：返回 `[]byte` 的语义没变，后续 `Format()` 仍能按 `.go` 后缀决定是否 gofmt；
- 现有 119+ 个 Pair 的 TemplateID 都以 `.tpl` 结尾，merge 这条改动单独不影响行为；只有当 Phase 1.2 把伪模板改名后，新分支才真正激活。

### 3.3 伪模板改名规则

**对 226 个 `.tpl` 文件做内容扫描，零 `{{` 的全部去掉 `.tpl` 后缀。**

- `pkg/ptr/ptr.go.tpl` → `pkg/ptr/ptr.go`
- `pkg/util/strings/strings.go.tpl` → `pkg/util/strings/strings.go`
- 真模板（含 `{{ }}`）保持不变。

### 3.4 Pair `TemplateID` 同步更新

webserver.go / cli.go / worker.go / cmd_add.go / codegen_test.go 中所有匹配新规则的 TemplateID 字符串，去掉末尾 `.tpl`。

## 4. 实施步骤

| 步骤 | 内容 | 预估 | 风险 | 单独可提交 |
|---|---|---|---|---|
| **1.0** | 给 Engine 加 raw copy 分支 + 单测 | 30 min | 极低 | ✅ 已完成 |
| **1.1** | 模板根目录 `templates/` → `_templates/` + 更新 embed 指令 | 30 min | 中 | ✅ |
| **1.2** | 批量替换 119+5+7+7+3 处 `"templates/` → `"_templates/`（webserver / cli / worker / cmd_add / codegen_test） | 30 min | 中 | 与 1.1 同 commit |
| **1.3** | 跑 `go build && go test ./...` 验收 1.0+1.1+1.2 | 10 min | — | — |
| **1.4** | 脚本化识别 83 个伪模板 + `git mv` 去后缀 | 30 min | 低 | ✅ |
| **1.5** | 同步更新 webserver.go 等里 83 处 TemplateID（去 `.tpl` 后缀） | 30 min | 低 | 与 1.4 同 commit |
| **1.6** | smoke test 路径修正（`stream_a_smoke_test.go` 等） | 15 min | 低 | 与 1.4 同 commit |
| **1.7** | 端到端验证：`linctl new myblog ... && diff 字节级核对` | 20 min | — | — |

### 4.1 步骤 1.0 详细（已完成）

修改 `lin/internal/template/engine.go` 的 `Render`：

```go
func (e *Engine) Render(tplPath string, data any) ([]byte, error) {
    if e == nil { return nil, ... }
    if !strings.HasSuffix(tplPath, ".tpl") {
        content, err := fs.ReadFile(e.fs, tplPath)
        if err != nil { return nil, ... }
        return content, nil
    }
    // 原有 text/template 逻辑
}
```

新增 3 个单测（已落 `engine_test.go`）：
- `TestRender_RawCopy_NonTpl`：验证非 .tpl 文件直接 raw copy
- `TestRender_RawCopy_NotFound`：验证 raw 路径下文件不存在的错误语义
- `TestRender_TplVsRawDispatch`：同名同内容的 `.tpl` 和 `.go` 验证两条路径互不串扰

### 4.2 步骤 1.1 + 1.2 详细

```bash
# 1.1: 物理改名
git mv lin/internal/template/templates lin/internal/template/_templates

# 1.2: embed 指令
# 编辑 lin/internal/template/embed.go：
#   //go:embed all:templates  →  //go:embed all:_templates

# 1.2 续: 批量字符串替换（仅 .go 文件 + TemplateID 字段）
rg -l '"templates/' lin/internal/component/ lin/internal/cli/ lin/internal/codegen/ \
  | xargs sed -i '' 's|"templates/|"_templates/|g'
```

#### 影响范围预估（基于 grep）

| 文件 | TemplateID 出现次数 |
|---|---|
| `lin/internal/component/webserver.go` | 119 |
| `lin/internal/component/cli.go` | 7 |
| `lin/internal/component/worker.go` | 7 |
| `lin/internal/cli/cmd_add.go` | 3 |
| `lin/internal/codegen/codegen_test.go` | 5 |
| **合计** | **141 处** |

> ⚠️ 文档（`docs/09-component-design.md`、`docs/08-feature-system.md`、`docs/11-implementation-plan.md`）中也提及 `templates/` 路径，但这些是文档描述、不影响行为，可在 Phase 1 完成后单独 PR 同步更新。

### 4.3 步骤 1.4 + 1.5 详细

```bash
# 1.4: 找出并改名 83 个伪模板
cd lin/internal/template/_templates
for f in $(find . -type f -name "*.tpl"); do
  if ! grep -q '{{' "$f"; then
    git mv "$f" "${f%.tpl}"
  fi
done

# 1.5: 把 webserver.go 等里所有 "(_templates/...).tpl" 模式去掉 .tpl
# 必须用更精确的正则避免误伤真模板的 TemplateID
# (具体脚本待 1.4 完成后根据改名清单生成)
```

#### 自动化校验脚本（建议）

```bash
# 校验：每个 _templates 下不带 .tpl 的文件，必须能在 webserver.go 等里找到对应 TemplateID
for f in $(find _templates -type f ! -name "*.tpl"); do
  rel=${f#_templates/}
  if ! rg -F "$rel" lin/internal/component/ > /dev/null; then
    echo "ORPHAN: $f"
  fi
done
```

### 4.4 步骤 1.6 详细

`lin/internal/template/stream_a_smoke_test.go` 等 smoke test 中硬编码的路径，按以下规则同步：

- `"templates/web-gin/pkg/ptr/ptr.go.tpl"` → `"_templates/web-gin/pkg/ptr/ptr.go"`
- `"templates/web-gin/pkg/store/store.go.tpl"` → `"_templates/web-gin/pkg/store/store.go.tpl"`（保留 .tpl 因含 `{{ .Module }}`）

### 4.5 步骤 1.7 验证清单

```bash
# 1. 编译/单测
make build
go test ./...

# 2. 端到端：用新 binary 生成项目
cd /tmp && rm -rf myblog
/path/to/linctl new myblog --module github.com/foo/myblog --framework gin --storage gorm-postgres -y

# 3. 字节级核对（与 Phase 1 之前的产物对比）
git -C myblog diff --stat <pre-phase-1-snapshot>
# 期望: 0 个文件 diff
```

## 5. 风险与缓解

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 批量字符串替换误伤（如文档里的 `templates/`） | 中 | 文档不一致 | 仅在 .go 文件且 TemplateID 字段附近替换；docs/ 单独 PR 更新 |
| 1.4 改名脚本判断错（把真模板当伪模板） | 低 | 渲染失败 | 改名前先 grep 双重确认；改名后 smoke test 必须全绿 |
| `_templates/` 在某些 IDE / 工具下被忽略（如 file watcher） | 低 | 开发体验差 | VS Code / GoLand 默认不忽略 `_*` 目录（仅 Go 编译器忽略） |
| smoke test 漏改导致 CI 挂 | 中 | merge 阻塞 | 步骤 1.6 显式跑全量 smoke test 验收 |
| 改名后 `git blame` 历史断裂 | 高 | 影响代码考古 | 用 `git mv` 而非 rm + add；启用 `--follow` 或 `.gitattributes` |

## 6. 后续阶段衔接

Phase 1 是方案 ⑥ "Patch-friendly Vendoring" 的第一步。它**只解决可读性**，不动升级机制。后续阶段：

| 阶段 | 主题 | 主要交付物 |
|---|---|---|
| **Phase 2** | 溯源标头 | 给所有 vendored 文件加 `// linctl-source: github.com/onexstack/onexstack/pkg/X@vY.Z sha:abc (vendored 2026-04-28)` |
| **Phase 3** | 简单替换路径 | 把 44 个仅做 `{{ .Module }}` 替换的真模板改成 `string.ReplaceAll`，进一步缩小 .tpl 文件数到 ≈3 个 |
| **Phase 4** | 物理分层 | 拆出 `pkg/runtime/`（CLI 管）和 `pkg/app/`（用户管），明确"upgrade 影响范围" |
| **Phase 5** | `linctl upgrade` | 实现 3-way merge，让"用户改动 + 上游修复"同时存活 |
| **Phase 6** | `make sync-templates` | linctl 内部维护工具：从上游 onexstack 自动拉取并打 .tpl 后缀，告别人肉同步 |

> 每一阶段都能独立交付价值。Phase 1 完成后已经能解决"拆盲盒"的 60% 痛点。

## 7. 决策日志

- **2026-04-28** —— 方案 ⑥ 采纳；Phase 1 拆分为 7 步；步骤 1.0 完成（Engine raw copy 分支 + 3 单测）；后续步骤待 shell 工具恢复后批量执行。

## 8. 参考

- ADR-002: 用 `//go:embed` 替代 `rakyll/statik`（[`adr/002-use-embed-not-statik.md`](../adr/002-use-embed-not-statik.md)）
- 设计文档：模板系统（[`05-template-system.md`](../05-template-system.md)）
- 设计文档：codegen pipeline（[`06-codegen-pipeline.md`](../06-codegen-pipeline.md)）
- Go 文档：[`go:embed` directive](https://pkg.go.dev/embed)、`_*` directory exclusion

---

_Last updated: 2026-04-28_
