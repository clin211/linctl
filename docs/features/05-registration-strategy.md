# 05. 资源注册策略（AST 注入）

> **前置阅读**：[03-resource-scaffold.md](./03-resource-scaffold.md) §4「AST 注入点」
>
> 本文档定义 `lin add` 如何通过 AST 注入将新资源连接到中央接口、proto、错误码注册表。

---

## 1. 注册策略全景

### 1.1 两类注册：约定式 vs AST 注入

`lin add Post` 触发的注册操作**分为两类**：

| 类别 | 文件 | 实现 | 是否需要 lin 介入 |
| --- | --- | --- | --- |
| **约定式注册** | `internal/<app>/handler/post.go` | `init() + Register()` 闭包 | ❌ 无需 AST，handler 文件本身即注册 |
| **AST 注入** | 4 个中央文件（见下表） | `dave/dst` / 文本插入 | ✅ 由 lin 自动改动 |

> **handler 路由 ≠ AST 注入**：handler 通过 miniblog-v4 风格的 `init() { Register(...) }` 闭包**自注册**到全局路由表。`lin add` 只**创建** handler 文件，**不**注入到任何中央文件。

### 1.2 4 类 AST 注入

`lin add Post` 触发的 AST 注入仅作用于以下中央文件：

| # | 目标文件 | 注入类型 | 库 |
| --- | --- | --- | --- |
| 1 | `internal/<app>/biz/biz.go` | Go 接口扩展 + 工厂方法追加 | `dave/dst` |
| 2 | `internal/<app>/store/store.go` | 同上 | `dave/dst` |
| 3 | `pkg/api/<app>/v1/<app>.proto` | proto import 追加 | `bufbuild/protocompile` + 文本插入 |
| 4 | `internal/pkg/errno/register.go` | 函数体内追加调用 | `dave/dst` |

