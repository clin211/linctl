package check_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/clin211/lin/internal/check"
)

func TestDoctor_Offline_ReturnsValidReport(t *testing.T) {
	report, err := check.Doctor(check.DoctorOptions{Offline: true})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	if report == nil {
		t.Fatal("Doctor returned nil report")
	}
	if len(report.Items) == 0 {
		t.Error("Doctor returned empty items")
	}

	total := report.Summary.OK + report.Summary.Warnings + report.Summary.Errors + report.Summary.Info
	if total != len(report.Items) {
		t.Errorf("summary total %d != items count %d", total, len(report.Items))
	}
}

func TestDoctor_GoVersion_OK(t *testing.T) {
	report, err := check.Doctor(check.DoctorOptions{
		Offline: true,
		Checks:  []string{"go/version"},
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	if len(report.Items) == 0 {
		t.Fatal("expected at least one item")
	}
	for _, item := range report.Items {
		if item.Name == "go" {
			if item.Status == "error" {
				t.Errorf("go/version check failed: %s (hint: %s)", item.Message, item.Hint)
			}
			return
		}
	}
	t.Error("expected 'go' item in report")
}

func TestDoctor_GOFLAGSWarning(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=vendor")

	report, err := check.Doctor(check.DoctorOptions{
		Offline: true,
		Checks:  []string{"go/goflags"},
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	for _, item := range report.Items {
		if item.Name == "GOFLAGS" {
			if item.Status != "warning" {
				t.Errorf("expected GOFLAGS=warning, got %s", item.Status)
			}
			return
		}
	}
	t.Error("expected GOFLAGS item in report")
}

func TestDoctor_GOFLAGSClean(t *testing.T) {
	orig := os.Getenv("GOFLAGS")
	os.Unsetenv("GOFLAGS")
	defer os.Setenv("GOFLAGS", orig)

	report, err := check.Doctor(check.DoctorOptions{
		Offline: true,
		Checks:  []string{"go/goflags"},
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	for _, item := range report.Items {
		if item.Name == "GOFLAGS" {
			if item.Status != "ok" {
				t.Errorf("expected GOFLAGS=ok, got %s", item.Status)
			}
			return
		}
	}
	t.Error("expected GOFLAGS item in report")
}

func TestDoctor_SkipChecks(t *testing.T) {
	report, err := check.Doctor(check.DoctorOptions{
		Offline: true,
		Skip:    []string{"go/version", "go/goflags"},
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}
	for _, item := range report.Items {
		if item.Name == "go" || item.Name == "GOFLAGS" {
			t.Errorf("skipped check %q appeared in report", item.Name)
		}
	}
}

func TestDoctor_PrintReport_Text(t *testing.T) {
	report, _ := check.Doctor(check.DoctorOptions{Offline: true})

	var buf bytes.Buffer
	if err := check.PrintReport(report, "text", &buf); err != nil {
		t.Fatalf("PrintReport error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "ok") && !strings.Contains(output, "warning") && !strings.Contains(output, "error") {
		t.Error("text report missing expected status symbols")
	}
}

func TestDoctor_PrintReport_JSON(t *testing.T) {
	report, _ := check.Doctor(check.DoctorOptions{Offline: true})

	var buf bytes.Buffer
	if err := check.PrintReport(report, "json", &buf); err != nil {
		t.Fatalf("PrintReport JSON error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"summary"`) {
		t.Errorf("JSON report missing 'summary' field; got: %s", output[:min(len(output), 200)])
	}
	if !strings.Contains(output, `"items"`) {
		t.Errorf("JSON report missing 'items' field")
	}
}

func TestDoctor_PrintReport_TextStatusSymbols(t *testing.T) {
	report := &check.Report{
		Items: []check.Item{
			{Name: "go", Status: "ok", Message: "1.22"},
			{Name: "protoc", Status: "warning", Message: "not found", Hint: "install it"},
			{Name: "broken", Status: "error", Message: "error!", Hint: "fix it"},
			{Name: "terminal", Status: "info", Message: "tty=true"},
		},
		Summary: check.Summary{OK: 1, Warnings: 1, Errors: 1, Info: 1},
	}

	var buf bytes.Buffer
	if err := check.PrintReport(report, "text", &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "✔") {
		t.Error("text report missing ✔ symbol for ok")
	}
	if !strings.Contains(output, "⚠") {
		t.Error("text report missing ⚠ symbol for warning")
	}
	if !strings.Contains(output, "✘") {
		t.Error("text report missing ✘ symbol for error")
	}
	if !strings.Contains(output, "ℹ") {
		t.Error("text report missing ℹ symbol for info")
	}
	if !strings.Contains(output, "1 ok / 1 warning / 1 error") {
		t.Errorf("text report missing summary line; got:\n%s", output)
	}
}

func TestDoctor_SubsetChecks(t *testing.T) {
	report, err := check.Doctor(check.DoctorOptions{
		Offline: true,
		Checks:  []string{"git/version", "system/terminal"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Items) != 2 {
		t.Errorf("expected exactly 2 items (git + terminal), got %d", len(report.Items))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
