package ast

import (
	"bytes"
	"go/parser"
	"go/token"

	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// parseFile 把 Go 源码字节解析为 *dst.File。
func parseFile(filename string, content []byte) (*dst.File, *decorator.Decorator, error) {
	dec := decorator.NewDecorator(token.NewFileSet())
	file, err := dec.ParseFile(filename, content, parser.ParseComments)
	if err != nil {
		return nil, nil, linctlerr.Wrapf(linctlerr.ErrASTInjection, err,
			"parse %s", filename)
	}
	return file, dec, nil
}

// printFile 把修改后的 *dst.File 序列化回字节。
func printFile(file *dst.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, file); err != nil {
		return nil, linctlerr.Wrap(linctlerr.ErrASTInjection, err, "print dst.File")
	}
	return buf.Bytes(), nil
}

// parseExpr 把字符串表达式（如 "newPostStore(s.store)"）解析为 dst.Expr。
//
// 注意：这是 osbuilder 反模式的纠正——绝不能用 &dst.Ident{Name: "..."} 塞整段表达式。
func parseExpr(src string) (dst.Expr, error) {
	expr, err := parser.ParseExpr(src)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrASTInjection, err,
			"parse expression %q", src)
	}
	dec := decorator.NewDecorator(token.NewFileSet())
	dstNode, err := dec.DecorateNode(expr)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrASTInjection, err,
			"decorate expr %q", src)
	}
	return dstNode.(dst.Expr), nil
}
