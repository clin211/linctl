package check_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clin211/lin/internal/check"
)

// makeTestProject creates a minimal lin v2 project in a temp directory.
func makeTestProject(t *testing.T, appName string) string {
	t.Helper()
	dir := t.TempDir()

	writeTestFile(t, filepath.Join(dir, "go.mod"),
		fmt.Sprintf("module github.com/test/%s\n\ngo 1.22\n", appName))

	mustMkdirAll(t, filepath.Join(dir, "cmd", appName))
	writeTestFile(t, filepath.Join(dir, "cmd", appName, "main.go"),
		"package main\n\nfunc main() {}\n")

	for _, sub := range []string{"handler", "biz", "store", "model"} {
		mustMkdirAll(t, filepath.Join(dir, "internal", appName, sub))
	}

	mustMkdirAll(t, filepath.Join(dir, "internal", "pkg", "errno"))

	return dir
}

const bizCentral = `package biz

import (
	"github.com/test/testapp/internal/testapp/store"
)

type IBiz interface {
}

type biz struct {
	store store.IStore
}

var _ IBiz = (*biz)(nil)

func NewBiz(s store.IStore) *biz { return &biz{store: s} }
`

const storeCentral = `package store

type IStore interface {
}

type memoryStore struct {
}

var _ IStore = (*memoryStore)(nil)
`

const registerCentral = `package errno

func RegisterAll() {
}
`

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func setupCentralFiles(t *testing.T, dir, appName string) {
	t.Helper()
	writeTestFile(t, filepath.Join(dir, "internal", appName, "biz", "biz.go"), bizCentral)
	writeTestFile(t, filepath.Join(dir, "internal", appName, "store", "store.go"), storeCentral)
	writeTestFile(t, filepath.Join(dir, "internal", "pkg", "errno", "register.go"), registerCentral)
}

func TestLint_DirChecks_OK(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	report, err := check.Lint(dir, check.LintOptions{
		Rules: []string{"dir/cmd-app", "dir/internal-app"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Summary.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", report.Summary.Errors)
	}
}

func TestLint_DirChecks_MissingInternalDir(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	os.RemoveAll(filepath.Join(dir, "internal", "testapp", "model"))

	report, err := check.Lint(dir, check.LintOptions{
		Rules: []string{"dir/internal-app"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, it := range report.Items {
		if it.Name == "dir/internal-app" && it.Status == "error" {
			found = true
		}
	}
	if !found {
		t.Error("expected dir/internal-app error when model dir is missing")
	}
}

func TestLint_RegisterConsistency_Missing(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	mustMkdirAll(t, filepath.Join(dir, "internal", "testapp", "biz", "v1", "post"))

	report, err := check.Lint(dir, check.LintOptions{
		Rules: []string{"register/biz-impl"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, it := range report.Items {
		if it.Name == "register/biz-impl" && it.Status == "warning" &&
			strings.Contains(it.Message, "post") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about unregistered post resource")
	}
}

func TestLint_PostProtocPlaceholder(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	mustMkdirAll(t, filepath.Join(dir, "pkg", "api", "testapp", "v1"))
	writeTestFile(t,
		filepath.Join(dir, "pkg", "api", "testapp", "v1", "post_lin.go"),
		"package v1\n")

	report, err := check.Lint(dir, check.LintOptions{
		Rules: []string{"lin/post-protoc-placeholder"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, it := range report.Items {
		if it.Name == "lin/post-protoc-placeholder" && it.Status == "info" {
			found = true
			if !strings.Contains(it.Message, "post_lin.go") {
				t.Errorf("expected post_lin.go in message, got: %s", it.Message)
			}
		}
	}
	if !found {
		t.Error("expected lin/post-protoc-placeholder info item")
	}
}

func TestLint_SkipRules(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	report, err := check.Lint(dir, check.LintOptions{
		Skip: []string{"dir/cmd-app"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, it := range report.Items {
		if it.Name == "dir/cmd-app" {
			t.Errorf("skipped rule dir/cmd-app appeared in report")
		}
	}
}

func TestLint_InvalidProjectRoot(t *testing.T) {
	_, err := check.Lint(t.TempDir(), check.LintOptions{})
	if err == nil {
		t.Fatal("expected error for invalid project root, got nil")
	}
}

func TestLint_JSONFormat_Output(t *testing.T) {
	dir := makeTestProject(t, "testapp")
	setupCentralFiles(t, dir, "testapp")

	report, err := check.Lint(dir, check.LintOptions{
		Rules: []string{"dir/cmd-app"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var buf bytes.Buffer
	if err := check.PrintReport(report, "json", &buf); err != nil {
		t.Fatalf("PrintReport json error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"summary"`) {
		t.Error("json report missing 'summary' field")
	}
	if !strings.Contains(output, `"items"`) {
		t.Error("json report missing 'items' field")
	}
}
