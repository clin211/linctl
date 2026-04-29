package templatesync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/clin211/lin/internal/gitmerge"
	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/templatesync/transform"
)

// LinTemplatesRoot 是 lin 仓库内 web-gin 模板根目录的相对路径（相对 lin/）。
const LinTemplatesRoot = "internal/template/templates/web-gin"

// ActionKind 是 templatesync 流水线对单个文件的处理动作。
type ActionKind string

const (
	// ActionCreate：dst 不存在，需新建（owner=upstream/shared 时）
	ActionCreate ActionKind = "create"

	// ActionUpdate：dst 已存在，且与上次同步内容相符；可直接覆盖（owner=upstream，或 shared 但用户没改）
	ActionUpdate ActionKind = "update"

	// ActionSkip：dst 已是最新（src 与 lock 一致）
	ActionSkip ActionKind = "skip"

	// ActionMergeNeeded：owner=shared 且双方都改了，需要 3-way merge（U2 阶段引入实际处理；U1 仅打标）
	ActionMergeNeeded ActionKind = "merge-needed"

	// ActionConflict：合并失败（同 ActionMergeNeeded 但已尝试 merge 后剩余冲突）
	ActionConflict ActionKind = "conflict"

	// ActionPreserve：owner=linSpecific，永不动
	ActionPreserve ActionKind = "preserve"

	// ActionMissingRule：上游有新文件但 sync.yaml 没声明（按 newFilePolicy 处理）
	ActionMissingRule ActionKind = "missing-rule"

	// ActionUpstreamGone：sync.yaml 声明的 src 在上游不存在（可能被删）
	ActionUpstreamGone ActionKind = "upstream-gone"
)

// FileAction 是 plan 中针对单个文件的待执行动作。
type FileAction struct {
	Kind             ActionKind
	Src              string // 上游相对路径（splits 时为父 src）
	Dst              string // 镜像相对路径
	Owner            Owner
	Reason           string
	NewContent       []byte // 渲染后内容（plan 阶段已渲染，apply 阶段可直接写）
	OldDstHash       string // 当前磁盘 dst 的 hash（如存在）
	NewSrcHash       string // 当前磁盘 src 的 hash
	BaseSrcHash      string // 上次同步时记录在 lockfile 的 src hash（base）
	BaseDstHash      string // 上次同步时记录在 lockfile 的 dst hash
	TransformsApplied []string
}

// Plan 是一次完整 sync 的待执行清单。
type Plan struct {
	Actions      []FileAction
	UnknownNew   []string // 上游有但 sync.yaml 没声明的 src 列表（newFilePolicy=warn 时打印）
}

// RunnerOptions 是 Runner 的构造选项。
type RunnerOptions struct {
	// LinRoot 是 lin/ 根目录绝对路径
	LinRoot string

	// ManifestPath 是 sync.yaml 绝对路径；空时取 LinRoot/internal/template/templates/web-gin/.sync.yaml
	ManifestPath string

	// LockPath 是 lockfile 绝对路径；空时取 LinRoot/.linctl/upstream-sync.lock.json
	LockPath string

	// Registry 是 transform 注册中心；空时使用 transform.DefaultRegistry
	Registry *transform.Registry

	// LinctlVersion 是写入 lockfile 的 linctl 版本字符串
	LinctlVersion string
}

// Runner 是 templatesync 的核心执行器，串联 Manifest + Lockfile + Transform。
type Runner struct {
	linRoot       string
	manifest      *Manifest
	lockfile      *Lockfile
	registry      *transform.Registry
	cache         *baseCache
	linctlVersion string
}

// NewRunner 初始化 Runner：加载 manifest、lockfile，设置 registry。
func NewRunner(opts RunnerOptions) (*Runner, error) {
	if opts.LinRoot == "" {
		return nil, linctlerr.New(linctlerr.ErrConfigInvalid,
			"NewRunner: LinRoot is required")
	}
	absRoot, err := filepath.Abs(opts.LinRoot)
	if err != nil {
		return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"resolve LinRoot %s", opts.LinRoot)
	}

	manifestPath := opts.ManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(absRoot, LinTemplatesRoot, ".sync.yaml")
	}
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	lockPath := opts.LockPath
	if lockPath == "" {
		lockPath = filepath.Join(absRoot, ".linctl", "upstream-sync.lock.json")
	}
	lf, err := LoadLockfile(lockPath)
	if err != nil {
		return nil, err
	}

	registry := opts.Registry
	if registry == nil {
		registry = transform.DefaultRegistry
	}

	return &Runner{
		linRoot:       absRoot,
		manifest:      m,
		lockfile:      lf,
		registry:      registry,
		cache:         newBaseCache(absRoot),
		linctlVersion: opts.LinctlVersion,
	}, nil
}

