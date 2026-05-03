package check

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DoctorOptions controls the behaviour of Doctor.
type DoctorOptions struct {
	Strict       bool
	Offline      bool
	ReportFormat string
	Checks       []string // subset of check IDs to run
	Skip         []string // check IDs to skip
}

// Doctor inspects the local environment for tools required by lin.
//
// 退出码规则（02 §6.5）：
//
//	0  → all error-level checks pass
//	35 → at least one error item
//	36 → --strict and at least one warning
func Doctor(opts DoctorOptions) (*Report, error) {
	type checkFn struct {
		id  string
		run func() Item
	}

	checks := []checkFn{
		{"go/version", checkGoVersion},
		{"go/goflags", checkGoFlags},
		{"git/version", checkGitVersion},
		{"tools/protoc", checkProtoc},
		{"tools/protoc-gen-go", checkProtocGenGo},
		{"tools/wire", checkWire},
		{"system/terminal", checkTerminal},
	}

	if !opts.Offline {
		strict := opts.Strict
		checks = append(checks, checkFn{
			"network/proxy",
			func() Item { return checkProxy(strict) },
		})
	}

	skipSet := make(map[string]bool, len(opts.Skip))
	for _, s := range opts.Skip {
		skipSet[s] = true
	}
	enableSet := make(map[string]bool, len(opts.Checks))
	for _, c := range opts.Checks {
		enableSet[c] = true
	}

	report := &Report{}
	for _, c := range checks {
		if skipSet[c.id] {
			continue
		}
		if len(enableSet) > 0 && !enableSet[c.id] {
			continue
		}
		report.addItem(c.run())
	}

	return report, nil
}

func checkGoVersion() Item {
	goVer := strings.TrimPrefix(runtime.Version(), "go")
	parts := strings.Split(goVer, ".")
	if len(parts) >= 2 {
		major, _ := strconv.Atoi(parts[0])
		minor, _ := strconv.Atoi(parts[1])
		if major > 1 || (major == 1 && minor >= 22) {
			return Item{Category: "go", Name: "go", Status: "ok", Message: goVer}
		}
		return Item{
			Category: "go",
			Name:     "go",
			Status:   "error",
			Message:  fmt.Sprintf("go %s", goVer),
			Hint:     "linctl requires go >= 1.22; upgrade at https://go.dev/dl/",
		}
	}
	return Item{Category: "go", Name: "go", Status: "ok", Message: goVer}
}

func checkGoFlags() Item {
	goflags := os.Getenv("GOFLAGS")
	if strings.Contains(goflags, "-mod=vendor") {
		return Item{
			Category: "go",
			Name:     "GOFLAGS",
			Status:   "warning",
			Message:  fmt.Sprintf("GOFLAGS=%s", goflags),
			Hint:     "linctl scaffold conflicts with -mod=vendor; unset or change GOFLAGS",
		}
	}
	return Item{Category: "go", Name: "GOFLAGS", Status: "ok", Message: "no -mod=vendor"}
}

func checkGitVersion() Item {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return Item{
			Category: "git",
			Name:     "git",
			Status:   "warning",
			Message:  "not found",
			Hint:     "install git >= 2.30 from https://git-scm.com/",
		}
	}

	verStr := strings.TrimPrefix(strings.TrimSpace(string(out)), "git version ")
	// macOS git appends extra "(Apple Git-143)" style info
	if sp := strings.Fields(verStr); len(sp) > 0 {
		verStr = sp[0]
	}

	parts := strings.Split(verStr, ".")
	if len(parts) >= 2 {
		major, _ := strconv.Atoi(parts[0])
		minor, _ := strconv.Atoi(parts[1])
		if major > 2 || (major == 2 && minor >= 30) {
			return Item{Category: "git", Name: "git", Status: "ok", Message: verStr}
		}
		return Item{
			Category: "git",
			Name:     "git",
			Status:   "warning",
			Message:  verStr,
			Hint:     "linctl recommends git >= 2.30; upgrade at https://git-scm.com/",
		}
	}
	return Item{Category: "git", Name: "git", Status: "ok", Message: verStr}
}

func checkProtoc() Item {
	path, err := exec.LookPath("protoc")
	if err != nil {
		return Item{
			Category: "protoc",
			Name:     "protoc",
			Status:   "warning",
			Message:  "not found",
			Hint:     "needed for `linctl add --with proto`; install from https://github.com/protocolbuffers/protobuf/releases",
		}
	}
	out, _ := exec.Command("protoc", "--version").Output()
	return Item{
		Category: "protoc",
		Name:     "protoc",
		Status:   "ok",
		Message:  fmt.Sprintf("%s (%s)", strings.TrimSpace(string(out)), path),
	}
}

func checkProtocGenGo() Item {
	path, err := exec.LookPath("protoc-gen-go")
	if err != nil {
		return Item{
			Category: "protoc",
			Name:     "protoc-gen-go",
			Status:   "warning",
			Message:  "not found",
			Hint:     "install via: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		}
	}
	return Item{Category: "protoc", Name: "protoc-gen-go", Status: "ok", Message: path}
}

func checkWire() Item {
	path, err := exec.LookPath("wire")
	if err != nil {
		return Item{
			Category: "system",
			Name:     "wire",
			Status:   "warning",
			Message:  "not found",
			Hint:     "install via: go install github.com/google/wire/cmd/wire@latest",
		}
	}
	return Item{Category: "system", Name: "wire", Status: "ok", Message: path}
}

func checkTerminal() Item {
	fi, _ := os.Stderr.Stat()
	isTTY := fi != nil && (fi.Mode()&os.ModeCharDevice) != 0

	width := "unknown"
	if out, err := exec.Command("tput", "cols").Output(); err == nil {
		width = strings.TrimSpace(string(out))
	}

	hasColor := os.Getenv("COLORTERM") != "" ||
		strings.Contains(os.Getenv("TERM"), "256color") ||
		os.Getenv("TERM") == "xterm"

	return Item{
		Category: "system",
		Name:     "terminal",
		Status:   "info",
		Message:  fmt.Sprintf("tty=%v colors=%v width=%s", isTTY, hasColor, width),
	}
}

func checkProxy(strict bool) Item {
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := client.Get("https://proxy.golang.org/github.com/clin211/lin/@v/list")
	if err != nil {
		hint := "check your internet connection or use --offline"
		status := "info"
		if strict {
			status = "error"
		}
		return Item{
			Category: "network",
			Name:     "proxy.golang.org",
			Status:   status,
			Message:  fmt.Sprintf("unreachable: %v", err),
			Hint:     hint,
		}
	}
	return Item{
		Category: "network",
		Name:     "proxy.golang.org",
		Status:   "info",
		Message:  "reachable",
	}
}
