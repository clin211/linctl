package templatesync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunner_E2E_FullSyncCycle 是 templatesync 流水线的端到端测试：
//
//  1. 构造一个 mini upstream（含 1 个 .go 文件）
//  2. 写一份对应的 sync.yaml
//  3. 执行 Plan → Apply（force 策略）
//  4. 验证：dst 已写盘 + transform 已应用 + lockfile / cache 已落地
//  5. 再跑 Plan：所有文件应是 ActionSkip（in-sync）
//
// 这个测试覆盖了 U1+U2 的核心 happy path。
func TestRunner_E2E_FullSyncCycle(t *testing.T) {
	// ------------------------------ Arrange ------------------------------
	rootDir := t.TempDir()

	upstreamDir := filepath.Join(rootDir, "upstream")
	templatesDir := filepath.Join(rootDir, "internal/template/templates/web-gin")
	if err := os.MkdirAll(upstreamDir, 0o755); err != nil {
		t.Fatalf("mkdir upstream: %v", err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}

	// 上游：一个简单 Go 文件
	srcRel := "internal/pkg/contextx/contextx.go"
	srcAbs := filepath.Join(upstreamDir, srcRel)
	if err := os.MkdirAll(filepath.Dir(srcAbs), 0o755); err != nil {
		t.Fatalf("mkdir upstream src: %v", err)
	}
	originalSrc := `// Copyright 2026 Foo. All rights reserved.

package contextx

import (
	"context"

	"github.com/clin211/miniblog-v4/internal/pkg/known"
)

var _ = known.Role
var _ = context.Background
`
	if err := os.WriteFile(srcAbs, []byte(originalSrc), 0o644); err != nil {
		t.Fatalf("write upstream src: %v", err)
	}

	// sync.yaml
	manifestPath := filepath.Join(templatesDir, ".sync.yaml")
	manifestYaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: e2e-test
upstream:
  name: miniblog-v4
  rootPath: ../../../../upstream
  moduleOld: github.com/clin211/miniblog-v4
defaultTransforms:
  - kind: rewriteImports
    from: '{{ .Upstream.ModuleOld }}'
    to: github.com/example/replaced
  - kind: stripCopyrightHeader
files:
  - src: internal/pkg/contextx/contextx.go
    dst: internal/pkg/contextx/contextx.go.tpl
    owner: shared
`
	if err := os.WriteFile(manifestPath, []byte(strings.TrimLeft(manifestYaml, "\n")), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// ------------------------------ Act #1 (initial sync) ------------------------------
	r, err := NewRunner(RunnerOptions{
		LinRoot:       rootDir,
		LinctlVersion: "test",
	})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	plan1, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan #1: %v", err)
	}
	if got := len(plan1.Actions); got != 1 {
		t.Fatalf("expected 1 action; got %d", got)
	}
	if plan1.Actions[0].Kind != ActionCreate {
		t.Errorf("expected ActionCreate; got %s", plan1.Actions[0].Kind)
	}

	rep1, err := r.Apply(context.Background(), plan1, StrategyForce, false)
	if err != nil {
		t.Fatalf("Apply #1: %v", err)
	}
	if len(rep1.Created) != 1 {
		t.Errorf("expected 1 created; got %v", rep1.Created)
	}

	// ------------------------------ Assert: dst 内容正确 ------------------------------
	dstAbs := filepath.Join(templatesDir, "internal/pkg/contextx/contextx.go.tpl")
	dstContent, err := os.ReadFile(dstAbs)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	dstStr := string(dstContent)
	if strings.Contains(dstStr, "Copyright 2026 Foo") {
		t.Errorf("copyright not stripped:\n%s", dstStr)
	}
	if strings.Contains(dstStr, "github.com/clin211/miniblog-v4") {
		t.Errorf("import not rewritten:\n%s", dstStr)
	}
	if !strings.Contains(dstStr, "github.com/example/replaced/internal/pkg/known") {
		t.Errorf("expected rewritten import; got\n%s", dstStr)
	}

	// lockfile 应有该条目
	lockedFile, ok := r.Lockfile().Get("internal/pkg/contextx/contextx.go.tpl")
	if !ok {
		t.Fatal("lockfile missing entry")
	}
	if lockedFile.SrcHashAtSync == "" || lockedFile.DstHashAtSync == "" {
		t.Errorf("lockfile entry missing hashes: %+v", lockedFile)
	}

	// cache 应已写入对应 dstHashAtSync
	if !r.BaseCache().Has(lockedFile.DstHashAtSync) {
		t.Errorf("base cache missing %s", lockedFile.DstHashAtSync)
	}

	// ------------------------------ Act #2 (idempotent re-sync) ------------------------------
	// 重新构造一个 Runner 以模拟新进程
	r2, err := NewRunner(RunnerOptions{LinRoot: rootDir, LinctlVersion: "test"})
	if err != nil {
		t.Fatalf("NewRunner #2: %v", err)
	}
	plan2, err := r2.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan #2: %v", err)
	}
	if plan2.Actions[0].Kind != ActionSkip {
		t.Errorf("re-sync should be ActionSkip; got %s reason=%q",
			plan2.Actions[0].Kind, plan2.Actions[0].Reason)
	}
}

func TestRunner_E2E_ConflictDetection(t *testing.T) {
	rootDir := t.TempDir()
	upstreamDir := filepath.Join(rootDir, "upstream")
	templatesDir := filepath.Join(rootDir, "internal/template/templates/web-gin")
	for _, d := range []string{upstreamDir, templatesDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	srcAbs := filepath.Join(upstreamDir, "pkg/log/log.go")
	if err := os.MkdirAll(filepath.Dir(srcAbs), 0o755); err != nil {
		t.Fatalf("mkdir log: %v", err)
	}
	originalSrc := "package log\n\nfunc Default() {}\n"
	if err := os.WriteFile(srcAbs, []byte(originalSrc), 0o644); err != nil {
		t.Fatalf("write upstream: %v", err)
	}

	manifestPath := filepath.Join(templatesDir, ".sync.yaml")
	manifest := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: conflict-e2e
upstream:
  name: miniblog-v4
  rootPath: ../../../../upstream
  moduleOld: example.com/m
files:
  - src: pkg/log/log.go
    dst: pkg/log/log.go.tpl
    owner: shared
`
	if err := os.WriteFile(manifestPath, []byte(strings.TrimLeft(manifest, "\n")), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// 第一次同步建基线
	r, _ := NewRunner(RunnerOptions{LinRoot: rootDir, LinctlVersion: "test"})
	plan1, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan #1: %v", err)
	}
	if _, err := r.Apply(context.Background(), plan1, StrategyForce, false); err != nil {
		t.Fatalf("Apply #1: %v", err)
	}

	// lin-side 改 dst（用户编辑），upstream 也改 src
	dstAbs := filepath.Join(templatesDir, "pkg/log/log.go.tpl")
	if err := os.WriteFile(dstAbs, []byte("package log\n\nfunc Default() { /*lin-edit*/ }\n"), 0o644); err != nil {
		t.Fatalf("modify dst: %v", err)
	}
	if err := os.WriteFile(srcAbs, []byte("package log\n\nfunc Default() { /*upstream-edit*/ }\n"), 0o644); err != nil {
		t.Fatalf("modify upstream: %v", err)
	}

	r2, _ := NewRunner(RunnerOptions{LinRoot: rootDir, LinctlVersion: "test"})
	plan2, err := r2.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan #2: %v", err)
	}
	a := plan2.Actions[0]
	if a.Kind != ActionMergeNeeded {
		t.Errorf("expected ActionMergeNeeded; got %s reason=%q", a.Kind, a.Reason)
	}

	// 用 strategy=ask 走 git merge-file → 应写入 conflict markers
	rep2, err := r2.Apply(context.Background(), plan2, StrategyAsk, false)
	if err != nil {
		t.Fatalf("Apply #2: %v", err)
	}
	if len(rep2.Conflict) != 1 {
		t.Errorf("expected 1 conflict; got %v", rep2.Conflict)
	}
	merged, _ := os.ReadFile(dstAbs)
	if !strings.Contains(string(merged), "<<<<<<<") {
		t.Errorf("expected conflict markers in dst; got:\n%s", merged)
	}
}
