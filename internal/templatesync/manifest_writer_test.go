package templatesync

import (
	"strings"
	"testing"
)

func TestInjectFileEntry_AppendsAfterLastFileEntry(t *testing.T) {
	yaml := strings.TrimLeft(`
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./up
  moduleOld: m
files:
  # --- comment for first ---
  - src: a.go
    dst: a.go.tpl
    owner: upstream

  - src: b.go
    dst: b.go.tpl
    owner: upstream

# ============================================================
# linSpecific block
# ============================================================
linSpecific: []
`, "\n")

	out, err := injectFileEntry([]byte(yaml), FileMapping{
		Src:   "c.go",
		Dst:   "c.go.tpl",
		Owner: OwnerShared,
	})
	if err != nil {
		t.Fatalf("injectFileEntry: %v", err)
	}
	got := string(out)

	// 新条目应该出现在 b.go 之后、linSpecific 注释块之前
	idxB := strings.Index(got, "src: b.go")
	idxC := strings.Index(got, "src: c.go")
	idxComment := strings.Index(got, "# linSpecific block")
	idxLinSpec := strings.Index(got, "linSpecific:")

	if idxC < 0 {
		t.Fatalf("new entry not added:\n%s", got)
	}
	if !(idxB < idxC) {
		t.Errorf("new entry should be after b.go; got\n%s", got)
	}
	if !(idxC < idxComment) {
		t.Errorf("new entry should be before linSpecific comment block; got\n%s", got)
	}
	if !(idxComment < idxLinSpec) {
		t.Errorf("comment block should still be before linSpecific:; got\n%s", got)
	}
	if !strings.Contains(got, "owner: shared") {
		t.Errorf("owner should be 'shared'; got\n%s", got)
	}
}

func TestInjectFileEntry_EmptyArrayShorthand(t *testing.T) {
	yaml := strings.TrimLeft(`
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./up
  moduleOld: m
files: []
linSpecific: []
`, "\n")

	out, err := injectFileEntry([]byte(yaml), FileMapping{
		Src:   "a.go",
		Dst:   "a.go.tpl",
		Owner: OwnerUpstream,
	})
	if err != nil {
		t.Fatalf("injectFileEntry: %v", err)
	}
	got := string(out)

	// 新条目应该取代 "files: []" 形式
	if strings.Contains(got, "files: []") {
		t.Errorf("should rewrite 'files: []' into block form; got\n%s", got)
	}
	if !strings.Contains(got, "files:\n  - src: a.go") {
		t.Errorf("expected new block after 'files:'; got\n%s", got)
	}
}

func TestInjectFileEntry_MissingFilesKey(t *testing.T) {
	yaml := strings.TrimLeft(`
apiVersion: linctl-internal/v1
kind: TemplateUpstreamSync
metadata:
  name: x
upstream:
  name: y
  rootPath: ./up
  moduleOld: m
linSpecific: []
`, "\n")

	out, err := injectFileEntry([]byte(yaml), FileMapping{
		Src:   "a.go",
		Dst:   "a.go.tpl",
		Owner: OwnerUpstream,
	})
	if err != nil {
		t.Fatalf("injectFileEntry: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "files:") {
		t.Errorf("expected 'files:' to be created; got\n%s", got)
	}
	if !strings.Contains(got, "src: a.go") {
		t.Errorf("entry not added; got\n%s", got)
	}
}

func TestYamlScalar_QuotesSpecialChars(t *testing.T) {
	tests := map[string]string{
		"plain/path.go":       "plain/path.go",
		"path:with:colon":     `"path:with:colon"`,
		"#starts-with-hash":   `"#starts-with-hash"`,
		"normal":              "normal",
	}
	for in, want := range tests {
		if got := yamlScalar(in); got != want {
			t.Errorf("yamlScalar(%q) = %q; want %q", in, got, want)
		}
	}
}
