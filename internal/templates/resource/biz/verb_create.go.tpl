package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Create creates a new {{.Resource | Pascal}}.
func (b *{{.Resource | LowerCamel}}Biz) Create(ctx context.Context, req *v1.Create{{.Resource | Pascal}}Request) (*v1.Create{{.Resource | Pascal}}Response, error) {
	// TODO: implement {{.Resource | Pascal}} creation logic
	// 1. convert request to model
	// 2. call store
	// 3. convert model to response
	return &v1.Create{{.Resource | Pascal}}Response{}, nil
}
