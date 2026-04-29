package fs_test

import (
	"strings"
	"testing"

	lfs "github.com/clin211/linctl/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashContent_Deterministic(t *testing.T) {
	a := lfs.HashContent([]byte("hello"))
	b := lfs.HashContent([]byte("hello"))
	c := lfs.HashContent([]byte("world"))
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Len(t, a, 64) // sha256 hex = 64 chars
}

func TestShortHash(t *testing.T) {
	h := lfs.ShortHash([]byte("hello"))
	assert.Len(t, h, 12)
}

func TestAppendExtractHashComment_Go(t *testing.T) {
	src := []byte("package main\n")
	hash := lfs.HashContent(src)

	with := lfs.AppendHashComment(src, hash, ".go")
	assert.Contains(t, string(with), "// linctl: hash="+hash)

	got, ok := lfs.ExtractHashComment(with)
	require.True(t, ok)
	assert.Equal(t, hash, got)
}

func TestAppendExtractHashComment_YAML(t *testing.T) {
	src := []byte("name: x\n")
	hash := lfs.HashContent(src)

	with := lfs.AppendHashComment(src, hash, ".yaml")
	assert.Contains(t, string(with), "# linctl: hash="+hash)

	got, ok := lfs.ExtractHashComment(with)
	require.True(t, ok)
	assert.Equal(t, hash, got)
}

func TestAppendExtractHashComment_HTML(t *testing.T) {
	src := []byte("<html></html>\n")
	hash := lfs.HashContent(src)

	with := lfs.AppendHashComment(src, hash, ".html")
	assert.Contains(t, string(with), "<!-- linctl: hash="+hash+" -->")

	got, ok := lfs.ExtractHashComment(with)
	require.True(t, ok)
	assert.Equal(t, hash, got)
}

func TestAppendHashComment_UnsupportedExt(t *testing.T) {
	src := []byte{0xde, 0xad, 0xbe, 0xef}
	out := lfs.AppendHashComment(src, "abc", ".png")
	assert.Equal(t, src, out, "binary file should be untouched")
}

func TestAppendHashComment_EnsureTrailingNewline(t *testing.T) {
	// 没有结尾换行的情况
	src := []byte("package main")
	hash := lfs.HashContent(src)
	out := lfs.AppendHashComment(src, hash, ".go")
	assert.True(t, strings.HasSuffix(string(out), "\n"))
}

func TestExtractHashComment_NoMatch(t *testing.T) {
	_, ok := lfs.ExtractHashComment([]byte("nothing here"))
	assert.False(t, ok)
}

func TestStripHashComment(t *testing.T) {
	src := []byte("package main\n\n// some comment\nfunc x() {}\n")
	hash := lfs.HashContent(src)
	with := lfs.AppendHashComment(src, hash, ".go")

	stripped := lfs.StripHashComment(with)
	_, ok := lfs.ExtractHashComment(stripped)
	assert.False(t, ok)
	assert.Contains(t, string(stripped), "func x() {}")
	assert.Contains(t, string(stripped), "// some comment")
}

func TestStripHashComment_NoMatchReturnsOriginal(t *testing.T) {
	src := []byte("plain content")
	out := lfs.StripHashComment(src)
	assert.Equal(t, src, out)
}
