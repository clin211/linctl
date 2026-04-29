package templatesync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadLockfile_NotExist_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "absent.lock.json")
	lf, err := LoadLockfile(path)
	if err != nil {
		t.Fatalf("expected no error for missing lockfile; got %v", err)
	}
	if lf == nil {
		t.Fatal("expected non-nil lockfile for missing path")
	}
	if len(lf.Files) != 0 {
		t.Errorf("expected empty files map; got %d entries", len(lf.Files))
	}
}

func TestLockfile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upstream-sync.lock.json")

	lf, err := LoadLockfile(path)
	if err != nil {
		t.Fatalf("LoadLockfile: %v", err)
	}
	lf.LastSyncAt = time.Date(2026, 4, 28, 13, 45, 0, 0, time.UTC)
	lf.LinctlVersion = "v0.3.0"
	lf.Upstream = LockUpstream{Name: "miniblog-v4", RootPath: "../../"}
	lf.Update("internal/pkg/contextx/contextx.go.tpl", LockedFile{
		Src:               "internal/pkg/contextx/contextx.go",
		Owner:             OwnerShared,
		SrcHashAtSync:     "sha256:abc",
		DstHashAtSync:     "sha256:def",
		TransformsApplied: []string{"rewriteImports", "stripCopyrightHeader"},
		LastSyncAt:        time.Date(2026, 4, 28, 13, 45, 0, 0, time.UTC),
	})
	if err := lf.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 重新加载，验证字段一一对应
	lf2, err := LoadLockfile(path)
	if err != nil {
		t.Fatalf("second LoadLockfile: %v", err)
	}
	if lf2.LinctlVersion != "v0.3.0" {
		t.Errorf("LinctlVersion=%q; want v0.3.0", lf2.LinctlVersion)
	}
	got, ok := lf2.Get("internal/pkg/contextx/contextx.go.tpl")
	if !ok {
		t.Fatal("expected key present after reload")
	}
	if got.Owner != OwnerShared {
		t.Errorf("owner=%q; want shared", got.Owner)
	}
	if got.SrcHashAtSync != "sha256:abc" {
		t.Errorf("srcHashAtSync mismatch: %v", got)
	}
	if len(got.TransformsApplied) != 2 || got.TransformsApplied[0] != "rewriteImports" {
		t.Errorf("transformsApplied mismatch: %v", got.TransformsApplied)
	}
}

func TestLockfile_BadSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.lock.json")
	bad := `{"schemaVersion":"99","files":{}}`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadLockfile(path)
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Errorf("expected schemaVersion error; got %v", err)
	}
}

func TestLockfile_UpdateAndDelete(t *testing.T) {
	lf := &Lockfile{
		SchemaVersion: lockSchemaVersion,
		Files:         make(map[string]LockedFile),
	}
	lf.SetPath("/tmp/x.json") // 仅为通过 Save 检查；本测试不调 Save

	lf.Update("a", LockedFile{Owner: OwnerUpstream})
	lf.Update("b", LockedFile{Owner: OwnerShared})
	if len(lf.Files) != 2 {
		t.Errorf("expected 2 entries; got %d", len(lf.Files))
	}

	all := lf.AllDsts()
	if len(all) != 2 || all[0] != "a" || all[1] != "b" {
		t.Errorf("AllDsts=%v; want [a b]", all)
	}

	lf.Delete("a")
	if _, ok := lf.Get("a"); ok {
		t.Error("'a' should be deleted")
	}
	if _, ok := lf.Get("b"); !ok {
		t.Error("'b' should still exist")
	}
}
