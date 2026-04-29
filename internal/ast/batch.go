package ast

import (
	"context"

	"github.com/clin211/linctl/internal/fs"
	"github.com/clin211/linctl/internal/linctlerr"
)

// Batch 把多个 Mutator 按 File 分组、合并执行。
//
// 性能优化（详见 docs/07-ast-injection.md §7.X）：
//   - 同一文件多 mutator 只 parse 一次、改 N 次、write 一次
//   - 文件不存在时跳过（plan 阶段会提前检测）
//
// 错误策略：
//   - 任一 Mutator 失败 → 立即返回（不写盘）
//   - 单 Mutator 返回 *ConflictError → 收集到 result.Conflicts 但继续处理其他文件
type Batch struct {
	fm *fs.FileManager
}

// NewBatch 构造 Batch。
func NewBatch(fm *fs.FileManager) *Batch {
	return &Batch{fm: fm}
}

// BatchResult 是 Batch.Apply 的结果。
type BatchResult struct {
	ModifiedFiles []string         // 实际被修改的文件相对路径
	Conflicts     []*ConflictError // 检测到的冲突（apply 已跳过这些文件）
}

// Apply 执行所有 mutator。
func (b *Batch) Apply(ctx context.Context, mutators []ASTMutator) (*BatchResult, error) {
	if len(mutators) == 0 {
		return &BatchResult{}, nil
	}

	// 按 File 分组
	byFile := make(map[string][]ASTMutator, 8)
	for _, m := range mutators {
		byFile[m.File()] = append(byFile[m.File()], m)
	}

	result := &BatchResult{}

	for file, ms := range byFile {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if !b.fm.Exists(file) {
			// 文件不存在直接跳过（plan 阶段会检测；apply 阶段继续容错）
			continue
		}
		content, err := b.fm.Read(file)
		if err != nil {
			return result, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"batch read %s", file)
		}

		modified := false
		for _, m := range ms {
			newContent, mod, err := m.Apply(ctx, content)
			if err != nil {
				if conflict, ok := err.(*ConflictError); ok {
					result.Conflicts = append(result.Conflicts, conflict)
					continue
				}
				return result, err
			}
			if mod {
				content = newContent
				modified = true
			}
		}

		if modified {
			if err := b.fm.AtomicWrite(file, content, 0o644); err != nil {
				return result, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"batch write %s", file)
			}
			result.ModifiedFiles = append(result.ModifiedFiles, file)
		}
	}

	return result, nil
}
