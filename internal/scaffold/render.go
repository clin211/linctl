package scaffold

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/clin211/linctl/internal/pkg/errs"
	"github.com/clin211/linctl/internal/pkg/fsx"
)

// templateVars 是传递给模板执行的变量集合。
// 字段名对应模板中的 {{.Module}}、{{.AppName}} 等。
type templateVars struct {
	Module      string
	AppName     string
	ProjectName string
	Storage     string
	Cache       string
	Framework   string
	Features    []string
	Resource    string // PascalCase 资源名（如 "Post"），仅 linctl add 时设置
	Author      string
	Email       string
	Year        int
	GoVersion   string
	LinVersion  string
}

// newTemplateVars 从 Context 构建模板变量。
func newTemplateVars(ctx *Context) templateVars {
	return templateVars{
		Module:      ctx.Module,
		AppName:     ctx.AppName,
		ProjectName: ctx.ProjectName,
		Storage:     ctx.Storage,
		Cache:       ctx.Cache,
		Framework:   ctx.Framework,
		Features:    ctx.Features,
		Resource:    ctx.Resource,
		Author:      ctx.Author,
		Email:       ctx.Email,
		Year:        ctx.Year,
		GoVersion:   ctx.GoVersion,
		LinVersion:  ctx.LinVersion,
	}
}

// Render 接受一个 Plan，将其 Creates 全部写入磁盘。
// 设计来源：04 §13.3 文件类型与处理方式、§9 加载实现。
func Render(ctx *Context, plan *Plan) error {
	vars := newTemplateVars(ctx)
	for _, spec := range plan.Creates {
		if err := renderOne(ctx, spec, vars); err != nil {
			return err
		}
	}
	return nil
}

// renderOne 渲染单个 FileSpec 到磁盘。
func renderOne(ctx *Context, spec FileSpec, vars templateVars) error {
	// 1. 安全路径拼接，防路径穿越
	dst, err := fsx.SafeJoin(ctx.RootDir, spec.DestPath)
	if err != nil {
		return err
	}

	// 2. 空模板路径 → 按位拷贝（当前不含二进制文件，保留接口）
	if spec.TemplatePath == "" {
		return fsx.WriteFileAtomic(dst, nil, spec.Permissions)
	}

	// 3. 从 Loader 加载模板
	t, err := ctx.Templates.Load(spec.TemplatePath)
	if err != nil {
		return renderDiag(spec.TemplatePath, err)
	}

	// 4. 执行模板
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return renderDiag(spec.TemplatePath, errs.Wrap(errs.CodeTplExecError,
			fmt.Sprintf("tpl: execute %s", spec.TemplatePath), err))
	}

	// 5. 统一 LF 行尾符（.bat/.cmd 除外）
	content := buf.Bytes()
	if !isCRLFFile(spec.DestPath) {
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	}

	// 6. 原子写入
	perm := spec.Permissions
	if perm == 0 {
		perm = 0o644
	}
	if err := fsx.WriteFileAtomic(dst, content, perm); err != nil {
		return err
	}
	return nil
}

// isCRLFFile 报告文件是否应使用 CRLF 行尾符（.bat/.cmd）。
func isCRLFFile(destPath string) bool {
	lower := strings.ToLower(destPath)
	return strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".cmd")
}

// renderDiag 生成三段式渲染诊断信息（详见 04 §8）。
func renderDiag(tplPath string, err error) error {
	var e *errs.Error
	msg := err.Error()
	hint := ""
	code := errs.CodeRenderFailed
	if asErr, ok := err.(*errs.Error); ok {
		e = asErr
		msg = e.Message
		hint = e.Hint
		code = e.Code
	}
	diag := fmt.Sprintf("template render failed\n  Template: %s\n  Error:    %s", tplPath, msg)
	if hint != "" {
		diag += "\n  Hint:     " + hint
	}
	_ = e // 抑制未使用变量告警
	return errs.New(code, diag)
}
