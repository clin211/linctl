package gitmerge

import (
	"bytes"
	"fmt"
	"strings"
)

// mergeWithFallback 是不依赖系统 git 的 3-way merge 兜底实现。
//
// 算法：行级别 LCS-based 3-way merge（参考 GNU diff3 的简化版）。
//
// 限制（对比真 git merge-file）：
//   - 仅按行处理，不做 word-level 细化合并
//   - 不实现 --no-diff3（等价于始终输出 base 段）
//   - 不实现 rename detection
//
// 已知的边界 case 比 git 弱，但对常见场景（用户改了 5 行 + upstream 改了另外 5 行）
// 能产生与 git 一致的结果。冲突时按 git 风格输出 markers。
func mergeWithFallback(f Files, strat Strategy) (*Result, error) {
	baseLines := splitLines(f.Base)
	oursLines := splitLines(f.Ours)
	theirsLines := splitLines(f.Theirs)

	hunks := computeHunks(baseLines, oursLines, theirsLines)

	var buf bytes.Buffer
	hasConflict := false
	for _, h := range hunks {
		switch h.kind {
		case hunkStable:
			for _, ln := range h.base {
				buf.WriteString(ln)
			}
		case hunkOursOnly:
			for _, ln := range h.ours {
				buf.WriteString(ln)
			}
		case hunkTheirsOnly:
			for _, ln := range h.theirs {
				buf.WriteString(ln)
			}
		case hunkBothEqual:
			// 双方做了同样的改动 → 直接采纳
			for _, ln := range h.ours {
				buf.WriteString(ln)
			}
		case hunkConflict:
			hasConflict = true
			switch strat {
			case StrategyOurs:
				for _, ln := range h.ours {
					buf.WriteString(ln)
				}
			case StrategyTheirs:
				for _, ln := range h.theirs {
					buf.WriteString(ln)
				}
			case StrategyUnion:
				for _, ln := range h.ours {
					buf.WriteString(ln)
				}
				for _, ln := range h.theirs {
					buf.WriteString(ln)
				}
			default:
				writeConflictMarkers(&buf, h, f.Labels)
			}
		}
	}

	return &Result{
		Content:     buf.Bytes(),
		HasConflict: hasConflict,
		Backend:     "diff3-fallback",
	}, nil
}

// hunk 是 base/ours/theirs 共划出的对应区间。
type hunk struct {
	kind   hunkKind
	base   []string
	ours   []string
	theirs []string
}

type hunkKind int

const (
	hunkStable     hunkKind = iota // 三方一致
	hunkOursOnly                   // 只有 ours 改了
	hunkTheirsOnly                 // 只有 theirs 改了
	hunkBothEqual                  // 双方改成相同内容（自动采纳）
	hunkConflict                   // 双方都改且不同
)

// computeHunks 是简化版 diff3：
//
//   1. 用 LCS(base, ours) 找出 base 与 ours 的差异区间
//   2. 用 LCS(base, theirs) 找出 base 与 theirs 的差异区间
//   3. 把两个差异序列按 base 索引对齐合成 hunks
//
// 当前实现是 O(N*M)（足以应对单文件几千行场景）。
func computeHunks(base, ours, theirs []string) []hunk {
	oursDelta := diffByLine(base, ours)
	theirsDelta := diffByLine(base, theirs)

	// 把 deltas 按 base 索引对齐
	hunks := []hunk{}
	bi := 0 // base index
	oi := 0 // index into oursDelta
	ti := 0 // index into theirsDelta

	for bi <= len(base) {
		// 拿到 oursDelta / theirsDelta 中下一个开始位置 >= bi 的 hunk
		var oursH, theirsH *deltaHunk
		if oi < len(oursDelta) && oursDelta[oi].baseStart == bi {
			oursH = &oursDelta[oi]
		}
		if ti < len(theirsDelta) && theirsDelta[ti].baseStart == bi {
			theirsH = &theirsDelta[ti]
		}

		switch {
		case oursH == nil && theirsH == nil:
			// 三方在当前位置都未改：消费一行 stable
			if bi == len(base) {
				// 走到末尾
				bi++
			} else {
				hunks = append(hunks, hunk{
					kind: hunkStable,
					base: []string{base[bi]},
				})
				bi++
			}
		case oursH != nil && theirsH == nil:
			hunks = append(hunks, hunk{
				kind:   hunkOursOnly,
				base:   base[oursH.baseStart:oursH.baseEnd],
				ours:   oursH.replacement,
				theirs: base[oursH.baseStart:oursH.baseEnd],
			})
			bi = oursH.baseEnd
			oi++
		case oursH == nil && theirsH != nil:
			hunks = append(hunks, hunk{
				kind:   hunkTheirsOnly,
				base:   base[theirsH.baseStart:theirsH.baseEnd],
				ours:   base[theirsH.baseStart:theirsH.baseEnd],
				theirs: theirsH.replacement,
			})
			bi = theirsH.baseEnd
			ti++
		default:
			// 两侧都改了相同区间或重叠区间
			endBase := oursH.baseEnd
			if theirsH.baseEnd > endBase {
				endBase = theirsH.baseEnd
			}
			h := hunk{
				base:   base[bi:endBase],
				ours:   oursH.replacement,
				theirs: theirsH.replacement,
			}
			if equalLines(h.ours, h.theirs) {
				h.kind = hunkBothEqual
			} else {
				h.kind = hunkConflict
			}
			hunks = append(hunks, h)
			bi = endBase
			oi++
			ti++
		}
	}
	return hunks
}

