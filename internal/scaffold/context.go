// Package scaffold 是 lin v2 的骨架生成核心。
//
// 设计来源：lin/docs/features/01-architecture-blueprint.md §3「模块职责矩阵」§7「关键抽象」。
//
// 模块职责（L2 核心生成层）：
//   - context.go : 项目上下文（Module/AppName/Storage 推断）
//   - plan.go    : 文件清单 + 注入清单
//   - render.go  : 模板渲染封装
//   - project.go : lin new 入口
//   - resource.go: lin add 入口（后续 stage）
package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/pkg/tpl"
	"github.com/clin211/lin/internal/templates"
)

// Context 持有当前命令的全部上下文信息。
//
// 字段来源详见：
//   - lin new : 由命令行 flags 直接构造
//   - lin add : 由 LoadContext 从 ./go.mod / ./cmd/* 推断（详见 02 §4.4）
type Context struct {
	// ProjectName 是用户传入的项目名（如 "myblog"）。
	ProjectName string

	// RootDir 是项目根的绝对路径（必填）。
	RootDir string

	// Module 是 Go module 路径，从 go.mod 推断（add）或由用户传入（new）。
	Module string

	// AppName 是应用名，从 cmd/<app>/ 推断（add）或由 project-name 派生（new）。
	AppName string

	// Storage 表示存储后端：memory | gorm-postgres | gorm-mysql | gorm-sqlite | mongo。
	Storage string

	// Cache 表示缓存后端：none | redis | bigcache。
	Cache string

	// Framework 表示 Web 框架：MVP 阶段仅支持 gin。
	Framework string

	// Features 是启用的可选特性：otel | healthz | user | swagger | preloader 等。
	Features []string

	// Resource 仅在 lin add 时有效：PascalCase 资源名（如 "Post"）。
	Resource string

	// Templates 是模板加载器（按 04 §2 优先级查找）。
	Templates *tpl.Loader

	// DryRun 为 true 时仅打印计划，不写入磁盘。
	DryRun bool

	// Force 为 true 时允许覆盖既有文件（含 .lin/.backup/<ts>/ 备份）。
	Force bool

	// 元数据
	Author     string
	Email      string
	Year       int
	GoVersion  string
	LinVersion string
}

// Flags 是 NewContextFromFlags 的输入：从命令行解析的原始 flags。
//
// 设计意图：保持 cli 层与 scaffold 层解耦——cli 仅负责把 cobra flags 装进 Flags 结构体，
// scaffold 接管推断与校验。
type Flags struct {
	ProjectName string
	Module      string
	AppName     string
	Chdir       string
	Storage     string
	Cache       string
	Framework   string
	Features    []string
	Resource    string
	TemplateDir string
	OutputDir   string
	DryRun      bool
	Force       bool
	NonInteract bool
	Author      string
	Email       string
}

