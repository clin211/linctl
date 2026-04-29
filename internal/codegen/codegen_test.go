package codegen_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/clin211/lin/internal/codegen"
	"github.com/clin211/lin/internal/fs"
	"github.com/clin211/lin/internal/template"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPairBuilder_DedupeByDst(t *testing.T) {
	b := codegen.NewPairBuilder()
	b.Add(codegen.Pair{Dst: "a.go", TemplateID: "t1", Owner: "first"})
	b.Add(codegen.Pair{Dst: "a.go", TemplateID: "t2", Owner: "second"})
	b.Add(codegen.Pair{Dst: "b.go", TemplateID: "t3", Owner: "first"})

	pairs := b.Build()
	require.Len(t, pairs, 2)
	for _, p := range pairs {
		if p.Dst == "a.go" {
			assert.Equal(t, "t2", p.TemplateID)
			assert.Equal(t, "second", p.Owner)
		}
	}
}

func TestPlan_DigestStable(t *testing.T) {
	p1 := &codegen.Plan{Actions: []codegen.Action{
		{Kind: codegen.ActionCreate, Dst: "a", NewHash: "x"},
		{Kind: codegen.ActionUpdate, Dst: "b", NewHash: "y"},
	}}
	p2 := &codegen.Plan{Actions: []codegen.Action{
		{Kind: codegen.ActionUpdate, Dst: "b", NewHash: "y"}, // 顺序不同
		{Kind: codegen.ActionCreate, Dst: "a", NewHash: "x"},
	}}
	assert.Equal(t, p1.ComputeDigest(), p2.ComputeDigest())

	p3 := &codegen.Plan{Actions: []codegen.Action{
		{Kind: codegen.ActionCreate, Dst: "a", NewHash: "z"}, // hash 不同
	}}
	assert.NotEqual(t, p1.ComputeDigest(), p3.ComputeDigest())
}

func TestPlanner_Plan_CreatesNewFiles(t *testing.T) {
	p, fm := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n// hello {{ .Name }}\n",
	})

	pairs := []codegen.Pair{
		{Dst: "main.go", TemplateID: "templates/x.go.tpl", Owner: "test"},
	}
	plan, err := p.Plan(context.Background(), map[string]any{"Name": "world"}, pairs)
	require.NoError(t, err)

	require.Len(t, plan.Actions, 1)
	assert.Equal(t, codegen.ActionCreate, plan.Actions[0].Kind)
	assert.Equal(t, 1, plan.Stats.Create)

	_ = fm
}

func TestPlanner_Plan_SkipsUnchanged(t *testing.T) {
	plnr, fm := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n",
	})

	// 预先写入与渲染结果一致的内容
	require.NoError(t, fm.AtomicWrite("main.go", []byte("package main\n"), 0o644))

	pairs := []codegen.Pair{
		{Dst: "main.go", TemplateID: "templates/x.go.tpl", Owner: "test"},
	}
	plan, err := plnr.Plan(context.Background(), nil, pairs)
	require.NoError(t, err)
	assert.Equal(t, codegen.ActionSkip, plan.Actions[0].Kind)
	assert.Equal(t, 1, plan.Stats.Skip)
}

func TestApplier_Apply_CreatesFiles(t *testing.T) {
	p, fm := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n",
	})

	app, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: extractEngine(p),
		FM:     fm,
	})
	require.NoError(t, err)

	pairs := []codegen.Pair{
		{Dst: "main.go", TemplateID: "templates/x.go.tpl", Owner: "test"},
	}
	plan, err := p.Plan(context.Background(), nil, pairs)
	require.NoError(t, err)

	rep, err := app.Apply(context.Background(), nil, pairs, plan)
	require.NoError(t, err)
	assert.Equal(t, []string{"main.go"}, rep.Created)
	assert.True(t, fm.Exists("main.go"))

	written, err := fm.Read("main.go")
	require.NoError(t, err)
	// v0.3.x 起不再附加 hash 注释。
	assert.NotContains(t, string(written), "linctl: hash=")
}