// deltaHunk 是 diffByLine 的输出条目：base[baseStart:baseEnd] 被 replacement 替换。
type deltaHunk struct {
	baseStart   int
	baseEnd     int
	replacement []string
}

// diffByLine 计算 base → other 的差异条目（按行 LCS）。
func diffByLine(base, other []string) []deltaHunk {
	// 构造 LCS 长度表
	n, m := len(base), len(other)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if base[i-1] == other[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// 逆推得到逐行 op
	type op struct {
		kind byte // '=' / '-' / '+'
		base int
		other int
	}
	var ops []op
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && base[i-1] == other[j-1]:
			ops = append(ops, op{'=', i - 1, j - 1})
			i--
			j--
		case j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]):
			ops = append(ops, op{'+', -1, j - 1})
			j--
		default:
			ops = append(ops, op{'-', i - 1, -1})
			i--
		}
	}
	for k, l := 0, len(ops)-1; k < l; k, l = k+1, l-1 {
		ops[k], ops[l] = ops[l], ops[k]
	}

	// 把连续的 '-'/'+' 合并为一个 deltaHunk
	var hunks []deltaHunk
	k := 0
	for k < len(ops) {
		if ops[k].kind == '=' {
			k++
			continue
		}
		hunkStart := k
		for k < len(ops) && ops[k].kind != '=' {
			k++
		}
		// [hunkStart, k) 是连续的非 '=' 序列
		bs := -1
		be := -1
		var repl []string
		for _, o := range ops[hunkStart:k] {
			switch o.kind {
			case '-':
				if bs == -1 {
					bs = o.base
				}
				be = o.base + 1
			case '+':
				repl = append(repl, other[o.other])
			}
		}
		if bs == -1 {
			// 纯插入：base 索引取 hunkStart 之前的 '=' 末尾
			if hunkStart == 0 {
				bs = 0
			} else {
				bs = ops[hunkStart-1].base + 1
			}
			be = bs
		}
		hunks = append(hunks, deltaHunk{baseStart: bs, baseEnd: be, replacement: repl})
	}
	return hunks
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// splitLines 按 \n 切分但保留行尾换行符。
func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// writeConflictMarkers 输出 git 风格的 conflict markers。
func writeConflictMarkers(buf *bytes.Buffer, h hunk, lbl Labels) {
	const sz = 7
	mark := strings.Repeat("<", sz)
	sep := strings.Repeat("=", sz)
	end := strings.Repeat(">", sz)

	if lbl.Ours == "" {
		buf.WriteString(fmt.Sprintf("%s\n", mark))
	} else {
		buf.WriteString(fmt.Sprintf("%s %s\n", mark, lbl.Ours))
	}
	for _, ln := range h.ours {
		buf.WriteString(ln)
	}

	// diff3 模式：插入 base
	mid := strings.Repeat("|", sz)
	if lbl.Base == "" {
		buf.WriteString(fmt.Sprintf("%s\n", mid))
	} else {
		buf.WriteString(fmt.Sprintf("%s %s\n", mid, lbl.Base))
	}
	for _, ln := range h.base {
		buf.WriteString(ln)
	}

	buf.WriteString(fmt.Sprintf("%s\n", sep))
	for _, ln := range h.theirs {
		buf.WriteString(ln)
	}
	if lbl.Theirs == "" {
		buf.WriteString(fmt.Sprintf("%s\n", end))
	} else {
		buf.WriteString(fmt.Sprintf("%s %s\n", end, lbl.Theirs))
	}
}
