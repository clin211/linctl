package transform

import (
	"context"
	"path/filepath"
	"strings"
)

// AddExtension 是路径变换 transform：本身不改 content，仅在 dst 路径推断阶段
// 用于把上游 src 的文件名转换为 lin 镜像内 dst 文件名（如加 .tpl 后缀）。
//
// 设计要点：
//   - sync.yaml 的 files[].dst 是显式声明的，正常情况下不需要本 transform
//     在内容阶段做任何事。
//   - 但 templatesync.Runner 在 status 命令中**反向推断**：上游某个新文件应该
//     映射到镜像下哪个 dst 路径。这时使用本 transform 的 InferDst 辅助函数。
//   - Apply() 对 content 是 no-op（保留接口一致性）。
//
// sync.yaml 配置：
//
//	- kind: addExtension
//	  suffix: .tpl
//	  appliesTo: ['*.go', '*.proto', '*.yaml']
type AddExtension struct {
	Suffix    string
	AppliesTo []string // 文件名 glob 列表（如 "*.go"）
}

// Kind implements Transform.
func (a *AddExtension) Kind() string { return "addExtension" }

// Apply implements Transform —— content 阶段为 no-op。
func (a *AddExtension) Apply(_ context.Context, _ *RuntimeContext, content []byte) ([]byte, error) {
	return content, nil
}

// InferDst 根据上游 src 路径推断对应的 dst 路径。
//
// 例如 a.Suffix=".tpl"，src="internal/pkg/contextx/contextx.go"
//
//	→ "internal/pkg/contextx/contextx.go.tpl"
//
// 当 src 不命中 AppliesTo 中任何 glob 时返回 src 原路径（pass-through）。
func (a *AddExtension) InferDst(src string) string {
	if a.Suffix == "" {
		return src
	}
	if !a.matchesAppliesTo(src) {
		return src
	}
	return src + a.Suffix
}

func (a *AddExtension) matchesAppliesTo(src string) bool {
	if len(a.AppliesTo) == 0 {
		return true
	}
	base := filepath.Base(src)
	for _, pat := range a.AppliesTo {
		ok, err := filepath.Match(pat, base)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func addExtensionFactory(rawCfg map[string]any) (Transform, error) {
	suffix, _ := asString(rawCfg, "suffix", false)
	if suffix == "" {
		suffix = ".tpl"
	}
	appliesTo, err := asStringSlice(rawCfg, "appliesTo")
	if err != nil {
		return nil, err
	}
	// 兼容老格式 "ext" 单字符串（不推荐）
	if appliesTo == nil {
		if extStr, _ := asString(rawCfg, "ext", false); extStr != "" {
			appliesTo = []string{"*" + strings.TrimPrefix(extStr, "*")}
		}
	}
	return &AddExtension{Suffix: suffix, AppliesTo: appliesTo}, nil
}

func init() {
	DefaultRegistry.MustRegister("addExtension", addExtensionFactory)
}
