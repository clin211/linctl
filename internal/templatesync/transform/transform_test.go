package transform

import (
	"context"
	"strings"
	"testing"
)

// fakeAccessor 是测试用的 ManifestAccessor。
type fakeAccessor struct{}

func (fakeAccessor) UpstreamModuleOld() string        { return "github.com/clin211/miniblog-v4" }
func (fakeAccessor) UpstreamBinaryNameOld() string    { return "blog-apiserver" }
func (fakeAccessor) UpstreamComponentNameOld() string { return "apiserver" }
func (fakeAccessor) UpstreamName() string             { return "miniblog-v4" }

func TestRewriteImports_BasicGoFile(t *testing.T) {
	src := `package contextx

import (
	"context"

	"github.com/clin211/miniblog-v4/internal/pkg/known"
	"github.com/clin211/miniblog-v4/pkg/log"
)

var _ = context.Background
var _ = known.Role
var _ = log.Default
`
	tr := &RewriteImports{
		From: "github.com/clin211/miniblog-v4",
		To:   "EXAMPLE.COM/MODULE",
	}
	rc := &RuntimeContext{
		SrcPath:  "internal/pkg/contextx/contextx.go",
		DstPath:  "internal/pkg/contextx/contextx.go.tpl",
		Manifest: fakeAccessor{},
	}
	out, err := tr.Apply(context.Background(), rc, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"EXAMPLE.COM/MODULE/internal/pkg/known"`) {
		t.Errorf("rewritten import not found in output:\n%s", got)
	}
	if strings.Contains(got, "github.com/clin211/miniblog-v4") {
		t.Errorf("old import still present:\n%s", got)
	}
}

func TestRewriteImports_ManifestRefExpansion(t *testing.T) {
	src := `package x

import "github.com/clin211/miniblog-v4/pkg/log"
`
	tr := &RewriteImports{
		From: "{{ .Upstream.ModuleOld }}",
		To:   "EXAMPLE.COM/MODULE",
	}
	rc := &RuntimeContext{
		SrcPath:  "x.go",
		DstPath:  "x.go.tpl",
		Manifest: fakeAccessor{},
	}
	out, err := tr.Apply(context.Background(), rc, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "EXAMPLE.COM/MODULE/pkg/log") {
		t.Errorf("expected expanded import; got:\n%s", got)
	}
}

func TestRewriteImports_NonGoFile_NoOp(t *testing.T) {
	src := `# this is a yaml file
foo: github.com/clin211/miniblog-v4
`
	tr := &RewriteImports{From: "github.com/clin211/miniblog-v4", To: "X"}
	rc := &RuntimeContext{SrcPath: "configs/app.yaml", DstPath: "configs/app.yaml.tpl"}
	out, err := tr.Apply(context.Background(), rc, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if string(out) != src {
		t.Errorf("non-go file should be no-op; got changed:\n%s", string(out))
	}
}

func TestReplaceLiteral_PairsOrderSensitive(t *testing.T) {
	src := "blog-apiserver and apiserver"
	tr := &ReplaceLiteral{
		Pairs: []LiteralPair{
			// 长串先：避免 "apiserver" 提前替换 "blog-apiserver"
			{From: "blog-apiserver", To: "X"},
			{From: "apiserver", To: "Y"},
		},
	}
	out, err := tr.Apply(context.Background(), &RuntimeContext{}, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if got := string(out); got != "X and Y" {
		t.Errorf("want %q, got %q", "X and Y", got)
	}
}

func TestStripCopyrightHeader_Go(t *testing.T) {
	src := `// Copyright 2026 Foo Bar Inc.
// All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package contextx

func F() {}
`
	tr := &StripCopyrightHeader{}
	rc := &RuntimeContext{SrcPath: "internal/pkg/contextx/contextx.go"}
	out, err := tr.Apply(context.Background(), rc, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "Copyright") {
		t.Errorf("copyright not stripped:\n%s", got)
	}
	if !strings.HasPrefix(got, "package contextx") {
		t.Errorf("expected output to start with 'package contextx'; got:\n%s", got)
	}
}

func TestStripCopyrightHeader_NoOpWhenNoCopyright(t *testing.T) {
	src := `// doc.go for the contextx package
package contextx
`
	tr := &StripCopyrightHeader{}
	rc := &RuntimeContext{SrcPath: "doc.go"}
	out, err := tr.Apply(context.Background(), rc, []byte(src))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if string(out) != src {
		t.Errorf("non-copyright comment should be preserved")
	}
}

func TestAddExtension_InferDst(t *testing.T) {
	tr := &AddExtension{Suffix: ".tpl", AppliesTo: []string{"*.go", "*.proto"}}
	cases := map[string]string{
		"internal/pkg/contextx/contextx.go": "internal/pkg/contextx/contextx.go.tpl",
		"api/user.proto":                    "api/user.proto.tpl",
		"configs/app.yaml":                  "configs/app.yaml", // appliesTo 不含 yaml → 原样
	}
	for src, want := range cases {
		got := tr.InferDst(src)
		if got != want {
			t.Errorf("InferDst(%q) = %q; want %q", src, got, want)
		}
	}
}

func TestInsertLinctlHeader_DefaultTemplate(t *testing.T) {
	tr := &InsertLinctlHeader{UpstreamCommit: "abc123", SyncManifest: ".sync.yaml"}
	rc := &RuntimeContext{
		SrcPath:  "internal/pkg/contextx/contextx.go",
		Manifest: fakeAccessor{},
	}
	out, err := tr.Apply(context.Background(), rc, []byte("package contextx\n"))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "linctl 模板系统从 miniblog-v4@abc123") {
		t.Errorf("header missing or wrong:\n%s", got)
	}
	if !strings.Contains(got, "package contextx") {
		t.Errorf("body lost:\n%s", got)
	}
}

func TestInsertLinctlHeader_Idempotent(t *testing.T) {
	tr := &InsertLinctlHeader{UpstreamCommit: "abc123"}
	rc := &RuntimeContext{
		SrcPath:  "x.go",
		Manifest: fakeAccessor{},
	}
	first, _ := tr.Apply(context.Background(), rc, []byte("package x\n"))
	second, _ := tr.Apply(context.Background(), rc, first)
	if !strings.HasSuffix(string(second), "package x\n") {
		t.Errorf("second pass output unexpected:\n%s", string(second))
	}
	// 头部 marker 不应出现两次
	if strings.Count(string(second), "linctl 模板系统从") != 1 {
		t.Errorf("header should be idempotent; got count %d:\n%s",
			strings.Count(string(second), "linctl 模板系统从"), string(second))
	}
}

func TestRegistry_Build(t *testing.T) {
	tests := []struct {
		kind    string
		rawCfg  map[string]any
		wantErr bool
	}{
		{"rewriteImports", map[string]any{"from": "a", "to": "b"}, false},
		{"replaceLiteral", map[string]any{"pairs": []any{
			map[string]any{"from": "x", "to": "y"},
		}}, false},
		{"stripCopyrightHeader", map[string]any{}, false},
		{"addExtension", map[string]any{"suffix": ".tpl"}, false},
		{"insertLinctlHeader", map[string]any{}, false},
		{"unknownTransform", map[string]any{}, true},
		{"rewriteImports", map[string]any{}, true}, // missing 'from'
	}
	for _, tc := range tests {
		t.Run(tc.kind, func(t *testing.T) {
			_, err := DefaultRegistry.Build(tc.kind, tc.rawCfg)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for kind=%s", tc.kind)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for kind=%s: %v", tc.kind, err)
			}
		})
	}
}
