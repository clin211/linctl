package scaffold

import (
	"os"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// PlanKind 区分 plan 的类型（项目骨架 / 资源骨架）。
type PlanKind int

const (
	// PlanKindProject 表示 lin new 生成的项目骨架计划。
	PlanKindProject PlanKind = iota
	// PlanKindResource 表示 lin add 生成的资源骨架计划。
	PlanKindResource
)

// Plan 描述一次生成操作的全部副作用。
//
// 设计来源：01 §7.2「scaffold.Plan」。
type Plan struct {
	Kind    PlanKind
	Creates []FileSpec
	Injects []InjectSpec
}

// FileSpec 描述要创建/写入的单个文件。
type FileSpec struct {
	// TemplatePath 是模板源相对路径（如 "project/cmd/app/main.go.tpl"）。
	// 若为空表示按位拷贝（详见 04 §13.3）。
	TemplatePath string

	// DestPath 是输出文件的相对路径（相对 Context.RootDir）。
	DestPath string

	// Permissions 是文件权限（详见 04 §13.4）。
	Permissions os.FileMode
}

// MutatorKind 标识 AST mutator 的种类。
type MutatorKind string

const (
	MutatorInterface MutatorKind = "interface"
	MutatorProto     MutatorKind = "proto"
	MutatorRegister  MutatorKind = "register"
)

// InjectSpec 描述一次 AST 注入。
//
// 详细注入语义见 lin/docs/features/05-registration-strategy.md §3。
type InjectSpec struct {
	File    string      // 目标文件相对路径
	Mutator MutatorKind // 注入类型
	Payload any         // 由具体 mutator 解释的负载
}

// errPhase1Stub 是 Phase 1 阶段对未实现方法的统一占位错误。
var errPhase1Stub = errs.New(
	errs.CodeUnknown,
	"scaffold: feature not implemented in Phase 1 skeleton",
).WithHint("see lin/docs/features/06-migration-plan.md for the implementation roadmap")
