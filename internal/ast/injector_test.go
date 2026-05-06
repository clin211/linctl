package ast_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clin211/linctl/internal/ast"
)

func TestBackupAndRestore(t *testing.T) {
	root := t.TempDir()

	relPath := filepath.Join("internal", "myblog", "biz", "biz.go")
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("package biz\n// original\n")
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	ts := ast.NewTimestamp()
	if ts == "" {
		t.Error("NewTimestamp returned empty string")
	}

	backupPath, err := ast.BackupFile(root, relPath, ts)
	if err != nil {
		t.Fatalf("BackupFile: %v", err)
	}
	if backupPath == "" {
		t.Error("empty backup path")
	}

	if err := os.WriteFile(absPath, []byte("package biz\n// modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ast.RestoreFromBackup(backupPath, absPath); err != nil {
		t.Fatalf("RestoreFromBackup: %v", err)
	}

	restored, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(content) {
		t.Errorf("restored content mismatch: got %q, want %q", restored, content)
	}

	if err := ast.CleanupBackup(root, ts); err != nil {
		t.Fatalf("CleanupBackup: %v", err)
	}
	backupDir := filepath.Join(root, ".linctl", ".backup", ts)
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Error("backup dir should be removed after cleanup")
	}
}

func TestInjector_Inject(t *testing.T) {
	root := t.TempDir()

	bizDir := filepath.Join(root, "internal", "myblog", "biz")
	if err := os.MkdirAll(bizDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bizSrc := `package biz

import (
	"github.com/test/myblog/internal/myblog/store"
)

type IBiz interface {
}

type biz struct {
	store store.IStore
}

var _ IBiz = (*biz)(nil)

func NewBiz(s store.IStore) *biz { return &biz{store: s} }
`
	bizFile := filepath.Join(bizDir, "biz.go")
	if err := os.WriteFile(bizFile, []byte(bizSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	errnoPkgDir := filepath.Join(root, "internal", "pkg", "errno")
	if err := os.MkdirAll(errnoPkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registerSrc := `package errno

type BizError struct{}

func RegisterErrors(errs ...*BizError) { _ = errs }

func PostErrors() []*BizError { return nil }

func RegisterAll() {
}
`
	registerFile := filepath.Join(errnoPkgDir, "register.go")
	if err := os.WriteFile(registerFile, []byte(registerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	injector := ast.NewInjector(root)
	plan := ast.InjectPlan{
		Specs: []ast.InjectSpec{
			{
				File:    filepath.Join("internal", "myblog", "biz", "biz.go"),
				Mutator: ast.MutatorKindInterface,
				Payload: ast.InterfacePayload{
					InterfaceName: "IBiz",
					StructName:    "biz",
					Method:        "PostV1",
					ReturnType:    "postv1.PostBiz",
					ImportAlias:   "postv1",
					ImportPath:    "github.com/test/myblog/internal/myblog/biz/v1/post",
					ImplBody:      "return postv1.New(b.store)",
				},
			},
			{
				File:    filepath.Join("internal", "pkg", "errno", "register.go"),
				Mutator: ast.MutatorKindRegister,
				Payload: ast.RegisterPayload{
					FunctionName: "RegisterAll",
					Statement:    "RegisterErrors(PostErrors()...)",
				},
			},
		},
	}

	if err := injector.Inject(plan); err != nil {
		t.Fatalf("Inject: %v", err)
	}

	bizData, err := os.ReadFile(bizFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bizData), "PostV1()") {
		t.Errorf("biz.go missing PostV1 after injection\ngot:\n%s", bizData)
	}

	regData, err := os.ReadFile(registerFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(regData), "RegisterErrors(PostErrors()...)") {
		t.Errorf("register.go missing RegisterErrors call\ngot:\n%s", regData)
	}
}
