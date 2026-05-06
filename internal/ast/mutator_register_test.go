package ast_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clin211/linctl/internal/ast"
)

func TestAppendRegistration(t *testing.T) {
	src := `package errno

type BizError struct{}

func RegisterErrors(errs ...*BizError) { _ = errs }

func PostErrors() []*BizError { return nil }

func RegisterAll() {
}
`

	tests := []struct {
		name      string
		statement string
		wantIn    string
	}{
		{
			name:      "inject PostErrors",
			statement: "RegisterErrors(PostErrors()...)",
			wantIn:    "RegisterErrors(PostErrors()...)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			goFile := filepath.Join(dir, "register.go")
			if err := os.WriteFile(goFile, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}

			p := ast.RegisterPayload{
				FunctionName: "RegisterAll",
				Statement:    tt.statement,
			}
			if err := ast.AppendRegistration(goFile, p); err != nil {
				t.Fatalf("AppendRegistration: %v", err)
			}

			data, err := os.ReadFile(goFile)
			if err != nil {
				t.Fatal(err)
			}
			result := string(data)
			if !strings.Contains(result, tt.wantIn) {
				t.Errorf("result missing %q\ngot:\n%s", tt.wantIn, result)
			}

			data1 := string(data)
			if err := ast.AppendRegistration(goFile, p); err != nil {
				t.Fatalf("second AppendRegistration: %v", err)
			}
			data2, _ := os.ReadFile(goFile)
			if string(data2) != data1 {
				t.Errorf("not idempotent: result changed on second call\nbefore:\n%s\nafter:\n%s", data1, string(data2))
			}
		})
	}
}

func TestAppendRegistration_MissingFunction(t *testing.T) {
	src := `package errno

func SomeOther() {}
`
	dir := t.TempDir()
	goFile := filepath.Join(dir, "register.go")
	if err := os.WriteFile(goFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	p := ast.RegisterPayload{
		FunctionName: "RegisterAll",
		Statement:    "RegisterErrors(PostErrors()...)",
	}
	err := ast.AppendRegistration(goFile, p)
	if err == nil {
		t.Error("expected error for missing function, got nil")
	}
}
