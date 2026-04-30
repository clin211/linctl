package validation

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// ValidateCreate{{.Resource | Pascal}}Request validates the Create{{.Resource | Pascal}} request.
func (v *Validator) ValidateCreate{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Create{{.Resource | Pascal}}Request) error {
	// TODO: add field-level validation
	return nil
}

// ValidateUpdate{{.Resource | Pascal}}Request validates the Update{{.Resource | Pascal}} request.
func (v *Validator) ValidateUpdate{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Update{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateDelete{{.Resource | Pascal}}Request validates the Delete{{.Resource | Pascal}} request.
func (v *Validator) ValidateDelete{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Delete{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateGet{{.Resource | Pascal}}Request validates the Get{{.Resource | Pascal}} request.
func (v *Validator) ValidateGet{{.Resource | Pascal}}Request(_ context.Context, _ *v1.Get{{.Resource | Pascal}}Request) error {
	return nil
}

// ValidateList{{.Resource | Pascal}}Request validates the List{{.Resource | Pascal}} request.
func (v *Validator) ValidateList{{.Resource | Pascal}}Request(_ context.Context, _ *v1.List{{.Resource | Pascal}}Request) error {
	return nil
}
