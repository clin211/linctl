package gitmerge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/clin211/lin/internal/linctlerr"
)

// Files 是 3-way merge 的三个版本。
//
// 字段语义（与 git merge-file 一致）：
//   - Base：共同祖先版本（例如上次同步时的内容）
//   - Ours：当前侧版本（在 templatesync 场景下是 lin/templates/web-gin/ 现状）
//   - Theirs：另一侧版本（在 templatesync 场景下是新模板渲染产物）
//
// 注意：Base 不应为 nil；如果没有 base 信息（如首次同步且文件已存在），
// 调用方应改用 lifecycle 文档 §4.2 的 "userTouched=false → Update" 路径，
// 不进入 3-way merge。
type Files struct {
	Base   []byte
	Ours   []byte
	Theirs []byte
	Labels Labels
}

// Labels 控制 git merge-file 写入的 marker 标签（默认无 label）。
type Labels struct {
	Ours   string
	Base   string
	Theirs string
}

// Result 是 merge 的输出。
type Result struct {
	Content     []byte // 合并后的内容（可能含 conflict markers）
	HasConflict bool   // git merge-file 退出码 > 0 时为 true
	Backend     string // "git" / "diff3-fallback"
}

// Strategy 控制 ours/theirs 偏好（同 git merge-file --ours/--theirs）。
type Strategy string

const (
	// StrategyAuto：纯 3-way merge；冲突时输出 markers。
	StrategyAuto Strategy = ""
	// StrategyOurs：冲突 hunk 全部保留 ours。
	StrategyOurs Strategy = "ours"
	// StrategyTheirs：冲突 hunk 全部保留 theirs。
	StrategyTheirs Strategy = "theirs"
	// StrategyUnion：冲突 hunk 保留双方（git --union）。
	StrategyUnion Strategy = "union"
)

// MergeFile 执行 3-way merge。优先调用系统 `git merge-file`；如果不可用回退到内置 diff3。
//
// HasConflict=true 时：
//   - StrategyAuto：Content 含 conflict markers
//   - StrategyOurs/Theirs/Union：Content 不含 markers（git 已自动取舍）
func MergeFile(ctx context.Context, f Files, strat Strategy) (*Result, error) {
	if f.Base == nil || f.Ours == nil || f.Theirs == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal,
			"gitmerge.MergeFile: Base/Ours/Theirs cannot be nil (use empty []byte{} explicitly if intentional)")
	}

	// 优先 git
	res, err := mergeWithGit(ctx, f, strat)
	if err == nil {
		return res, nil
	}
	if !errors.Is(err, errGitUnavailable) {
		return nil, err
	}
	// fallback
	return mergeWithFallback(f, strat)
}

// errGitUnavailable 是 mergeWithGit 在系统无 git 时返回的哨兵错误。
var errGitUnavailable = errors.New("system git not available")

// mergeWithGit 调用 `git merge-file` 命令。
func mergeWithGit(ctx context.Context, f Files, strat Strategy) (*Result, error) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return nil, errGitUnavailable
	}

	// git merge-file 要求三个真实文件路径，写到 temp dir
	tmpDir, err := os.MkdirTemp("", "linctl-gitmerge-*")
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"create temp dir for git merge-file")
	}
	defer os.RemoveAll(tmpDir)

	type tmpFile struct {
		name    string
		path    string
		content []byte
	}
	files := []tmpFile{
		{name: "ours", path: filepath.Join(tmpDir, "ours"), content: f.Ours},
		{name: "base", path: filepath.Join(tmpDir, "base"), content: f.Base},
		{name: "theirs", path: filepath.Join(tmpDir, "theirs"), content: f.Theirs},
	}
	for _, tf := range files {
		if err := os.WriteFile(tf.path, tf.content, 0o644); err != nil {
			return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"write tmp %s for git merge-file", tf.name)
		}
	}

	args := []string{"merge-file", "-p", "--diff3", "--marker-size=7"}
	switch strat {
	case StrategyOurs:
		args = append(args, "--ours")
	case StrategyTheirs:
		args = append(args, "--theirs")
	case StrategyUnion:
		args = append(args, "--union")
	}
	if f.Labels.Ours != "" {
		args = append(args, "-L", f.Labels.Ours)
	}
	if f.Labels.Base != "" {
		args = append(args, "-L", f.Labels.Base)
	}
	if f.Labels.Theirs != "" {
		args = append(args, "-L", f.Labels.Theirs)
	}
	args = append(args, files[0].path, files[1].path, files[2].path)

	cmd := exec.CommandContext(ctx, gitBin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	hasConflict := false
	if err != nil {
		// git merge-file 在有冲突时返回非零退出码（具体值 = 冲突 hunk 数，>=1）。
		// errors.As 拿到 ExitError 后区分"真错误"vs"仅冲突"。
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			if ee.ExitCode() > 0 {
				hasConflict = true
			} else {
				// 极少：git 未启动成功
				return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"git merge-file failed: %s", stderr.String())
			}
		} else {
			return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"git merge-file: %s", stderr.String())
		}
	}

	return &Result{
		Content:     stdout.Bytes(),
		HasConflict: hasConflict,
		Backend:     "git",
	}, nil
}

// HasConflictMarkers 检测内容是否含 git 冲突标记（用于 resolve 命令验证）。
func HasConflictMarkers(content []byte) bool {
	return bytes.Contains(content, []byte("\n<<<<<<<")) ||
		bytes.HasPrefix(content, []byte("<<<<<<<"))
}

// IsAvailable 判断当前系统是否可调用 git merge-file。
func IsAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// _ 抑制 fmt 未使用警告（保留 import 以便未来扩展 verbose 输出）。
var _ = fmt.Sprintf
