# 10. 技术栈与依赖选型

> 本文档定义 linctl 自身（不是它生成的项目）使用的所有外部依赖、版本下限、平台支持矩阵，以及"为什么不选"的备选方案对比。任何新增依赖都必须先更新本文档 + 关联的 ADR。

## 10.1 选型原则

| 原则 | 解释 | 反例（osbuilder 痛点） |
| --- | --- | --- |
| **最小依赖** | 直接依赖 ≤ 18 个**（仅含运行时直接依赖，不计 test/dev 工具）**；间接依赖 ≤ 60 个 | osbuilder ~45 直接 + ~160 间接 |
| **避免归档/僵尸库** | 仅选过去 12 个月有 commit 的包 | osbuilder 用 archived 的 `rakyll/statik` |
| **优先标准库** | 凡能用标准库解决的，绝不引外部库 | osbuilder 多套日志：klog + apex + fmt |
| **跨平台** | 必须支持 darwin/linux/windows × amd64/arm64 | osbuilder 含 `procfs` 仅 Linux |
| **可替换** | 关键依赖必须有抽象层，便于未来切换 | dst → ASTMutator 接口 |
| **零本机依赖** | 不假设用户机器上装了 protoc/buf/git（除非可选检查） | osbuilder 强依赖 protoc |

## 10.2 Go 版本与平台支持

### 10.2.1 Go 版本下限

- **最低**：Go 1.22
- **推荐**：Go 1.22+（最新 stable）
- **理由**：
  - `log/slog` 在 1.21 进入标准库，1.22 完善 attrs API。
  - `slices`/`maps` 包稳定，避免引入 `samber/lo`。
  - `errors.Join` (1.20) 简化错误聚合。
  - `range over int` (1.22) 让模板循环代码更简洁。

### 10.2.2 平台支持矩阵

| 平台 | 架构 | 状态 | CI 测试 |
| --- | --- | --- | --- |
| darwin | amd64 | ✅ Tier 1 | ✅ |
| darwin | arm64 | ✅ Tier 1 | ✅ |
| linux | amd64 | ✅ Tier 1 | ✅ |
| linux | arm64 | ✅ Tier 1 | ✅ |
| windows | amd64 | ✅ Tier 2 | ✅（限制：无 fork/exec 的 hook 测试） |
| windows | arm64 | ⚠️ Tier 3 | 不在 CI，手动验证 |
| linux | 386 / arm | ❌ | 不支持 |
| freebsd / openbsd | * | ❌ | 不支持 |

> **Tier 1**：完整测试 + 官方二进制。  
> **Tier 2**（**这里 Tier 指平台支持等级**，与 ADR-004 的能力 Tier 不同）：CI 测试 + 官方二进制，但功能受限（如 Hook 执行策略中的 unrestricted 仅 POSIX 平台支持）。  
> **Tier 3**：仅 best-effort，无官方二进制（用户自行 `go install`）。

## 10.3 直接依赖清单（≤ 18，仅运行时直接依赖）

按用途分组。每个依赖都标注：版本下限、用途、ADR 引用、替代方案。

### 10.3.1 CLI 框架

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/spf13/cobra` | v1.8+ | 命令树、flag、completion | `urfave/cli`（API 不如 cobra 一致） |
| `github.com/spf13/pflag` | v1.0.5+ | POSIX 风格 flag | `flag`（标准库无 GNU 风格 long flag） |

### 10.3.2 配置 / 校验

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `gopkg.in/yaml.v3` | v3.0+ | YAML 序列化（支持 KnownFields 严格模式） | `goccy/go-yaml`（性能稍优但与 v3 兼容性差） |
| `github.com/go-playground/validator/v10` | v10.16+ | struct tag 校验 | `gookit/validate`（社区小） |

### 10.3.3 文件系统

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/spf13/afero` | v1.11+ | FS 抽象，便于测试用 MemMapFs | `io/fs`（只读） |

