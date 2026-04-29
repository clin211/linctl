package template_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	tpl "github.com/clin211/linctl/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEngineWithFS(t *testing.T, files map[string]string) *tpl.Engine {
	t.Helper()
	fs := fstest.MapFS{}
	for path, content := range files {
		fs[path] = &fstest.MapFile{Data: []byte(content)}
	}
	e, err := tpl.New(tpl.WithFS(fs))
	require.NoError(t, err)
	return e
}

func TestRender_Basic(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{
		"templates/main.go.tpl": "package main\n\nfunc main() { println({{ .Greeting | printf \"%q\" }}) }\n",
	})

	out, err := e.RenderToString("templates/main.go.tpl", map[string]any{
		"Greeting": "hello",
	})
	require.NoError(t, err)
	assert.Contains(t, out, `println("hello")`)
}

func TestRender_MissingKeyIsError(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{
		"templates/x.tpl": "{{ .Missing }}",
	})

	_, err := e.RenderToString("templates/x.tpl", map[string]any{})
	require.Error(t, err)

	var rerr *tpl.RenderError
	require.True(t, errors.As(err, &rerr))
	assert.Equal(t, "templates/x.tpl", rerr.Template)
}

func TestRender_TemplateNotFound(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{})
	_, err := e.RenderToString("templates/missing.tpl", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing.tpl")
}

func TestRender_Concurrent(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{
		"templates/a.tpl": "A:{{ .V }}",
		"templates/b.tpl": "B:{{ .V }}",
	})

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			out, err := e.RenderToString("templates/a.tpl", map[string]any{"V": "x"})
			assert.NoError(t, err)
			assert.Equal(t, "A:x", out)
		}()
		go func() {
			defer wg.Done()
			out, err := e.RenderToString("templates/b.tpl", map[string]any{"V": "y"})
			assert.NoError(t, err)
			assert.Equal(t, "B:y", out)
		}()
	}
	wg.Wait()
}

func TestFormat_GoFile(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{})
	src := []byte("package main\nfunc main() {println(\"x\")}\n")
	out, err := e.Format(src, ".go")
	require.NoError(t, err)
	assert.Contains(t, string(out), "func main()")
}

func TestFormat_NonGoNoop(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{})
	src := []byte("any text")
	out, err := e.Format(src, ".txt")
	require.NoError(t, err)
	assert.Equal(t, "any text", string(out))
}

func TestFormat_BadGoSourceReturnsOriginal(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{})
	src := []byte("package not_go {{")
	out, err := e.Format(src, ".go")
	require.Error(t, err)
	assert.Equal(t, "package not_go {{", string(out)) // 原内容保持
}

func TestRenderError_Error(t *testing.T) {
	e := &tpl.RenderError{
		Template: "x.tpl",
		Err:      errors.New("oops"),
		Snippet:  "snippet",
	}
	msg := e.Error()
	assert.Contains(t, msg, "x.tpl")
	assert.Contains(t, msg, "oops")
	assert.Contains(t, msg, "snippet")
}

func TestEmptyFSStillInitializes(t *testing.T) {
	e, err := tpl.New(tpl.WithFS(fstest.MapFS{}))
	require.NoError(t, err)
	require.NotNil(t, e)
}

func TestNewWithCustomFuncMap(t *testing.T) {
	e, err := tpl.New(
		tpl.WithFS(fstest.MapFS{
			"templates/x.tpl": &fstest.MapFile{Data: []byte("{{ shout .Word }}")},
		}),
		tpl.WithFuncMap(map[string]any{
			"shout": func(s string) string { return strings.ToUpper(s) + "!" },
		}),
	)
	require.NoError(t, err)
	out, err := e.RenderToString("templates/x.tpl", map[string]any{"Word": "hi"})
	require.NoError(t, err)
	assert.Equal(t, "HI!", out)
}

// TestRender_RawCopy_NonTpl 验证 Phase 1 引入的 raw copy 分支：tplPath 不以
// .tpl 结尾时，引擎直接返回 fs 中的原始字节，不经过 text/template 解析。
//
// 这是"伪模板"优化的回归保护：哪怕文件内容长得像模板（例如包含 {{ 字面量），
// 只要 tplPath 没有 .tpl 后缀就一律 raw copy，不会触发渲染逻辑。
func TestRender_RawCopy_NonTpl(t *testing.T) {
	raw := "package ptr\n\n// {{ this is not a template }}\nfunc To[T any](v T) *T { return &v }\n"
	e := newEngineWithFS(t, map[string]string{
		"_templates/web-gin/pkg/ptr/ptr.go": raw,
	})

	out, err := e.RenderToString("_templates/web-gin/pkg/ptr/ptr.go", map[string]any{
		"Anything": "ignored",
	})
	require.NoError(t, err)
	assert.Equal(t, raw, out)
}

// TestRender_RawCopy_NotFound 验证 raw copy 分支对不存在文件的错误语义。
func TestRender_RawCopy_NotFound(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{})
	_, err := e.RenderToString("_templates/missing.go", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing.go")
}

// TestRender_TplVsRawDispatch 同名同内容（去掉 .tpl）两份文件，验证 .tpl 走
// 渲染、非 .tpl 走 raw copy，两条路径互不串扰。
func TestRender_TplVsRawDispatch(t *testing.T) {
	e := newEngineWithFS(t, map[string]string{
		"_templates/foo.go.tpl": "package foo\n// hi {{ .Who }}\n",
		"_templates/foo.go":     "package foo\n// hi {{ .Who }}\n",
	})

	rendered, err := e.RenderToString("_templates/foo.go.tpl", map[string]any{"Who": "world"})
	require.NoError(t, err)
	assert.Contains(t, rendered, "// hi world")
	assert.NotContains(t, rendered, "{{")

	raw, err := e.RenderToString("_templates/foo.go", map[string]any{"Who": "world"})
	require.NoError(t, err)
	assert.Contains(t, raw, "{{ .Who }}")
}
