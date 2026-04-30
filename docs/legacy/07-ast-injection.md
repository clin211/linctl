# 07. AST 注入机制（Go + Proto）

## 7.1 为什么需要 AST 注入

osbuilder 的核心创新是 "**新增一个 REST 资源只改动必要的几处**" —— 这要求工具能：

1. 给 `IBiz` 接口加一行方法
2. 给 `biz` struct 加一个方法
3. 给 `IStore` 接口加一行方法
4. 给 `store` struct 加一个方法
5. 给 `apiserver.proto` 的 service 块加 5 个 RPC 方法
6. 给 `all.go` 加一个 `_ "..."` 匿名 import

**不能**简单地"重新生成整个文件"，因为：
- 用户可能在这些文件里加了自己的方法（如自定义的 `MyCustom() MyStore`）。
- 模板渲染会丢失用户的修改。

**只能**"在已有文件上做精确的局部修改"。这就是 AST 注入。

## 7.2 osbuilder 的 AST 实现及其问题

osbuilder 使用 `go/ast`（标准库）：

```go
// osbuilder/internal/osbuilder/file/method.go (节选)
fset := token.NewFileSet()
node, _ := parser.ParseFile(fset, filePath, nil, parser.AllErrors|parser.ParseComments)

// 给接口加方法
field := &ast.Field{
    Names: []*ast.Ident{ast.NewIdent("Posts")},
    Type:  &ast.Ident{Name: "PostStore"},
}

// 给 struct 加方法
funcDecl := &ast.FuncDecl{
    Recv: &ast.FieldList{...},
    Name: ast.NewIdent("Posts"),
    Body: &ast.BlockStmt{
        List: []ast.Stmt{
            &ast.ReturnStmt{
                Results: []ast.Expr{
                    &ast.Ident{Name: "newPostStore(store)"}, // ⚠️ 把表达式塞进 Ident.Name！
                },
            },
        },
    },
}
```

**问题**：
- `*ast.Ident{Name: "newPostStore(store)"}` 把整段表达式塞进 `Ident.Name`，这**不是标准用法**（`Ident` 应只持有标识符）。
- `printer.Fprint` 侥幸能原样打印出来，但任何 AST analyzer / 后续 `ast.Walk` 都会失败。
- 复杂表达式（含泛型、接口断言、换行）会断裂。
- **关键缺陷**：`go/ast` 在 Print 时会**丢失原始注释和空行**，导致已有文件越改越丑。

## 7.3 linctl 的方案：`dave/dst` + `parser.ParseExpr`

