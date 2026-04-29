package fsx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/clin211/lin/internal/pkg/errs"
)

// WriteFileAtomic 原子地写入 dst：
//  1. 写入 <dst>.<pid>.tmp
//  2. 成功后 rename 为 dst（同 FS 内的 rename 是原子操作）
//
// 自动 mkdir 父目录（0755）。
func WriteFileAtomic(dst string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errs.Wrap(errs.CodeWriteFailed, "fsx: mkdir parent", err)
	}

	tmp := fmt.Sprintf("%s.%d.tmp", dst, os.Getpid())
	if err := os.WriteFile(tmp, content, perm); err != nil {
		return errs.Wrap(errs.CodeWriteFailed, "fsx: write tmp", err)
	}

	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return errs.Wrap(errs.CodeWriteFailed, "fsx: rename tmp→dst", err)
	}
	return nil
}

// FileExists 报告 path 是否存在（且不是目录）。
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// DirExists 报告 path 是否存在且为目录。
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