### 10.3.4 模板 / 格式化

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| **标准库** `text/template` | - | 主模板引擎 | `quicktemplate`（编译型，过于复杂） |
| `mvdan.cc/gofumpt` | v0.7+ | Go 源码格式化（比 gofmt 严格）；CI 与 Makefile 必须 pin 到 §10.7 的具体 tag | `gofmt`（标准库，规则更宽） |
| `github.com/duke-git/lancet/v2` | v2.3+ | 字符串 case 转换工具集 | 自研 `internal/template/strutil/`（重复造轮子） |
| `github.com/gobuffalo/flect` | v1.0+ | 复数化 / 单数化 | `jinzhu/inflection`（更小但 API 老） |

### 10.3.5 AST

| 依赖 | 版本 | 用途 | ADR | 替代方案（已否决） |
| --- | --- | --- | --- | --- |
| `github.com/dave/dst` | v0.27+ | Go AST + 注释保留 | [ADR-001](./adr/001-use-dst-not-goast.md) | `go/ast`（丢注释） |
| `github.com/bufbuild/protocompile` | v0.10+ | Proto 解析 | [ADR-003](./adr/003-use-protocompile-for-proto.md) | 字符串扫描（不可靠） |
| `google.golang.org/protobuf` | v1.34+ | descriptor 类型 | （随 protocompile） | - |

### 10.3.6 文本处理 / Diff

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/hexops/gotextdiff` | v1.0+ | 文本 diff（plan 阶段显示 diff） | `sergi/go-diff`（API 旧） |

### 10.3.7 终端 UI

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/fatih/color` | v1.16+ | 跨平台彩色输出 | `gookit/color`（依赖较多） |
| `github.com/enescakir/emoji` | v1.0+ | emoji 别名 | 自研（维护成本） |
| `github.com/briandowns/spinner` | v1.23+ | 长任务 spinner | `charmbracelet/bubbletea`（功能太重） |
| **标准库** `text/tabwriter` | - | 对齐表格输出 | `olekukonko/tablewriter`（依赖大） |

### 10.3.8 日志 / 错误

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| **标准库** `log/slog` | - | 结构化日志 | `zap` / `logrus` / `apex`（多套日志是 osbuilder 的痛） |

### 10.3.9 国际化（i18n）

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/nicksnyder/go-i18n/v2` | v2.4+ | CLI 输出 / 错误消息 / 帮助文本的多语言资源管理（详见 [13-coding-standards.md §13.9](./13-coding-standards.md)） | `golang.org/x/text/message`（功能少）；自研 map（早期方案，规模化后维护成本高） |
| `golang.org/x/text` | v0.14+ | `language` / `message`，go-i18n 的依赖 | - |

> **位置**：`internal/i18n/`，资源文件 `messages_*.yaml` 通过 `//go:embed` 嵌入二进制。

### 10.3.10 测试（仅在 `_test.go` 中使用）

| 依赖 | 版本 | 用途 | 替代方案（已否决） |
| --- | --- | --- | --- |
| `github.com/stretchr/testify` | v1.9+ | assert + require | 标准库 `testing`（断言代码冗长） |

### 10.3.11 依赖总览

```
直接依赖（≤ 18 个，对应 §10.1「最小依赖」原则；仅运行时，不计 test/dev 工具）：
├── CLI: cobra, pflag
├── 配置: yaml.v3, validator/v10
├── FS: afero
├── 模板: gofumpt, lancet/v2, flect
├── AST: dst, protocompile, protobuf
├── Diff: gotextdiff
├── UI: color, emoji, spinner
├── i18n: go-i18n, x/text
└── 测试: testify

间接依赖（估算 ≤ 50 个，详见 go.sum）
```

> CI 中加 `make deps-audit` 命令：检查 `go mod why` 后 `go list -m all | wc -l` ≤ 65。

