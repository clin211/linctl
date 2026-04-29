// Package validate 封装 go-playground/validator/v10，并注册 linctl 项目所需的
// 自定义校验规则（modulePath / projectName / kindName / featureName / componentName）。
//
// 设计要点（与 docs/04-config-schema.md §4.5 严格对齐）：
//   - 全局 Validator 单例，并发安全
//   - 自定义规则的 regexp 在包初始化期 MustCompile 一次，运行时仅执行 MatchString
//   - 任何 error 通过 internal/linctlerr.LinctlError 暴露，自带 hint
package validate

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// 自定义校验规则使用的预编译正则（全局编译一次，运行时仅 MatchString）。
//
// 严格遵循 docs/04-config-schema.md §4.5：
//   - reModulePath：Go module 路径（如 github.com/clin211/myblog）
//   - reProjectName：kebab-case 项目名（小写起头，1..40 长度）
//   - reKindName：宽松大小写的 Pascal/Snake 名（用于 Resource / NamedSpec.Name）
//   - reFeatureName：kebab-case 的 feature 名
//   - reCompName：kebab-case 的 component 名
var (
	reModulePath  = regexp.MustCompile(`^([a-zA-Z0-9\-]+\.)+[a-zA-Z0-9\-]+(/[a-zA-Z0-9_.\-]+)*$`)
	reProjectName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)
	reKindName    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_/-]{0,40}$`)
	reFeatureName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)
	reCompName    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)
)

// customRule 把 tag 名与对应的 validator.Func 绑在一起。
type customRule struct {
	tag string
	fn  validator.Func
}

// allCustomRules 返回所有需要注册的规则。函数式而非全局变量，便于测试时按需重置。
func allCustomRules() []customRule {
	return []customRule{
		{"modulepath", validateModulePath},
		{"projectname", validateProjectName},
		{"kindname", validateKindName},
		{"featurename", validateFeatureName},
		{"componentname", validateComponentName},
	}
}

// RegisterCustomRules 把 linctl 自定义的所有 validator 规则注册到给定的 *validator.Validate。
//
// 重复注册相同 tag 会返回 error；调用方通常只需一次注册（参见 [NewValidator]）。
func RegisterCustomRules(v *validator.Validate) error {
	if v == nil {
		return nil
	}
	for _, r := range allCustomRules() {
		if err := v.RegisterValidation(r.tag, r.fn); err != nil {
			return err
		}
	}
	return nil
}

// validateModulePath 校验 Go module 路径，例如 `github.com/foo/bar` 或 `gitlab.com/x/y/z.v2`。
//
// 规则：至少一个含点的域名段 + 0..N 个 `/segment` 子路径段。
func validateModulePath(fl validator.FieldLevel) bool {
	return reModulePath.MatchString(fl.Field().String())
}

// validateProjectName 校验项目名（用于生成目录、二进制名）。kebab-case，小写起头。
func validateProjectName(fl validator.FieldLevel) bool {
	return reProjectName.MatchString(fl.Field().String())
}

// validateKindName 校验 Kubernetes-style 宽松命名（兼容 Pascal / snake_case 与路径）。
func validateKindName(fl validator.FieldLevel) bool {
	return reKindName.MatchString(fl.Field().String())
}

// validateFeatureName 校验 feature 名（与 internal/feature 注册中心 key 一致）。
func validateFeatureName(fl validator.FieldLevel) bool {
	return reFeatureName.MatchString(fl.Field().String())
}

// validateComponentName 校验 component 名（与 metadata.name 同样的 kebab-case 规则）。
func validateComponentName(fl validator.FieldLevel) bool {
	return reCompName.MatchString(fl.Field().String())
}

// HintForTag 返回某个失败 tag 的修复建议（用户可读）。未知 tag 返回空字符串。
//
// 该函数被 [errors.go] 用于把 validator.ValidationErrors 渲染为带 Hint 的 LinctlError。
func HintForTag(tag string) string {
	switch tag {
	case "modulepath":
		return "Module path must look like 'github.com/<org>/<repo>', e.g. github.com/clin211/lin."
	case "projectname":
		return "Project name must be kebab-case: lowercase letters, digits, dashes; 1-40 chars; start with a letter."
	case "kindname":
		return "Kind name must start with a letter and use only [A-Za-z0-9_/-]; max 41 chars."
	case "featurename":
		return "Feature name must be kebab-case: lowercase letters, digits, dashes; max 41 chars."
	case "componentname":
		return "Component name must be kebab-case: lowercase letters, digits, dashes; max 41 chars."
	case "required":
		return "Field is required."
	case "oneof":
		return "Value must be one of the allowed enum values."
	case "min":
		return "Value is below the minimum."
	case "max":
		return "Value exceeds the maximum."
	case "email":
		return "Value must be a valid email address (RFC 5321)."
	case "hostname_port":
		return "Value must be in 'host:port' format, e.g. kafka.svc:9092."
	case "startswith":
		return "Value must start with the required prefix."
	}
	return ""
}