[`dave/dst`](https://github.com/dave/dst) 是 `go/ast` 的"装饰版"，叫 **D**ecorated **S**yntax **T**ree，专门解决"AST 修改后丢失注释/空行"的问题。

### 7.3.1 dst 的核心 API

```go
import (
    "github.com/dave/dst"
    "github.com/dave/dst/decorator"
)

// Decorator: 把 go/ast.File 转成 dst.File（保留所有装饰信息）
dec := decorator.NewDecorator(fset)
file, err := dec.Parse([]byte(src))

// 修改 file.Decls / file.Imports

// Restorer: 把 dst.File 转回 go/ast.File
res := decorator.NewRestorer()
out, err := res.Print(file) // 含原注释、空行
```

### 7.3.2 关键好处

| 维度 | go/ast | dst |
| --- | --- | --- |
| 修改后保留注释 | ❌ | ✅ |
| 修改后保留空行 | ❌ | ✅ |
| 表达式构造 | 手写 `*ast.CallExpr` 套娃 | 用 `parser.ParseExpr` 直接解析字符串 |
| 学习曲线 | 中 | 中（API 与 ast 几乎一致） |

### 7.3.3 表达式构造的最佳实践

❌ **错误做法**（osbuilder）：
```go
&ast.Ident{Name: "newPostStore(store)"}
```

✅ **正确做法**（linctl）：
```go
expr, err := parser.ParseExpr("newPostStore(store)")
if err != nil { return err }

// 转成 dst
dstExpr := decorator.NewDecorator(fset).Decorate(expr).(dst.Expr)
```

✅ **更好的做法**（推荐）：用 `dstutil.Apply` 高阶 API：

```go
import "github.com/dave/dst/dstutil"

dstutil.Apply(file, func(c *dstutil.Cursor) bool {
    // 在 cursor 当前位置插入新节点
    return true
}, nil)
```

## 7.4 ASTInjector 接口设计

```go
// internal/ast/mutator.go
package ast

type Layer string
const (
    LayerStore Layer = "store"
    LayerBiz   Layer = "biz"
    LayerProto Layer = "proto"
    LayerAllGo Layer = "all_go"  // jobserver/cmd 的匿名 import 注入
)

// ASTMutator 描述一次 AST 修改
type ASTMutator interface {
    Layer() Layer
    TargetFile() string  // 项目相对路径
    Description() string // 用于 plan 报告
    Apply(content []byte) (newContent []byte, modified bool, err error)
}

// 内置 Mutator
type AddInterfaceMethodMutator struct {
    File          string
    InterfaceName string  // "IBiz" / "IStore"
    MethodName    string  // "Posts"
    ReturnType    string  // "PostBiz" / "PostStore"
    DocComment    string  // 可选，给方法加注释
}

type AddStructMethodMutator struct {
    File         string
    StructName   string  // "biz" / "store"
    MethodName   string
    ReceiverName string  // "b" / "s"
    ReceiverType string  // "*biz" / "*store"
    ReturnType   string
    BodyExpr     string  // "return newPostStore(s.store)"
}

type AddImportMutator struct {
    File       string
    Alias      string  // 可选 alias
    ImportPath string
    Anonymous  bool    // 是否 `_ "..."`
}

type AddProtoRPCMutator struct {
    File         string
    ServiceName  string  // "APIServer"
    Methods      []ProtoRPCMethod
    Imports      []string // 需要 import 的其他 .proto 文件
}

type ProtoRPCMethod struct {
    Name        string  // "CreatePost"
    RequestType string  // "CreatePostRequest"
    ResponseType string // "CreatePostResponse"
    Streaming   StreamingMode  // none/server/client/bidi
    HTTPRule    string  // 可选：google.api.http 规则
}
```

## 7.5 Go AST 注入实现

### 7.5.1 给接口加方法

```go
// internal/ast/go_inject.go
package ast

import (
    "fmt"
    "go/parser"
    "go/token"

    "github.com/dave/dst"
    "github.com/dave/dst/decorator"
    "mvdan.cc/gofumpt/format"
)

func (m *AddInterfaceMethodMutator) Apply(content []byte) ([]byte, bool, error) {
    fset := token.NewFileSet()
    dec := decorator.NewDecorator(fset)
    file, err := dec.Parse(content)
    if err != nil {
        return nil, false, fmt.Errorf("parse %s: %w", m.File, err)
    }

    // Step 1: 找到目标 interface
    var iface *dst.InterfaceType
    for _, decl := range file.Decls {
        gen, ok := decl.(*dst.GenDecl)
        if !ok || gen.Tok != token.TYPE {
            continue
        }
        for _, spec := range gen.Specs {
            ts, ok := spec.(*dst.TypeSpec)
            if !ok || ts.Name.Name != m.InterfaceName {
                continue
            }
            i, ok := ts.Type.(*dst.InterfaceType)
            if ok {
                iface = i
                break
            }
        }
        if iface != nil { break }
    }
    if iface == nil {
        return nil, false, fmt.Errorf("interface %s not found in %s", m.InterfaceName, m.File)
    }

    // Step 2: 检查方法是否已存在 (按"名字 + 签名" 匹配，详见本文档 §7.7 方法去重策略)
    //   - 名字 + 签名完全匹配 → 跳过（幂等）
    //   - 名字相同但签名不同 → 写 warn + 标记为 Conflict，由 Plan 报告并交由用户处理
    for _, field := range iface.Methods.List {
        if len(field.Names) == 0 || field.Names[0].Name != m.MethodName {
            continue
        }
        existingSig := normalizeFuncSig(field.Type)
        wantSig := wantFuncSig(m)
        if existingSig == wantSig {
            return content, false, nil // 完全一致，幂等返回
        }
        log.L().Warn("interface method exists with different signature; recording as Conflict",
            "interface", m.InterfaceName, "method", m.MethodName,
            "existing", existingSig, "want", wantSig)
        return content, false, &ConflictError{
            Kind: "interface_method_signature_mismatch",
            File: m.File, Symbol: m.InterfaceName + "." + m.MethodName,
            Existing: existingSig, Want: wantSig,
        }
    }

    // Step 3: 构造新方法字段
    methodField := &dst.Field{
        Names: []*dst.Ident{{Name: m.MethodName}},
        Type: &dst.FuncType{
            Params: &dst.FieldList{},
            Results: &dst.FieldList{
                List: []*dst.Field{{
                    Type: &dst.Ident{Name: m.ReturnType},
                }},
            },
        },
    }
    if m.DocComment != "" {
        methodField.Decs.Start.Append("// " + m.DocComment)
        methodField.Decs.Before = dst.NewLine
    }

    // Step 4: 追加到 interface
    iface.Methods.List = append(iface.Methods.List, methodField)

    // Step 5: Restore + format
    return finalize(fset, file)
}

func finalize(fset *token.FileSet, file *dst.File) ([]byte, bool, error) {
    res := decorator.NewRestorer()
    var buf bytes.Buffer
    if err := res.Fprint(&buf, file); err != nil {
        return nil, false, fmt.Errorf("restore: %w", err)
    }
    formatted, err := format.Source(buf.Bytes(), format.Options{LangVersion: "1.22"})
    if err != nil {
        return nil, false, fmt.Errorf("gofumpt: %w\n--- raw ---\n%s", err, buf.String())
    }
    return formatted, true, nil
}
```

### 7.5.2 给 struct 加方法

```go
func (m *AddStructMethodMutator) Apply(content []byte) ([]byte, bool, error) {
    fset := token.NewFileSet()
    dec := decorator.NewDecorator(fset)
    file, err := dec.Parse(content)
    if err != nil {
        return nil, false, err
    }

    // Step 1: 检查方法是否已存在（按"名字 + 签名"匹配）
    //   - 完全一致 → 幂等返回
    //   - 同名但签名/接收者不同 → warn + Conflict（由 Plan 报告）
    wantSig := structMethodSig(m)
    for _, decl := range file.Decls {
        fn, ok := decl.(*dst.FuncDecl)
        if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
            continue
        }
        if fn.Name.Name != m.MethodName {
            continue
        }
        existingSig := normalizeFuncDecl(fn)
        if existingSig == wantSig {
            return content, false, nil
        }
        log.L().Warn("struct method exists with different signature; recording as Conflict",
            "type", recvType(fn), "method", m.MethodName,
            "existing", existingSig, "want", wantSig)
        return content, false, &ConflictError{
            Kind: "struct_method_signature_mismatch",
            File: m.File, Symbol: recvType(fn) + "." + m.MethodName,
            Existing: existingSig, Want: wantSig,
        }
    }

    // Step 2: 用 parser.ParseExpr 解析 body 表达式
    bodyExpr, err := parser.ParseExpr(m.BodyExpr)
    if err != nil {
        return nil, false, fmt.Errorf("parse body expr %q: %w", m.BodyExpr, err)
    }
    dstBodyExpr := decorator.NewDecorator(fset).Decorate(bodyExpr).(dst.Expr)

    // Step 3: 构造完整 FuncDecl
    fn := &dst.FuncDecl{
        Recv: &dst.FieldList{
            List: []*dst.Field{{
                Names: []*dst.Ident{{Name: m.ReceiverName}},
                Type:  &dst.StarExpr{X: &dst.Ident{Name: stripPointer(m.ReceiverType)}},
            }},
        },
        Name: &dst.Ident{Name: m.MethodName},
        Type: &dst.FuncType{
            Params: &dst.FieldList{},
            Results: &dst.FieldList{
                List: []*dst.Field{{Type: &dst.Ident{Name: m.ReturnType}}},
            },
        },
        Body: &dst.BlockStmt{
            List: []dst.Stmt{
                &dst.ReturnStmt{
                    Results: []dst.Expr{dstBodyExpr},
                },
            },
        },
    }
    fn.Decs.Before = dst.EmptyLine // 与上一个 decl 之间留空行

    // Step 4: 追加到文件末尾
    file.Decls = append(file.Decls, fn)

    return finalize(fset, file)
}

func recvType(fn *dst.FuncDecl) string {
    if fn.Recv == nil || len(fn.Recv.List) == 0 { return "" }
    switch t := fn.Recv.List[0].Type.(type) {
    case *dst.StarExpr:
        if id, ok := t.X.(*dst.Ident); ok { return "*" + id.Name }
    case *dst.Ident:
        return t.Name
    }
    return ""
}

func stripPointer(s string) string {
    if strings.HasPrefix(s, "*") { return s[1:] }
    return s
}
```

### 7.5.3 添加 import

```go
func (m *AddImportMutator) Apply(content []byte) ([]byte, bool, error) {
    fset := token.NewFileSet()
    dec := decorator.NewDecorator(fset)
    file, err := dec.Parse(content)
    if err != nil {
        return nil, false, err
    }

    // Step 1: 检查 import 是否已存在
    for _, imp := range file.Imports {
        path := strings.Trim(imp.Path.Value, `"`)
        if path == m.ImportPath {
            return content, false, nil
        }
    }

    // Step 2: 找到第一个 import block 或创建一个
    var importDecl *dst.GenDecl
    for _, decl := range file.Decls {
        if g, ok := decl.(*dst.GenDecl); ok && g.Tok == token.IMPORT {
            importDecl = g
            break
        }
    }
    if importDecl == nil {
        importDecl = &dst.GenDecl{Tok: token.IMPORT, Lparen: true, Rparen: true}
        // 插入到 package 声明之后
        file.Decls = append([]dst.Decl{importDecl}, file.Decls...)
    }

    // Step 3: 构造新 ImportSpec
    importSpec := &dst.ImportSpec{
        Path: &dst.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", m.ImportPath)},
    }
    if m.Alias != "" {
        importSpec.Name = &dst.Ident{Name: m.Alias}
    }
    if m.Anonymous {
        importSpec.Name = &dst.Ident{Name: "_"}
    }

    importDecl.Specs = append(importDecl.Specs, importSpec)

    return finalize(fset, file)
}
```

## 7.6 Proto AST 注入实现

### 7.6.1 osbuilder 的方案及其问题

osbuilder 使用**字符串行扫描**：

```go
// osbuilder/internal/osbuilder/file/proto.go
func applyUpdates(src, kind, grpcServiceName, importPath string) (string, bool, error) {
    lines := strings.Split(src, "\n")
    for i, line := range lines {
        if strings.Contains(line, "service "+grpcServiceName+" {") {
            // 在这一行后追加 5 个 rpc method
        }
    }
    // ...
}
```

**问题**：
- 注释里写了 `service Foo {` → 误判
- gRPC streaming / google.api.http 注解 → 不支持
- 多个 service 定义 → 处理不正确
- 没有重复检测

### 7.6.2 linctl 的方案：`bufbuild/protocompile` 的 descriptor 回写（**唯一 canonical 路径**）

> **决策**：此前草稿同时存在「descriptor 回写」与「文本插入」两套 `Apply()` 实现，
> 已**删除文本插入版本**，确保 `AddProtoRPCMutator` 在所有场景下走同一条路径。
> 这避免了「Phase 1 用方案 A、Phase 4 用方案 B」可能造成的行为差异。

**唯一 Apply 流程**：
1. 用 `protocompile/parser` 解析 .proto 文件 → `FileDescriptorProto`。
2. 在 descriptor 上做 **结构化** 修改（添加 method、import）。
3. 通过 `printProtoFile()` 把 descriptor 写回 .proto 文本（保留原注释/格式 → §7.6.3 详述）。
4. （可选）`buf format` 美化最终输出。

```go
// internal/ast/proto_inject.go
package ast

import (
    "context"
    "fmt"
    "strings"

    "github.com/bufbuild/protocompile/parser"
    "github.com/bufbuild/protocompile/reporter"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/descriptorpb"
)

// Apply 是 AddProtoRPCMutator 的**唯一** canonical 实现（descriptor 回写）。
// 详见本文档 §7.6.2 与 §7.7 方法去重策略：
//   - 不再保留"文本插入"作为 Phase 1 替代方案
//   - 同名异签的 rpc 发 warn 并返回 ConflictError（与 Go AST 路径一致）
func (m *AddProtoRPCMutator) Apply(content []byte) ([]byte, bool, error) {
    handler := reporter.NewHandler(reporter.NewReporter(
        func(err reporter.ErrorWithPos) error { return err },
        func(reporter.ErrorWithPos) {},
    ))

    res, err := parser.Parse(m.File, strings.NewReader(string(content)), handler)
    if err != nil {
        return nil, false, fmt.Errorf("proto parse: %w", err)
    }

    fileDesc := res.FileDescriptorProto()

    // Step 1: 找到目标 service
    var svc *descriptorpb.ServiceDescriptorProto
    for _, s := range fileDesc.Service {
        if s.GetName() == m.ServiceName {
            svc = s
            break
        }
    }
    if svc == nil {
        return nil, false, fmt.Errorf("service %s not found", m.ServiceName)
    }

    // Step 2: 按"名字 + 签名"判定方法是否已存在
    existing := make(map[string]*descriptorpb.MethodDescriptorProto)
    for _, method := range svc.Method {
        existing[method.GetName()] = method
    }

    var added []ProtoRPCMethod
    for _, method := range m.Methods {
        wantSig := protoMethodSig(method)
        if existed, ok := existing[method.Name]; ok {
            if methodSigOf(existed) == wantSig {
                continue // 完全一致，幂等
            }
            // 同名异签 → warn + Conflict（与 Go AST 路径一致）
            log.L().Warn("proto rpc exists with different signature",
                "service", m.ServiceName, "method", method.Name,
                "existing", methodSigOf(existed), "want", wantSig)
            return content, false, &ConflictError{
                Kind: "proto_rpc_signature_mismatch",
                File: m.File, Symbol: m.ServiceName + "." + method.Name,
                Existing: methodSigOf(existed), Want: wantSig,
            }
        }
        md := &descriptorpb.MethodDescriptorProto{
            Name:       proto.String(method.Name),
            InputType:  proto.String("." + method.RequestType),
            OutputType: proto.String("." + method.ResponseType),
        }
        switch method.Streaming {
        case StreamingServer:
            md.ServerStreaming = proto.Bool(true)
        case StreamingClient:
            md.ClientStreaming = proto.Bool(true)
        case StreamingBidi:
            md.ServerStreaming = proto.Bool(true)
            md.ClientStreaming = proto.Bool(true)
        }
        svc.Method = append(svc.Method, md)
        added = append(added, method)
    }

    if len(added) == 0 {
        return content, false, nil
    }

    // Step 3: 添加 imports
    for _, imp := range m.Imports {
        if !contains(fileDesc.Dependency, imp) {
            fileDesc.Dependency = append(fileDesc.Dependency, imp)
        }
    }

    // Step 4: descriptor → 文本（见 §7.6.3）
    return printProtoFile(fileDesc, content)
}
```

### 7.6.3 `printProtoFile`：descriptor → text 的分阶段实现

`bufbuild/protocompile` 提供 AST 解析与 descriptor 模型，但**没有**直接的 descriptor → text 输出。`printProtoFile` 是上层封装，**对外只暴露一个稳定入口**，内部按 Phase 切换实现细节：

| Phase | 内部策略 | 何时切换 |
| --- | --- | --- |
| **Phase 1** | 在原始文本上**结构化定位**（用 protocompile 给出的 service `{}` 位置）后插入新方法行 + imports；最终 `buf format` 兜底 | MVP 即可工作；保留原注释/空行最稳妥 |
| **Phase 4** | 完全基于 descriptor 模型重新生成 .proto；用模板 + `buf format` 输出 | 当用户需要更复杂的 descriptor 改动（移除 method、改 streaming）时 |

> 关键：调用方（`Apply`）**永远只调 `printProtoFile`**；切换 Phase 不影响调用方代码或测试断言。这就是"唯一 canonical 路径"的含义。

```go
// 对外契约：仅此一个签名
func printProtoFile(desc *descriptorpb.FileDescriptorProto, original []byte) ([]byte, error)
```

### 7.6.4 端到端可执行示例

```go
// 端到端使用：与 §7.6.2 的 Apply 完全一致，无任何替代分支
mutator := &AddProtoRPCMutator{
    File:        "pkg/api/myblog/v1/myblog.proto",
    ServiceName: "APIServer",
    Methods: []ProtoRPCMethod{
        {Name: "CreatePost", RequestType: "CreatePostRequest", ResponseType: "CreatePostResponse"},
    },
    Imports: []string{"v1/post.proto"},
}
out, modified, err := mutator.Apply(content)
```

## 7.7 时序图：AST 注入完整流程

```mermaid
sequenceDiagram
    autonumber
    participant Apply as Applier
    participant Mutator as ASTMutator
    participant FM as fs.FileManager
    participant Decor as decorator.Decorator
    participant File as dst.File
    participant Parser as parser.ParseExpr
    participant Restor as decorator.Restorer
    participant Gofumpt as gofumpt.Source

    Apply->>Mutator: Apply(content)
    Mutator->>FM: (no read; content 已从 plan 阶段读取)
    
    Mutator->>Decor: NewDecorator(fset)
    Mutator->>Decor: Parse(content)
    Decor-->>Mutator: dst.File
    
    Note over Mutator,File: 检查目标 (interface/struct/import)<br/>是否已存在
    Mutator->>File: 遍历 Decls
    File-->>Mutator: target node 或 nil
    
    alt target 已存在 (重复)
        Mutator-->>Apply: (content, modified=false, nil)
    else target 不存在
        alt 需要构造表达式 (如 struct method body)
            Mutator->>Parser: ParseExpr("newPostStore(s.store)")
            Parser-->>Mutator: ast.Expr
            Mutator->>Decor: Decorate(expr)
            Decor-->>Mutator: dst.Expr
        end
        
        Mutator->>File: 修改 (Append field/decl/import)
        Mutator->>Restor: NewRestorer()
        Mutator->>Restor: Print(file)
        Restor-->>Mutator: 字节流
        Mutator->>Gofumpt: Source(bytes)
        Gofumpt-->>Mutator: formatted bytes
        Mutator-->>Apply: (formatted, modified=true, nil)
    end

    Apply->>FM: Write(file, formatted)
```

> 完整源文件见 [diagrams/seq-ast-injection.mmd](./diagrams/seq-ast-injection.mmd)。

## 7.8 测试 AST 注入

由于 AST 操作是最容易出错的部分，单测必须覆盖：

```go
// internal/ast/go_inject_test.go
package ast

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestAddInterfaceMethod(t *testing.T) {
    cases := []struct {
        name     string
        input    string
        mutator  AddInterfaceMethodMutator
        want     string
        modified bool
    }{
        {
            name: "add to empty interface",
            input: `package store

type IStore interface {
}
`,
            mutator: AddInterfaceMethodMutator{
                InterfaceName: "IStore",
                MethodName:    "Posts",
                ReturnType:    "PostStore",
            },
            want: `package store

type IStore interface {
	Posts() PostStore
}
`,
            modified: true,
        },
        {
            name: "skip if already exists",
            input: `package store

type IStore interface {
	Posts() PostStore
}
`,
            mutator: AddInterfaceMethodMutator{
                InterfaceName: "IStore",
                MethodName:    "Posts",
                ReturnType:    "PostStore",
            },
            want:     /* same as input */,
            modified: false,
        },
        {
            name: "preserve existing comments",
            input: `package store

// IStore is the main store interface.
type IStore interface {
	// Users returns the user store.
	Users() UserStore
}
`,
            mutator: AddInterfaceMethodMutator{
                InterfaceName: "IStore",
                MethodName:    "Posts",
                ReturnType:    "PostStore",
            },
            want: `package store

// IStore is the main store interface.
type IStore interface {
	// Users returns the user store.
	Users() UserStore
	Posts() PostStore
}
`,
            modified: true,
        },
        // 更多 case: 添加到 struct / import / 嵌套类型...
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got, modified, err := tc.mutator.Apply([]byte(tc.input))
            require.NoError(t, err)
            require.Equal(t, tc.modified, modified)
            require.Equal(t, tc.want, string(got))
        })
    }
}
```

## 7.9 边界情况与缓解

> 关于"方法去重"的统一规则（决策见 [META-fix-decisions §1.13](./META-fix-decisions-2026-04-25.md) 同精神）：
> **所有 Go AST mutator 一律按"方法名 + 签名"匹配**，避免仅按名字误判幂等。

| 边界情况 | 处理 |
| --- | --- |
| 文件包含语法错误 | parse 时报错，返回 LinctlError 给用户 |
| 接口/结构体不存在 | 报错并提示用户检查 |
| 方法已存在且名字+签名完全一致 | 幂等跳过（modified=false） |
| 方法已存在但签名不同（同名异签） | warn 日志 + 返回 `ConflictError`，由 Plan 标记为 Conflict 由用户处理 |
| 文件含泛型 | dst 完全支持 Go 1.18+ 泛型；签名比较时归一化类型参数顺序 |
| import 已存在且 path+alias 一致 | 幂等跳过 |
| import 已存在但 alias 不同 | 默认跳过；`--strict` 模式报错 |
| .proto 文件格式错乱 | protocompile 报错；提示用户先 `buf format` |
| .proto 中 service 不存在 | 报错并列出可用 service |
| .proto rpc 同名异签 | 与 Go 一致：warn + Conflict |
| Diff 触发 git merge conflict | 检测 `<<<<<<<` markers，跳过修改 |

## 7.10 性能考虑

- AST 操作单次 < 50ms（小文件）/ ~200ms（大文件 5K 行）。
- 同一文件多个 mutator 应**合并**为单次 parse + 多次修改 + 单次 write。

```go
// internal/ast/batch.go
type Batch struct {
    file      string
    mutators  []ASTMutator
}