## 10.4 关键依赖的替代方案对比

> 本节展开几个**容易引发争论**的选型决策。

### 10.4.1 yaml.v3 vs goccy/go-yaml

| 维度 | yaml.v3 | goccy/go-yaml |
| --- | --- | --- |
| 性能 | 1× | 1.5× |
| 严格模式（KnownFields） | ✅ | ❌（要自己实现） |
| 与 K8s/Helm 生态兼容 | ✅ 标准 | ⚠️（少数 edge case 不一致） |
| 维护活跃度 | upstream 慢但稳定 | 活跃 |
| 我们的选择 | ✅ | ❌ |

**结论**：linctl 的瓶颈不在 YAML 解析（< 50ms），不需要更快的实现，选稳定者。

### 10.4.2 validator/v10 vs 自研 Schema 校验

| 维度 | validator/v10 | 自研 |
| --- | --- | --- |
| 即用 | ✅ | ❌ |
| 与 JSON Schema 互通 | ⚠️ 通过 `tags` 转换 | ✅ 完全控制 |
| 自定义规则 | ✅ `RegisterValidation` | ✅ |
| 错误信息可读性 | ⚠️ 默认输出难懂 | ✅ |
| 依赖体积 | ~150 KB | 0 |
| 我们的选择 | ✅ | ❌（重新发明轮子） |

**结论**：validator/v10 + 自定义错误格式化 wrapper（`internal/validate/format.go`），既得自研可读性，又借力开源。

### 10.4.3 charmbracelet/bubbletea vs briandowns/spinner

| 维度 | bubbletea | spinner |
| --- | --- | --- |
| 功能 | 完整 TUI 框架（可做交互式表单） | 仅 spinner |
| 依赖体积 | ~3 MB | ~50 KB |
| 学习曲线 | 高（Elm 风格） | 极低 |
| 我们的需求 | 仅需要 spinner + 简单 prompt | ✅ 满足 |
| 我们的选择 | ❌ | ✅ |

**结论**：linctl 不是 TUI 工具，spinner + cobra 自带的 survey 已足够。如果未来真的需要复杂 TUI，再升级。

### 10.4.4 cobra vs urfave/cli

| 维度 | cobra | urfave/cli |
| --- | --- | --- |
| 子命令树 | ✅ 自动嵌套 | ⚠️ 手动管理 |
| Auto-completion | ✅ 内置 bash/zsh/fish | ⚠️ 需要配合插件 |
| 文档生成 | ✅ 内置 `cobra-cli` | ⚠️ |
| K8s/docker/hugo 等都用 | ✅ | ❌ |
| 我们的选择 | ✅ | ❌ |

## 10.5 依赖治理

### 10.5.1 添加新依赖的流程

```
1. PR 描述中说明：
   - 用途
   - 选它而不选其他的理由
   - 二进制大小影响（go build -ldflags="-s -w" + ls -la）
   - 是否引入 cgo（cgo 是 ❌）
2. 更新本文档（10-tech-stack.md）
3. 如果是关键决策（影响接口/性能），写 ADR
4. CI 自动检查依赖数量 + 体积
```

### 10.5.2 依赖升级策略

| 升级类型 | 频率 | 操作 |
| --- | --- | --- |
| 安全补丁 | 立即 | 自动化 PR（dependabot） |
| Patch 版本 | 每月 | 一次性 PR `make deps-update-patch` |
| Minor 版本 | 每季度 | 评估变更日志后 PR |
| Major 版本 | 不定 | 必须有 ADR 评估 |

### 10.5.3 禁止的依赖类型

- ❌ **cgo 依赖**：破坏 cross-compile，破坏静态链接。
- ❌ **闭源 SDK**：linctl 是 MIT 项目。
- ❌ **archived 仓库**：除非有可信 fork。
- ❌ **GPL/AGPL 协议**：与 MIT 不兼容。
- ❌ **`replace` 到非官方 fork**：除非 ADR 明确说明。