// BaseCache 暴露内部 cache（仅供 cmd 层 / 测试访问）。
func (r *Runner) BaseCache() *baseCache { return r.cache }

// Manifest 返回内部 manifest 引用（仅供调试 / 测试）。
func (r *Runner) Manifest() *Manifest { return r.manifest }

// Lockfile 返回内部 lockfile 引用（仅供调试 / 测试）。
func (r *Runner) Lockfile() *Lockfile { return r.lockfile }

// LinTemplatesAbsRoot 返回 lin 仓库内 web-gin 模板根的绝对路径。
func (r *Runner) LinTemplatesAbsRoot() string {
	return filepath.Join(r.linRoot, LinTemplatesRoot)
}

// Plan 算出从当前 upstream 状态 + lockfile 到 sync.yaml 期望状态的待执行清单。
//
// 不写盘；不修改 manifest / lockfile。
func (r *Runner) Plan(ctx context.Context) (*Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	upstreamRoot, err := r.manifest.AbsUpstreamRoot()
	if err != nil {
		return nil, err
	}

	plan := &Plan{Actions: make([]FileAction, 0, len(r.manifest.Files)+len(r.manifest.LinSpecific))}

	// 1) 处理 manifest.Files：对每个 src 算出 action
	for _, fm := range r.manifest.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(fm.Splits) > 0 {
			// U1 不处理 splits（U2/U3 引入）；占位
			for _, s := range fm.Splits {
				plan.Actions = append(plan.Actions, FileAction{
					Kind:   ActionMissingRule,
					Src:    fm.Src,
					Dst:    s.Dst,
					Owner:  fm.Owner,
					Reason: "splits not supported in U1; defer to U2",
				})
			}
			continue
		}

		action, err := r.computeAction(ctx, fm, upstreamRoot)
		if err != nil {
			return nil, err
		}
		plan.Actions = append(plan.Actions, action)
	}

	// 2) 处理 linSpecific：始终 Preserve
	for _, e := range r.manifest.LinSpecific {
		plan.Actions = append(plan.Actions, FileAction{
			Kind:   ActionPreserve,
			Dst:    e.Dst,
			Owner:  OwnerLinSpecific,
			Reason: e.Reason,
		})
	}

	// 3) 检测上游新文件（不在 sync.yaml / ignored / linSpecific 中）
	unknown, err := r.detectUnknownUpstreamFiles(upstreamRoot)
	if err != nil {
		return nil, err
	}
	plan.UnknownNew = unknown

	// 4) 排序输出（按 dst 字典序，便于稳定 diff）
	sort.SliceStable(plan.Actions, func(i, j int) bool {
		return plan.Actions[i].Dst < plan.Actions[j].Dst
	})

	return plan, nil
}

