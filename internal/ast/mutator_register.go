package ast

import (
	"fmt"
	"strings"

	"github.com/dave/dst"

	"github.com/clin211/lin/internal/pkg/errs"
)

// RegisterPayload carries parameters for AppendRegistration.
//
// The insertion target is the Body of the top-level function FunctionName.
// No anchor comments are required.
type RegisterPayload struct {
	// FunctionName is the top-level function whose body will receive the
	// new statement (e.g. "RegisterAll").
	FunctionName string
	// Statement is the call expression to append, written as a Go expression
	// such as "RegisterErrors(PostErrors()...)".
	Statement string
}

// AppendRegistration appends a call statement to the Body of FunctionName.
// Idempotent: if the same statement is already present in the Body, returns nil.
func AppendRegistration(file string, p RegisterPayload) error {
	f, err := ParseFile(file)
	if err != nil {
		return err
	}

	fn := findFuncDecl(f, p.FunctionName)
	if fn == nil {
		return errs.New(errs.CodeASTApplyError,
			fmt.Sprintf("ast: function %q not found in %s", p.FunctionName, file)).
			WithHint(fmt.Sprintf("declare `func %s() { ... }` in the target file", p.FunctionName))
	}
	if fn.Body == nil {
		return errs.New(errs.CodeASTApplyError,
			fmt.Sprintf("ast: function %q has no body in %s", p.FunctionName, file))
	}

	stmt := &dst.ExprStmt{X: parseCallExpr(p.Statement)}

	if hasEquivalentStmt(fn.Body, stmt) {
		fmt.Printf("⊝ skipped (already exists): %s in %s\n", p.Statement, file)
		return nil
	}

	stmt.Decs.Before = dst.NewLine
	fn.Body.List = append(fn.Body.List, stmt)

	return WriteFile(file, f)
}

// findFuncDecl locates a top-level (non-receiver) function declaration by name.
func findFuncDecl(f *dst.File, name string) *dst.FuncDecl {
	for _, decl := range f.Decls {
		fd, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			continue
		}
		if fd.Name != nil && fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

// hasEquivalentStmt reports whether body already contains a statement whose
// printed form equals stmt's printed form. We compare via canonical forms
// produced by exprString to ignore positions and decorations.
func hasEquivalentStmt(body *dst.BlockStmt, stmt *dst.ExprStmt) bool {
	want := canonicalCallString(stmt.X)
	if want == "" {
		return false
	}
	for _, s := range body.List {
		es, ok := s.(*dst.ExprStmt)
		if !ok {
			continue
		}
		if canonicalCallString(es.X) == want {
			return true
		}
	}
	return false
}

// canonicalCallString renders a *dst.CallExpr (or any expression) into a stable
// textual form suitable for equality comparison. Uses the dst printer indirectly
// by re-implementing the small subset we generate via parseCallExpr.
func canonicalCallString(expr dst.Expr) string {
	var sb strings.Builder
	writeExpr(&sb, expr)
	return sb.String()
}

func writeExpr(sb *strings.Builder, expr dst.Expr) {
	switch v := expr.(type) {
	case *dst.Ident:
		sb.WriteString(v.Name)
	case *dst.SelectorExpr:
		writeExpr(sb, v.X)
		sb.WriteByte('.')
		sb.WriteString(v.Sel.Name)
	case *dst.StarExpr:
		sb.WriteByte('*')
		writeExpr(sb, v.X)
	case *dst.CallExpr:
		writeExpr(sb, v.Fun)
		sb.WriteByte('(')
		for i, a := range v.Args {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeExpr(sb, a)
		}
		if v.Ellipsis {
			sb.WriteString("...")
		}
		sb.WriteByte(')')
	case *dst.BasicLit:
		sb.WriteString(v.Value)
	default:
		fmt.Fprintf(sb, "<%T>", v)
	}
}
