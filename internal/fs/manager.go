// Package fs 提供 linctl 的文件系统抽象层。
//
// 核心特性（详见 docs/06-codegen-pipeline.md）：
//   - 基于 spf13/afero，便于单元测试用 MemMapFs 注入
//   - 原子写（写到 .tmp 再 rename，保证中断时无半成品）
//   - 路径强制 SafeJoin（防止 ../../etc/passwd 类越界）
//   - hash 注释附加 / 提取（按文件类型选注释格式）
//   - 项目级 flock（详见 lock.go 与 SSOT §5.5）
package fs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/spf13/afero"
)

// FileManager 是 linctl 文件操作的统一入口。
//
// 所有路径都相对 RootDir；外部传入的相对路径都会经过 SafeJoin 校验。
type FileManager struct {
	fs      afero.Fs
	rootDir string
}

// Options 是 FileManager 构造选项。
type Options struct {
	// FS 是底层文件系统，默认 afero.OsFs。测试时可注入 afero.NewMemMapFs()。
	FS afero.Fs

	// RootDir 是项目根目录绝对路径。所有相对路径都基于此。
	RootDir string
}

// NewFileManager 构造一个 FileManager。RootDir 为空或不存在时返回错误。
func NewFileManager(opts Options) (*FileManager, error) {
	if opts.RootDir == "" {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"FileManager: RootDir is empty",
			"Pass an absolute path of the project root directory")
	}
	if opts.FS == nil {
		opts.FS = afero.NewOsFs()
	}
	abs, err := filepath.Abs(opts.RootDir)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"FileManager: resolve root %s", opts.RootDir)
	}
	return &FileManager{fs: opts.FS, rootDir: abs}, nil
}

// RootDir 返回 FileManager 持有的根目录绝对路径。
func (m *FileManager) RootDir() string {
	return m.rootDir
}

// FS 返回底层 afero.Fs（仅用于高级场景，例如测试和插件实现）。
func (m *FileManager) FS() afero.Fs {
	return m.fs
}

// Read 读取相对路径对应的文件内容。
func (m *FileManager) Read(rel string) ([]byte, error) {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return nil, err
	}
	data, err := afero.ReadFile(m.fs, abs)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "read %s", rel)
	}
	return data, nil
}

// Exists 判断相对路径下文件是否存在。错误（如权限）被视为不存在。
func (m *FileManager) Exists(rel string) bool {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return false
	}
	exists, _ := afero.Exists(m.fs, abs)
	return exists
}

// Stat 返回文件 / 目录的 FileInfo。
func (m *FileManager) Stat(rel string) (os.FileInfo, error) {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return nil, err
	}
	fi, err := m.fs.Stat(abs)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "stat %s", rel)
	}
	return fi, nil
}

// MkdirAll 递归创建目录。
func (m *FileManager) MkdirAll(rel string, perm os.FileMode) error {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return err
	}
	if err := m.fs.MkdirAll(abs, perm); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "mkdir %s", rel)
	}
	return nil
}

// AtomicWrite 原子地写入文件：先写到 <abs>.tmp，再 rename 到目标。
//
// 中断时（进程被杀、磁盘满等）保证不会出现半成品文件。
//
// perm 应用于最终文件；父目录必须已经存在（自行调用 MkdirAll）。
func (m *FileManager) AtomicWrite(rel string, content []byte, perm os.FileMode) error {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return err
	}

	dir := filepath.Dir(abs)
	if err := m.fs.MkdirAll(dir, 0o755); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "mkdir %s", filepath.Dir(rel))
	}

	tmpPath := abs + ".tmp"
	tmp, err := m.fs.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "open temp %s", rel)
	}

	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = m.fs.Remove(tmpPath)
		}
	}()

	if _, err := io.WriteString(tmp, string(content)); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "write temp %s", rel)
	}
	if syncer, ok := tmp.(interface{ Sync() error }); ok {
		_ = syncer.Sync()
	}
	if err := tmp.Close(); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "close temp %s", rel)
	}

	if err := m.fs.Rename(tmpPath, abs); err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "rename %s", rel)
	}
	committed = true
	return nil
}

// Walk 遍历项目目录，对每个文件 / 目录调用 walkFn。
//
// 自动跳过：.git/、_output/、.linctl/、node_modules/、vendor/。
// 调用方可在 walkFn 中返回 fs.SkipDir 跳过当前目录。
func (m *FileManager) Walk(walkFn fs.WalkDirFunc) error {
	return afero.Walk(m.fs, m.rootDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 计算相对路径
		rel, rerr := filepath.Rel(m.rootDir, p)
		if rerr != nil {
			return rerr
		}
		// 跳过常见无关目录
		if info.IsDir() && shouldSkipDir(rel) {
			return filepath.SkipDir
		}
		// 把 os.FileInfo 适配为 fs.DirEntry
		dirEntry := &fileInfoEntry{info: info}
		return walkFn(rel, dirEntry, nil)
	})
}

// 跳过的目录前缀。
var skipDirs = []string{
	".git",
	".linctl",
	"_output",
	"node_modules",
	"vendor",
	"dist",
	".idea",
	".vscode",
}

func shouldSkipDir(rel string) bool {
	if rel == "." {
		return false
	}
	first := rel
	if i := indexRune(rel, '/'); i >= 0 {
		first = rel[:i]
	}
	for _, s := range skipDirs {
		if first == s {
			return true
		}
	}
	return false
}

func indexRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// fileInfoEntry 把 os.FileInfo 适配成 fs.DirEntry。
type fileInfoEntry struct {
	info os.FileInfo
}

func (e *fileInfoEntry) Name() string               { return e.info.Name() }
func (e *fileInfoEntry) IsDir() bool                { return e.info.IsDir() }
func (e *fileInfoEntry) Type() os.FileMode          { return e.info.Mode().Type() }
func (e *fileInfoEntry) Info() (os.FileInfo, error) { return e.info, nil }

// Remove 删除文件。不存在时不报错。
func (m *FileManager) Remove(rel string) error {
	abs, err := SafeJoin(m.rootDir, rel)
	if err != nil {
		return err
	}
	if err := m.fs.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err, "remove %s", rel)
	}
	return nil
}
