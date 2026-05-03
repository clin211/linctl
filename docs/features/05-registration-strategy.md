# 05. 资源注册策略（AST 结构识别）

> **前置阅读**：[03-resource-scaffold.md](./03-resource-scaffold.md) §4「AST 注入点」
>
> 本文档定义 `linctl add` 如何把新资源连接到中央接口、proto、错误码注册表。
>
> ⚠️ 设计变更（2026-04-30）：linctl v2 已完全去除「锚点注释」（`// lin: inject-region:*`）。所有 AST 注入直接基于 Go 语法结构（接口名、receiver 名、函数名）定位插入点。模板里**不会**也**不应该**出现任何注入区注释。

---

## 1. 注册策略全景

### 1.1 两类注册：约定式 vs AST 注入

`linctl add Post` 触发的注册操作分为两类：

| 类别 | 文件 | 实现 | 是否需要 lin 介入 |
| --- | --- | --- | --- |
| **约定式注册** | `internal/<app>/handler/post.go` | `init() + Register()` 闭包 | ❌ 无需 AST，handler 文件本身即注册 |
| **AST 注入** | 4 个中央文件（见下表） | `dave/dst` / 文本插入 | ✅ 由 lin 自动改动 |

> **handler 路由 ≠ AST 注入**：handler 通过 miniblog-v4 风格的 `init() { Register(...) }` 闭包**自注册**到全局路由表。`linctl add` 只**创建** handler 文件，**不**注入到任何中央文件。

### 1.2 4 类 AST 注入

| # | 目标文件 | 注入类型 | 定位方式 | 库 |
| --- | --- | --- | --- | --- |
| 1 | `internal/<app>/biz/biz.go` | 接口扩展 + 工厂方法追加 | 接口名 `IBiz` / 结构体 `biz` | `dave/dst` |
| 2 | `internal/<app>/store/store.go` | 同上 | 接口名 `IStore` / 结构体 `memoryStore` 或 `datastore` | `dave/dst` |
| 3 | `pkg/api/<app>/v1/<app>.proto` | proto import 追加 | header（syntax/package/option）后或现有 import 块 | 文本插入（保留用户排序意图） |
| 4 | `internal/pkg/errno/register.go` | 函数体内追加调用 | 顶层函数 `RegisterAll` 的 Body | `dave/dst` |

> **决策回溯**：本设计是 [00 §10 §5.2](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29) 「B：AST 注入」的细化——B 选项的本意是「自动改 `biz.go`/`store.go`/`router.go`」中的「中央文件」，而非"所有"。handler 走约定式而非 AST，是因为 handler 文件本身可以闭包注册，无需修改中央文件。这种「分而治之」是最佳实践。

---

## 2. 关键设计原则

### 2.1 结构识别（Structural Targeting）

被注入的中央文件**不需要**任何特殊注释或占位符，注入点完全由 Go AST 结构决定：

| 注入类型 | 定位逻辑 |
| --- | --- |
| 接口方法追加 | 遍历 `*dst.File.Decls`，查找 `*dst.GenDecl{Tok: TYPE}` 且 `TypeSpec.Name == "<InterfaceName>"`，取其 `*dst.InterfaceType` |
| Receiver 方法追加 | 直接 `f.Decls = append(f.Decls, fd)`，新 FuncDecl 的 Recv 为 `(*<StructName>)` |
| 顶层函数体内语句追加 | 遍历 `*dst.File.Decls`，查找 `*dst.FuncDecl{Recv == nil, Name == "<FunctionName>"}`，操作其 `Body.List` |
| Proto import 追加 | 文本扫描 `import "..."` 行，根据现有顺序意图（字母序 / 追加）决定插入点 |

**好处**：
- 用户**无须维护**任何注入区注释，删错也不影响 AST 注入
- 模板更清爽（生产代码就是生产代码）
- 减少了一类错误："锚点 missing/duplicate/malformed" 全部消失

