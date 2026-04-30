package ast_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clin211/lin/internal/ast"
)

func TestAddProtoImport(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		importVal  string
		wantLine   string
		wantBefore string // line that should appear before the inserted import (empty = no constraint)
		idempotent bool
	}{
		{
			name: "insert sorted between existing imports",
			initial: `syntax = "proto3";
package myblog.v1;

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";
`,
			importVal:  "post.proto",
			wantLine:   `import "post.proto";`,
			wantBefore: `import "protoc-gen-openapiv2`,
		},
		{
			name: "insert after package when no imports",
			initial: `syntax = "proto3";
package myblog.v1;

option go_package = "myblog/v1;v1";
`,
			importVal: "post.proto",
			wantLine:  `import "post.proto";`,
		},
		{
			name: "idempotent: already exists",
			initial: `syntax = "proto3";
package myblog.v1;

import "post.proto";
`,
			importVal:  "post.proto",
			wantLine:   `import "post.proto";`,
			idempotent: true,
		},
		{
			name: "insert unsorted: append to end of imports",
			initial: `syntax = "proto3";
package myblog.v1;

import "z_last.proto";
import "a_first.proto";
`,
			importVal: "post.proto",
			wantLine:  `import "post.proto";`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			protoFile := filepath.Join(dir, "test.proto")
			if err := os.WriteFile(protoFile, []byte(tt.initial), 0o644); err != nil {
				t.Fatal(err)
			}

			p := ast.ProtoPayload{
				File:   protoFile,
				Import: tt.importVal,
			}
			if err := ast.AddProtoImport(protoFile, p); err != nil {
				t.Fatalf("AddProtoImport: %v", err)
			}

			// Read result
			data, err := os.ReadFile(protoFile)
			if err != nil {
				t.Fatal(err)
			}
			result := string(data)

			if !strings.Contains(result, tt.wantLine) {
				t.Errorf("result missing %q\ngot:\n%s", tt.wantLine, result)
			}

			// Run again: should be idempotent
			if err := ast.AddProtoImport(protoFile, p); err != nil {
				t.Fatalf("second AddProtoImport: %v", err)
			}
			data2, _ := os.ReadFile(protoFile)
			if string(data2) != result {
				t.Errorf("not idempotent: result changed on second call")
			}
		})
	}
}
