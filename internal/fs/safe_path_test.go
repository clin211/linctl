package fs_test

import (
	"runtime"
	"testing"

	lfs "github.com/clin211/linctl/internal/fs"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeJoin_HappyPath(t *testing.T) {
	root := mustRoot(t)
	cases := []string{
		"a.go",
		"sub/dir/file.txt",
		"./relative/x",
		".",
	}
	for _, rel := range cases {
		out, err := lfs.SafeJoin(root, rel)
		require.NoError(t, err, "rel=%s", rel)
		assert.Contains(t, out, root, "rel=%s", rel)
	}
}

func TestSafeJoin_RejectsEscape(t *testing.T) {
	root := mustRoot(t)
	cases := []string{
		"../etc/passwd",
		"sub/../../bad",
		"a/b/../../../../tmp",
	}
	for _, rel := range cases {
		_, err := lfs.SafeJoin(root, rel)
		require.Error(t, err, "rel=%s should escape", rel)
		code, _ := linctlerr.CodeOf(err)
		assert.Equal(t, linctlerr.ErrUnsafePath, code, "rel=%s", rel)
	}
}

func TestSafeJoin_RejectsAbsolute(t *testing.T) {
	root := mustRoot(t)
	abs := "/etc/passwd"
	if runtime.GOOS == "windows" {
		abs = `C:\Windows`
	}
	_, err := lfs.SafeJoin(root, abs)
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrUnsafePath, code)
}

func TestSafeJoin_RejectsNUL(t *testing.T) {
	root := mustRoot(t)
	_, err := lfs.SafeJoin(root, "x\x00y")
	require.Error(t, err)
}

func TestSafeJoin_EmptyRoot(t *testing.T) {
	_, err := lfs.SafeJoin("", "a.go")
	require.Error(t, err)
}

func mustRoot(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return `C:\proj`
	}
	return "/proj"
}