// 合法的 Go module path 正则：domain.tld/owner/name（至少两段，含点）。
var moduleRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*\.[a-zA-Z]{2,}(/[a-zA-Z0-9][a-zA-Z0-9._~-]*)+$`)

// 合法的 appName 正则：[a-z0-9_]+，首字符非数字。
var appNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var validStorages = map[string]bool{
	"memory":        true,
	"gorm-postgres": true,
	"gorm-mysql":    true,
	"gorm-sqlite":   true,
	"mongo":         true,
}

var validCaches = map[string]bool{
	"none":     true,
	"redis":    true,
	"bigcache": true,
}

var validFeatures = map[string]bool{
	"healthz":   true,
	"otel":      true,
	"user":      true,
	"swagger":   true,
	"preloader": true,
}

// ValidateProjectName 校验目录名将能规范为合法的 AppName（[a-z][a-z0-9_]*）。
func ValidateProjectName(projectName string) error {
	if strings.TrimSpace(projectName) == "" {
		return errs.New(errs.CodeInvalidArg, "scaffold: project name is empty")
	}
	app := strings.ToLower(strings.TrimSpace(projectName))
	app = strings.ReplaceAll(app, "-", "_")
	if !appNameRe.MatchString(app) {
		return errs.New(errs.CodeInvalidArg,
			fmt.Sprintf("invalid project name %q (normalized %q)", projectName, app)).
			WithHint("use letters, digits, hyphen or underscore; must not start with a digit after normalization")
	}
	return nil
}

// ValidateModulePath 校验 Go module path（domain.tld/owner/...）。
func ValidateModulePath(module string) error {
	mod := strings.TrimSpace(module)
	if mod == "" {
		return errs.New(errs.CodeInvalidArg, "scaffold: module path is required").
			WithHint("example: github.com/yourname/myproject")
	}
	if !moduleRe.MatchString(mod) {
		return errs.New(errs.CodeBadModule,
			fmt.Sprintf("scaffold: invalid module path %q", mod)).
			WithHint("module must be like domain.tld/owner/name")
	}
	return nil
}

// NewContextFromFlags 用于 lin new：直接由 flags 构造（不需要推断）。
func NewContextFromFlags(flags Flags) (*Context, error) {
	if err := ValidateProjectName(flags.ProjectName); err != nil {
		return nil, err
	}
	if err := ValidateModulePath(flags.Module); err != nil {
		return nil, err
	}

	// 2. 派生 AppName
	appName := flags.AppName
	if appName == "" {
		appName = flags.ProjectName
	}
	appName = strings.ToLower(appName)
	appName = strings.ReplaceAll(appName, "-", "_")
	if !appNameRe.MatchString(appName) {
		return nil, errs.New(errs.CodeInvalidArg,
			fmt.Sprintf("scaffold: invalid app name %q (derived from project name)", appName)).
			WithHint("app name must match [a-z][a-z0-9_]*")
	}

	// 3. 校验 Storage
	storage := flags.Storage
	if storage == "" {
		storage = "memory"
	}
	if !validStorages[storage] {
		return nil, errs.New(errs.CodeInvalidArg,
			fmt.Sprintf("scaffold: unsupported storage %q", storage)).
			WithHint("valid values: memory, gorm-postgres, gorm-mysql, gorm-sqlite, mongo")
	}

	cache := strings.TrimSpace(flags.Cache)
	if cache == "" {
		cache = "none"
	}
	if !validCaches[cache] {
		return nil, errs.New(errs.CodeInvalidArg,
			fmt.Sprintf("scaffold: unsupported cache %q", cache)).
			WithHint("valid values: none, redis, bigcache")
	}

	// 4. 校验 Features
	for _, f := range flags.Features {
		if !validFeatures[f] {
			return nil, errs.New(errs.CodeInvalidArg,
				fmt.Sprintf("scaffold: unsupported feature %q", f)).
				WithHint("valid features: healthz, otel, user, swagger, preloader")
		}
	}

	// 5. Framework（MVP 仅 gin）
	framework := flags.Framework
	if framework == "" {
		framework = "gin"
	}

	// 6. RootDir = abs(OutputDir / ProjectName)
	outDir := flags.OutputDir
	if outDir == "" {
		outDir = "."
	}
	rootDir, err := filepath.Abs(filepath.Join(outDir, flags.ProjectName))
	if err != nil {
		return nil, errs.Wrap(errs.CodeUnknown, "scaffold: resolve root dir", err)
	}

	// 7. Author / Email：从 git config 推断（若未传）
	author := flags.Author
	email := flags.Email
	if author == "" {
		author = gitConfigValue("user.name")
	}
	if email == "" {
		email = gitConfigValue("user.email")
	}

	// 8. GoVersion 从 runtime.Version() 派生
	goVer := runtime.Version()
	goVer = strings.TrimPrefix(goVer, "go")
	// 只保留 major.minor（如 1.25）
	parts := strings.Split(goVer, ".")
	if len(parts) >= 2 {
		goVer = parts[0] + "." + parts[1]
	}

	// 9. 构建 Templates loader（含 EmbeddedFS）
	loader, err := tpl.NewLoader(tpl.Options{
		TemplateDir: flags.TemplateDir,
		EmbeddedFS:  templates.FS,
	})
	if err != nil {
		return nil, errs.Wrap(errs.CodeUnknown, "scaffold: init template loader", err)
	}

	return &Context{
		ProjectName: flags.ProjectName,
		RootDir:     rootDir,
		Module:      flags.Module,
		AppName:     appName,
		Storage:     storage,
		Cache:       cache,
		Framework:   framework,
		Features:    flags.Features,
		Templates:   loader,
		DryRun:      flags.DryRun,
		Force:       flags.Force,
		Author:      author,
		Email:       email,
		Year:        time.Now().Year(),
		GoVersion:   goVer,
		LinVersion:  "2.0.0-rc1",
	}, nil
}

// LoadContext 用于 lin add：从已有项目根推断元信息（go.mod / cmd/<app>/ / store 中的 import）。
//
// 实现来源：02 §4.4 + 04 §4.1。
func LoadContext(rootDir string, flags Flags) (*Context, error) {
	// 0. 如果传入了 Chdir，先切换目录
	if flags.Chdir != "" {
		abs, err := filepath.Abs(flags.Chdir)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInvalidArg, "scaffold: resolve chdir", err)
		}
		rootDir = abs
	}

	// 1. 从 rootDir 向上查找含 go.mod 的目录
	projectRoot, err := findProjectRoot(rootDir)
	if err != nil {
		return nil, err
	}

	// 2. 读 go.mod 第一行获取 module 路径
	module, err := readModulePath(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return nil, err
	}

	// 3. 推断 AppName：列 cmd/* 子目录
	appName := flags.AppName
	if appName == "" {
		appName, err = inferAppName(projectRoot)
		if err != nil {
			return nil, err
		}
	}

	// 4. 检查必要的目录结构（model/ 由 lin add 创建，所以不强制要求）
	requiredDirs := []string{
		filepath.Join(projectRoot, "internal", appName, "handler"),
		filepath.Join(projectRoot, "internal", appName, "biz"),
		filepath.Join(projectRoot, "internal", appName, "store"),
	}
	for _, d := range requiredDirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			return nil, errs.New(errs.CodeNotProjectRoot,
				fmt.Sprintf("scaffold: expected directory %q not found", d)).
				WithHint("linctl v2 expects miniblog-v4 layout")
		}
	}

	// 5. 推断 Storage（从 store/store.go 的 import 检测）
	storage := flags.Storage
	if storage == "" {
		storage = inferStorage(filepath.Join(projectRoot, "internal", appName, "store", "store.go"))
		if storage == "memory" {
			if m := inferStorageFromGoMod(filepath.Join(projectRoot, "go.mod")); m != "" {
				storage = m
			}
		}
	}
	if !validStorages[storage] {
		storage = "memory"
	}

	cache := strings.TrimSpace(flags.Cache)
	if cache == "" {
		cache = "none"
	}
	if !validCaches[cache] {
		cache = "none"
	}

	// 6. GoVersion
	goVer := runtime.Version()
	goVer = strings.TrimPrefix(goVer, "go")
	parts := strings.Split(goVer, ".")
	if len(parts) >= 2 {
		goVer = parts[0] + "." + parts[1]
	}

	// 7. Templates loader
	loader, err := tpl.NewLoader(tpl.Options{
		TemplateDir: flags.TemplateDir,
		ProjectRoot: projectRoot,
		EmbeddedFS:  templates.FS,
	})
	if err != nil {
		return nil, errs.Wrap(errs.CodeUnknown, "scaffold: init template loader", err)
	}

	return &Context{
		ProjectName: appName,
		RootDir:     projectRoot,
		Module:      module,
		AppName:     appName,
		Storage:     storage,
		Cache:       cache,
		Framework:   "gin",
		Templates:   loader,
		DryRun:      flags.DryRun,
		Force:       flags.Force,
		Author:      gitConfigValue("user.name"),
		Email:       gitConfigValue("user.email"),
		Year:        time.Now().Year(),
		GoVersion:   goVer,
		LinVersion:  "2.0.0-rc1",
	}, nil
}

// findProjectRoot walks up from startDir looking for a directory with go.mod.
func findProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", errs.Wrap(errs.CodeNotProjectRoot, "scaffold: resolve start dir", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}
	return "", errs.New(errs.CodeNotProjectRoot,
		fmt.Sprintf("scaffold: no go.mod found from %q upward", startDir)).
		WithHint("run `linctl add` from inside a Go project (must have go.mod)")
}

// readModulePath reads the first "module ..." line from go.mod.
func readModulePath(gomodPath string) (string, error) {
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", errs.Wrap(errs.CodeBadGoMod, "scaffold: read go.mod", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			mod := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if mod == "" {
				return "", errs.New(errs.CodeBadGoMod, "scaffold: empty module path in go.mod")
			}
			return mod, nil
		}
	}
	return "", errs.New(errs.CodeBadGoMod, "scaffold: module declaration not found in go.mod")
}

// inferAppName lists cmd/* subdirectories to find the app name.
func inferAppName(projectRoot string) (string, error) {
	cmdDir := filepath.Join(projectRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return "", errs.Wrap(errs.CodeMultiAppNoFlag,
			fmt.Sprintf("scaffold: cannot read cmd/ in %s", projectRoot), err)
	}

	var apps []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Check if it has a main.go
		if _, err := os.Stat(filepath.Join(cmdDir, e.Name(), "main.go")); err == nil {
			apps = append(apps, e.Name())
		}
	}

	switch len(apps) {
	case 0:
		return "", errs.New(errs.CodeMultiAppNoFlag,
			"scaffold: no app found in cmd/ (need a subdirectory with main.go)").
			WithHint("create cmd/<app>/main.go or pass --app <name>")
	case 1:
		return apps[0], nil
	default:
		return "", errs.New(errs.CodeMultiAppNoFlag,
			fmt.Sprintf("scaffold: multiple apps found in cmd/: %v", apps)).
			WithHint("pass --app <name> to specify which app to add to")
	}
}

// inferStorage reads store.go and checks imports to determine the storage backend.
func inferStorage(storePath string) string {
	data, err := os.ReadFile(storePath)
	if err != nil {
		return "memory"
	}
	src := string(data)
	switch {
	case strings.Contains(src, "gorm.io") && strings.Contains(src, "driver/postgres"):
		return "gorm-postgres"
	case strings.Contains(src, "gorm.io") && strings.Contains(src, "driver/mysql"):
		return "gorm-mysql"
	case strings.Contains(src, "gorm.io") && strings.Contains(src, "driver/sqlite"):
		return "gorm-sqlite"
	case strings.Contains(src, "go.mongodb.org/mongo-driver"):
		return "mongo"
	case strings.Contains(src, "gorm.io"):
		return "gorm-mysql" // gorm without explicit driver → default to mysql
	default:
		return "memory"
	}
}

// inferStorageFromGoMod 检测 go.mod 是否将 mongo-driver 列为**直接**依赖（排除 // indirect），
// 与「mongo 脚手架下 store 仍为 memory 形状」的场景配套。
func inferStorageFromGoMod(gomodPath string) string {
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return ""
	}
	return parseGoModForMongoDriver(string(data))
}

func parseGoModForMongoDriver(content string) string {
	lines := strings.Split(content, "\n")
	inRequireBlock := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock {
			if t == ")" {
				inRequireBlock = false
				continue
			}
			if strings.Contains(t, "// indirect") {
				continue
			}
			fields := strings.Fields(t)
			if len(fields) > 0 && strings.HasPrefix(fields[0], "go.mongodb.org/mongo-driver") {
				return "mongo"
			}
			continue
		}
		if strings.HasPrefix(t, "require ") && !strings.Contains(t, "(") {
			rest := strings.TrimSpace(strings.TrimPrefix(t, "require "))
			if strings.Contains(rest, "// indirect") {
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) > 0 && strings.HasPrefix(fields[0], "go.mongodb.org/mongo-driver") {
				return "mongo"
			}
		}
	}
	return ""
}

// gitConfigValue 读取 git 全局配置的值（如 user.name/user.email）。
func gitConfigValue(key string) string {
	out, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
