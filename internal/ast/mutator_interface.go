package ast

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// InterfacePayload carries parameters for AddInterfaceMethod.
//
// Insertion locations are derived purely from Go AST structure
// (interface name, receiver struct name) — no anchor comments are required.
type InterfacePayload struct {
	// InterfaceName is the name of the interface type (e.g. "IBiz").
	InterfaceName string
	// StructName is the receiver type (e.g. "biz").
	StructName string

	// Method is the method name to add (e.g. "PostV1").
	Method string
	// ReturnType is the return type string (e.g. "postv1.PostBiz").
	ReturnType string

	// ImportAlias and ImportPath form the import to add (optional).
	ImportAlias string
	ImportPath  string

	// ImplBody is the function body expression (e.g. "return postv1.New(b.store)").
	// When empty, an empty body is emitted.
	ImplBody string
}

// AddInterfaceMethod injects a method into a Go interface and adds a corresponding
// receiver method, with proper import. Idempotent.
//
// All insertion points are computed from AST structure:
//   - Interface method  → appended to *dst.InterfaceType.Methods.List of InterfaceName.
//   - Receiver method   → appended to f.Decls (top-level decls).
//   - Import            → prepended to the first import GenDecl, or a new one if absent.
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

// hasImport reports whether f already imports the given path.
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

// addImport prepends an import spec to the first import GenDecl in the file.
// If no import block exists, one is created at the top of Decls.
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

// findInterfaceDecl locates an interface type declaration by name.
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

// hasMethodInInterface reports whether the interface already declares a method with the given name.
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

// appendMethodToInterface appends a method field to an interface.
func appendMethodToInterface(iface *dst.InterfaceType, method, returnType string) {
	field := &dst.Field{
		Names: []*dst.Ident{dst.NewIdent(method)},
		Type:  buildFuncTypeExpr(returnType),
	}
	field.Decs.Before = dst.NewLine
	iface.Methods.List = append(iface.Methods.List, field)
}

// buildFuncTypeExpr constructs a *dst.FuncType that returns the given type.
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

// parseTypeExpr converts a type string to a dst.Expr.
// Handles "pkg.Type" → SelectorExpr, "Type" → Ident, "*Type" → StarExpr.
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

// hasReceiverMethod reports whether file already has func (*structName) method(...).
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

// appendReceiverMethod appends a receiver method to the file's top-level Decls.
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

// parseCallExpr builds a dst.Expr from a call expression string like "postv1.New(b.store)".
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

// splitArgs splits a comma-separated argument list, respecting nested parentheses.
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

// ParseFileWithDecorator parses a Go file using the dst decorator (exported for testing).
func ParseFileWithDecorator(src []byte) (*dst.File, *decorator.Decorator, error) {
	d := decorator.NewDecorator(token.NewFileSet())
	f, err := d.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	return f, d, nil
}
