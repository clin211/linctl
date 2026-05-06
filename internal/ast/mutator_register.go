package ast

import (
	"fmt"
	"strings"

	"github.com/dave/dst"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// RegisterPayload 是 AppendRegistration 的入参。
//
// 插入目标是顶层函数 FunctionName 的函数体；
// 不依赖任何锚点注释。
type RegisterPayload struct {
	// FunctionName 是用于接收新增语句的顶层函数名（如 "RegisterAll"）。
	FunctionName string
	// Statement 是要追加的调用表达式（Go 语法），
	// 例如 "RegisterErrors(PostErrors()...)"。
	Statement string
}

// AppendRegistration 将一条调用语句追加到 FunctionName 的函数体内。
// 注入幂等：若该函数体内已存在等价语句则直接返回 nil。
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

// findFuncDecl 按名称定位顶层（非 receiver）函数声明。
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

// hasEquivalentStmt 判断 body 中是否已经存在与 stmt 文本等价的语句。
// 通过 canonicalCallString 生成的稳定文本进行比较，忽略位置与装饰信息。
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

// canonicalCallString 将 *dst.CallExpr（或任意表达式）渲染为稳定的文本形式，
// 用于等值比较。该函数在 parseCallExpr 生成的子集上等价于 dst printer。
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
