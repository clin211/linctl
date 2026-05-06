package ast

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// InterfacePayload 是 AddInterfaceMethod 的入参。
//
// 插入点完全由 Go AST 结构（接口名、receiver 名）决定 ——
// 不依赖任何锚点注释。
type InterfacePayload struct {
	// InterfaceName 是接口类型名（如 "IBiz"）。
	InterfaceName string
	// StructName 是 receiver 类型名（如 "biz"）。
	StructName string

	// Method 是要新增的方法名（如 "PostV1"）。
	Method string
	// ReturnType 是返回类型字符串（如 "postv1.PostBiz"）。
	ReturnType string

	// ImportAlias 与 ImportPath 共同构成需要新增的 import（可选）。
	ImportAlias string
	ImportPath  string

	// ImplBody 是函数体表达式（如 "return postv1.New(b.store)"）。
	// 留空时生成空函数体。
	ImplBody string
}

// AddInterfaceMethod 将一个方法注入到 Go 接口，并补充对应的 receiver 方法
// 与 import。注入幂等。
//
// 所有插入点均由 AST 结构推导得出：
//   - 接口方法 → 追加到 InterfaceName 对应 *dst.InterfaceType.Methods.List。
//   - receiver 方法 → 追加到 f.Decls（顶层 decl 列表）。
//   - import → 插入到第一个 import GenDecl 的开头；若不存在则新建。
func AddInterfaceMethod(file string, p InterfacePayload) error {
	f, err := ParseFile(file)
	if err != nil {
		return err
	}

	skippedImport := false
	skippedIface := false
	skippedImpl := false

	if p.ImportAlias != "" && p.ImportPath != "" {
		if hasImport(f, p.ImportPath) {
			skippedImport = true
		} else {
			addImport(f, p.ImportAlias, p.ImportPath)
		}
	} else {
		skippedImport = true
	}

	iface := findInterfaceDecl(f, p.InterfaceName)
	if iface == nil {
		return errs.New(errs.CodeASTApplyError,
			fmt.Sprintf("ast: interface %q not found in %s", p.InterfaceName, file)).
			WithHint(fmt.Sprintf("declare `type %s interface { ... }` in the target file", p.InterfaceName))
	}
	if hasMethodInInterface(iface, p.Method) {
		skippedIface = true
	} else {
		appendMethodToInterface(iface, p.Method, p.ReturnType)
	}

	if hasReceiverMethod(f, p.StructName, p.Method) {
		skippedImpl = true
	} else {
		appendReceiverMethod(f, p.StructName, p.Method, p.ReturnType, p.ImplBody)
	}

	if skippedImport && skippedIface && skippedImpl {
		fmt.Printf("⊝ skipped (already exists): %s.%s in %s\n", p.StructName, p.Method, file)
		return nil
	}

	return WriteFile(file, f)
}

// hasImport 判断 f 是否已经 import 了指定路径。
func hasImport(f *dst.File, path string) bool {
	for _, decl := range f.Decls {
		gd, ok := decl.(*dst.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gd.Specs {
			is, ok := spec.(*dst.ImportSpec)
			if !ok {
				continue
			}
			if strings.Trim(is.Path.Value, `"`) == path {
				return true
			}
		}
	}
	return false
}

// addImport 在文件第一个 import GenDecl 的开头插入一条 import spec；
// 若不存在则在 Decls 顶部新建一个 import 块。
func addImport(f *dst.File, alias, path string) {
	spec := &dst.ImportSpec{
		Path: &dst.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", path)},
	}
	if alias != "" {
		spec.Name = dst.NewIdent(alias)
	}

	for _, decl := range f.Decls {
		gd, ok := decl.(*dst.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		gd.Specs = append([]dst.Spec{spec}, gd.Specs...)
		return
	}

	newDecl := &dst.GenDecl{
		Tok:    token.IMPORT,
		Lparen: true,
		Specs:  []dst.Spec{spec},
	}
	f.Decls = append([]dst.Decl{newDecl}, f.Decls...)
}

// findInterfaceDecl 按名称定位接口类型声明。
func findInterfaceDecl(f *dst.File, name string) *dst.InterfaceType {
	for _, decl := range f.Decls {
		gd, ok := decl.(*dst.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*dst.TypeSpec)
			if !ok || ts.Name.Name != name {
				continue
			}
			it, ok := ts.Type.(*dst.InterfaceType)
			if ok {
				return it
			}
		}
	}
	return nil
}

// hasMethodInInterface 判断接口是否已声明了同名方法。
func hasMethodInInterface(iface *dst.InterfaceType, name string) bool {
	for _, field := range iface.Methods.List {
		for _, n := range field.Names {
			if n.Name == name {
				return true
			}
		}
	}
	return false
}

// appendMethodToInterface 向接口追加一个方法 field。
func appendMethodToInterface(iface *dst.InterfaceType, method, returnType string) {
	field := &dst.Field{
		Names: []*dst.Ident{dst.NewIdent(method)},
		Type:  buildFuncTypeExpr(returnType),
	}
	field.Decs.Before = dst.NewLine
	iface.Methods.List = append(iface.Methods.List, field)
}

// buildFuncTypeExpr 构造一个返回值为 returnType 的 *dst.FuncType。
func buildFuncTypeExpr(returnType string) *dst.FuncType {
	return &dst.FuncType{
		Params: &dst.FieldList{},
		Results: &dst.FieldList{
			List: []*dst.Field{
				{Type: parseTypeExpr(returnType)},
			},
		},
	}
}

// parseTypeExpr 将类型字符串转换为 dst.Expr。
// 支持 "pkg.Type" → SelectorExpr、"Type" → Ident、"*Type" → StarExpr。
func parseTypeExpr(s string) dst.Expr {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "*") {
		return &dst.StarExpr{X: parseTypeExpr(s[1:])}
	}
	if idx := strings.Index(s, "."); idx >= 0 {
		return &dst.SelectorExpr{
			X:   dst.NewIdent(s[:idx]),
			Sel: dst.NewIdent(s[idx+1:]),
		}
	}
	return dst.NewIdent(s)
}

