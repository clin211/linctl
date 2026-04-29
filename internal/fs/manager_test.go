package fs_test

import (
	"errors"
	stdfs "io/fs"
	"os"
	"sort"
	"testing"

	lfs "github.com/clin211/linctl/internal/fs"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMemManager(t *testing.T) (*lfs.FileManager, afero.Fs) {
	t.Helper()
	memFs := afero.NewMemMapFs()
	require.NoError(t, memFs.MkdirAll("/proj", 0o755))
	m, err := lfs.NewFileManager(lfs.Options{FS: memFs, RootDir: "/proj"})
	require.NoError(t, err)
	return m, memFs
}

func TestNewFileManager_RequiresRootDir(t *testing.T) {
	_, err := lfs.NewFileManager(lfs.Options{})
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrConfigInvalid, code)
}

func TestAtomicWrite_HappyPath(t *testing.T) {
	m, memFs := newMemManager(t)

	require.NoError(t, m.AtomicWrite("a/b/file.txt", []byte("hello"), 0o644))

	content, err := afero.ReadFile(memFs, "/proj/a/b/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	// .tmp 不应残留
	exists, _ := afero.Exists(memFs, "/proj/a/b/file.txt.tmp")
	assert.False(t, exists)
}

func TestAtomicWrite_RejectsEscapingPath(t *testing.T) {
	m, _ := newMemManager(t)
	err := m.AtomicWrite("../../etc/passwd", []byte("x"), 0o600)
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrUnsafePath, code)
}

func TestExistsRead(t *testing.T) {
	m, memFs := newMemManager(t)

	require.NoError(t, afero.WriteFile(memFs, "/proj/foo.txt", []byte("bar"), 0o644))
	assert.True(t, m.Exists("foo.txt"))
	assert.False(t, m.Exists("missing.txt"))

	data, err := m.Read("foo.txt")
	require.NoError(t, err)
	assert.Equal(t, "bar", string(data))
}

func TestRemove_Idempotent(t *testing.T) {
	m, memFs := newMemManager(t)
	require.NoError(t, afero.WriteFile(memFs, "/proj/x.txt", []byte("y"), 0o644))

	require.NoError(t, m.Remove("x.txt"))
	require.NoError(t, m.Remove("x.txt")) // second call ok
}

func TestWalk_SkipsBlackList(t *testing.T) {
	m, memFs := newMemManager(t)

	require.NoError(t, afero.WriteFile(memFs, "/proj/keep.txt", []byte("a"), 0o644))
	require.NoError(t, afero.WriteFile(memFs, "/proj/.git/HEAD", []byte("b"), 0o644))
	require.NoError(t, afero.WriteFile(memFs, "/proj/_output/bin/x", []byte("c"), 0o644))
	require.NoError(t, afero.WriteFile(memFs, "/proj/.linctl/lock.yaml", []byte("d"), 0o644))

	var visited []string
	err := m.Walk(func(p string, d stdfs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			visited = append(visited, p)
		}
		return nil
	})
	require.NoError(t, err)

	sort.Strings(visited)
	assert.Equal(t, []string{"keep.txt"}, visited)
}

func TestStat(t *testing.T) {
	m, memFs := newMemManager(t)
	require.NoError(t, afero.WriteFile(memFs, "/proj/x.txt", []byte("hi"), 0o644))

	fi, err := m.Stat("x.txt")
	require.NoError(t, err)
	assert.Equal(t, int64(2), fi.Size())

	_, err = m.Stat("nope.txt")
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}
