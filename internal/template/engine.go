package template

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	texttemplate "text/template"

	"github.com/clin211/lin/internal/linctlerr"
)

// Engine 是 linctl 的模板渲染引擎。
//
// 核心特性：
//   - 业务模板首次 Render 时 Parse 并缓存
//   - missingkey=error 严格模式（模板访问不存在字段直接报错）
//   - 失败时返回 RenderError 携带模板路径 + 数据上下文
//   - 并发安全（parsedCache 用 sync.Map）
//
// 用法：
//
//	eng, err := template.New(template.WithFS(myFS))
//	if err != nil { return err }
//	out, err := eng.Render("templates/component/webserver/main.go.tpl", data)
type Engine struct {
	fs          fs.FS
	funcMap     texttemplate.FuncMap
	parsedCache sync.Map // map[string]*texttemplate.Template
}

// Option 是 Engine 构造选项。
type Option func(*Engine)

// WithFS 注入自定义 fs.FS（如内存测试 FS）。默认为 TemplatesFS。
func WithFS(f fs.FS) Option {
	return func(e *Engine) { e.fs = f }
}

// WithFuncMap 追加 / 覆盖模板函数（合并到 DefaultFuncMap 之上）。
func WithFuncMap(extra texttemplate.FuncMap) Option {
	return func(e *Engine) {
		if e.funcMap == nil {
			e.funcMap = make(texttemplate.FuncMap, len(extra))
		}
		for k, v := range extra {
			e.funcMap[k] = v
		}
	}
}

// New 构造一个 Engine。
//
// 默认 fs 是嵌入的 TemplatesFS。返回 error 仅为接口稳定性预留（当前实现不会失败）。
func New(opts ...Option) (*Engine, error) {
	e := &Engine{
		fs:      TemplatesFS,
		funcMap: DefaultFuncMap(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e, nil
}

// Render 渲染指定模板文件。返回原始字节（未格式化）。
//
// tplPath 为相对 fs 根的路径，如 "templates/component/webserver/main.go.tpl"。
//
// 路径分支语义（Phase 1 引入的"伪模板"优化）：
//   - 以 .tpl 结尾：走 text/template 渲染，按 missingkey=error 严格模式执行
//   - 不以 .tpl 结尾：走 raw copy，直接返回 fs 中的原始字节，不解析任何 {{ }}
//
// 引入 raw copy 分支的动机：约 2/3 的内置模板从未真正用到 {{ }} 模板语法
// （是从上游 vendored 过来的 .go 源码），把它们的 .tpl 后缀去掉后既能让
// IDE 正常高亮、又能跳过昂贵的 text/template Parse + Execute 流程。
//
// 失败时返回 *RenderError 包装的错误，可通过 errors.As 提取上下文。
func (e *Engine) Render(tplPath string, data any) ([]byte, error) {
	if e == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "engine is nil")
	}
	if !strings.HasSuffix(tplPath, ".tpl") {
		content, err := fs.ReadFile(e.fs, tplPath)
		if err != nil {
			return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
				"read raw template %s", tplPath)
		}
		return content, nil
	}
	tpl, err := e.lookupOrParse(tplPath)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		line, col := extractRenderPos(err)
		return nil, &RenderError{
			Template: tplPath,
			Err:      err,
			Data:     data,
			Snippet:  snippetAround(err),
			Line:     line,
			Col:      col,
		}
	}
	return buf.Bytes(), nil
}

// RenderToString 是 Render 的 string 版本，便于测试断言。
func (e *Engine) RenderToString(tplPath string, data any) (string, error) {
	out, err := e.Render(tplPath, data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Format 用 go/format 格式化 Go 源码。失败时返回原内容 + 错误（不破坏原有内容）。
//
// 注意：MVP 阶段使用标准库 go/format；Phase 2+ 可切换到 gofumpt（需引入 gofumpt API 或外部命令）。
func (e *Engine) Format(content []byte, ext string) ([]byte, error) {
	if !strings.EqualFold(ext, ".go") {
		return content, nil
	}
	formatted, err := format.Source(content)
	if err != nil {
		return content, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err, "go format")
	}
	return formatted, nil
}

// lookupOrParse 命中缓存返回；否则 Parse + 缓存。
//
// 每个业务模板独立 Parse 出独立的 *Template，互不干扰；missingkey=error 严格模式开启。
func (e *Engine) lookupOrParse(tplPath string) (*texttemplate.Template, error) {
	if cached, ok := e.parsedCache.Load(tplPath); ok {
		return cached.(*texttemplate.Template), nil
	}
	content, err := fs.ReadFile(e.fs, tplPath)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
			"read template %s", tplPath)
	}
	parsed, err := texttemplate.New(filepath.Base(tplPath)).
		Funcs(e.funcMap).
		Option("missingkey=error").
		Parse(string(content))
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
			"parse template %s", tplPath)
	}
	// 缓存可能被并发写入；sync.Map 自带去重，多次写入是无副作用的（参数相同）
	e.parsedCache.Store(tplPath, parsed)
	return parsed, nil
}

// snippetAround 从 text/template 错误信息提取「<line>:<col>」片段（仅展示用）。
func snippetAround(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, " executing "); i > 0 {
		return msg[i:]
	}
	return fmt.Sprintf("%v", err)
}
