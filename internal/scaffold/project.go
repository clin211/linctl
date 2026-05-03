package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/templates"
)

// NewProject 是 lin new 的核心：根据 ctx 计算 Plan 并执行。
func NewProject(ctx *Context) error {
	// 1. 校验 RootDir 不存在（除非 ctx.Force）
	if _, err := os.Stat(ctx.RootDir); err == nil && !ctx.Force {
		return errs.New(errs.CodeTargetExists,
			fmt.Sprintf("target directory %q already exists", ctx.RootDir)).
			WithHint("use --force to overwrite or choose a different project name")
	}

	// 2. 计算 Plan
	plan, err := BuildPlan(ctx, PlanKindProject)
	if err != nil {
		return err
	}

	// 3. DryRun：打印摘要，不写盘
	if ctx.DryRun {
		printPlanSummary(plan, ctx)
		return nil
	}

	// 4. 渲染
	if err := Render(ctx, plan); err != nil {
		return err
	}

	// 5. 打印 Next steps
	printNextSteps(ctx)
	return nil
}

// BuildPlan 根据 ctx 与 kind 计算当前操作的 Plan。
func BuildPlan(ctx *Context, kind PlanKind) (*Plan, error) {
	if ctx == nil {
		return nil, errs.New(errs.CodeInvalidArg, "scaffold.BuildPlan: nil ctx")
	}

	switch kind {
	case PlanKindProject:
		return buildProjectPlan(ctx)
	case PlanKindResource:
		return buildResourcePlan(ctx)
	default:
		return nil, errs.New(errs.CodeInvalidArg,
			fmt.Sprintf("scaffold.BuildPlan: unknown plan kind %d", kind))
	}
}

// buildProjectPlan 遍历 templates.FS 的 project/ 目录，构建文件清单。
func buildProjectPlan(ctx *Context) (*Plan, error) {
	var paths []string

	// 收集所有模板路径（稳定排序）
	err := fs.WalkDir(templates.FS, "project", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, errs.Wrap(errs.CodeUnknown, "scaffold: walk embedded templates", err)
	}

	sort.Strings(paths)

	plan := &Plan{Kind: PlanKindProject}

	for _, tplPath := range paths {
		// 根据 storage 跳过不需要的文件
		if shouldSkipTemplate(tplPath, ctx) {
			continue
		}

		destPath := templatePathToDestPath(tplPath, ctx.AppName)
		perm := permForPath(destPath)

		plan.Creates = append(plan.Creates, FileSpec{
			TemplatePath: tplPath,
			DestPath:     destPath,
			Permissions:  perm,
		})
	}

	return plan, nil
}

// shouldSkipTemplate 判断是否应跳过指定模板（基于 ctx.Storage/Features）。
func shouldSkipTemplate(tplPath string, ctx *Context) bool {
	// memory storage：跳过 gorm 相关的 pkg 包
	if ctx.Storage == "memory" {
		if strings.HasPrefix(tplPath, "project/pkg/db/") {
			return true
		}
		if strings.HasPrefix(tplPath, "project/pkg/store/") {
			return true
		}
	}
	p := filepath.ToSlash(tplPath)
	if ctx.Storage == "mongo" && strings.HasSuffix(p, "/pkg/db/open.go.tpl") {
		return true
	}
	if ctx.Cache != "redis" && strings.HasSuffix(p, "/pkg/cache/redis.go.tpl") {
		return true
	}
	if ctx.Cache != "bigcache" && strings.HasSuffix(p, "/pkg/cache/bigcache.go.tpl") {
		return true
	}
	return false
}

// templatePathToDestPath 将模板路径转换为目标文件相对路径。
//
// 规则：
//  1. 去掉 "project/" 前缀
//  2. 找到第一个名为 "app" 的目录分段，替换为 appName
//  3. 文件名若以 "app." 开头（如 app.proto.tpl），则 "app" 也替换为 appName
//  4. 去掉 ".tpl" 后缀
func templatePathToDestPath(tplPath, appName string) string {
	// 1. 去掉 "project/" 前缀
	rel := strings.TrimPrefix(tplPath, "project/")

	parts := strings.Split(rel, "/")
	replaced := false // 标记是否已完成第一个 app 目录替换

	for i, part := range parts {
		if i < len(parts)-1 {
			// 目录分段：只替换第一次出现的 "app"
			if !replaced && part == "app" {
				parts[i] = appName
				replaced = true
			}
		} else {
			// 文件名分段：去掉 .tpl 后缀
			name := strings.TrimSuffix(part, ".tpl")
			// 若文件名以 "app." 开头（如 app.proto → appname.proto）
			if strings.HasPrefix(name, "app.") {
				name = appName + name[3:]
			} else if name == "app" {
				name = appName
			}
			parts[i] = name
		}
	}

	return filepath.Join(parts...)
}

// permForPath 根据文件路径推断文件权限（详见 04 §13.4）。
func permForPath(destPath string) os.FileMode {
	switch {
	case strings.HasSuffix(destPath, ".sh"):
		return 0o755
	case filepath.Base(destPath) == "boot.sh":
		return 0o755
	default:
		return 0o644
	}
}

// printPlanSummary 在 dry-run 模式下打印计划摘要。
func printPlanSummary(plan *Plan, ctx *Context) {
	fmt.Printf("✔ project plan computed (%d files)\n", len(plan.Creates))
	fmt.Printf("  Module:   %s\n", ctx.Module)
	fmt.Printf("  AppName:  %s\n", ctx.AppName)
	fmt.Printf("  Storage:  %s\n", ctx.Storage)
	if ctx.Cache != "" && ctx.Cache != "none" {
		fmt.Printf("  Cache:    %s\n", ctx.Cache)
	}
	if len(ctx.Features) > 0 {
		fmt.Printf("  Features: %s\n", strings.Join(ctx.Features, ", "))
	}
	fmt.Printf("  Output:   %s\n", ctx.RootDir)
	fmt.Printf("  [dry-run] no files written\n")
	for _, spec := range plan.Creates {
		fmt.Printf("    + %s\n", spec.DestPath)
	}
}

// printNextSteps 打印项目生成后的提示步骤。
func printNextSteps(ctx *Context) {
	fmt.Printf("✔ scaffold rendered into ./%s\n", ctx.ProjectName)
	fmt.Printf("📦 Next steps:\n")
	fmt.Printf("   cd %s\n", ctx.ProjectName)
	fmt.Printf("   make deps\n")
	fmt.Printf("   make protoc\n")
	fmt.Printf("   make build\n")
}
