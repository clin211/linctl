package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// List retrieves a list of {{.Resource | Pascal}} entries.
func (b *{{.Resource | LowerCamel}}Biz) List(ctx context.Context, req *v1.List{{.Resource | Pascal}}Request) (*v1.List{{.Resource | Pascal}}Response, error) {
	// TODO: implement {{.Resource | Pascal}} list logic
	return &v1.List{{.Resource | Pascal}}Response{}, nil
}
