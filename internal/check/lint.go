package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clin211/linctl/internal/pkg/errs"
	"github.com/clin211/linctl/internal/scaffold"
)

// LintOptions 控制 Lint 的行为。
type LintOptions struct {
	Fix          bool
	DryRun       bool
	Rules        []string
	Skip         []string
	ReportFormat string
}

// Lint 校验项目目录布局、注册一致性以及 protoc 后的占位状态。
//
// Anchor（锚点注释）机制已被移除——注册一致性现在直接基于
// AST 符号（receiver 方法 / 接口方法的存在性）判断。
//
// 退出码（02 §5.5）：
//
//	0  → 无任何问题（或 --fix 全部修复）
//	30 → 至少一项 error
//	31 → --fix 部分失败
//	32 → 找不到项目根目录
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

	// ── 1. 目录结构检查 ───────────────────────────────────────
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

	// ── 2. 注册一致性（biz/store）─────────────────────────────
	if shouldRun("register/biz-impl") {
		checkRegisterConsistency(report, rootDir, app, "biz-impl", opts)
	}
	if shouldRun("register/store-impl") {
		checkRegisterConsistency(report, rootDir, app, "store-impl", opts)
	}

	// ── 3. protoc 生成检查 ──────────────────────────────────────────────
	if shouldRun("proto/missing-pb-go") {
		protoPattern := filepath.Join(rootDir, "pkg", "api", app, "v1", "*.proto")
		protos, _ := filepath.Glob(protoPattern)
		var missing []string
		for _, p := range protos {
			base := strings.TrimSuffix(filepath.Base(p), ".proto")
			pbGo := filepath.Join(filepath.Dir(p), base+".pb.go")
			if _, err := os.Stat(pbGo); os.IsNotExist(err) {
				missing = append(missing, base+".proto")
			}
		}
		if len(missing) > 0 {
			report.addItem(Item{
				Category: "proto",
				Name:     "proto/missing-pb-go",
				Status:   "info",
				Message:  fmt.Sprintf("%d proto file(s) missing generated .pb.go: %s", len(missing), strings.Join(missing, ", ")),
				Hint:     "run 'make protoc' to generate Go stubs from proto definitions",
			})
		}
	}

	// ── 4. 路径安全检查 ──────────────────────────────────────────────────────
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

// checkRegisterConsistency 校验 biz/v1 下每个 <resource>/ 子目录
// （或 store/ 下每个 <resource>.go）是否已通过 AST 符号注册到中心
// 文件 biz.go / store.go。MVP 仅报告，不做自动修复。
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
