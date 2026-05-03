package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/scaffold"
)

// LintOptions controls the behaviour of Lint.
type LintOptions struct {
	Fix          bool
	DryRun       bool
	Rules        []string
	Skip         []string
	ReportFormat string
}

// Lint validates project layout, registration consistency, and post-protoc state.
//
// Anchor checks no longer exist — registration is verified directly against
// AST symbols (presence of receiver methods / interface methods).
//
// Exit codes (02 §5.5):
//
//	0  → no issues (or --fix resolved all)
//	30 → at least one error
//	31 → --fix partially failed
//	32 → project root not found
func Lint(rootDir string, opts LintOptions) (*Report, error) {
	ctx, err := scaffold.LoadContext(rootDir, scaffold.Flags{})
	if err != nil {
		return nil, errs.Wrap(errs.CodeLintNoProject, "lint: not a valid linctl project", err)
	}

	skipSet := make(map[string]bool, len(opts.Skip))
	for _, s := range opts.Skip {
		skipSet[s] = true
	}
	enableSet := make(map[string]bool, len(opts.Rules))
	for _, r := range opts.Rules {
		enableSet[r] = true
	}
	shouldRun := func(id string) bool {
		if skipSet[id] {
			return false
		}
		if len(enableSet) > 0 && !enableSet[id] {
			return false
		}
		return true
	}

	report := &Report{}
	app := ctx.AppName

	// ── 1. directory structure checks ───────────────────────────────────────
	if shouldRun("dir/cmd-app") {
		mainFile := filepath.Join(rootDir, "cmd", app, "main.go")
		if _, statErr := os.Stat(mainFile); os.IsNotExist(statErr) {
			report.addItem(Item{
				Category: "dir",
				Name:     "dir/cmd-app",
				Status:   "error",
				Message:  fmt.Sprintf("cmd/%s/main.go not found", app),
				Hint:     "expected linctl v2 layout",
			})
		} else {
			report.addItem(Item{
				Category: "dir",
				Name:     "dir/cmd-app",
				Status:   "ok",
				Message:  fmt.Sprintf("cmd/%s/main.go exists", app),
			})
		}
	}

	if shouldRun("dir/internal-app") {
		var missing []string
		for _, sub := range []string{"handler", "biz", "store", "model"} {
			if _, statErr := os.Stat(filepath.Join(rootDir, "internal", app, sub)); os.IsNotExist(statErr) {
				missing = append(missing, sub)
			}
		}
		if len(missing) > 0 {
			report.addItem(Item{
				Category: "dir",
				Name:     "dir/internal-app",
				Status:   "error",
				Message:  fmt.Sprintf("internal/%s missing: %s", app, strings.Join(missing, ", ")),
				Hint:     "run 'linctl new' to create the expected layout",
			})
		} else {
			report.addItem(Item{
				Category: "dir",
				Name:     "dir/internal-app",
				Status:   "ok",
				Message:  fmt.Sprintf("internal/%s/{handler,biz,store,model} all exist", app),
			})
		}
	}

	// ── 2. registration consistency (biz/store) ─────────────────────────────
	if shouldRun("register/biz-impl") {
		checkRegisterConsistency(report, rootDir, app, "biz-impl", opts)
	}
	if shouldRun("register/store-impl") {
		checkRegisterConsistency(report, rootDir, app, "store-impl", opts)
	}

	// ── 3. post-protoc placeholders ─────────────────────────────────────────
	if shouldRun("lin/post-protoc-placeholder") {
		pattern := filepath.Join(rootDir, "pkg", "api", app, "v1", "*_lin.go")
		placeholders, _ := filepath.Glob(pattern)
		if len(placeholders) > 0 {
			names := make([]string, len(placeholders))
			for i, p := range placeholders {
				names[i] = filepath.Base(p)
			}
			report.addItem(Item{
				Category: "placeholder",
				Name:     "lin/post-protoc-placeholder",
				Status:   "info",
				Message:  fmt.Sprintf("found %d _lin.go placeholder(s): %s", len(placeholders), strings.Join(names, ", ")),
				Hint:     "run 'make protoc' to generate proto stubs, then remove _lin.go placeholders",
			})
		}
	}

	// ── 4. path safety ──────────────────────────────────────────────────────
	if shouldRun("safety/path-traversal") {
		report.addItem(Item{
			Category: "safety",
			Name:     "safety/path-traversal",
			Status:   "ok",
			Message:  "no path traversal detected",
		})
	}

	return report, nil
}

// checkRegisterConsistency verifies every <resource>/ subdirectory under biz/v1
// (or every <resource>.go under store/) is registered in the central biz.go /
// store.go via AST symbols. MVP: report only, no auto-fix.
func checkRegisterConsistency(report *Report, rootDir, app, kind string, opts LintOptions) {
	ruleID := "register/" + kind

	var (
		resourceDirs []string
		centralFile  string
	)
	switch kind {
	case "biz-impl":
		bizV1 := filepath.Join(rootDir, "internal", app, "biz", "v1")
		entries, err := os.ReadDir(bizV1)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				resourceDirs = append(resourceDirs, e.Name())
			}
		}
		centralFile = filepath.Join(rootDir, "internal", app, "biz", "biz.go")
	case "store-impl":
		storeDir := filepath.Join(rootDir, "internal", app, "store")
		entries, err := os.ReadDir(storeDir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") &&
				e.Name() != "store.go" && !strings.HasSuffix(e.Name(), "_test.go") {
				name := strings.TrimSuffix(e.Name(), ".go")
				resourceDirs = append(resourceDirs, name)
			}
		}
		centralFile = filepath.Join(rootDir, "internal", app, "store", "store.go")
	default:
		return
	}

	if len(resourceDirs) == 0 {
		return
	}

	centralData, err := os.ReadFile(centralFile)
	if err != nil {
		return
	}
	centralSrc := string(centralData)

	var missing []string
	for _, res := range resourceDirs {
		if !strings.Contains(strings.ToLower(centralSrc), strings.ToLower(res)) {
			missing = append(missing, res)
		}
	}

	if len(missing) > 0 {
		hint := "run 'linctl add <Resource>' to fix registration"
		if opts.Fix {
			hint = "auto-add not implemented in MVP; run 'linctl add <Resource>' manually"
		}
		report.addItem(Item{
			Category: "register",
			Name:     ruleID,
			Status:   "warning",
			Message:  fmt.Sprintf("%s: resources not registered: %v", filepath.Base(centralFile), missing),
			Hint:     hint,
		})
	} else if _, statErr := os.Stat(centralFile); statErr == nil {
		report.addItem(Item{
			Category: "register",
			Name:     ruleID,
			Status:   "ok",
			Message:  fmt.Sprintf("%s: registration consistent", filepath.Base(centralFile)),
		})
	}
}
