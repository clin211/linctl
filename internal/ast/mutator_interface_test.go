package ast_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clin211/lin/internal/ast"
)

const bizSourceClean = `package biz

import (
	"github.com/test/myblog/internal/myblog/store"
)

// IBiz defines the methods.
type IBiz interface {
}

type biz struct {
	store store.IStore
}

var _ IBiz = (*biz)(nil)

func NewBiz(s store.IStore) *biz { return &biz{store: s} }
`

func TestAddInterfaceMethod(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		payload   ast.InterfacePayload
		wantIface string
		wantImpl  string
		wantErr   bool
	}{
		{
			name: "add new method",
			src:  bizSourceClean,
			payload: ast.InterfacePayload{
				InterfaceName: "IBiz",
				StructName:    "biz",
				Method:        "PostV1",
				ReturnType:    "postv1.PostBiz",
				ImportAlias:   "postv1",
				ImportPath:    "github.com/test/myblog/internal/myblog/biz/v1/post",
				ImplBody:      "return postv1.New(b.store)",
			},
			wantIface: "PostV1()",
			wantImpl:  "func (b *biz) PostV1()",
		},
		{
			name: "missing interface",
			src: `package biz

type OtherIface interface{}
`,
			payload: ast.InterfacePayload{
				InterfaceName: "IBiz",
				StructName:    "biz",
				Method:        "PostV1",
				ReturnType:    "postv1.PostBiz",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			goFile := filepath.Join(dir, "biz.go")
			if err := os.WriteFile(goFile, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}

			err := ast.AddInterfaceMethod(goFile, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("AddInterfaceMethod: %v", err)
			}

			data, err := os.ReadFile(goFile)
			if err != nil {
				t.Fatal(err)
			}
			result := string(data)

			if tt.wantIface != "" && !strings.Contains(result, tt.wantIface) {
				t.Errorf("interface missing %q\ngot:\n%s", tt.wantIface, result)
			}
			if tt.wantImpl != "" && !strings.Contains(result, tt.wantImpl) {
				t.Errorf("receiver method missing %q\ngot:\n%s", tt.wantImpl, result)
			}

			data1 := string(data)
			if err := ast.AddInterfaceMethod(goFile, tt.payload); err != nil {
				t.Fatalf("second AddInterfaceMethod: %v", err)
			}
			data2, _ := os.ReadFile(goFile)
			if string(data2) != data1 {
				t.Errorf("not idempotent: changed on second call\nbefore:\n%s\nafter:\n%s", data1, string(data2))
			}
		})
	}
}