// hasReceiverMethod 判断文件中是否已存在 func (*structName) method(...) 形式的方法。
func hasReceiverMethod(f *dst.File, structName, method string) bool {
	for _, decl := range f.Decls {
		fd, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}
		if fd.Name.Name != method {
			continue
		}
		if fd.Recv == nil || len(fd.Recv.List) == 0 {
			continue
		}
		recv := fd.Recv.List[0]
		if se, ok := recv.Type.(*dst.StarExpr); ok {
			if id, ok := se.X.(*dst.Ident); ok && id.Name == structName {
				return true
			}
		}
		if id, ok := recv.Type.(*dst.Ident); ok && id.Name == structName {
			return true
		}
	}
	return false
}

// appendReceiverMethod 将一个 receiver 方法追加到文件的顶层 Decls。
func appendReceiverMethod(f *dst.File, structName, method, returnType, body string) {
	recv := &dst.FieldList{
		List: []*dst.Field{
			{
				Names: []*dst.Ident{dst.NewIdent("b")},
				Type:  &dst.StarExpr{X: dst.NewIdent(structName)},
			},
		},
	}

	var stmts []dst.Stmt
	if body != "" {
		body = strings.TrimPrefix(strings.TrimSpace(body), "return ")
		stmts = []dst.Stmt{
			&dst.ReturnStmt{
				Results: []dst.Expr{parseCallExpr(body)},
			},
		}
	}

	fd := &dst.FuncDecl{
		Recv: recv,
		Name: dst.NewIdent(method),
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{Type: parseTypeExpr(returnType)},
				},
			},
		},
		Body: &dst.BlockStmt{List: stmts},
	}
	fd.Decs.Before = dst.EmptyLine
	f.Decls = append(f.Decls, fd)
}

// parseCallExpr 从形如 "postv1.New(b.store)" 的字符串构造 dst.Expr。
func parseCallExpr(s string) dst.Expr {
	s = strings.TrimSpace(s)

	parenIdx := strings.Index(s, "(")
	if parenIdx < 0 {
		return parseTypeExpr(s)
	}

	funStr := s[:parenIdx]
	rest := s[parenIdx+1:]
	ellipsis := false

	if last := len(rest) - 1; last >= 0 && rest[last] == ')' {
		rest = rest[:last]
	}

	if strings.HasSuffix(rest, "...") {
		ellipsis = true
		rest = rest[:len(rest)-3]
	}

	funExpr := parseTypeExpr(funStr)

	var args []dst.Expr
	rest = strings.TrimSpace(rest)
	if rest != "" {
		for _, part := range splitArgs(rest) {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "(") {
				args = append(args, parseCallExpr(part))
			} else {
				args = append(args, parseTypeExpr(part))
			}
		}
	}

	return &dst.CallExpr{
		Fun:      funExpr,
		Args:     args,
		Ellipsis: ellipsis,
	}
}

// splitArgs 将以逗号分隔的参数列表切分开，正确处理嵌套括号。
func splitArgs(s string) []string {
	var result []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// ParseFileWithDecorator 使用 dst decorator 解析 Go 文件（仅供测试导出使用）。
func ParseFileWithDecorator(src []byte) (*dst.File, *decorator.Decorator, error) {
	d := decorator.NewDecorator(token.NewFileSet())
	f, err := d.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	return f, d, nil
}