### 2.2 幂等性（Idempotency）

每次注入前 mutator 必须先检查目标是否已存在：

| 注入类型 | 幂等检查 |
| --- | --- |
| `IBiz.PostV1()` 接口方法 | 遍历 `iface.Methods.List`，检查 `field.Names` 中是否包含 `PostV1` |
| `*biz.PostV1()` receiver 方法 | 遍历 `f.Decls` 中所有 FuncDecl：Recv 类型 `*biz` && Name `PostV1` |
| import alias `postv1 "..."` | 遍历 import GenDecl，比较 `is.Path.Value` 是否等于目标路径 |
| proto `import "post.proto";` | 文本对比每行是否等于 `import "post.proto";` |
| `RegisterErrors(PostErrors()...)` | 遍历 `RegisterAll().Body.List`，比较 ExprStmt 的规范化字符串 |

**约定**：检测到已存在 → 跳过、打印 `⊝ skipped`，不算错误。

### 2.3 事务性（Atomicity）

`linctl add Post` 的所有操作（创建 + 注入）作为单一事务：

```
1. 创建临时备份点（.lin/.backup/<ts>/）
2. 创建文件（顺序）
3. 执行 AST 注入（顺序）
4. 校验：渲染完的项目能 go build（如启用 --strict）
5. 任意步骤失败 → 回滚
   - 删除已创建的新文件
   - 恢复修改过的中央文件（从备份）
6. 成功 → 提交（删除备份）
```

### 2.4 失败可恢复

注入失败时输出完整诊断：

```
✗ AST injection failed
  File:    internal/myblog/biz/biz.go
  Mutator: AddInterfaceMethod
  Method:  PostV1() postv1.PostBiz
  Error:   ast: interface "IBiz" not found in internal/myblog/biz/biz.go

  Hint:
    declare `type IBiz interface { ... }` in the target file

  Rollback:
    ✔ Removed 8 created files
    ✔ Restored 3 modified files
```

---

## 3. Mutator 详解

### 3.1 `mutator_interface.go` — 接口扩展

**用途**：向 `biz.go` / `store.go` 的接口块追加方法，并在结构体上追加 receiver method；可选地补 import alias。

**输入**：

```go
type InterfacePayload struct {
    InterfaceName string // "IBiz" / "IStore"
    StructName    string // "biz" / "memoryStore" / "datastore"

    Method     string // "PostV1"
    ReturnType string // "postv1.PostBiz"

    ImportAlias string // "postv1"     (可选)
    ImportPath  string // "github.com/foo/myblog/internal/myblog/biz/v1/post"

    ImplBody string // "return postv1.New(b.store)"
}
```

**算法**（伪代码）：

```go
func AddInterfaceMethod(file string, p InterfacePayload) error {
    f, _ := decorator.Parse(readFile(file))

    // 1. import alias 不存在则添加
    if p.ImportAlias != "" && !hasImport(f, p.ImportPath) {
        addImport(f, p.ImportAlias, p.ImportPath)
    }

    // 2. 直接通过名称定位接口
    iface := findInterfaceDecl(f, p.InterfaceName)
    if iface == nil {
        return errs.New(CodeASTApplyError,
            "ast: interface "+p.InterfaceName+" not found")
    }
    if !hasMethodInInterface(iface, p.Method) {
        appendMethodToInterface(iface, p.Method, p.ReturnType)
    }

    // 3. 直接通过 (struct, method) 检测 receiver method
    if !hasReceiverMethod(f, p.StructName, p.Method) {
        appendReceiverMethod(f, p.StructName, p.Method, p.ReturnType, p.ImplBody)
    }

    return writeFile(file, decorator.Restore(f))
}
```

**关键库选型**：使用 `dave/dst`（不是标准库 `go/ast`）：

