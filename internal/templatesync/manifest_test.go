package templatesync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write tmp manifest: %v", err)
	}
	// 在 dir 旁建一个空的 upstream root 目录，避免 AbsUpstreamRoot 校验失败
	upstreamDir := filepath.Join(dir, "upstream")
	if err := os.MkdirAll(upstreamDir, 0o755); err != nil {
		t.Fatalf("mkdir upstream: %v", err)
	}
	return path
}

func TestLoadManifest_Minimal(t *testing.T) {
	yaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: test-sync
upstream:
  name: miniblog-v4
  rootPath: ./upstream
  moduleOld: github.com/clin211/miniblog-v4
defaultTransforms:
  - kind: rewriteImports
    from: github.com/clin211/miniblog-v4
    to: NEW
files:
  - src: pkg/log/log.go
    dst: pkg/log/log.go.tpl
    owner: upstream
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest error: %v", err)
	}
	if m.Metadata.Name != "test-sync" {
		t.Errorf("metadata.name=%q; want test-sync", m.Metadata.Name)
	}
	if got := len(m.DefaultTransforms); got != 1 {
		t.Errorf("len(defaultTransforms)=%d; want 1", got)
	}
	if got := m.DefaultTransforms[0].Kind; got != "rewriteImports" {
		t.Errorf("transform[0].Kind=%q; want rewriteImports", got)
	}
	if from := m.DefaultTransforms[0].RawCfg["from"]; from != "github.com/clin211/miniblog-v4" {
		t.Errorf("transform.from=%v; want github.com/clin211/miniblog-v4", from)
	}
	if got := len(m.Files); got != 1 {
		t.Errorf("len(files)=%d; want 1", got)
	}
	if m.Files[0].Owner != OwnerUpstream {
		t.Errorf("files[0].owner=%q; want upstream", m.Files[0].Owner)
	}
	if got := m.NewFilePolicy.Default; got != "warn" {
		t.Errorf("default newFilePolicy.default=%q; want warn (auto-applied)", got)
	}
}

func TestLoadManifest_BadAPIVersion(t *testing.T) {
	yaml := `
apiVersion: wrong/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./upstream
  moduleOld: m
`
	path := writeTempManifest(t, yaml)
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "apiVersion") {
		t.Errorf("expected apiVersion error; got %v", err)
	}
}

func TestLoadManifest_DuplicateDst(t *testing.T) {
	yaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./upstream
  moduleOld: m
files:
  - src: a.go
    dst: a.go.tpl
    owner: upstream
  - src: b.go
    dst: a.go.tpl
    owner: upstream
`
	path := writeTempManifest(t, yaml)
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "duplicates") {
		t.Errorf("expected duplicate dst error; got %v", err)
	}
}

func TestLoadManifest_InvalidOwner(t *testing.T) {
	yaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./upstream
  moduleOld: m
files:
  - src: a.go
    dst: a.go.tpl
    owner: alien
`
	path := writeTempManifest(t, yaml)
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "owner=") {
		t.Errorf("expected invalid owner error; got %v", err)
	}
}

func TestManifest_AbsUpstreamRoot(t *testing.T) {
	yaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./upstream
  moduleOld: m
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	abs, err := m.AbsUpstreamRoot()
	if err != nil {
		t.Fatalf("AbsUpstreamRoot: %v", err)
	}
	expected := filepath.Join(filepath.Dir(path), "upstream")
	if abs != expected {
		t.Errorf("AbsUpstreamRoot=%q; want %q", abs, expected)
	}
}

func TestManifest_AllDsts(t *testing.T) {
	yaml := `
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./upstream
  moduleOld: m
files:
  - src: pkg/b.go
    dst: pkg/b.go.tpl
    owner: upstream
  - src: pkg/a.go
    dst: pkg/a.go.tpl
    owner: shared
linSpecific:
  - dst: lin/c.go.tpl
    reason: for-test
`
	path := writeTempManifest(t, yaml)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	all := m.AllDsts()
	wantOrder := []string{"lin/c.go.tpl", "pkg/a.go.tpl", "pkg/b.go.tpl"}
	if len(all) != len(wantOrder) {
		t.Fatalf("len(all)=%d; want %d", len(all), len(wantOrder))
	}
	for i, want := range wantOrder {
		if all[i] != want {
			t.Errorf("all[%d]=%q; want %q", i, all[i], want)
		}
	}
}
