package feature_test

import (
	"context"
	texttemplate "text/template"
	"testing"

	"github.com/clin211/linctl/internal/ast"
	"github.com/clin211/linctl/internal/codegen"
	"github.com/clin211/linctl/internal/feature"
	"github.com/clin211/linctl/internal/feature/builtin"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	r := feature.NewRegistry()
	require.NoError(t, r.Register(builtin.NewHealthz()))

	got, err := r.Get("healthz")
	require.NoError(t, err)
	assert.Equal(t, "healthz", got.Name())

	// Duplicate
	err = r.Register(builtin.NewHealthz())
	require.Error(t, err)
}

func TestRegistry_GetUnknown(t *testing.T) {
	r := feature.NewRegistry()
	_, err := r.Get("unknown")
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrFeatureDependency, code)
}

// fakeFeature 是测试用的最小 Feature 实现。
type fakeFeature struct {
	name     string
	requires []string
	order    int
}

func (f *fakeFeature) Name() string                                      { return f.name }
func (f *fakeFeature) Requires() []string                                { return f.requires }
func (f *fakeFeature) AppliesTo() []string                               { return []string{"WebServer"} }
func (f *fakeFeature) Apply(_ context.Context, _ project.Component) ([]codegen.Pair, error) {
	return nil, nil
}
func (f *fakeFeature) ResourceContributions(_ project.Component) []project.Resource { return nil }
func (f *fakeFeature) Mutators(_ *project.Project, _ project.Component) []ast.ASTMutator {
	return nil
}
func (f *fakeFeature) FuncMap() texttemplate.FuncMap { return nil }
func (f *fakeFeature) Defaults(_ *project.Project, _ project.Component) map[string]any {
	return nil
}
func (f *fakeFeature) Validate(_ *project.Project, _ project.Component) error { return nil }
func (f *fakeFeature) Order() int                                              { return f.order }

func TestResolveOrder_NoDeps(t *testing.T) {
	r := feature.NewRegistry()
	require.NoError(t, r.Register(&fakeFeature{name: "a", order: 100}))
	require.NoError(t, r.Register(&fakeFeature{name: "b", order: 200}))

	out, err := r.ResolveOrder([]string{"a", "b"})
	require.NoError(t, err)
	require.Len(t, out, 2)
	// Order 升序
	assert.Equal(t, "a", out[0].Name())
	assert.Equal(t, "b", out[1].Name())
}

func TestResolveOrder_WithDeps(t *testing.T) {
	r := feature.NewRegistry()
	require.NoError(t, r.Register(&fakeFeature{name: "base"}))
	require.NoError(t, r.Register(&fakeFeature{name: "dep", requires: []string{"base"}}))

	out, err := r.ResolveOrder([]string{"dep"})
	require.NoError(t, err)
	require.Len(t, out, 2)
	// base 必须在 dep 之前
	assert.Equal(t, "base", out[0].Name())
	assert.Equal(t, "dep", out[1].Name())
}

func TestResolveOrder_TransitiveDeps(t *testing.T) {
	r := feature.NewRegistry()
	require.NoError(t, r.Register(&fakeFeature{name: "a"}))
	require.NoError(t, r.Register(&fakeFeature{name: "b", requires: []string{"a"}}))
	require.NoError(t, r.Register(&fakeFeature{name: "c", requires: []string{"b"}}))

	out, err := r.ResolveOrder([]string{"c"})
	require.NoError(t, err)
	require.Len(t, out, 3)
	assert.Equal(t, "a", out[0].Name())
	assert.Equal(t, "b", out[1].Name())
	assert.Equal(t, "c", out[2].Name())
}

func TestResolveOrder_Cycle(t *testing.T) {
	r := feature.NewRegistry()
	require.NoError(t, r.Register(&fakeFeature{name: "a", requires: []string{"b"}}))
	require.NoError(t, r.Register(&fakeFeature{name: "b", requires: []string{"a"}}))

	_, err := r.ResolveOrder([]string{"a"})
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrFeatureDependency, code)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestResolveOrder_UnknownFeature(t *testing.T) {
	r := feature.NewRegistry()
	_, err := r.ResolveOrder([]string{"nope"})
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrFeatureDependency, code)
}

func TestHealthzFeature_Basic(t *testing.T) {
	h := builtin.NewHealthz()
	assert.Equal(t, "healthz", h.Name())
	assert.Empty(t, h.Requires())
	assert.Equal(t, []string{"WebServer"}, h.AppliesTo())
	assert.Equal(t, 200, h.Order())
	assert.NoError(t, h.Validate(nil, project.Component{}))
}

func TestHealthzFeature_Apply(t *testing.T) {
	// healthz 自 v0.3.0 起被纳入 WebServer 基础骨架，Feature 的 Apply 不再贡献 Pair。
	h := builtin.NewHealthz()
	pairs, err := h.Apply(context.Background(), project.Component{Name: "api"})
	require.NoError(t, err)
	assert.Empty(t, pairs)
}