// computeAction 给单个 FileMapping 算 action。
func (r *Runner) computeAction(ctx context.Context, fm FileMapping, upstreamRoot string) (FileAction, error) {
	srcAbs := filepath.Join(upstreamRoot, fm.Src)
	dstAbs := filepath.Join(r.LinTemplatesAbsRoot(), fm.Dst)

	srcContent, err := os.ReadFile(srcAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return FileAction{
				Kind:   ActionUpstreamGone,
				Src:    fm.Src,
				Dst:    fm.Dst,
				Owner:  fm.Owner,
				Reason: "src not found in upstream",
			}, nil
		}
		return FileAction{}, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"read upstream src %s", fm.Src)
	}

	srcHash := hashBytes(srcContent)
	locked, hasLock := r.lockfile.Get(fm.Dst)

	// 应用 transforms 算出 newContent
	newContent, applied, err := r.applyTransforms(ctx, fm, upstreamRoot, srcContent)
	if err != nil {
		return FileAction{}, err
	}

	dstExists, oldDstContent, err := r.readDstIfExists(dstAbs)
	if err != nil {
		return FileAction{}, err
	}
	oldDstHash := ""
	if dstExists {
		oldDstHash = hashBytes(oldDstContent)
	}

	action := FileAction{
		Src:               fm.Src,
		Dst:               fm.Dst,
		Owner:             fm.Owner,
		NewContent:        newContent,
		OldDstHash:        oldDstHash,
		NewSrcHash:        srcHash,
		BaseSrcHash:       locked.SrcHashAtSync,
		BaseDstHash:       locked.DstHashAtSync,
		TransformsApplied: applied,
	}

	newHash := hashBytes(newContent)

	switch {
	case !dstExists:
		action.Kind = ActionCreate
		action.Reason = "dst missing"

	case oldDstHash == newHash:
		// 已是最新，无需操作
		action.Kind = ActionSkip
		if hasLock && locked.SrcHashAtSync == srcHash {
			action.Reason = "no change"
		} else {
			action.Reason = "content matches"
		}

	case fm.Owner == OwnerUpstream:
		// upstream 主导：直接覆盖（未来加 force/protect 策略）
		action.Kind = ActionUpdate
		action.Reason = "upstream changed; owner=upstream"

	case fm.Owner == OwnerShared:
		// shared：决策矩阵
		userTouched := dstExists && hasLock && locked.DstHashAtSync != "" && locked.DstHashAtSync != oldDstHash
		upstreamChanged := !hasLock || locked.SrcHashAtSync != srcHash
		switch {
		case !userTouched && upstreamChanged:
			action.Kind = ActionUpdate
			action.Reason = "upstream changed; lin-side untouched"
		case userTouched && !upstreamChanged:
			action.Kind = ActionSkip
			action.Reason = "lin-side has edits; upstream unchanged"
		case userTouched && upstreamChanged:
			action.Kind = ActionMergeNeeded
			action.Reason = "both upstream and lin-side changed; 3-way merge required (U2)"
		default:
			// userTouched=false && upstreamChanged=false 但 hash 不等：
			// 这意味着 transform 行为变化（比如 sync.yaml 改了规则）
			action.Kind = ActionUpdate
			action.Reason = "transform output changed"
		}

	case fm.Owner == OwnerLinSpecific:
		// 不应到达：LinSpecific 不应出现在 Files 中
		action.Kind = ActionPreserve
		action.Reason = "linSpecific in files block; check sync.yaml"

	default:
		action.Kind = ActionMergeNeeded
		action.Reason = fmt.Sprintf("unknown owner %q", fm.Owner)
	}

	return action, nil
}

// applyTransforms 串行应用 default + extra transforms。
func (r *Runner) applyTransforms(
	ctx context.Context, fm FileMapping, upstreamRoot string, content []byte,
) ([]byte, []string, error) {
	rc := &transform.RuntimeContext{
		UpstreamRoot: upstreamRoot,
		SrcPath:      fm.Src,
		DstPath:      fm.Dst,
		Owner:        string(fm.Owner),
		Manifest:     manifestAccessor{m: r.manifest},
	}

	all := append([]TransformConfig{}, r.manifest.DefaultTransforms...)
	all = append(all, fm.ExtraTransforms...)

	applied := make([]string, 0, len(all))
	cur := content
	for _, cfg := range all {
		// triggeredBy=manual 的 transform 跳过自动 apply 路径
		if v, ok := cfg.RawCfg["triggeredBy"]; ok {
			if s, _ := v.(string); s == "manual" {
				continue
			}
		}
		t, err := r.registry.Build(cfg.Kind, cfg.RawCfg)
		if err != nil {
			return nil, nil, err
		}
		next, err := t.Apply(ctx, rc, cur)
		if err != nil {
			return nil, nil, err
		}
		cur = next
		applied = append(applied, cfg.Kind)
	}

	return cur, applied, nil
}

// readDstIfExists 读 dst 文件；不存在返回 (false, nil, nil)。
func (r *Runner) readDstIfExists(dstAbs string) (bool, []byte, error) {
	data, err := os.ReadFile(dstAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil, nil
		}
		return false, nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"read dst %s", dstAbs)
	}
	return true, data, nil
}

