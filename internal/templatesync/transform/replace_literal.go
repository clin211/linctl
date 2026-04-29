package transform

import (
	"context"
	"strings"
)

// ReplaceLiteral 是字面量替换 transform。
//
// 与 RewriteImports 互补：后者只改 Go import 路径；本 transform 改任何字符串
// 出现位置（含字面量、注释、文件名提示等）。配合 sync.yaml 例如：
//
//	- kind: replaceLiteral
//	  pairs:
//	    - from: blog-apiserver
//	      to:   '{{`{{ .Component.Name }}`}}'
//	    - from: github.com/clin211/miniblog-v4
//	      to:   '{{`{{ .Project.Module }}`}}'
//
// 注意事项：
//   - pairs 顺序敏感：先做长字符串、再做短字符串，避免短串提前替换破坏长串。
//   - 本 transform 不区分 Go / proto / yaml；大文件场景下用 strings.NewReplacer
//     一次性批量替换（O(n + m)）。
type ReplaceLiteral struct {
	Pairs []LiteralPair
}

// Kind implements Transform.
func (r *ReplaceLiteral) Kind() string { return "replaceLiteral" }

// Apply implements Transform.
func (r *ReplaceLiteral) Apply(_ context.Context, rc *RuntimeContext, content []byte) ([]byte, error) {
	if len(r.Pairs) == 0 {
		return content, nil
	}
	// 解析 manifest 占位
	args := make([]string, 0, len(r.Pairs)*2)
	for _, p := range r.Pairs {
		from := p.From
		to := p.To
		if rc.Manifest != nil {
			from = expandManifestRefs(from, rc.Manifest)
			to = expandManifestRefs(to, rc.Manifest)
		}
		if from == "" {
			continue
		}
		args = append(args, from, to)
	}
	if len(args) == 0 {
		return content, nil
	}
	r2 := strings.NewReplacer(args...)
	return []byte(r2.Replace(string(content))), nil
}

// replaceLiteralFactory 解析 sync.yaml 中的 pairs 字段构造 ReplaceLiteral。
func replaceLiteralFactory(rawCfg map[string]any) (Transform, error) {
	pairs, err := asPairs(rawCfg, "pairs")
	if err != nil {
		return nil, err
	}
	// 兼容简写：直接 from/to 单对
	from, _ := asString(rawCfg, "from", false)
	to, _ := asString(rawCfg, "to", false)
	if from != "" && to != "" {
		pairs = append(pairs, LiteralPair{From: from, To: to})
	}
	return &ReplaceLiteral{Pairs: pairs}, nil
}

func init() {
	DefaultRegistry.MustRegister("replaceLiteral", replaceLiteralFactory)
}
