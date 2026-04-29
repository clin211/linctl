package linctlerr_test

import (
	"errors"
	"testing"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "bad config", "fix it")
	require.NotNil(t, err)
	assert.Equal(t, linctlerr.ErrConfigInvalid, err.Code)
	assert.Equal(t, "bad config", err.Message)
	assert.Equal(t, "fix it", err.Hint)
	assert.Nil(t, err.Cause)
	assert.Contains(t, err.Error(), "[config_invalid]")
	assert.Contains(t, err.Error(), "bad config")
}

func TestNewMultipleHints(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "msg", "hint1", "hint2")
	assert.Equal(t, "hint1\nhint2", err.Hint)
}

func TestNewf(t *testing.T) {
	err := linctlerr.Newf(linctlerr.ErrConfigInvalid, "field %q invalid: %d", "port", 0)
	assert.Equal(t, `field "port" invalid: 0`, err.Message)
}

func TestWrap(t *testing.T) {
	cause := errors.New("io error")
	err := linctlerr.Wrap(linctlerr.ErrEnvironment, cause, "read file")
	require.NotNil(t, err)
	assert.Equal(t, linctlerr.ErrEnvironment, err.Code)
	assert.Equal(t, "read file", err.Message)
	assert.Same(t, cause, err.Cause)
	assert.Contains(t, err.Error(), "io error")
}

func TestWrapNilReturnsNil(t *testing.T) {
	assert.Nil(t, linctlerr.Wrap(linctlerr.ErrEnvironment, nil, "msg"))
	assert.Nil(t, linctlerr.Wrapf(linctlerr.ErrEnvironment, nil, "%s", "x"))
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("base")
	err := linctlerr.Wrap(linctlerr.ErrEnvironment, cause, "wrap")
	assert.True(t, errors.Is(err, cause))
}

func TestIsByCode(t *testing.T) {
	a := linctlerr.New(linctlerr.ErrConfigInvalid, "a")
	b := linctlerr.New(linctlerr.ErrConfigInvalid, "b")
	c := linctlerr.New(linctlerr.ErrEnvironment, "c")

	assert.True(t, errors.Is(a, b), "same code should be Is-equal")
	assert.False(t, errors.Is(a, c), "different code should not be Is-equal")
}

func TestErrorsAs(t *testing.T) {
	cause := errors.New("base")
	err := linctlerr.Wrap(linctlerr.ErrEnvironment, cause, "wrap")

	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrEnvironment, lerr.Code)
}

func TestWithHint(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "msg", "h1")
	err2 := err.WithHint("h2")

	assert.Equal(t, "h1", err.Hint, "original should be unchanged (immutable)")
	assert.Equal(t, "h1\nh2", err2.Hint)
}

func TestCodeOf(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "msg")
	wrapped := errors.New("plain")

	code, ok := linctlerr.CodeOf(err)
	assert.True(t, ok)
	assert.Equal(t, linctlerr.ErrConfigInvalid, code)

	_, ok = linctlerr.CodeOf(wrapped)
	assert.False(t, ok)

	_, ok = linctlerr.CodeOf(nil)
	assert.False(t, ok)
}

func TestNilSafety(t *testing.T) {
	var e *linctlerr.LinctlError
	assert.Equal(t, "", e.Error())
	assert.Nil(t, e.Unwrap())
	assert.False(t, e.Is(linctlerr.New(linctlerr.ErrConfigInvalid, "x")))
	assert.Nil(t, e.WithHint("x"))
	assert.Nil(t, e.WithDocLink("x"))
	assert.Equal(t, "", e.Pretty(true))
}

func TestWithDocLink(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "msg")
	err2 := err.WithDocLink("https://example.com/docs")

	assert.Equal(t, "", err.DocLink, "original should be unchanged (immutable)")
	assert.Equal(t, "https://example.com/docs", err2.DocLink)
}

func TestPretty_NoColor(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "bad config", "fix the value").
		WithDocLink("https://example.com/docs/cfg")

	out := err.Pretty(true)
	assert.Contains(t, out, "Error:")
	assert.Contains(t, out, "[config_invalid]")
	assert.Contains(t, out, "bad config")
	assert.Contains(t, out, "Hint:")
	assert.Contains(t, out, "fix the value")
	assert.Contains(t, out, "Doc:")
	assert.Contains(t, out, "https://example.com/docs/cfg")
	// 不含 ANSI 控制字符
	assert.NotContains(t, out, "\x1b[")
}

func TestPretty_WithColor(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "bad")
	out := err.Pretty(false)
	// 带 ANSI 颜色
	assert.Contains(t, out, "\x1b[31m")
	assert.Contains(t, out, "\x1b[0m")
}

func TestPretty_NoHintNoDoc(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "bad")
	out := err.Pretty(true)
	assert.Contains(t, out, "[config_invalid]")
	assert.NotContains(t, out, "Hint:")
	assert.NotContains(t, out, "Doc:")
}

func TestPretty_MultilineHint(t *testing.T) {
	err := linctlerr.New(linctlerr.ErrConfigInvalid, "bad", "line1", "line2")
	out := err.Pretty(true)
	// 多行 hint：每行单独一个 "Hint:" 前缀
	assert.Contains(t, out, "line1")
	assert.Contains(t, out, "line2")
}

// TestErrorChain_DeepUnwrap 验证 LinctlError 包装层叠后仍可 Unwrap 到底层 error。
func TestErrorChain_DeepUnwrap(t *testing.T) {
	base := errors.New("base io error")
	level1 := linctlerr.Wrap(linctlerr.ErrEnvironment, base, "read file")
	level2 := linctlerr.Wrap(linctlerr.ErrInternal, level1, "load config")

	// errors.Is 沿整条链传递
	assert.True(t, errors.Is(level2, base), "deep Unwrap must reach base error")

	// errors.As 提取最近的 LinctlError
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(level2, &lerr))
	assert.Equal(t, linctlerr.ErrInternal, lerr.Code)
}
