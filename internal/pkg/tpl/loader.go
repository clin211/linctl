// Package tpl 实现 lin v2 的模板加载（embed + 外部目录覆盖）。
//
// 设计来源：lin/docs/features/04-template-system.md §2「模板查找优先级」、§9「模板加载实现」。
//
// 加载优先级（高 → 低）：
//  1. 命令行 --template-dir <abs-path>
//  2. 项目根目录 ./.lin/templates/
//  3. 用户目录 ~/.lin/templates/
//  4. embed.FS（本仓库内 internal/templates/）
//
// MVP 阶段 embed.FS 与具体模板内容由后续 stage 提供；本文件先建立加载骨架。
package tpl

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/pkg/fsx"
)

// Options 控制 Loader 的覆盖目录。
type Options struct {
	TemplateDir string // --template-dir 指定的绝对路径（最高优先级）
	ProjectRoot string // 当前项目根（用于 ./.lin/templates/）
	EmbeddedFS  fs.FS  // 内嵌模板 FS（在 Phase 2 由 internal/templates 提供）
}

// Loader 按优先级查找并解析模板。
type Loader struct {
	overrideDirs []string
	embedded     fs.FS
}

// NewLoader 按 Options 构建 Loader（仅校验路径存在；不实际打开文件）。
func NewLoader(opts Options) (*Loader, error) {
	var dirs []string

	if opts.TemplateDir != "" {
		abs, err := filepath.Abs(opts.TemplateDir)
		if err != nil {
			return nil, errs.Wrap(errs.CodeInvalidArg, "tpl: abs template-dir", err)
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			dirs = append(dirs, abs)
		}
	}

	if opts.ProjectRoot != "" {
		p := filepath.Join(opts.ProjectRoot, ".lin", "templates")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			dirs = append(dirs, p)
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".lin", "templates")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			dirs = append(dirs, p)
		}
	}

	return &Loader{overrideDirs: dirs, embedded: opts.EmbeddedFS}, nil
}

// Load 按优先级查找并解析 relPath 对应的模板。
//
// relPath 必须使用斜杠分隔（如 "project/cmd/app/main.go.tpl"）；
// 含 ".." 等穿越段会被拒绝（CodeTplPathTraversal）。
func (l *Loader) Load(relPath string) (*template.Template, error) {
	rel := filepath.ToSlash(filepath.Clean(relPath))
	if strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return nil, errs.New(
			errs.CodeTplPathTraversal,
			fmt.Sprintf("tpl: template path traversal: %s", relPath),
		)
	}

	for _, dir := range l.overrideDirs {
		full, err := fsx.SafeJoin(dir, rel)
		if err != nil {
			continue
		}
		if info, statErr := os.Stat(full); statErr == nil && !info.IsDir() {
			return parseFromFile(full)
		}
	}

	if l.embedded != nil {
		return parseFromFS(l.embedded, rel)
	}

	return nil, errs.New(errs.CodeTplNotFound, fmt.Sprintf("tpl: not found: %s", relPath))
}

// DefaultFuncs 返回 lin v2 的 funcMap（详见 04 §5）。MVP 阶段先留接口，
// 在 Phase 2 由后续 stage 充实（strcase / inflection / Pascal / Has 等）。
func DefaultFuncs() template.FuncMap {
	return template.FuncMap{}
}

func parseFromFile(path string) (*template.Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errs.Wrap(errs.CodeTplParseError, "tpl: read file", err)
	}
	t, err := template.New(filepath.Base(path)).Funcs(DefaultFuncs()).Parse(string(stripBOM(raw)))
	if err != nil {
		return nil, errs.Wrap(errs.CodeTplParseError, fmt.Sprintf("tpl: parse %s", path), err)
	}
	return t, nil
}

func parseFromFS(filesystem fs.FS, relPath string) (*template.Template, error) {
	raw, err := fs.ReadFile(filesystem, relPath)
	if err != nil {
		return nil, errs.Wrap(errs.CodeTplNotFound, fmt.Sprintf("tpl: embed not found: %s", relPath), err)
	}
	t, err := template.New(filepath.Base(relPath)).Funcs(DefaultFuncs()).Parse(string(stripBOM(raw)))
	if err != nil {
		return nil, errs.Wrap(errs.CodeTplParseError, fmt.Sprintf("tpl: parse embed %s", relPath), err)
	}
	return t, nil
}

// stripBOM 去除 UTF-8 BOM（详见 04 §13.2）。
func stripBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}