// detectUnknownUpstreamFiles 列出"上游有但 sync.yaml 没声明 / 没 ignore / 不在 linSpecific"的文件。
//
// 扫描范围由 sync.yaml 的 upstream.scanPaths 控制：
//   - scanPaths 非空 → 只扫这些子目录（相对 upstream rootPath）
//   - scanPaths 为空 → 全 walk，但跳过常见无关目录（.git / _output / data / etc.）
//
// 扩展名过滤：只关注代码 / 配置 / 文档相关文件（详见 allowedExts）。
func (r *Runner) detectUnknownUpstreamFiles(upstreamRoot string) ([]string, error) {
	known := make(map[string]struct{}, len(r.manifest.Files)+len(r.manifest.Ignored))
	for _, fm := range r.manifest.Files {
		if fm.Src != "" {
			known[fm.Src] = struct{}{}
		}
	}
	for _, ig := range r.manifest.Ignored {
		known[ig.Src] = struct{}{}
	}

	allowedExts := map[string]struct{}{
		".go": {}, ".proto": {}, ".yaml": {}, ".yml": {}, ".json": {},
		".sh": {}, ".bash": {}, ".sql": {}, ".md": {},
	}

	scanRoots := r.manifest.Upstream.ScanPaths
	if len(scanRoots) == 0 {
		scanRoots = []string{""} // empty means walk upstream root
	}

	var unknown []string
	seen := make(map[string]struct{}, 64)
	for _, sub := range scanRoots {
		root := upstreamRoot
		if sub != "" {
			root = filepath.Join(upstreamRoot, sub)
		}
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue // scanPath 配错时跳过；不报错（不阻塞 status）
			}
			return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"stat scanPath %s", sub)
		}
		walkErr := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(p)
				switch base {
				case ".git", "_output", "node_modules", "vendor",
					".idea", ".vscode", ".claude", "data",
					"dist", "tmp", ".cache":
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(p)
			if _, ok := allowedExts[ext]; !ok {
				return nil
			}
			rel, err := filepath.Rel(upstreamRoot, p)
			if err != nil {
				return nil
			}
			if _, ok := known[rel]; ok {
				return nil
			}
			if _, ok := seen[rel]; ok {
				return nil
			}
			seen[rel] = struct{}{}
			unknown = append(unknown, rel)
			return nil
		})
		if walkErr != nil {
			return nil, linctlerr.Wrapf(linctlerr.ErrEnvironment, walkErr,
				"walk upstream %s", root)
		}
	}

	sort.Strings(unknown)
	return unknown, nil
}

// Strategy 决定 apply 时如何处理冲突 / merge 等场景。
type Strategy string

const (
	// StrategyForce：所有 owner 都直接覆盖（U1 仅支持此模式）。
	//   - upstream → 覆盖
	//   - shared   → 覆盖（用户改动会丢；用此策略时调用方应已 backup）
	//   - linSpecific → 仍然保留（永远不动）
	StrategyForce Strategy = "force"

	// StrategyAbort：遇到 ActionMergeNeeded 立即中止整个 apply（不写盘）。
	StrategyAbort Strategy = "abort"

	// StrategyAsk：交互式询问每个 mergeNeeded 文件（U2 引入）。
	StrategyAsk Strategy = "ask"

	// StrategyOurs：保留 lin-side 改动（跳过 upstream 改动）。
	StrategyOurs Strategy = "ours"

	// StrategyTheirs：用上游覆盖 lin-side 改动。
	StrategyTheirs Strategy = "theirs"
)

// Report 是 apply 完成后的总结。
type Report struct {
	Created  []string
	Updated  []string
	Skipped  []string
	Merged   []string
	Conflict []string
	Preserved []string
	DryRun   bool
}

