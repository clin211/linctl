# ADR-002: 用 `//go:embed` 替代 `rakyll/statik` 嵌入模板资产

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态** | ✅ Accepted (2026-04-25) |
| **日期** | 2026-04-25 |
| **作者** | @clin211 |
| **审阅人** | TBD |
| **相关 ADR** | 无 |
| **影响范围** | `internal/template/`、`templates/`、构建流程 |
| **相关 issue/PR** | - |

---

## 1. 背景

osbuilder 使用 [`rakyll/statik`](https://github.com/rakyll/statik) 把 `internal/osbuilder/tpl/` 目录打包成单文件二进制（`statik/statik.go`）。问题：

1. **`rakyll/statik` 已归档**：upstream 仓库 archived，不再维护。
2. **构建步骤多余**：每次模板源变更都需要先跑 `go generate ./statik/...` 才能 build，开发者经常忘记。
3. **编辑器索引差**：模板源被打包成 `statik.go` 中的 base64 字符串，IDE 无法语法高亮 / Go to Definition / 重命名重构。
4. **二进制 bloat**：base64 编码后体积比原始 + gzip 大约 1.3×。

Go 1.16 以后，标准库提供了 `//go:embed`，能完美解决以上所有问题。

## 2. 决策

我们决定用 `//go:embed` 替代 `rakyll/statik`。

关键变更：

```go
// internal/template/embed.go
package template

import "embed"

//go:embed all:../../templates
var TemplatesFS embed.FS
```

- 模板源直接放在仓库 `templates/` 目录下，文件结构 = 嵌入结构。
- 删除 `internal/osbuilder/statik/` 整个目录及 `go generate` 指令。
- 用 `embed.FS` 实现 `fs.FS` 接口，可直接传给 `text/template.ParseFS` / `fs.WalkDir`。

## 3. 备选方案

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | **`//go:embed`**（chosen） | 标准库；零额外步骤；IDE 友好；二进制紧凑 | Go 1.16+ 才支持 | 我们要求 Go ≥ 1.22 |
| 2 | `rakyll/statik`（保留） | 与 osbuilder 兼容 | 已归档；构建多一步；IDE 不友好 | 维护风险 + 用户体验差 |
| 3 | 运行时从远端下载模板 | 模板可独立更新 | 离线不可用；引入网络依赖；版本一致性问题 | 违背"零外部依赖"原则 |
| 4 | `pkger`（Markus Wüstenberg） | 同样 archived | 同 statik | 社区已弃用 |
| 5 | 用 `*.go` 文件直接 `var X = "..."` 内嵌 | 无依赖 | 模板内容大量转义；维护噩梦 | 工程上不可接受 |

## 4. 后果

### 4.1 正面

- **零额外构建步骤**：改模板 → `go build` → 直接生效。
- **编辑器索引完整**：VSCode/GoLand 可以直接打开 `templates/**/*.tpl` 编辑、跳转。
- **二进制体积减少 ~15-20%**：相比 statik 的 base64，embed 用 binary 存储更紧凑。
- **测试更直接**：测试用 `os.ReadFile("templates/...")` 即可，无需 mock embed.FS。

### 4.2 负面 / Trade-offs

- **要求 Go 1.16+**：linctl 要求 Go ≥ 1.22，无影响。
- **`embed` 只能 embed 同 module 的相对路径**：模板必须在 linctl 仓库内，不能是 git submodule。但这本来就是我们的目标。
- **第一次 build 时把所有模板载入二进制**：约 3 MB 的模板（估算），二进制大小增加可控。

### 4.3 中性

- 删除 `rakyll/statik` 直接依赖（-1 直接依赖）。
- `templates/` 目录从 `internal/osbuilder/tpl/` 提到仓库根的 `templates/`，目录结构调整。

## 5. 实施步骤

1. 把 osbuilder 的 `internal/osbuilder/tpl/**/*` 移动到 linctl 仓库的 `templates/`。
2. 在 `internal/template/embed.go` 用 `//go:embed all:../../templates` 嵌入。
3. 替换所有 `statik.FS()` 调用为 `template.TemplatesFS`。
4. 删除 `statik/` 目录、相关 `go generate` 指令。
5. 验证：`go build` 后二进制体积应 < osbuilder 体积；`linctl new` 后生成的项目与 osbuilder 输出 diff = 0。

## 6. 参考资料

- [Go 1.16 release notes - embed](https://go.dev/doc/go1.16#embed)
- [rakyll/statik 归档说明](https://github.com/rakyll/statik#deprecated)
- [设计文档：05-template-system.md §5.3](../05-template-system.md)

---

_Last reviewed: 2026-04-25_