- `dst` 在解析与序列化时保留所有注释和空白，避免 reformat 整个文件
- `dst` 节点带 `Decs` 字段，方便插入新 decl 时控制空行（如 `EmptyLine` / `NewLine`）

### 3.2 `mutator_proto.go` — Proto Import 追加

**用途**：向 `<app>.proto` 添加 `import "post.proto";` 语句。

**输入**：

```go
type ProtoPayload struct {
    File   string // pkg/api/myblog/v1/myblog.proto
    Import string // "post.proto"
}
```

**算法**：

```go
func AddProtoImport(file string, p ProtoPayload) error {
    lines := strings.Split(readFile(file), "\n")

    // 1. 幂等：已存在 → 跳过
    if linesContain(lines, `import "`+p.Import+`";`) { return nil }

    // 2. 收集现有 import 行
    importLineNos := detectImportLines(lines)

    // 3. 选择插入点
    if len(importLineNos) == 0 {
        // 无 import 段：插到 syntax/package/option 等 header 行之后
        return writeLines(file, insertAfterHeader(lines, importLine))
    }

    existing := extractImportValues(lines, importLineNos)
    if isSortedImports(existing) {
        // 用户字母序：按字母位置插入
        return writeLines(file, insertSorted(lines, importLineNos, importLine, p.Import))
    }
    // 用户无明显排序：追加到末尾
    return writeLines(file, insertAfterLast(lines, importLineNos, importLine))
}
```

**用户排序意图保护**：插入位置由「用户的现有排序意图」决定：

| 用户既有 import 排序 | 检测方式 | 插入位置 |
| --- | --- | --- |
| **已字母升序** | `sort.StringsAreSorted(existing)` | 插入到字母序对应位置 |
| **无明显排序** | 上面检测为 false | 追加到最后一个 `import` 行后 |
| **无任何 import** | 文件中没有 `import` | 在最末一个 `syntax`/`package`/`option` 行后插入 + 必要空行 |

> **简化策略**：proto 修改采用有限文本编辑而非完全 AST 反序列化。原因：proto 标准库主要用于读取/校验，序列化能力有限。文本插入足够且能保留用户意图。

### 3.3 `mutator_register.go` — 函数体内追加语句

**用途**：向 `errno/register.go` 的 `RegisterAll()` 函数体追加调用语句。

**输入**：

```go
type RegisterPayload struct {
    FunctionName string // "RegisterAll"
    Statement    string // "RegisterErrors(PostErrors()...)"
}
```

**算法**：

```go
func AppendRegistration(file string, p RegisterPayload) error {
    f, _ := decorator.Parse(readFile(file))

    // 1. 直接通过函数名定位顶层（非 receiver）函数
    fn := findFuncDecl(f, p.FunctionName)
    if fn == nil || fn.Body == nil {
        return errs.New(CodeASTApplyError,
            "ast: function "+p.FunctionName+" not found")
    }

    // 2. 解析待注入语句
    stmt := &dst.ExprStmt{X: parseCallExpr(p.Statement)}

    // 3. 幂等：检查 Body 中是否已有等价语句
    if hasEquivalentStmt(fn.Body, stmt) { return nil }

    // 4. 在 Body 末尾追加
    stmt.Decs.Before = dst.NewLine
    fn.Body.List = append(fn.Body.List, stmt)

    return writeFile(file, decorator.Restore(f))
}
```

幂等比较通过将 `dst.CallExpr` 序列化为 canonical 字符串（`Pkg.Func(arg1,arg2,...)` 形式）实现，避免位置/装饰差异导致误判。

---

## 4. 模板要求（项目骨架阶段）

`linctl new` 生成的初始 `biz.go` / `store.go` / `register.go` **必须**满足以下结构：

### 4.1 `internal/<app>/biz/biz.go.tpl`

