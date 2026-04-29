// Package builtin 包含 linctl 的内置 Feature 实现。
//
// Phase 1 范围：仅交付 healthz Feature（最小可工作示例）；
// opentelemetry / user / websocket / preloader 在后续 Phase 引入。
package builtin

import (
	"context"
	texttemplate "text/template"

	"github.com/clin211/lin/internal/ast"
	"github.com/clin211/lin/internal/codegen"
	"github.com/clin211/lin/internal/project"
)

// HealthzFeature 注入 /healthz 端点（最小开销，几乎所有 Web 服务都需要）。
//
// 适用范围：仅 WebServer。
type HealthzFeature struct{}

// NewHealthz 构造 HealthzFeature 实例。
func NewHealthz() *HealthzFeature { return &HealthzFeature{} }

// Name 返回 "healthz"。
func (*HealthzFeature) Name() string { return "healthz" }

// Requires 无依赖。
func (*HealthzFeature) Requires() []string { return nil }

// AppliesTo 仅适用于 WebServer。
func (*HealthzFeature) AppliesTo() []string { return []string{"WebServer"} }

// Apply 不贡献任何 Pair。
//
// healthz 自从 v0.3.0 起被纳入 WebServer 基础骨架（始终生成 internal/<app>/handler/healthz.go），
// 因此 Feature 仅作为占位保留，避免破坏既有 spec.components[].features 列表的兼容性。
func (*HealthzFeature) Apply(_ context.Context, _ project.Component) ([]codegen.Pair, error) {
	return nil, nil
}

// ResourceContributions 不注入新 Resource。
func (*HealthzFeature) ResourceContributions(_ project.Component) []project.Resource {
	return nil
}

// Mutators 在 router.go 中注册 /healthz 路由。Phase 1 暂不做 AST 注入，
// 由 router.go.tpl 通过 hasFeature 判断条件渲染（Phase 2 改为 AST）。
func (*HealthzFeature) Mutators(_ *project.Project, _ project.Component) []ast.ASTMutator {
	return nil
}

// FuncMap 不贡献额外模板函数。
func (*HealthzFeature) FuncMap() texttemplate.FuncMap { return nil }

// Defaults 不贡献额外默认值。
func (*HealthzFeature) Defaults(_ *project.Project, _ project.Component) map[string]any {
	return nil
}

// Validate 永远成功（healthz 无前置条件）。
func (*HealthzFeature) Validate(_ *project.Project, _ project.Component) error { return nil }

// Order 200（按 SSOT §1.15 示例：UserFeature=100，HealthzFeature=200）。
func (*HealthzFeature) Order() int { return 200 }
