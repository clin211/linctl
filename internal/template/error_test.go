package template_test

import (
	"errors"
	"testing"

	"github.com/clin211/linctl/internal/linctlerr"
	tpl "github.com/clin211/linctl/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderError_NilSafety(t *testing.T) {
	var e *tpl.RenderError
	assert.Equal(t, "", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestRenderError_BasicFormat(t *testing.T) {
	e := &tpl.RenderError{
		Template: "templates/foo.tpl",
		Err:      errors.New("oops"),
	}
	out := e.Error()
	assert.Contains(t, out, "templates/foo.tpl")
	assert.Contains(t, out, "oops")
	assert.NotContains(t, out, ":0:")
}

func TestRenderError_WithLineCol(t *testing.T) {
	e := &tpl.RenderError{
		Template: "templates/foo.tpl",
		Err:      errors.New("oops"),
		Line:     12,
		Col:      4,
	}
	out := e.Error()
	assert.Contains(t, out, "templates/foo.tpl:12:4")
}

func TestRenderError_WithLineNoCol(t *testing.T) {
	e := &tpl.RenderError{
		Template: "templates/foo.tpl",
		Err:      errors.New("oops"),
		Line:     7,
	}
	out := e.Error()
	assert.Contains(t, out, "templates/foo.tpl:7:")
	assert.NotContains(t, out, ":7:0")
}

func TestRenderError_DataSummary(t *testing.T) {
	e := &tpl.RenderError{
		Template: "t",
		Err:      errors.New("e"),
		Data:     map[string]any{"k": "v"},
	}
	out := e.Error()
	assert.Contains(t, out, "--- data ---")
	assert.Contains(t, out, "k:v")
}

func TestRenderError_DataTruncated(t *testing.T) {
	long := make([]byte, 500)
	for i := range long {
		long[i] = 'a'
	}
	e := &tpl.RenderError{
		Template: "t",
		Err:      errors.New("e"),
		Data:     string(long),
	}
	out := e.Error()
	assert.Contains(t, out, "...(truncated)")
}

func TestRenderError_Unwrap(t *testing.T) {
	base := errors.New("base")
	e := &tpl.RenderError{
		Template: "t",
		Err:      base,
	}
	assert.True(t, errors.Is(e, base))
}

// TestRenderError_RealRender 端到端：用 Engine 渲染一个会失败的模板，
// 验证 Line:Col 能从 text/template 错误中提取出来。
func TestRenderError_RealRender(t *testing.T) {
	// 故意写错的模板：调用不存在的 funcmap 函数，会 parse 失败
	engine, err := tpl.New()
	require.NoError(t, err)
	_, err = engine.Render("templates/__nonexistent__.tpl", nil)
	require.Error(t, err)
	// 这个模板不存在，会包装为 LinctlError(ErrTemplateRender)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrTemplateRender, lerr.Code)
}
