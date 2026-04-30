package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Get retrieves a single {{.Resource | Pascal}}.
func (b *{{.Resource | LowerCamel}}Biz) Get(ctx context.Context, req *v1.Get{{.Resource | Pascal}}Request) (*v1.Get{{.Resource | Pascal}}Response, error) {
	// TODO: implement {{.Resource | Pascal}} get logic
	return &v1.Get{{.Resource | Pascal}}Response{}, nil
}