// Apply 执行 plan：按 strategy 写盘。
//
// U1 仅支持 strategy=force / abort：
//   - force：所有 ActionCreate / Update / MergeNeeded 都直接覆盖（不调用 git merge）
//   - abort：遇到 MergeNeeded 立即返回错误，不写盘
//
// dryRun=true 时不写盘，仅输出 report。
func (r *Runner) Apply(ctx context.Context, plan *Plan, strategy Strategy, dryRun bool) (*Report, error) {
	if plan == nil {
		return nil, linctlerr.New(linctlerr.ErrInternal, "Apply: plan is nil")
	}

	rep := &Report{DryRun: dryRun}

	// abort 模式：先扫一遍，有 mergeNeeded 直接挂
	if strategy == StrategyAbort {
		for _, a := range plan.Actions {
			if a.Kind == ActionMergeNeeded || a.Kind == ActionConflict {
				return rep, linctlerr.Newf(linctlerr.ErrFileConflict,
					"templatesync apply aborted: %d files need merge (first: %s)",
					countMergeNeeded(plan), a.Dst)
			}
		}
	}

	for _, a := range plan.Actions {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		if err := r.applyOne(ctx, a, strategy, dryRun, rep); err != nil {
			return rep, err
		}
	}

	if !dryRun {
		// 更新 lockfile metadata
		r.lockfile.SchemaVersion = lockSchemaVersion
		r.lockfile.SyncManifest = LinTemplatesRoot + "/.sync.yaml"
		r.lockfile.LastSyncAt = time.Now().UTC()
		r.lockfile.LinctlVersion = r.linctlVersion
		r.lockfile.Upstream = LockUpstream{
			Name:     r.manifest.Upstream.Name,
			RootPath: r.manifest.Upstream.RootPath,
		}
		if err := r.lockfile.Save(); err != nil {
			return rep, err
		}
	}

	return rep, nil
}

func (r *Runner) applyOne(ctx context.Context, a FileAction, strategy Strategy, dryRun bool, rep *Report) error {
	dstAbs := filepath.Join(r.LinTemplatesAbsRoot(), a.Dst)

	switch a.Kind {
	case ActionPreserve:
		rep.Preserved = append(rep.Preserved, a.Dst)
		return nil

	case ActionSkip:
		rep.Skipped = append(rep.Skipped, a.Dst)
		return nil

	case ActionUpstreamGone:
		// U1 不自动删 dst（避免误删 lin-side）；仅 skipped
		rep.Skipped = append(rep.Skipped, a.Dst+" (upstream-gone)")
		return nil

	case ActionMissingRule:
		// 由调用方在 status / plan 输出时单独提示
		return nil

	case ActionMergeNeeded, ActionConflict:
		return r.applyMerge(ctx, a, strategy, dstAbs, dryRun, rep)

	case ActionCreate, ActionUpdate:
		return r.writeAndLock(a, a.NewContent, dstAbs, dryRun, rep)
	}

	return linctlerr.Newf(linctlerr.ErrInternal,
		"applyOne: unknown action kind %q for %s", a.Kind, a.Dst)
}

