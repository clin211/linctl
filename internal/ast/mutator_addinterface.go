package ast

import (
	"context"
	"fmt"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/dave/dst"
)

// AddInterfaceMethodMutator 在指定 interface 中添加一个方法签名。
//
// 例：在 IBiz 中添加 `Posts() PostBiz`。
//
// 幂等：
//   - 名字+签名完全匹配 → 跳过
//   - 名字相同但签名不同 → 返回 *ConflictError
type AddInterfaceMethodMutator struct {
	FilePath      string
	InterfaceName string // "IBiz"
	MethodName    string // "Posts"
	Params        string // 参数列表的 Go 源码，如 ""（无参）或 "id int64"
	Returns       string // 返回列表的 Go 源码，如 "PostBiz" 或 "(PostBiz, error)"
	Doc           string // 可选，方法上方的文档注释（不含 // 前缀）
}

func (m *AddInterfaceMethodMutator) File() string        { return m.FilePath }
func (m *AddInterfaceMethodMutator) Layer() Layer        { return LayerBiz }
func (m *AddInterfaceMethodMutator) Description() string {
	return fmt.Sprintf("add method %s(%s) %s to %s in %s",
		m.MethodName, m.Params, m.Returns, m.InterfaceName, m.FilePath)
}

func (m *AddInterfaceMethodMutator) Apply(_ context.Context, content []byte) ([]byte, bool, error) {
	if m.InterfaceName == "" || m.MethodName == "" {
		return content, false, linctlerr.New(linctlerr.ErrASTInjection,
			"AddInterfaceMethodMutator: InterfaceName / MethodName required")
	}

	file, _, err := parseFile(m.FilePath, content)
	if err != nil {
		return content, false, err
	}

	// 找到目标 interface
	var iface *dst.InterfaceType
	for _, decl := range file.Decls {
		gen, ok := decl.(*dst.GenDecl)
		if !ok || gen.Tok.String() != "type" {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*dst.TypeSpec)
			if !ok || ts.Name.Name != m.InterfaceName {
				continue
			}
			it, ok := ts.Type.(*dst.InterfaceType)
			if !ok {
				continue
			}
			iface = it
			break
		}
		if iface != nil {
			break
		}
	}
	if iface == nil {
		return content, false, linctlerr.Newf(linctlerr.ErrASTInjection,
			"interface %s not found in %s", m.InterfaceName, m.FilePath).
			WithHint("Make sure the interface is declared in this file before applying.")
	}

	// 检查方法是否已存在（按 名字 + 签名 匹配）
	wantSig := fmt.Sprintf("(%s) %s", m.Params, m.Returns)
	for _, field := range iface.Methods.List {
		if len(field.Names) == 0 || field.Names[0].Name != m.MethodName {
			continue
		}
		// 序列化已有方法签名作字符串比较
		existingSig, err := stringifyFuncType(field.Type)
		if err != nil {
			return content, false, err
		}
		if existingSig == wantSig {
			return content, false, nil // 完全一致，幂等
		}
		return content, false, &ConflictError{
			Kind:     "interface_method_signature_mismatch",
			File:     m.FilePath,
			Symbol:   m.InterfaceName + "." + m.MethodName,
			Existing: existingSig,
			Want:     wantSig,
			Hint: "Either rename your method, revert the existing signature, " +
				"or run with --strategy=overwrite to discard local changes (not recommended).",
		}
	}

	// 构造新方法 Field：用 parser.ParseExpr 解析整个 "func(params) returns"
	// 但 method-style 在 interface 里是 *dst.FuncType
	funcSrc := fmt.Sprintf("func(%s) %s", m.Params, m.Returns)
	expr, err := parseExpr(funcSrc)
	if err != nil {
		return content, false, err
	}
	funcType, ok := expr.(*dst.FuncType)
	if !ok {
		return content, false, linctlerr.Newf(linctlerr.ErrASTInjection,
			"failed to parse method signature: %s", funcSrc)
	}

	newField := &dst.Field{
		Names: []*dst.Ident{{Name: m.MethodName}},
		Type:  funcType,
	}
	if m.Doc != "" {
		newField.Decs.NodeDecs.Start.Append("// " + m.Doc)
	}

	// 追加到 interface 末尾
	iface.Methods.List = append(iface.Methods.List, newField)

	out, err := printFile(file)
	if err != nil {
		return content, false, err
	}
	return out, true, nil
}

// stringifyFuncType 把 *dst.FuncType 序列化成 "(params) returns" 字符串，用于签名比对。
func stringifyFuncType(t dst.Expr) (string, error) {
	ft, ok := t.(*dst.FuncType)
	if !ok {
		return "", linctlerr.Newf(linctlerr.ErrASTInjection,
			"expected *dst.FuncType, got %T", t)
	}

	params := stringifyFieldList(ft.Params)
	returns := ""
	if ft.Results != nil && len(ft.Results.List) > 0 {
		returns = stringifyFieldList(ft.Results)
	}
	return fmt.Sprintf("(%s) %s", params, returns), nil
}

func stringifyFieldList(fl *dst.FieldList) string {
	if fl == nil {
		return ""
	}
	out := ""
	for i, f := range fl.List {
		if i > 0 {
			out += ", "
		}
		// 字段名（可选）
		if len(f.Names) > 0 {
			for j, n := range f.Names {
				if j > 0 {
					out += ", "
				}
				out += n.Name
			}
			out += " "
		}
		// 类型
		out += stringifyExpr(f.Type)
	}
	return out
}

func stringifyExpr(e dst.Expr) string {
	switch x := e.(type) {
	case *dst.Ident:
		return x.Name
	case *dst.StarExpr:
		return "*" + stringifyExpr(x.X)
	case *dst.SelectorExpr:
		return stringifyExpr(x.X) + "." + x.Sel.Name
	case *dst.ArrayType:
		return "[]" + stringifyExpr(x.Elt)
	default:
		return fmt.Sprintf("%T", e)
	}
}
