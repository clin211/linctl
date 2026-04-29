// Package codegen 实现 linctl 的代码生成调度（Plan/Pair/Apply）。
//
// 核心抽象：
//   - Pair：单个文件级生成单元（目标路径 + 模板 ID + Owner）
//   - PairBuilder：累积来自 Component / Feature 的 Pair，自动按 Dst 去重
//   - Plan：Pair 集合 + Action 列表（Create/Update/Skip）
//   - Planner：从 Project + Components 生成 Plan
//   - Applier：执行 Plan（render + atomic write + hash）
//
// Phase 1 仅支持 Action: Create / Update / Skip（详见 docs/11-implementation-plan.md §11.2.0）。
// Phase 2 引入 Conflict / 3-way merge；Phase 4 引入 Delete / drift detection。
package codegen

import "sync"

// WriteMode 决定单个 Pair 的写入策略。
type WriteMode int

const (
	// WriteModeOverwrite 在文件存在时覆盖（仅当内容 hash 不同）。
	WriteModeOverwrite WriteMode = iota

	// WriteModeAppend 在文件存在时追加（用于 daemon 类清单文件）。
	WriteModeAppend

	// WriteModeOnce 仅在文件不存在时写入（保护用户手工编辑）。
	WriteModeOnce
)

// Pair 是单个文件的生成定义。
//
// 字段说明：
//   - Dst：目标文件相对路径（相对项目根）
//   - TemplateID：模板路径（相对 templates/ 目录）
//   - Mode：写入策略，默认 Overwrite
//   - Owner：生成方标识（用于 plan 报告中的「这个文件来自谁」）
//   - Data：可选模板数据覆盖（默认使用 Component-level data）
type Pair struct {
	Dst        string
	TemplateID string
	Mode       WriteMode
	Owner      string
	Data       any
}

// PairBuilder 是 Pair 的累积器。
//
// 多个 Component / Feature 可向同一 Builder 提交 Pair；
// 同 Dst 的后写覆盖前写（默认 warn；--strict 时 fail，详见 SSOT §1.14）。
//
// 并发安全：内部用 sync.Mutex 保护。
type PairBuilder struct {
	mu      sync.Mutex
	byDst   map[string]Pair // 按 Dst 去重的最终结果
	overrides []OverrideEvent // 被覆盖的事件（plan 阶段展示）
}

// OverrideEvent 描述一次 Pair 覆盖。
type OverrideEvent struct {
	Dst         string
	OldOwner    string
	OldTemplate string
	NewOwner    string
	NewTemplate string
}

// NewPairBuilder 创建一个空的 PairBuilder。
func NewPairBuilder() *PairBuilder {
	return &PairBuilder{
		byDst: make(map[string]Pair, 64),
	}
}

// Add 添加一个 Pair。同 Dst 时后写覆盖前写，覆盖事件记录到 Overrides()。
func (b *PairBuilder) Add(p Pair) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if existing, ok := b.byDst[p.Dst]; ok && (existing.Owner != p.Owner || existing.TemplateID != p.TemplateID) {
		b.overrides = append(b.overrides, OverrideEvent{
			Dst:         p.Dst,
			OldOwner:    existing.Owner,
			OldTemplate: existing.TemplateID,
			NewOwner:    p.Owner,
			NewTemplate: p.TemplateID,
		})
	}
	b.byDst[p.Dst] = p
}

// AddMany 是 Add 的批量版本。
func (b *PairBuilder) AddMany(pairs []Pair) {
	for _, p := range pairs {
		b.Add(p)
	}
}

// Build 返回最终的 Pair 切片，按 Dst 字典序排列（保证渲染顺序确定性）。
func (b *PairBuilder) Build() []Pair {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Pair, 0, len(b.byDst))
	for _, p := range b.byDst {
		out = append(out, p)
	}
	sortPairsByDst(out)
	return out
}

// Len 返回当前 Pair 数量。
func (b *PairBuilder) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.byDst)
}

func sortPairsByDst(p []Pair) {
	// 简单插入排序（项目通常 Pair 数 < 1000；避免引入 sort 依赖时的间接成本）
	for i := 1; i < len(p); i++ {
		j := i
		for j > 0 && p[j-1].Dst > p[j].Dst {
			p[j-1], p[j] = p[j], p[j-1]
			j--
		}
	}
}
