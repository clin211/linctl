package validate

import (
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Validator 是 linctl 全局共享的校验器。它包装 go-playground/validator/v10，
// 注册了 linctl 自定义规则（modulePath / projectName / kindName / ...）以及
// 字段名提取器（优先用 yaml tag 而不是 Go field 名作为错误路径）。
//
// **并发安全**：底层 *validator.Validate 实例是 goroutine-safe，可任意并发调用 Struct。
//
// 推荐使用 [Default] 拿到全局单例；只在写需要替换/扩展规则的高级场景中才创建独立 Validator。
type Validator struct {
	v *validator.Validate
}

// NewValidator 返回一个全新的 Validator 实例，已注册 linctl 全部自定义规则。
//
// 如果注册自定义规则失败，会 panic（属于编程错误而非运行时错误：tag 名重复或为空）。
func NewValidator() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(extractYAMLName)

	if err := RegisterCustomRules(v); err != nil {
		panic("validate: failed to register custom rules: " + err.Error())
	}
	return &Validator{v: v}
}

var (
	defaultOnce      sync.Once
	defaultValidator *Validator
)

// Default 返回进程级别的单例 Validator。第一次调用时初始化。
//
// 推荐所有业务代码通过 Default() 拿到 Validator；显式 NewValidator() 仅用于
// 测试隔离或需要替换规则的场景。
func Default() *Validator {
	defaultOnce.Do(func() {
		defaultValidator = NewValidator()
	})
	return defaultValidator
}

// Struct 校验给定 struct 指针，并返回 LinctlError（带 hint 的友好错误）。
//
// 边界：
//   - s == nil → 返回 nil（视为"无校验目标"，与 validator/v10 行为一致）
//   - s 非 struct/map → validator 自身会返回 InvalidValidationError，本方法 wrap 为 LinctlError
func (v *Validator) Struct(s any) error {
	if v == nil || v.v == nil {
		return nil
	}
	if s == nil {
		return nil
	}
	if err := v.v.Struct(s); err != nil {
		return ToLinctlError(err)
	}
	return nil
}

// Var 校验单个值（不嵌入到 struct 时使用）。tag 即 validate tag 字符串，例如 "required,oneof=a b"。
//
// 用于 CLI flag 等场景的轻量校验。
func (v *Validator) Var(value any, tag string) error {
	if v == nil || v.v == nil {
		return nil
	}
	if err := v.v.Var(value, tag); err != nil {
		return ToLinctlError(err)
	}
	return nil
}

// Raw 暴露底层 *validator.Validate，便于调用方注册额外规则或 struct-level 校验。
//
// **使用提醒**：注册扩展规则后请确保不与 linctl 内置 tag 名冲突。
func (v *Validator) Raw() *validator.Validate {
	if v == nil {
		return nil
	}
	return v.v
}

// extractYAMLName 让 validator 在生成错误路径时优先使用 yaml tag 名（而不是 Go 字段名）。
//
// 例如：字段 `Framework string yaml:"framework,omitempty"` 在错误中会显示为
// "framework" 而不是 "Framework"。
//
// 没有 yaml tag 或被 yaml:"-" 屏蔽时，回退使用 Go 字段名（保留 PascalCase，
// 由后续 normalizeFieldPath 再转小写）。
func extractYAMLName(fld reflect.StructField) string {
	tag := fld.Tag.Get("yaml")
	if tag == "" {
		return fld.Name
	}
	name := strings.SplitN(tag, ",", 2)[0]
	if name == "-" || name == "" {
		return fld.Name
	}
	return name
}