> **决策回溯**：本设计是 [00 §10 §5.2](./00-refactor-rationale.md#10-用户决策已确定--2026-04-29) 「B：AST 注入」的细化——B 选项的本意是「自动改 `biz.go`/`store.go`/`router.go`」中的「中央文件」，而非"所有"。handler 走约定式而非 AST，是因为 handler 文件本身可以闭包注册，无需修改中央文件。这种「分而治之」是最佳实践。

---

## 2. 关键设计原则

### 2.1 锚点注释（Inject Region）

被注入的中央文件必须包含**锚点注释**，让 AST 注入精确定位修改区域：

```go
// lin: inject-region:store-interface (do not remove this comment)
type IStore interface {
    DB(...) *gorm.DB
    TX(...) error
    User() UserStore
}
// lin: inject-region-end
```

**规则**：

- 锚点格式固定：`// lin: inject-region:<name>`、`// lin: inject-region-end`。
- `<name>` 列表（v1.0）：
  - `biz-imports`、`biz-interface`、`biz-impl`
  - `store-interface`、`store-impl`
  - `errno-register`
- 模板（`lin new` 时生成的初始 `biz.go`/`store.go`）必须包含完整锚点。
- 用户**不应删除**锚点注释；删除后 AST 注入将失败并提示恢复方法。

### 2.2 幂等性（Idempotency）

每次注入前 mutator 必须**先检查**目标是否已存在：

| 注入类型 | 幂等检查 |
| --- | --- |
| `IBiz.PostV1()` 接口方法 | AST 遍历接口字段，检查方法名是否存在 |
| `*biz.PostV1()` 实现方法 | AST 遍历文件 decl，检查 `func (b *biz) PostV1()` |
| import alias `postv1 "..."` | AST 遍历 import spec |
| proto `import "post.proto";` | protocompile 遍历 import |
| `RegisterErrors(PostErrors()...)` 调用 | AST 遍历 RegisterAll 函数体 |

**约定**：检测到已存在 → **跳过、打印 `⊝ skipped`**，不算错误。

### 2.3 事务性（Atomicity）

`lin add Post` 的所有操作（创建 + 注入）作为**单一事务**：

```
1. 创建临时备份点（git stash 或 .lin/backup/<ts>/）
2. 创建文件（顺序）
3. 执行 AST 注入（顺序）
4. 校验：渲染完的项目能 go build（如启用 --strict）
5. 任意步骤失败 → 回滚
   - 删除已创建的新文件
   - 恢复修改过的中央文件（从备份）
6. 成功 → 提交（删除备份）
```

### 2.4 失败可恢复

注入失败时输出**完整诊断**：

```
✗ AST injection failed
  File:    internal/myblog/biz/biz.go
  Mutator: addInterfaceMethod
  Method:  PostV1() postv1.PostBiz
  Error:   inject region "biz-interface" not found

  Hint:
    1. Ensure 'biz.go' contains:
         // lin: inject-region:biz-interface (do not remove this comment)
         ... (interface body) ...
         // lin: inject-region-end
    2. If you removed the comments, restore from a previous version
       or copy from internal/templates/project/internal/app/biz/biz.go.tpl

  Rollback:
    ✔ Removed 8 created files
    ✔ Restored 3 modified files
```

---

## 3. Mutator 详解

### 3.1 `mutator_interface.go` - 接口扩展

**用途**：向 `biz.go` / `store.go` 的接口块追加方法，并在结构体上追加实现方法。

**输入**：

```go
type InterfacePayload struct {
    File          string  // internal/myblog/biz/biz.go
    AnchorName    string  // "biz-interface" / "biz-impl" / "store-interface" / "store-impl"
    InterfaceName string  // "IBiz" / "IStore"
    StructName    string  // "biz" / "datastore"

    // 接口方法
    Method     string  // "PostV1"
    ReturnType string  // "postv1.PostBiz"

    // import 别名
    Import struct {
        Alias string  // "postv1"
        Path  string  // "github.com/foo/myblog/internal/myblog/biz/v1/post"
    }

    // 实现方法体
    ImplBody string  // "return postv1.New(b.store)"
}
```

**算法**（伪代码）：

```go
func AddInterfaceMethod(file string, p InterfacePayload) error {
    // 1. 解析为 dst.File
    f, err := decorator.Parse(readFile(file))

    // 2. 在 import 段添加 alias（若不存在）
    if !hasImport(f, p.Import.Path) {
        addImport(f, p.Import.Alias, p.Import.Path)
    }

    // 3. 在 inject-region:<AnchorName> 区域内的接口添加方法
    iface := findInterfaceInRegion(f, p.AnchorName, p.InterfaceName)
    if iface == nil { return ErrAnchorNotFound }
    if !hasMethodInInterface(iface, p.Method) {
        addMethodToInterface(iface, p.Method, p.ReturnType)
    }

    // 4. 在 inject-region:<implAnchor> 区域内追加 receiver method
    if !hasReceiverMethod(f, p.StructName, p.Method) {
        addReceiverMethod(f, p.StructName, p.Method, p.ReturnType, p.ImplBody)
    }

    // 5. 序列化回写
    return writeFile(file, decorator.Restore(f))
}
```

**关键库选型**：使用 `dave/dst`（不是标准库 `go/ast`）：

- `dst` 在解析与序列化时**保留所有注释和空白**，避免 reformat 整个文件。
- `dst` 支持精确定位锚点注释。

### 3.2 `mutator_proto.go` - Proto Import 追加

**用途**：向 `<app>.proto` 添加 `import "post.proto";` 语句。

**输入**：

```go
type ProtoPayload struct {
    File   string  // pkg/api/myblog/v1/myblog.proto
    Import string  // "post.proto"
}
```

**算法**：

```go
func AddProtoImport(file string, p ProtoPayload) error {
    // 1. 用 protocompile 解析（仅校验结构）
    parser := protoparse.Parser{}
    fd, err := parser.ParseFiles(file)
    if err != nil { return err }

    // 2. 检查 import 是否已存在（幂等）
    for _, imp := range fd[0].GetDependencies() {
        if imp.GetName() == p.Import { return nil }
    }

    // 3. 文本插入：尊重用户的现有排序意图（详见 §3.2.1）
    return insertImportLineRespectingOrder(file, p.Import)
}
```

#### 3.2.1 用户排序意图保护

文本插入位置由「用户的现有排序意图」决定，不是简单地"追加到最后一行"：

| 用户既有 import 排序 | 检测方式 | 插入位置 |
| --- | --- | --- |
| **已字母升序** | 现有 imports 按 `imp[i] <= imp[i+1]` 排序 | 插入到字母序对应位置 |
| **分组（系统在前 / 业务在后）** | 出现连续空行分隔 | 插入到末尾分组内的字母序位置 |
| **无明显排序** | 既有顺序无规律 | 追加到最后一个 `import` 行后 |
| **无任何 import** | 文件中没有 `import` | 在 `package` 行后插入 + 必要空行 |

**算法**（伪代码）：

```go
func insertImportLineRespectingOrder(file, newImport string) error {
    lines := readLines(file)
    importBlocks := detectImportBlocks(lines)  // [[start,end], ...]

    if len(importBlocks) == 0 {
        // 无 import 段：在 package 行后插入新段
        return insertAfterPackage(file, "import \""+newImport+"\";")
    }

    last := importBlocks[len(importBlocks)-1]
    existing := extractImports(lines[last.start:last.end])

    if isSorted(existing) {
        // 用户字母序：按字母位置插入
        idx := sort.SearchStrings(existing, newImport)
        return insertAt(file, last.start+idx, newImport)
    }
    // 用户无排序：追加到末尾
    return insertAt(file, last.end, newImport)
}
```

> **简化策略**：proto 修改采用**有限文本编辑**而非完全 AST 反序列化。原因：protocompile 主要用于读取/校验，序列化能力有限。文本插入足够且**保留用户意图**。

### 3.3 `mutator_register.go` - 注册函数追加

**用途**：向 `errno/register.go` 的 `RegisterAll()` 函数体追加调用。

**输入**：

```go
type RegisterPayload struct {
    File         string  // internal/pkg/errno/register.go
    AnchorName   string  // "errno-register"
    FunctionName string  // "RegisterAll"
    Statement    string  // "RegisterErrors(PostErrors()...)"
}
```

**算法**：

```go
func AppendRegistration(file string, p RegisterPayload) error {
    f, _ := decorator.Parse(readFile(file))

    fn := findFunction(f, p.FunctionName)
    if fn == nil { return ErrFunctionNotFound }

    // 在 inject-region:<AnchorName> 区域内检查并追加
    if !hasStatement(fn.Body, p.Statement) {
        appendStatementToBody(fn.Body, p.Statement)
    }

    return writeFile(file, decorator.Restore(f))
}
```

---

## 4. 锚点注释模板（项目骨架阶段）

`lin new` 生成的初始 `biz.go` / `store.go` / `register.go` 必须包含锚点：

### 4.1 `internal/<app>/biz/biz.go.tpl`

```go.tpl
package biz

import (
    // lin: inject-region:biz-imports (do not remove this comment)
    {{- if .Features | Has "user" }}
    userv1 "{{ .Module }}/internal/{{ .AppName }}/biz/v1/user"
    {{- end }}
    "{{ .Module }}/internal/{{ .AppName }}/store"
    // lin: inject-region-end

    "github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))

// IBiz 业务层接口集合.
// lin: inject-region:biz-interface (do not remove this comment)
type IBiz interface {
    {{- if .Features | Has "user" }}
    UserV1() userv1.UserBiz
    {{- end }}
}
// lin: inject-region-end

type biz struct {
    store store.IStore
}

var _ IBiz = (*biz)(nil)

func NewBiz(s store.IStore) *biz { return &biz{store: s} }

// lin: inject-region:biz-impl (do not remove this comment)
{{- if .Features | Has "user" }}
func (b *biz) UserV1() userv1.UserBiz { return userv1.New(b.store) }
{{- end }}
// lin: inject-region-end
```

### 4.2 `internal/pkg/errno/register.go.tpl`

```go.tpl
package errno

// RegisterAll 注册全部业务错误.
func RegisterAll() {
    // lin: inject-region:errno-register (do not remove this comment)
    {{- if .Features | Has "user" }}
    RegisterErrors(UserErrors()...)
    {{- end }}
    // lin: inject-region-end
}
```

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
        │ Step 1: 创建新文件（8 个）          │
        │   handler/post.go                  │
        │   biz/v1/post/post.go              │
        │   biz/v1/post/create.go            │
        │   ...                              │
        └─────────────────┬──────────────────┘
                          │
        ┌─────────────────▼──────────────────┐
        │ Step 2: AST 注入（4 个，按顺序）    │
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
        │ 提交     │        │ 回滚             │
        │ 删备份  │        │   - 删除新文件 │
        └─────────┘        │   - 恢复中央文件│
                            └────────────────┘
```

### 5.1 `.lin/` 工作目录与 `.gitignore` 契约

`lin add` 在项目根创建 `.lin/` 工作目录用于事务、备份、模板缓存：

```
<project-root>/
├── .lin/
│   ├── .backup/<timestamp>/      # AST 注入的临时备份（成功即删）
│   │   ├── biz.go
│   │   ├── store.go
│   │   └── ...
│   ├── templates/                 # 项目级模板覆盖（可选；用户提交）
│   └── .last-run.json             # 最近一次操作的 audit log（可选）
└── .gitignore                     # ← lin new 自动写入排除规则
```

**关键规则**：

| 路径 | 是否进 git | `.gitignore` 规则 |
| --- | --- | --- |
| `.lin/.backup/` | ❌ 永远不进 | `.lin/.backup/` |
| `.lin/.last-run.json` | ❌ 永远不进 | `.lin/.last-run.json` |
| `.lin/templates/` | ✅ 用户决定提交（团队共享） | 不排除 |
| `.lin/` 顶层 | 见上 | 仅 ignore 上述敏感文件 |

`lin new` 生成的 `.gitignore` 模板必须包含：

```gitignore
# lin scaffolding tool runtime files
.lin/.backup/
.lin/.last-run.json
```

`lin add` 执行前**强校验**：若项目 `.gitignore` 缺少 `.lin/.backup/` 行，会先自动追加（`⚠ updated .gitignore`），避免备份污染版本控制。

### 5.2 备份生命周期

| 时机 | 动作 |
| --- | --- |
| 注入前 | `cp <central-file> .lin/.backup/<ts>/<central-file>` |
| 注入成功 | `rm -rf .lin/.backup/<ts>/`（保留最近 3 次） |
| 注入失败 | `cp -f .lin/.backup/<ts>/* <to-original-paths>`，保留备份直到下次成功 |
| 用户主动 | `lin lint --restore-backup <ts>`（计划中，Phase 2+） |

---

## 6. `--no-inject` 模式

当用户希望仅生成文件、跳过注入时（用于调试或手工合并）：

```bash
lin add Post --no-inject
```

输出：

```
🎯 Adding resource: Post (--no-inject mode)

📦 Creating files...
   ✔ internal/myblog/handler/post.go
   ✔ internal/myblog/biz/v1/post/post.go
   ... (8 files)

⏭ Skipping AST injection.

⚠️  Manual injection required:
   Add to internal/myblog/biz/biz.go:

       PostV1() postv1.PostBiz

   Add to internal/myblog/biz/biz.go (struct method):

       func (b *biz) PostV1() postv1.PostBiz {
           return postv1.New(b.store)
       }

   Add to internal/myblog/store/store.go:

       Posts() PostStore

   Add to pkg/api/myblog/v1/myblog.proto:

       import "post.proto";

   Add to internal/pkg/errno/register.go:

       RegisterErrors(PostErrors()...)
```

---

## 7. 边界场景与处理

### 7.1 锚点状态机

锚点必须**成对**出现（`inject-region:<name>` 起 + `inject-region-end` 止）。任何半破坏均视为缺失：

| 锚点状态 | start 注释 | end 注释 | 处理 |
| --- | --- | --- | --- |
| 完整 | ✅ | ✅ | 正常注入 |
| 仅缺 start | ❌ | ✅ | **视为缺失**；exit 24；提示用 `lin lint --fix` 重建 |
| 仅缺 end | ✅ | ❌ | **视为缺失**；同上 |
| 完全缺失 | ❌ | ❌ | exit 24；可用 `lin lint --fix` 区域整体重写 |
| 嵌套（同一 name 出现两次） | ✅✅ | ✅✅ | exit 24；提示「duplicate anchor」 |
| 跨函数边界 | ✅ | ✅（错位） | exit 24；提示「anchor span across function」 |

**算法伪代码**：

```go
func detectAnchor(file *dst.File, name string) (start, end *dst.Comment, err error) {
    starts := findAll(file, "// lin: inject-region:" + name)
    ends := findAllAfter(file, starts, "// lin: inject-region-end")

    switch {
    case len(starts) == 0 && len(ends) == 0:
        return nil, nil, ErrAnchorMissing
    case len(starts) == 1 && len(ends) == 1 && spanIsValid(starts[0], ends[0]):
        return starts[0], ends[0], nil
    case len(starts) > 1 || len(ends) > 1:
        return nil, nil, ErrAnchorDuplicate
    default:
        return nil, nil, ErrAnchorMalformed
    }
}
```

### 7.2 其他边界场景

| 场景 | 处理 |
| --- | --- |
| 用户在锚点区域内手工添加了同名方法 | 幂等检查通过，跳过；记录 `⊝ skipped` |
| 用户在锚点区域外手工添加了方法 | 仍然注入到锚点区域内（生成代码会出现重复），报 warning |
| 文件被人为破坏（Go 语法错误） | dst.Parse 失败，注入中止，文件保持原样；exit 50 |
| 多 app 项目 | 通过 `--app=<name>` 路由到对应目录；详见 [02 §4.4](./02-command-set.md#44-上下文推断) |
| Windows 路径分隔符 | 内部路径全部 `filepath.ToSlash` |
| Windows 行尾符 (CRLF) | 解析时归一化为 LF；写入时按目标文件原行尾符保留 |
| 文件 git untracked | 注入正常进行；用户可后续 `git add` |
| 文件 git uncommitted（已 staged） | 注入正常；用户可看 `git diff` 验证；`--strict` 模式可要求工作区干净 |
| `.lin/.backup/` 已被 git tracked | warn；提示用户检查 `.gitignore`（详见 [§5.1](#51-lin-工作目录与-gitignore-契约)） |
| 非 `master`/`main` 分支 | 不影响；lin 不操作 git 引用 |

---

## 8. 测试策略

### 8.1 单测覆盖

| 测试文件 | 覆盖 | 数据 |
| --- | --- | --- |
| `mutator_interface_test.go` | AddInterfaceMethod 幂等性 / 锚点缺失 / 多 method | testdata/biz_*.go |
| `mutator_proto_test.go` | AddProtoImport 重复 / 多 import | testdata/*.proto |
| `mutator_register_test.go` | AppendRegistration 重复 / 锚点缺失 | testdata/register_*.go |
| `injector_test.go` | 全流程：创建文件 + 注入 + 回滚 | E2E |

测试方法：

- 输入：模拟 `biz.go`（含锚点）+ payload
- 输出：序列化后的文件应包含期望的代码片段
- 反向：再次执行同样注入 → 文件不变（幂等）

### 8.2 E2E 测试

```bash
# tests/e2e/add_resource_test.sh
lin new myblog --module github.com/test/myblog --storage memory
cd myblog
lin add Post
lin add Post   # 第二次：应幂等
go build ./...   # 必须通过
lin add Comment Tag --no-inject
ls internal/myblog/handler/  # 应有 post.go / comment.go / tag.go
```

---

## 9. 实现清单（开发顺序）

| Phase | 任务 | 工作量估计 |
| --- | --- | --- |
| 1 | `dst` 集成 + 锚点检测 | 1d |
| 2 | `mutator_interface` 实现 + 单测 | 2d |
| 3 | `mutator_proto` 实现 + 单测 | 1d |
| 4 | `mutator_register` 实现 + 单测 | 1d |
| 5 | `injector.go` 编排 + 备份/回滚 | 1d |
| 6 | E2E 集成测试 | 1d |
| **小计** | | **7d** |

---

## 10. 与当前 lin AST 模块的差异

| 维度 | 当前 `lin/internal/ast/` | 重构后 |
| --- | --- | --- |
| Mutator 数量 | 1（addinterface） | 3（interface / proto / register） |
| 锚点注释 | 未使用 | **强制使用**，提供精确定位 |
| 幂等性 | 部分 | **全面支持** |
| 事务 / 回滚 | 无 | **全流程事务** |
| Proto 修改 | 未实现 | 文本追加（基于 protocompile 校验） |
| 测试覆盖 | ~40% | 目标 ≥ 90% |

---

_Last reviewed: 2026-04-29_
