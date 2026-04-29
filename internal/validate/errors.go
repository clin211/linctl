package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/clin211/linctl/internal/linctlerr"
)

// FieldIssue 是单条字段校验失败的结构化描述。
//
// 字段语义：
//   - Field：人类可读的字段路径（例如 "spec.components[0].framework"）
//   - Tag：失败的 validate tag（如 "oneof"、"required"、"projectname"）
//   - Param：tag 的参数（如 oneof 的可选值列表）
//   - Value：当前实际值的字符串表达
//   - Message：渲染好的中性英文描述
//   - Hint：来自 [HintForTag] 的修复建议
type FieldIssue struct {
	Field   string `json:"field"             yaml:"field"`
	Tag     string `json:"tag"               yaml:"tag"`
	Param   string `json:"param,omitempty"   yaml:"param,omitempty"`
	Value   string `json:"value,omitempty"   yaml:"value,omitempty"`
	Message string `json:"message"           yaml:"message"`
	Hint    string `json:"hint,omitempty"    yaml:"hint,omitempty"`
}

// String 渲染为单行格式："spec.components[0].framework: must be one of [gin grpc] (got \"foo\")"
func (f FieldIssue) String() string {
	if f.Value != "" {
		return fmt.Sprintf("%s: %s (got %q)", f.Field, f.Message, f.Value)
	}
	return fmt.Sprintf("%s: %s", f.Field, f.Message)
}

// ToLinctlError 把 validator.ValidationErrors 转换为 *linctlerr.LinctlError，
// 自动聚合所有字段错误并渲染 hint。
//
// 边界处理：
//   - err 为 nil → 返回 nil
//   - err 不是 validator.ValidationErrors → wrap 为 ErrConfigInvalid，保留 cause
//   - err 为空 ValidationErrors → 视作"未通过但无明细"，返回带通用 hint 的错误
//
// 返回的 LinctlError：
//   - Code: linctlerr.ErrConfigInvalid
//   - Message: "schema validation failed: <逗号分隔的字段:消息>"
//   - Hint: 多行；每个失败字段一行 + HintForTag 给出的修复建议
//   - Cause: 原 ValidationErrors（保留链）
func ToLinctlError(err error) *linctlerr.LinctlError {
	if err == nil {
		return nil
	}

	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return linctlerr.Wrap(linctlerr.ErrConfigInvalid, err, "schema validation")
	}
	if len(ves) == 0 {
		return linctlerr.Wrap(linctlerr.ErrConfigInvalid, err, "schema validation").
			WithHint("validator returned empty error list; please check the spec manually")
	}

	issues := make([]FieldIssue, 0, len(ves))
	msgParts := make([]string, 0, len(ves))
	hintParts := make([]string, 0, len(ves))

	for _, fe := range ves {
		issue := newIssue(fe)
		issues = append(issues, issue)
		msgParts = append(msgParts, issue.String())
		if issue.Hint != "" {
			hintParts = append(hintParts, fmt.Sprintf("- %s: %s", issue.Field, issue.Hint))
		}
	}

	out := linctlerr.Wrap(
		linctlerr.ErrConfigInvalid,
		err,
		fmt.Sprintf("schema validation failed: %s", strings.Join(msgParts, "; ")),
	)
	if len(hintParts) > 0 {
		out = out.WithHint(strings.Join(hintParts, "\n"))
	}
	return out
}

// IssuesFromError 把任意 error 抽取为 []FieldIssue（用于 --output json/yaml 模式下的结构化输出）。
//
// 非 validator.ValidationErrors 返回 nil。
func IssuesFromError(err error) []FieldIssue {
	if err == nil {
		return nil
	}
	var ves validator.ValidationErrors
	if !errors.As(err, &ves) {
		return nil
	}
	out := make([]FieldIssue, 0, len(ves))
	for _, fe := range ves {
		out = append(out, newIssue(fe))
	}
	return out
}

// newIssue 把单个 validator.FieldError 抽取为 FieldIssue。
func newIssue(fe validator.FieldError) FieldIssue {
	field := normalizeFieldPath(fe.Namespace())
	tag := fe.Tag()
	param := fe.Param()
	val := fmt.Sprintf("%v", fe.Value())
	if val == "<nil>" {
		val = ""
	}
	return FieldIssue{
		Field:   field,
		Tag:     tag,
		Param:   param,
		Value:   val,
		Message: messageForTag(tag, param),
		Hint:    HintForTag(tag),
	}
}

// normalizeFieldPath 把 validator 默认的 PascalCase 路径转换为更友好的 yaml 风格小写驼峰。
//
// validator.FieldError.Namespace() 例：
//   - "Project.Spec.Components[0].Framework"
//
// 转换后：
//   - "spec.components[0].framework"
//
// 我们丢弃首段（顶层类型名），并把每段首字母转小写。
func normalizeFieldPath(ns string) string {
	if ns == "" {
		return ""
	}
	parts := strings.Split(ns, ".")
	if len(parts) > 1 {
		parts = parts[1:] // 丢弃 "Project"
	}
	for i, p := range parts {
		parts[i] = lowerFirstWithIndex(p)
	}
	return strings.Join(parts, ".")
}

// lowerFirstWithIndex 把 "Components[0]" 这种带方括号下标的段转换为 "components[0]"，
// 对纯单词段（"Framework"）转换为 "framework"。
func lowerFirstWithIndex(s string) string {
	if s == "" {
		return s
	}
	bracket := strings.IndexByte(s, '[')
	head := s
	tail := ""
	if bracket >= 0 {
		head = s[:bracket]
		tail = s[bracket:]
	}
	if head == "" {
		return s
	}
	first := head[0]
	if first >= 'A' && first <= 'Z' {
		head = string(first+32) + head[1:]
	}
	return head + tail
}

// messageForTag 给 tag 渲染中性英文消息。未知 tag 返回 "validation rule '<tag>' failed"。
func messageForTag(tag, param string) string {
	switch tag {
	case "required":
		return "field is required"
	case "oneof":
		return fmt.Sprintf("must be one of [%s]", param)
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "len":
		return fmt.Sprintf("length must be %s", param)
	case "email":
		return "must be a valid email"
	case "hostname_port":
		return "must be in 'host:port' format"
	case "startswith":
		return fmt.Sprintf("must start with %q", param)
	case "modulepath":
		return "must be a valid Go module path (e.g. github.com/foo/bar)"
	case "projectname":
		return "must be kebab-case (1-40 chars, lowercase, digits, dashes)"
	case "kindname":
		return "must be kind-name style (1-41 chars, [A-Za-z0-9_/-], starts with letter)"
	case "featurename":
		return "must be kebab-case (1-41 chars)"
	case "componentname":
		return "must be kebab-case (1-41 chars)"
	}
	return fmt.Sprintf("validation rule %q failed", tag)
}