```go
package biz

import (
    "{{.Module}}/internal/{{.AppName}}/store"
)

// IBiz defines the methods that must be implemented by the business layer.
//
// `linctl add <Resource>` appends new methods to this interface via AST.
type IBiz interface {
}

type biz struct {
    store store.IStore
}

var _ IBiz = (*biz)(nil)

func NewBiz(s store.IStore) *biz { return &biz{store: s} }
```

只要满足以下三点即可：
- 顶层声明 `type IBiz interface { ... }`
- 顶层声明 `type biz struct { ... }`（field 任意）
- 不需要任何"占位注释"

### 4.2 `internal/pkg/errno/register.go.tpl`

```go
package errno

func RegisterErrors(errs ...*BizError) { _ = errs }

func RegisterAll() {
}
```

只要顶层有 `func RegisterAll()` 即可，函数体可空可非空。

### 4.3 `pkg/api/<app>/v1/<app>.proto.tpl`

```proto
syntax = "proto3";
package myblog.v1;
option go_package = "...";

import "google/api/annotations.proto";

service MyblogService {
}
```

只要 import 段位置可识别（或没有 import 段时 header 行能识别）即可。

---

## 5. 注入顺序（事务保障）

```
        ┌────────────────────────────────────┐
        │ Step 0: 创建备份                    │
        │   .lin/.backup/<ts>/                │
        │     biz.go  store.go  ...           │
        └─────────────────┬──────────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │ Step 1: 创建新文件（最多 14 个）     │
        │   handler/post.go                  │
        │   biz/v1/post/post.go              │
        │   biz/v1/post/create.go            │
        │   ...                              │
        └─────────────────┬──────────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │ Step 2: AST 注入（最多 4 个，按序） │
        │   biz/biz.go                       │
        │   store/store.go                   │
        │   api/.../<app>.proto              │
        │   errno/register.go                │
        └─────────────────┬──────────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │ Step 3: 校验（可选 --strict）       │
        │   go build ./...                   │
        └────┬─────────────────┬─────────────┘
             │ 成功            │ 失败
             ▼                  ▼
        ┌─────────┐        ┌────────────────┐
        │ 提交    │        │ 回滚            │
        │ 删备份  │        │  - 删除新文件   │
        └─────────┘        │  - 恢复中央文件 │
                            └────────────────┘
```

### 5.1 `.lin/` 工作目录与 `.gitignore` 契约

`linctl add` 在项目根创建 `.lin/` 工作目录用于事务、备份、模板缓存：

```
<project-root>/
├── .lin/
│   ├── .backup/<timestamp>/      # AST 注入的临时备份（成功即删，保留最近 3 次）
│   │   ├── biz.go
│   │   ├── store.go
│   │   └── ...
│   ├── templates/                 # 项目级模板覆盖（可选；用户提交）
│   └── .last-run.json             # 最近一次操作的 audit log（可选）
└── .gitignore                     # ← linctl new 自动写入排除规则
```

**关键规则**：

| 路径 | 是否进 git | `.gitignore` 规则 |
| --- | --- | --- |
| `.lin/.backup/` | ❌ 永远不进 | `.lin/.backup/` |
| `.lin/.last-run.json` | ❌ 永远不进 | `.lin/.last-run.json` |
| `.lin/templates/` | ✅ 用户决定提交（团队共享） | 不排除 |

`linctl add` 执行前强校验：若项目 `.gitignore` 缺少 `.lin/.backup/` 行，会先自动追加（`⚠ updated .gitignore`）。

### 5.2 备份生命周期

| 时机 | 动作 |
| --- | --- |
| 注入前 | `cp <central-file> .lin/.backup/<ts>/<central-file>` |
| 注入成功 | `rm -rf .lin/.backup/<ts>/`（保留最近 3 次） |
| 注入失败 | 自动从 `.lin/.backup/<ts>/` 恢复中央文件 |

---

## 6. `--no-inject` 模式

当用户希望仅生成文件、跳过注入时（用于调试或手工合并）：

```bash
linctl add Post --no-inject
```

输出：