func TestApplier_DryRun_NoWrites(t *testing.T) {
	p, fm := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n",
	})

	app, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: extractEngine(p),
		FM:     fm,
		DryRun: true,
	})
	require.NoError(t, err)

	pairs := []codegen.Pair{
		{Dst: "main.go", TemplateID: "templates/x.go.tpl", Owner: "test"},
	}
	plan, err := p.Plan(context.Background(), nil, pairs)
	require.NoError(t, err)

	rep, err := app.Apply(context.Background(), nil, pairs, plan)
	require.NoError(t, err)
	assert.True(t, rep.DryRun)
	assert.False(t, fm.Exists("main.go"), "dry-run should not write files")
}

// ===== Applier basic tests =====

func TestNewApplier_BasicConstruction(t *testing.T) {
	p, fm := newTestPlannerEnv(t, map[string]string{"templates/x.go.tpl": "package x\n"})
	app, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: extractEngine(p),
		FM:     fm,
	})
	require.NoError(t, err)
	require.NotNil(t, app)
}

// TestApplier_UpdateOverwritesUserUntouched 验证：模板更新后，未被用户改动的
// 已存在文件会被新内容覆盖（v0.3.x 之后 Update 路径退化为「直接覆盖」语义）。
func TestApplier_UpdateOverwritesUserUntouched(t *testing.T) {
	plnr, fm := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n// v1\n",
	})
	pairs := []codegen.Pair{
		{Dst: "main.go", TemplateID: "templates/x.go.tpl", Owner: "test"},
	}

	app1, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: extractEngine(plnr),
		FM:     fm,
	})
	require.NoError(t, err)
	plan1, err := plnr.Plan(context.Background(), nil, pairs)
	require.NoError(t, err)
	_, err = app1.Apply(context.Background(), nil, pairs, plan1)
	require.NoError(t, err)

	plnr2, fm2 := newTestPlannerEnv(t, map[string]string{
		"templates/x.go.tpl": "package main\n// v2\n",
	})
	cur, err := fm.Read("main.go")
	require.NoError(t, err)
	require.NoError(t, fm2.AtomicWrite("main.go", cur, 0o644))

	plan2, err := plnr2.Plan(context.Background(), nil, pairs)
	require.NoError(t, err)
	require.Equal(t, codegen.ActionUpdate, plan2.Actions[0].Kind)

	app2, err := codegen.NewApplier(codegen.ApplierOptions{
		Engine: extractEngine(plnr2),
		FM:     fm2,
	})
	require.NoError(t, err)
	rep, err := app2.Apply(context.Background(), nil, pairs, plan2)
	require.NoError(t, err)
	assert.Equal(t, []string{"main.go"}, rep.Updated, "untouched files must be updated")
}

// ===== helpers =====

// engineHolder 暴露 Planner 内部 engine 给测试。
type engineHolder interface {
	Engine() *template.Engine
}

// extractEngine 从 testEnv 拿出 engine。我们不暴露 Planner.engine，所以测试通过传参用同一个。
// 这里的 plnr 来自 newTestPlannerEnv 内部构造，engine 是 enterprise-shared。
var sharedEngineForTest *template.Engine

func extractEngine(_ *codegen.Planner) *template.Engine {
	return sharedEngineForTest
}

func newTestPlannerEnv(t *testing.T, files map[string]string) (*codegen.Planner, *fs.FileManager) {
	t.Helper()
	tplFS := fstest.MapFS{}
	for path, c := range files {
		tplFS[path] = &fstest.MapFile{Data: []byte(c)}
	}
	eng, err := template.New(template.WithFS(tplFS))
	require.NoError(t, err)
	sharedEngineForTest = eng

	memFs := afero.NewMemMapFs()
	require.NoError(t, memFs.MkdirAll("/proj", 0o755))
	fm, err := fs.NewFileManager(fs.Options{FS: memFs, RootDir: "/proj"})
	require.NoError(t, err)

	plnr, err := codegen.NewPlanner(codegen.PlannerOptions{Engine: eng, FM: fm})
	require.NoError(t, err)
	return plnr, fm
}
