// Package feature 提供 linctl 的 Feature 抽象、Registry 与内置 Feature。
//
// Feature 是「贡献 Pair / Mutator / Resource 的横向能力」（详见 docs/08-feature-system.md）。
// 与 Component（纵向骨架）正交：一个 Feature 可贡献给多个 Component（如 healthz 既给
// WebServer 也给 Worker），Component 也可启用多个 Feature。
//
// SSOT 决策（详见 docs/META-fix-decisions-2026-04-25.md §1.15 / §1.16）：
//   - 依赖关系用 Requires() + 拓扑排序（不是 Order 总序）
//   - Order() 仅作同层 tie-break
//   - Apply 必须是纯函数：不修改入参；返回 []Pair
//   - Resource 注入由独立的 ResourceContributions() 完成
package feature

import (
	"context"
	texttemplate "text/template"

	"github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/project"
)

// Feature 是横向能力的统一接口。
type Feature interface {
	// Name 返回全局唯一名称（小写、kebab-case，如 "healthz"、"open-telemetry"）。
	Name() string

	// Requires 返回本 Feature 直接依赖的其它 Feature 名称列表。
	// 调度器用拓扑排序保证 Requires 中的 Feature 先于本 Feature 执行。
	Requires() []string

	// AppliesTo 返回本 Feature 适用的 Component Kind 列表（如 []string{"WebServer"}）。
	AppliesTo() []string

	// Apply 计算并返回本 Feature 贡献的 Pair 列表（纯函数，不修改入参）。
	Apply(ctx context.Context, c project.Component) ([]codegen.Pair, error)

	// ResourceContributions 返回本 Feature 要为该 Component 注入的 Resource 集合。
	// 替代旧版本 UserFeature.Apply 直接修改 c.Resources 的反模式。
	ResourceContributions(c project.Component) []project.Resource

	// Mutators 返回本 Feature 需要执行的 AST 修改（Phase 1 通常返回 nil）。
	Mutators(p *project.Project, c project.Component) []ast.ASTMutator

	// FuncMap 返回本 Feature 提供的模板函数（注册到全局 FuncMap）。
	FuncMap() texttemplate.FuncMap

	// Defaults 返回本 Feature 的默认配置贡献。必须返回新 map，禁止修改 p / c。
	Defaults(p *project.Project, c project.Component) map[string]any

	// Validate 校验本 Feature 在当前 Project 上的合法性。
	Validate(p *project.Project, c project.Component) error

	// Order 同层 tie-break。默认 0。
	Order() int
}