```
🎯 Adding resource: Post (--no-inject mode)

📦 Creating files...
   ✔ internal/myblog/handler/post.go
   ✔ internal/myblog/biz/v1/post/post.go
   ... (14 files)

⏭ Skipping AST injection.

⚠️  Manual injection required:
   internal/myblog/biz/biz.go        → IBiz: PostV1() postv1.PostBiz; func (b *biz) PostV1() ...
   internal/myblog/store/store.go    → IStore: Posts() PostStore; func (b *memoryStore) Posts() ...
   pkg/api/myblog/v1/myblog.proto    → import "post.proto";
   internal/pkg/errno/register.go    → RegisterAll body: RegisterErrors(PostErrors()...)
```

---

## 7. 边界场景

| 场景 | 处理 |
| --- | --- |
| 用户已经手工添加了同名方法/语句 | 幂等检查通过，跳过；记录 `⊝ skipped` |
| 文件被人为破坏（Go 语法错误） | `decorator.Parse` 失败，注入中止，文件保持原样；exit 50 |
| 接口/函数被改名 | AST 找不到目标符号，返回 `ast: interface/function not found`；exit 24 |
| 多 app 项目 | 通过 `--app=<name>` 路由到对应目录；详见 [02 §4.4](./02-command-set.md#44-上下文推断) |
| Windows 路径分隔符 | 内部路径全部 `filepath.ToSlash` |
| Windows 行尾符 (CRLF) | 解析时归一化为 LF；写入时按目标文件原行尾符保留 |
| 文件 git untracked | 注入正常进行；用户可后续 `git add` |
| 文件 git uncommitted（已 staged） | 注入正常；`--strict` 模式可要求工作区干净 |
| `.lin/.backup/` 已被 git tracked | warn；提示用户检查 `.gitignore` |

---

## 8. 测试策略

### 8.1 单测覆盖

| 测试文件 | 覆盖 |
| --- | --- |
| `mutator_interface_test.go` | AddInterfaceMethod 添加方法、import alias、receiver；接口缺失场景；幂等性 |
| `mutator_proto_test.go` | AddProtoImport 字母序插入、追加、idempotent、无 import 段 |
| `mutator_register_test.go` | AppendRegistration 函数体追加、幂等、函数缺失场景 |
| `injector_test.go` | 全流程：备份 / 创建 / 多 mutator 顺序执行 / 回滚 |

测试方法：

- 输入：模拟 `biz.go`（不含锚点，仅是合法 Go 源码）+ payload
- 输出：序列化后的文件应包含期望的代码片段
- 反向：再次执行同样注入 → 文件不变（幂等）

### 8.2 E2E 测试

```bash
# tests/e2e/add_test.sh
linctl new myblog --module github.com/test/myblog --storage memory --yes --non-interactive
cd myblog
linctl add Post                                             # 第一次
linctl add Post --yes --non-interactive                     # 第二次：应幂等
go build ./...                                            # 必须通过
```

---

## 9. 与历史版本（含锚点）的差异

| 维度 | 旧版（v1 锚点版） | 新版（v2 结构识别） |
| --- | --- | --- |
| 注入点定位 | `// lin: inject-region:<name>` 注释 | Go AST 节点（接口/struct/函数名） |
| 模板要求 | 必须包含 6 处 `inject-region` 占位 | 仅需保留正常的接口/函数声明 |
| 错误模式 | anchor missing / duplicate / malformed | symbol missing（接口或函数未定义） |
| `linctl lint` 规则 | 8 条 `anchor/*` + register 规则 | 仅 `dir/*`、`register/*`、`safety/*` |
| `--fix` 自动恢复 | 模式 A/B 重建锚点；模式 C 重新 add | 不再需要——只有"重新 add"一种模式 |
| 误删容忍度 | 删锚点 → 注入失败 | 删占位 → 不影响（占位本来就不存在） |

---

_Last reviewed: 2026-04-30_
