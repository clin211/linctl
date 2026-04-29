package codegen

import (
	"context"

	"github.com/clin211/linctl/internal/fs"
	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/template"
)

// Planner 负责把 Pair 集合转换为 Plan：
//   - 渲染每个 Pair 得到 newContent
//   - 与磁盘对比（不存在 → Create；存在且 hash 同 → Skip；存在且 hash 不同 → Update）
//
// Phase 1 仅做整体 sha256 比对，不解析 embedded hash 注释（那是 Phase 2 引入）。
type Planner struct {
	engine *template.Engine
	fm     *fs.FileManager
}

// PlannerOptions 是 Planner 构造选项。
type PlannerOptions struct {
	Engine *template.Engine
	FM     *fs.FileManager
}

// NewPlanner 构造 Planner。Engine 和 FM 必填。
func NewPlanner(opts PlannerOptions) (*Planner, error) {
	if opts.Engine == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Planner: Engine is nil")
	}
	if opts.FM == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Planner: FM is nil")
	}
	return &Planner{engine: opts.Engine, fm: opts.FM}, nil
}

// renderedPair 是 Planner.compute 内部使用的中间结构。
type renderedPair struct {
	pair    Pair
	content []byte
	hash    string
}

// Plan 计算 Plan。
//
//   - data 是模板渲染上下文（通常是 *template.TemplateData）
//   - pairs 是来自 Component.BasePairs + Feature.Apply 聚合后的最终 Pair 集合
//
// 错误场景：
//   - 任一模板渲染失败 → 立即返回（Phase 1 的 fail-fast 策略）
//   - 计算过程中读盘失败 → wrap 为 LinctlError
func (p *Planner) Plan(ctx context.Context, data any, pairs []Pair) (*Plan, error) {
	if p == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Planner is nil")
	}

	rendered := make([]renderedPair, 0, len(pairs))

	for _, pair := range pairs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// 选用 pair-level data 或 fallback 到全局 data
		renderData := data
		if pair.Data != nil {
			renderData = pair.Data
		}

		out, err := p.engine.Render(pair.TemplateID, renderData)
		if err != nil {
			return nil, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
				"render %s (owner=%s)", pair.TemplateID, pair.Owner)
		}

		// Phase 1：根据扩展名做格式化（仅 .go），失败时返回原内容（不破坏 plan）
		if formatted, ferr := p.engine.Format(out, fileExt(pair.Dst)); ferr == nil {
			out = formatted
		}

		rendered = append(rendered, renderedPair{
			pair:    pair,
			content: out,
			hash:    fs.HashContent(out),
		})
	}

	plan := &Plan{Actions: make([]Action, 0, len(rendered))}

	for _, r := range rendered {
		action := Action{
			Dst:      r.pair.Dst,
			Owner:    r.pair.Owner,
			Template: r.pair.TemplateID,
			NewHash:  r.hash,
		}

		if !p.fm.Exists(r.pair.Dst) {
			action.Kind = ActionCreate
		} else {
			oldContent, err := p.fm.Read(r.pair.Dst)
			if err != nil {
				return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"read existing %s for plan", r.pair.Dst)
			}
			oldHash := fs.HashContent(stripHashCommentSafe(oldContent))
			action.OldHash = oldHash

			if oldHash == r.hash {
				action.Kind = ActionSkip
				action.Reason = "no change"
			} else {
				action.Kind = ActionUpdate
				action.Reason = "content differs"
			}
		}

		plan.Actions = append(plan.Actions, action)
	}

	plan.RecomputeStats()
	plan.Digest = plan.ComputeDigest()
	return plan, nil
}

// stripHashCommentSafe 移除 hash 注释行（hash 比较前归一化）。
// 包级辅助；将 fs.StripHashComment 间接化以便测试替换。
var stripHashCommentSafe = fs.StripHashComment

// fileExt 返回路径扩展名（含 .），全小写。
func fileExt(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '.' {
			return p[i:]
		}
		if p[i] == '/' {
			break
		}
	}
	return ""
}
