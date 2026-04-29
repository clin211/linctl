package template

// TemplateData 是渲染上下文的标准数据形状。
//
// 所有 linctl 内置模板都接收此结构作为根数据。Feature / Component 在 Apply 时
// 可以通过 Custom 字段附加额外数据。
//
// 字段语义：
//   - Project：当前项目的完整 spec（已 ApplyDefaults + Validate）
//   - Component：当前正在渲染的 Component
//   - Feature：当前正在贡献 Pair 的 Feature 名（可选；非 Feature 触发的渲染留空）
//   - CLIVersion：linctl 二进制版本，便于在生成代码中标注
//   - Custom：Feature/Component 自定义透传数据
//
// 注意：Project / Component 类型用 any 而非具体类型，避免本包对 internal/project 的强依赖
// （后者在 Stage B-1 完成后可在 Adapter 层做强类型化）。
type TemplateData struct {
	Project    any            `json:"project"    yaml:"project"`
	Component  any            `json:"component"  yaml:"component"`
	Feature    string         `json:"feature,omitempty"    yaml:"feature,omitempty"`
	CLIVersion string         `json:"cliVersion,omitempty" yaml:"cliVersion,omitempty"`
	Custom     map[string]any `json:"custom,omitempty"     yaml:"custom,omitempty"`
}

// WithCustom 返回一个浅拷贝，并设置/合并 custom 字段。便于链式构造。
func (d TemplateData) WithCustom(key string, value any) TemplateData {
	cp := d
	if cp.Custom == nil {
		cp.Custom = make(map[string]any, 4)
	} else {
		// 浅拷贝以保持不可变性
		next := make(map[string]any, len(cp.Custom)+1)
		for k, v := range cp.Custom {
			next[k] = v
		}
		cp.Custom = next
	}
	cp.Custom[key] = value
	return cp
}
