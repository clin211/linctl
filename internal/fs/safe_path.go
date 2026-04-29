package fs

import (
	"path/filepath"
	"strings"

	"github.com/clin211/lin/internal/linctlerr"
)

// SafeJoin 把 rel 拼到 rootDir 下，并校验结果仍在 rootDir 子树内。
//
// 防御目标（详见 docs/15-security-model.md §15.2.2 路径越界）：
//   - "../../etc/passwd" 类相对路径越界
//   - 绝对路径（必须为相对）
//   - 包含 NUL 字节
//
// 返回清理后的绝对路径，或 ErrUnsafePath 错误。
func SafeJoin(rootDir, rel string) (string, error) {
	if rootDir == "" {
		return "", linctlerr.New(linctlerr.ErrConfigInvalid,
			"SafeJoin: rootDir is empty",
			"Provide an absolute project root directory")
	}
	if filepath.IsAbs(rel) {
		return "", linctlerr.Newf(linctlerr.ErrUnsafePath,
			"path must be relative: %s", rel)
	}
	if strings.ContainsRune(rel, '\x00') {
		return "", linctlerr.New(linctlerr.ErrUnsafePath,
			"path contains NUL byte")
	}

	cleanRoot := filepath.Clean(rootDir)
	candidate := filepath.Clean(filepath.Join(cleanRoot, rel))

	rootWithSep := cleanRoot
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}

	if candidate != cleanRoot && !strings.HasPrefix(candidate, rootWithSep) {
		return "", linctlerr.Newf(linctlerr.ErrUnsafePath,
			"path escapes root: %s -> %s", rel, candidate).
			WithHint("Use a relative path that stays inside the project root.")
	}
	return candidate, nil
}