func (b *Batch) Apply(content []byte) ([]byte, bool, error) {
    fset := token.NewFileSet()
    dec := decorator.NewDecorator(fset)
    file, err := dec.Parse(content)
    if err != nil { return nil, false, err }

    modified := false
    for _, m := range b.mutators {
        // 注意: mutator 操作 dst.File 而不是 content
        // 需要 mutator 提供 ApplyDST(file *dst.File) bool 接口
        if m.ApplyDST(file) {
            modified = true
        }
    }
    if !modified { return content, false, nil }

    return finalize(fset, file)
}
```

## 7.11 与 osbuilder 的对比总结

| 维度 | osbuilder | linctl |
| --- | --- | --- |
| Go AST 库 | `go/ast`（标准库） | `dave/dst`（保留装饰） |
| Body 表达式构造 | `*ast.Ident{Name: "整段表达式"}` | `parser.ParseExpr` |
| 注释保留 | ❌ | ✅ |
| 空行保留 | ❌ | ✅ |
| Proto 修改 | 字符串行扫描 | `bufbuild/protocompile` descriptor 回写（统一 canonical 路径，见 §7.6） |
| 重复检测 | 仅按方法名 | 按方法名 + 签名（同名异签发 warn + 标 Conflict） |
| 单测 | 无 | 表驱动 + golden file |
| Batch 优化 | 无 | 同文件多 mutator 合并 |
| 错误处理 | 部分 silent fail | LinctlError + 完整上下文 |

---

下一步阅读：[08-feature-system.md](./08-feature-system.md)

_Last reviewed: 2026-04-25_
