package validation

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// ValidateCreate{{.Resource | Pascal}}Request 校验 Create{{.Resource | Pascal}} 请求。
func (v *Validator) ValidateCreate{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Create{{.Resource | Pascal}}Request) error {
	// TODO：补充字段级校验
	return nil
}

// ValidateUpdate{{.Resource | Pascal}}Request 校验 Update{{.Resource | Pascal}} 请求。
func (v *Validator) ValidateUpdate{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Update{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateDelete{{.Resource | Pascal}}Request 校验 Delete{{.Resource | Pascal}} 请求。
func (v *Validator) ValidateDelete{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Delete{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateGet{{.Resource | Pascal}}Request 校验 Get{{.Resource | Pascal}} 请求。
func (v *Validator) ValidateGet{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Get{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateList{{.Resource | Pascal}}Request 校验 List{{.Resource | Pascal}} 请求。
func (v *Validator) ValidateList{{.Resource | Pascal}}Request(_ context.Context, _ *v1.List{{.Resource | Pascal}}Request) error {
	return nil
}
