// Package fsx 提供 lin v2 的安全文件 IO 工具。
//
// 设计来源：lin/docs/features/04-template-system.md §10.1「路径穿越（Path Traversal）防护」。
//
// 核心约束：
//   - 所有写入磁盘的路径必须经过 SafeJoin 校验，杜绝路径穿越
//   - 模板渲染产物路径若跳出 RootDir，立即报错（CodeTplPathTraversal）
package fsx

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/clin211/linctl/internal/pkg/errs"
)

// SafeJoin 把 root 与 rel 安全拼接成绝对路径，并校验最终路径不会跳出 root。
//
// 返回的绝对路径已规范化（filepath.Abs + filepath.Clean）。
// 如果拼接结果跳出 root，返回 errs.Error{Code: CodeTplPathTraversal}。
//
// 用法示例：
//
//	abs, err := fsx.SafeJoin("/repo/myblog", "internal/myblog/handler/post.go")
//	// abs == "/repo/myblog/internal/myblog/handler/post.go"
//
//	_, err := fsx.SafeJoin("/repo/myblog", "../../etc/passwd")
//	// err.Code == errs.CodeTplPathTraversal
func SafeJoin(root, rel string) (string, error) {
	if root == "" {
		return "", errs.New(errs.CodeInvalidArg, "fsx.SafeJoin: empty root")
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", errs.Wrap(errs.CodeUnknown, "fsx.SafeJoin: abs root", err)
	}

	abs, err := filepath.Abs(filepath.Join(rootAbs, rel))
	if err != nil {
		return "", errs.Wrap(errs.CodeUnknown, "fsx.SafeJoin: abs join", err)
	}

	if abs != rootAbs && !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) {
		return "", errs.New(
			errs.CodeTplPathTraversal,
			fmt.Sprintf("fsx.SafeJoin: path %q escapes root %q", rel, root),
		).WithHint("relative paths must not contain '..' components that traverse out of root")
	}

	return abs, nil
}

// IsWithin 报告 child 是否位于 root 之内（基于绝对路径前缀比较）。
// root 与 child 都会先 filepath.Abs。
func IsWithin(root, child string) (bool, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false, err
	}
	if childAbs == rootAbs {
		return true, nil
	}
	return strings.HasPrefix(childAbs, rootAbs+string(filepath.Separator)), nil
}
