package codegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ActionKind 是 Plan 中单个 Action 的类型。
//
// Phase 1 仅支持 Create / Update / Skip（与 ADR-004 Tier 1 对齐）；
// Conflict / Delete 在 Phase 2-4 引入。
type ActionKind string

const (
	ActionCreate ActionKind = "create"
	ActionUpdate ActionKind = "update"
	ActionSkip   ActionKind = "skip"

	// Phase 2+ 引入的占位（当前未使用，仅保留常量稳定 API）
	ActionConflict ActionKind = "conflict"
	ActionDelete   ActionKind = "delete"
)

// Action 是 Plan 中针对单个文件的待执行动作。
type Action struct {
	Kind     ActionKind `json:"kind"`
	Dst      string     `json:"dst"`
	Owner    string     `json:"owner,omitempty"`
	Template string     `json:"template,omitempty"`
	NewHash  string     `json:"newHash,omitempty"` // 渲染结果的 hash
	OldHash  string     `json:"oldHash,omitempty"` // 磁盘文件的 hash（Update 时填）
	Reason   string     `json:"reason,omitempty"`  // 人类可读说明（如 "no change"）
}

// PlanStats 是 Plan 的聚合统计，用于 reporter 快速展示。
type PlanStats struct {
	Total    int `json:"total"`
	Create   int `json:"create"`
	Update   int `json:"update"`
	Skip     int `json:"skip"`
	Conflict int `json:"conflict"`
	Delete   int `json:"delete"`
}

// Plan 是从 Project + Pairs 计算出的"待执行变更清单"。
//
// 由 Planner.Plan() 产出；可通过 ComputeDigest 生成稳定 hash 用于 apply --plan 校验
// （详见 SSOT §1.12）。
type Plan struct {
	Actions []Action  `json:"actions"`
	Stats   PlanStats `json:"stats"`
	Digest  string    `json:"digest,omitempty"`
}

// ComputeDigest 计算 Plan 的稳定 SHA256，便于 apply --plan plan.json 校验。
//
// 算法：
//  1. Actions 按 Dst 字典序排序
//  2. 仅取 (Kind, Dst, NewHash) 三字段（避免 Reason 等不稳定字段干扰）
//  3. canonical JSON marshal → SHA256 → hex
func (p *Plan) ComputeDigest() string {
	if p == nil {
		return ""
	}
	type stableAction struct {
		Kind    ActionKind `json:"k"`
		Dst     string     `json:"d"`
		NewHash string     `json:"h,omitempty"`
	}
	stable := make([]stableAction, len(p.Actions))
	for i, a := range p.Actions {
		stable[i] = stableAction{Kind: a.Kind, Dst: a.Dst, NewHash: a.NewHash}
	}
	sort.Slice(stable, func(i, j int) bool { return stable[i].Dst < stable[j].Dst })

	data, err := json.Marshal(stable)
	if err != nil {
		// Marshal 不会失败（结构都是基础类型），但保险
		return fmt.Sprintf("digest-error-%v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// RecomputeStats 根据 Actions 重新计算 Stats（适合 Action 修改后调用）。
func (p *Plan) RecomputeStats() {
	if p == nil {
		return
	}
	stats := PlanStats{Total: len(p.Actions)}
	for _, a := range p.Actions {
		switch a.Kind {
		case ActionCreate:
			stats.Create++
		case ActionUpdate:
			stats.Update++
		case ActionSkip:
			stats.Skip++
		case ActionConflict:
			stats.Conflict++
		case ActionDelete:
			stats.Delete++
		}
	}
	p.Stats = stats
}

