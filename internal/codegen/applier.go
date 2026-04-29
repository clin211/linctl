package codegen

import (
	"context"
	"os"
	"strings"

	"github.com/clin211/lin/internal/fs"
	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/template"
)

// Applier 执行 Plan：按 Action 渲染并写盘。
//
//   - 串行写入（Skip 跳过；Create/Update 走 atomic write 直接覆盖）
//   - dryRun 模式不做实际写入，仅返回 Report
//
// 注：v0.3.x 起不再向生成文件追加 `// linctl: hash=...` 注释，
// 也不再做 hash drift 检测；Update 路径退化为「直接覆盖」语义。
type Applier struct {
	engine *template.Engine
	fm     *fs.FileManager
	dryRun bool
}

// ApplierOptions 是 Applier 构造选项。
type ApplierOptions struct {
	Engine *template.Engine
	FM     *fs.FileManager
	DryRun bool
}

// NewApplier 构造 Applier。Engine 和 FM 必填。
func NewApplier(opts ApplierOptions) (*Applier, error) {
	if opts.Engine == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Applier: Engine is nil")
	}
	if opts.FM == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Applier: FM is nil")
	}
	return &Applier{
		engine: opts.Engine,
		fm:     opts.FM,
		dryRun: opts.DryRun,
	}, nil
}

// Report 是 Apply 完成后的总结。
type Report struct {
	Created []string
	Updated []string
	Skipped []string
	DryRun  bool
}

// Apply 执行 plan。data 用于 Update / Create 时的二次渲染。
//
// 注意：Phase 1 的实现会**重新渲染**每个 Pair（Plan 阶段已渲染过一次）；
// 这是为了简化 Plan struct（不内嵌大量 content 字节）。Phase 2 可优化为 Plan
// 内嵌 content 缓存，避免重复渲染开销。
func (a *Applier) Apply(ctx context.Context, data any, pairs []Pair, plan *Plan) (*Report, error) {
	if a == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Applier is nil")
	}
	if plan == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Apply: plan is nil")
	}

	pairByDst := make(map[string]Pair, len(pairs))
	for _, p := range pairs {
		pairByDst[p.Dst] = p
	}

	rep := &Report{DryRun: a.dryRun}

	for _, action := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return rep, err
		}

		switch action.Kind {
		case ActionSkip:
			rep.Skipped = append(rep.Skipped, action.Dst)
			continue

		case ActionCreate, ActionUpdate:
			pair, ok := pairByDst[action.Dst]
			if !ok {
				return rep, linctlerr.Newf(linctlerr.ErrInternal,
					"Apply: pair missing for action %s", action.Dst)
			}

			renderData := data
			if pair.Data != nil {
				renderData = pair.Data
			}

			content, err := a.engine.Render(pair.TemplateID, renderData)
			if err != nil {
				return rep, linctlerr.Wrapf(linctlerr.ErrTemplateRender, err,
					"render %s for apply", pair.TemplateID)
			}

			if formatted, ferr := a.engine.Format(content, fileExt(pair.Dst)); ferr == nil {
				content = formatted
			}

			// 防御式剥离：即使下游模板、partials 或 Format 步骤意外引入了
			// `linctl: hash=...` 注释行，也保证最终落盘内容彻底不含指纹。
			// 循环调用直到稳定，避免历史模板存在多行残留时遗漏。
			content = stripAllHashComments(content)

			if a.dryRun {
				if action.Kind == ActionCreate {
					rep.Created = append(rep.Created, action.Dst)
				} else {
					rep.Updated = append(rep.Updated, action.Dst)
				}
				continue
			}

			if err := a.fm.AtomicWrite(action.Dst, content, perm(action.Dst)); err != nil {
				return rep, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"write %s", action.Dst)
			}

			if action.Kind == ActionCreate {
				rep.Created = append(rep.Created, action.Dst)
			} else {
				rep.Updated = append(rep.Updated, action.Dst)
			}

		default:
			return rep, linctlerr.Newf(linctlerr.ErrInternal,
				"Apply: unknown action kind %q for %s", action.Kind, action.Dst)
		}
	}

	return rep, nil
}

// executableExts 是按 Dst 文件后缀映射到 0o755 的可执行文件扩展名集合。
//
// 这些后缀通常对应 shell / awk / python / bash 脚本，落盘后用户期望可以
// 直接 `./script.sh` 执行；其他文件继续使用 0o644。
var executableExts = map[string]struct{}{
	".sh":   {},
	".bash": {},
	".awk":  {},
	".py":   {},
}

// perm 按 Dst 文件后缀返回新建文件权限。
//
// 默认 0o644；当后缀属于 executableExts（如 scripts/coverage.awk、
// scripts/startup-test.sh）时返回 0o755 以保持可执行语义。
func perm(dst string) os.FileMode {
	idx := strings.LastIndexByte(dst, '.')
	if idx < 0 {
		return 0o644
	}
	if slash := strings.LastIndexByte(dst, '/'); slash > idx {
		// 末段（basename）没有 .，扩展名其实在父目录里——按非脚本处理
		return 0o644
	}
	ext := strings.ToLower(dst[idx:])
	if _, ok := executableExts[ext]; ok {
		return 0o755
	}
	return 0o644
}

// stripAllHashComments 反复调用 fs.StripHashComment 直到内容稳定，用于在
// Apply 写盘前彻底清除可能存在的 linctl 指纹注释残留。
//
// 与 fs.StripHashComment 的区别：后者单次调用只移除首个匹配的整行（其语义在
// planner.go 的 hash 归一化场景中是足够的，不能擅自改动）。本函数循环调用以
// 覆盖多行残留场景；这是 v0.3.x 之后的防御式保险，绝大多数情况一轮即可终止。
func stripAllHashComments(content []byte) []byte {
	for {
		stripped := fs.StripHashComment(content)
		if len(stripped) == len(content) {
			return content
		}
		content = stripped
	}
}