### 10.5.4 间接依赖审计

```bash
# Makefile 目标
deps-audit:
    @go list -m all | wc -l | awk '{ if ($$1 > 65) { print "ERROR: too many deps:", $$1; exit 1 } else { print "OK:", $$1, "deps" } }'
    @go mod why -m github.com/<问题包>  # 找出谁引入了大依赖
    @go mod graph | head -50              # 依赖图概览
```

## 10.6 二进制体积预算

| 阶段 | 二进制大小目标 | 优化手段 |
| --- | --- | --- |
| Phase 1 (MVP) | ≤ 12 MB | 基础库 + 内嵌模板，**自研轻量 trace**（不引 OTel） |
| Phase 3 | ≤ 15 MB | AST 增强；OTel SDK 通过 **build tag 隔离**（默认 build 不含），仅 `-tags otel` 时编入 |
| Phase 5 | ≤ 20 MB | 含插件机制 |
| **绝不超过** | 25 MB | 触发优化 |

> **OTel 与体积约束**：`go.opentelemetry.io/otel` SDK 约 ~2 MB，全量纳入会突破 15MB 上限。
> 因此 OTel 仅作为 **可选 export**（详见 [14-observability.md §14.4.2](./14-observability.md)）：
> - 默认 build：`go build .` —— 不包含 OTel，二进制 ≤ 15 MB
> - 含 OTel build：`go build -tags otel .` —— 体积 ≤ 17 MB（用户主动选择）
> - 自研轻量 trace 在两种 build 下都可用；OTel 仅作为可选输出后端

### 10.6.1 体积优化技巧

```bash
# Release build flags
-ldflags="-s -w" # 去掉调试符号 + DWARF
-trimpath        # 去掉构建路径
-buildmode=pie   # PIE 二进制（仅 Linux）

# 使用 upx（可选，对 macOS Gatekeeper 不友好，不推荐）
upx --best --lzma _output/bin/linctl
```

### 10.6.2 监控体积

CI 加 `bloat check` 任务：

```yaml
- name: Binary size check
  run: |
    go build -ldflags="-s -w" -trimpath -o /tmp/linctl .
    SIZE=$(stat -f%z /tmp/linctl 2>/dev/null || stat -c%s /tmp/linctl)
    if [ $SIZE -gt 15728640 ]; then  # 15 MB
      echo "Binary too large: $SIZE bytes"
      exit 1
    fi
```

## 10.7 工具链依赖（开发期，非运行时）

linctl 自身的开发期工具，通过 `tools.go` + `Makefile` 管理，**不打包进二进制**。

```go
// tools/tools.go
//go:build tools

package tools

import (
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
    _ "github.com/goreleaser/goreleaser/v2"
    _ "mvdan.cc/gofumpt"
    _ "github.com/google/wire/cmd/wire"
    _ "honnef.co/go/tools/cmd/staticcheck"
    _ "go.uber.org/mock/mockgen"
)
```

通过 `make tools` 一键安装。**所有工具版本 pin 到具体 tag**，不使用浮动版本（latest 标签），便于：

1. **可复现构建**：CI 与本地、不同 contributor 之间使用同一版本，避免「我这能跑」
2. **明确升级窗口**：升级版本 = 一次显式 PR + lint 输出 diff review，便于排查破坏性 lint 规则变更
3. **审计追溯**：发布事故时可精确定位是否由工具升级触发

```makefile
# 工具版本统一管理；升级时改这里 + 跑一次 CI 验证
GOLANGCI_LINT_VERSION ?= v1.59.0
GORELEASER_VERSION    ?= v2.3.2
GOFUMPT_VERSION       ?= v0.7.0
STATICCHECK_VERSION   ?= 2024.1.1
MOCKGEN_VERSION       ?= v0.4.0
WIRE_VERSION          ?= v0.6.0

tools:
    @go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
    @go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
    @go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
    @go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
    @go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
    @go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
```