// applyMerge 处理 ActionMergeNeeded / ActionConflict：根据 strategy 决定路径。
//
// 策略矩阵：
//   - force / theirs：用 transform 渲染产物覆盖（lin-side 改动会丢；force 与 theirs 在
//     无 git 场景下行为等价；有 git 时 theirs 仍走 merge 但冲突 hunk 取 theirs）
//   - ours：保留 lin-side，跳过本次更新（lockfile 不动）
//   - abort：上层已检测并 fail；这里防御性兜底
//   - ask / 默认：走 git merge-file，3-way merge 写入工作区（含 conflict markers）
func (r *Runner) applyMerge(ctx context.Context, a FileAction, strategy Strategy, dstAbs string, dryRun bool, rep *Report) error {
	switch strategy {
	case StrategyForce:
		return r.writeAndLock(a, a.NewContent, dstAbs, dryRun, rep)

	case StrategyOurs:
		rep.Skipped = append(rep.Skipped, a.Dst+" (kept lin-side; upstream changes ignored)")
		return nil

	case StrategyAbort:
		// 防御兜底：Apply 主循环已经在前置扫描时短路
		return linctlerr.Newf(linctlerr.ErrFileConflict,
			"templatesync apply aborted: %s needs merge", a.Dst)

	case StrategyTheirs, StrategyAsk:
		// 走 git merge-file
	default:
		return linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"applyMerge: unknown strategy %q", strategy)
	}

	base, hasBase, err := r.cache.Get(a.BaseDstHash)
	if err != nil {
		return err
	}
	if !hasBase {
		// 没有 base 内容缓存 → 降级为 force（warning）
		// 用户首次升级到 U2 之前生成的 lockfile 缺缓存内容是常见情况
		return r.writeAndLock(a, a.NewContent, dstAbs, dryRun, rep)
	}

	// 读 ours（磁盘当前内容）
	ours, err := os.ReadFile(dstAbs)
	if err != nil {
		return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
			"applyMerge: read ours %s", a.Dst)
	}

	gmStrat := gitmerge.StrategyAuto
	if strategy == StrategyTheirs {
		gmStrat = gitmerge.StrategyTheirs
	}

	res, err := gitmerge.MergeFile(ctx, gitmerge.Files{
		Base:   base,
		Ours:   ours,
		Theirs: a.NewContent,
		Labels: gitmerge.Labels{
			Ours:   "lin-side",
			Base:   "previous-sync",
			Theirs: "miniblog-v4@upstream",
		},
	}, gmStrat)
	if err != nil {
		return err
	}

	if res.HasConflict {
		// 有冲突 → 写入带 markers 的内容；lockfile 不更新（让 status / check 仍标 dirty）
		if !dryRun {
			if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
				return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"mkdir %s", filepath.Dir(a.Dst))
			}
			if err := atomicWrite(dstAbs, res.Content, 0o644); err != nil {
				return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
					"write merge-with-markers %s", a.Dst)
			}
		}
		rep.Conflict = append(rep.Conflict, a.Dst)
		return nil
	}

	// 干净合并 → 写盘 + 更新 lock + 缓存新 base
	return r.writeAndLock(a, res.Content, dstAbs, dryRun, rep)
}

// writeAndLock 写入内容到 dst + 更新 lockfile + 把内容缓存为下次同步的 base。
//
// dryRun=true 时仅追加 report 不写盘 / 不动 lock。
func (r *Runner) writeAndLock(a FileAction, content []byte, dstAbs string, dryRun bool, rep *Report) error {
	if !dryRun {
		if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
			return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"mkdir %s", filepath.Dir(a.Dst))
		}
		if err := atomicWrite(dstAbs, content, 0o644); err != nil {
			return linctlerr.Wrapf(linctlerr.ErrEnvironment, err,
				"write %s", a.Dst)
		}
		dstHash := hashBytes(content)
		// 缓存这份"刚写盘的产物"，作为下次同步的 base
		if err := r.cache.Put(dstHash, content); err != nil {
			return err
		}
		r.lockfile.Update(a.Dst, LockedFile{
			Src:               a.Src,
			Owner:             a.Owner,
			SrcHashAtSync:     a.NewSrcHash,
			DstHashAtSync:     dstHash,
			TransformsApplied: a.TransformsApplied,
			LastSyncAt:        time.Now().UTC(),
		})
	}
	switch a.Kind {
	case ActionCreate:
		rep.Created = append(rep.Created, a.Dst)
	case ActionMergeNeeded, ActionConflict:
		rep.Merged = append(rep.Merged, a.Dst)
	default:
		rep.Updated = append(rep.Updated, a.Dst)
	}
	return nil
}

// hashBytes 返回 sha256:<hex> 形式。
func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// atomicWrite 原子写入：先写到 .tmp 再 rename。
func atomicWrite(absPath string, content []byte, perm os.FileMode) error {
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, content, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, absPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func countMergeNeeded(plan *Plan) int {
	c := 0
	for _, a := range plan.Actions {
		if a.Kind == ActionMergeNeeded || a.Kind == ActionConflict {
			c++
		}
	}
	return c
}

// manifestAccessor 是 *Manifest 对 transform.ManifestAccessor 的薄壳。
type manifestAccessor struct {
	m *Manifest
}

func (a manifestAccessor) UpstreamModuleOld() string        { return a.m.Upstream.ModuleOld }
func (a manifestAccessor) UpstreamBinaryNameOld() string    { return a.m.Upstream.BinaryNameOld }
func (a manifestAccessor) UpstreamComponentNameOld() string { return a.m.Upstream.ComponentNameOld }
func (a manifestAccessor) UpstreamName() string             { return a.m.Upstream.Name }
