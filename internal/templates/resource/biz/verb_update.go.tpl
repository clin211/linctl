package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Update updates an existing {{.Resource | Pascal}}.
func (b *{{.Resource | LowerCamel}}Biz) Update(ctx context.Context, req *v1.Update{{.Resource | Pascal}}Request) (*v1.Update{{.Resource | Pascal}}Response, error) {
	// TODO: implement {{.Resource | Pascal}} update logic
	return &v1.Update{{.Resource | Pascal}}Response{}, nil
}