> **CI 一致性保证**：`tools.go` 中也用相同版本 pin（`go mod tidy` 后版本与 Makefile 同步）；
> 任何浮动版本标签（如 `latest`）出现在 lint / CI 脚本中都视为 P0 bug，CI 通过 `scripts/check-tool-versions.sh` 强制扫描。

| 工具 | 用途 | 触发时机 |
| --- | --- | --- |
| `golangci-lint` | 静态分析（含 ~20 个 linter） | pre-commit + CI |
| `gofumpt` | 格式化 | pre-commit + CI |
| `staticcheck` | 高级静态分析 | CI |
| `goreleaser` | 跨平台发布 | tag 推送时（CI） |
| `mockgen` | 生成 mock | 手动 `go generate` |
| `wire` | 依赖注入（仅 linctl 自身使用） | 手动 |

## 10.8 第三方资产（templates/ 中）

linctl 内嵌的模板会引用一些**生成的项目**所需的依赖，这些**不算 linctl 自身依赖**，但需要稳定的版本承诺：

| 模板内引用 | 默认版本 | 升级策略 |
| --- | --- | --- |
| `github.com/gin-gonic/gin` | v1.10+ | 半年评估 |
| `github.com/grpc-ecosystem/grpc-gateway` | v2.20+ | 半年评估 |
| `gorm.io/gorm` | v1.25+ | 半年评估 |
| `go.opentelemetry.io/otel` | v1.28+ | 季度评估 |
| `github.com/redis/go-redis/v9` | v9.5+ | 半年评估 |

**模板版本管理**：每个模板里 `go.mod` 中的版本号通过模板变量 `{{.Versions.Gin}}` 注入，便于一次性升级。

## 10.9 与 osbuilder 依赖对比

| 维度 | osbuilder | linctl | 减少 |
| --- | --- | --- | --- |
| 直接依赖数 | ~45 | ≤ 18 | -60% |
| 间接依赖数 | ~160 | ≤ 50 | -69% |
| 含 k8s 依赖 | ✅（apimachinery 等数 MB） | ❌ | -数 MB |
| 含 archived 依赖 | ✅（rakyll/statik） | ❌ | - |
| 多套日志库 | klog + apex + fmt | 仅 slog | - |
| cgo 依赖 | 间接含 | ❌ | - |
| 二进制大小 | ~20 MB | ≤ 15 MB | -25% |

## 10.10 Open Questions

| 问题 | 状态 | 决议截止 |
| --- | --- | --- |
| 是否引入 `charmbracelet/lipgloss` 美化输出？ | 待评估，目前 fatih/color + tabwriter 够用 | Phase 2 |
| 是否引入 `mattn/go-isatty` 显式检测 TTY？ | 倾向于"否"，由 fatih/color 内置处理 | - |
| `mockgen` 是否换成 `vektra/mockery`？ | 标准化 mockgen 更主流；mockery 灵活但配置多 | Phase 1 锁定 mockgen |
| 是否在 Phase 5 内置 WASM runtime 支持插件？ | 远期评估，当前 stdio JSON-RPC 优先 | Phase 5+ |

---

## 修订记录

| 日期 | 版本 | 变更 |
| --- | --- | --- |
| 2026-04-25 | 0.1 | 初始版本 |
| 2026-04-25 | 0.2 | 按 [META-fix-decisions-2026-04-25](./META-fix-decisions-2026-04-25.md) 修订：(1) 直接依赖表新增 `nicksnyder/go-i18n` + `golang.org/x/text`；(2) §10.6 体积预算明确 OTel 通过 `-tags otel` 隔离，默认 ≤ 15 MB；(3) §10.7 工具链版本全部 pin 到具体 tag |

---

下一步阅读：[09-component-design.md](./09-component-design.md)

---

_Last reviewed: 2026-04-25_
