# ADR-001: 用 `dave/dst` 替代 `go/ast` 做 Go AST 注入

## 元数据

| 字段 | 值 |
| --- | --- |
| **状态** | ✅ Accepted (2026-04-25) |
| **日期** | 2026-04-25 |
| **作者** | @clin211 |
| **审阅人** | TBD |
| **相关 ADR** | 无 |
| **影响范围** | `internal/ast/`、`07-ast-injection.md` |
| **相关 issue/PR** | - |

---

## 1. 背景

osbuilder 的 AST 注入实现存在两类反模式：

1. **直接用标准库 `go/ast`**：
   ```go
   field := &ast.Field{
       Names: []*ast.Ident{ast.NewIdent("Posts")},
       Type:  &ast.Ident{Name: "PostStore"},
   }
   ```
   → 修改后 `printer.Fprint` 会**丢失原文件的注释和空行**，每次 `add api` 后已有文件越改越丑。

2. **把表达式塞进 Ident.Name**：
   ```go
   &ast.Ident{Name: "newPostStore(store)"}  // 整段表达式作为标识符名
   ```
   → 不符合 `ast.Ident` 的契约，下游任何 `ast.Walk` / `astutil.Apply` 都会失败；遇到含泛型/接口断言/换行的表达式直接断裂。

为了让 linctl 的 AST 注入既"**保留用户编辑过的注释/空行**"，又"**用标准方式构造表达式**"，必须换库。

## 2. 决策

我们决定用 [`github.com/dave/dst`](https://github.com/dave/dst) 替代 `go/ast`，并配合 `parser.ParseExpr` 构造表达式。

关键变更：

- 所有 `internal/ast/go_inject.go` 中的 AST 类型从 `*ast.X` 改为 `*dst.X`。
- 文件读写的入口/出口分别用 `decorator.Decorator.Parse` 和 `decorator.Restorer.Print`。
- 任何需要构造表达式的地方（如 struct method 的 `BodyExpr`），改用 `parser.ParseExpr` 后通过 `decorator.Decorate` 转成 `dst.Expr`。

## 3. 备选方案

| # | 方案 | 优势 | 劣势 | 否决原因 |
| --- | --- | --- | --- | --- |
| 1 | **`dave/dst`**（chosen） | 保留注释/空行；API 与 `go/ast` 几乎一致；活跃维护（最近一次 release 2025） | 小众库；可能有未发现的 bug | - |
| 2 | `go/ast` + `golang.org/x/tools/go/ast/astutil` | 标准库内 | astutil 不解决"丢注释"问题；只能修补 import | 没解决核心痛点 |
| 3 | 自研基于 `go/printer` 的 patcher | 完全可控 | 工作量极大；要重新实现 dst 的全部逻辑 | ROI 太低 |
| 4 | 字符串 + 正则替换 | 零依赖 | 任何复杂结构都会断裂；与 osbuilder Proto 同样的问题 | 已被 osbuilder 验证不可行 |

## 4. 后果

### 4.1 正面

- 修改后的文件**完整保留**用户原有注释、空行、缩进。
- 表达式构造走标准 parser，**任何复杂表达式**（含泛型 `Generic[T]`、断言 `x.(*T)`、闭包等）都能正确处理。
- 单测可对比"输入 → 输出"做精确字符串比对（不再有"今天和昨天 print 出的格式不同"的玄学问题）。

### 4.2 负面 / Trade-offs

- 引入一个**非标准库依赖**（约 20 KB，纯 Go）。
- 学习曲线：开发者需要理解 `decorator.Decorator` / `decorator.Restorer` / `dst.Decorations` 三个新概念。
- `dst` 库本身存在小众性风险，如果维护停滞，需要 fork 维护或换库。

### 4.3 中性

- 二进制大小增加约 **80~150 KB**（含 dst + 间接依赖）。
- 编译时间几乎无影响。

## 5. 风险缓解

- **抽象层**：所有 AST 操作都封装在 `internal/ast/` 包内的 `ASTMutator` 接口下。如果未来需要换库，只改这一个包即可，**不影响其他模块**。
- **CI 监控**：每月 CI 跑一次 `go list -m -u all` 检查 dst 是否有更新；超过 6 个月没更新触发 alert。
- **替代方案预案**：如果 dst 真的死了，备选项是 fork + 自维护；最终极方案是回退到 `go/ast` + 手写 comment-preserving printer。

## 6. 参考资料

- [dave/dst GitHub](https://github.com/dave/dst)
- [Why dst? Decorations - the godoc](https://pkg.go.dev/github.com/dave/dst)
- [设计文档：07-ast-injection.md §7.3](../07-ast-injection.md)
- [对比：go/ast 修改丢注释的 issue](https://github.com/golang/go/issues/20744)（标准库长期未修复）

---

_Last reviewed: 2026-04-25_
