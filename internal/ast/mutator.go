// Package ast 实现 linctl 的 Go AST 注入能力。
//
// 设计原则（详见 docs/07-ast-injection.md 与 ADR-001）：
//   - 基于 dst (dave/dst) 而非 go/ast，保留注释与空行
//   - 所有 Mutator 必须幂等（同样输入多次执行结果一致）
//   - 同名异签场景返回 ConflictError，由 plan 显式上报
//   - 同文件多 mutator 通过 Batch 合并：parse 一次 + 多次修改 + write 一次
package ast

import (
	"context"
	"fmt"
	"strings"
)

// Layer 标识 Mutator 作用的逻辑层级，用于 plan 报告分组。
type Layer string

const (
	LayerStore Layer = "store"
	LayerBiz   Layer = "biz"
	LayerProto Layer = "proto"
	LayerAllGo Layer = "all_go"
)

// ASTMutator 是对单个文件的 AST 修改单元。
//
// 契约：
//   - 必须幂等：同一份 input 多次调用 Apply 结果一致
//   - 不修改入参 content；返回新切片
//   - 失败返回 LinctlError；尤其是同名异签场景应返回 *ConflictError
type ASTMutator interface {
	// File 返回 Mutator 作用的目标文件相对路径（相对项目根）。
	File() string

	// Layer 返回逻辑层级（用于 plan 报告分组）。
	Layer() Layer

	// Description 返回用于 plan 报告的简短描述。
	Description() string

	// Apply 执行修改。
	//   - modified=false 且 err=nil 表示已存在且无需变更（幂等）
	//   - err 非 nil 时 newContent 应保持原样
	Apply(ctx context.Context, content []byte) ([]byte, bool, error)
}

// ConflictError 表示 Mutator 检测到「同名异签 / 不可幂等覆盖」类冲突。
//
// 该错误被 plan 收集并显示给用户；不会自动覆盖已存在的代码。
//
// Hint 字段（Story 2.6）：可执行的修复建议，例如：
//   - "rename your method or revert the existing signature"
//   - "use --strategy=overwrite to discard local changes（不推荐）"
type ConflictError struct {
	Kind     string // 冲突类型，如 "interface_method_signature_mismatch"
	File     string
	Symbol   string // "InterfaceName.MethodName" 或 "FuncName"
	Existing string // 已存在的签名/定义（人类可读）
	Want     string // 期望的签名/定义（人类可读）
	Hint     string // 可选：修复建议
}

// Error 实现 error 接口。
func (e *ConflictError) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ast conflict: %s in %s for %s\n  existing: %s\n  want    : %s",
		e.Kind, e.File, e.Symbol, e.Existing, e.Want)
	if e.Hint != "" {
		fmt.Fprintf(&b, "\n  hint    : %s", e.Hint)
	}
	return b.String()
}

// WithHint 在不可变模式下追加 Hint。
func (e *ConflictError) WithHint(hint string) *ConflictError {
	if e == nil {
		return nil
	}
	cp := *e
	if cp.Hint == "" {
		cp.Hint = hint
	} else {
		cp.Hint = cp.Hint + "\n" + hint
	}
	return &cp
}
