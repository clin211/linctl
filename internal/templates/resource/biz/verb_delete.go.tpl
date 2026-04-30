package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Delete deletes a {{.Resource | Pascal}}.
func (b *{{.Resource | LowerCamel}}Biz) Delete(ctx context.Context, req *v1.Delete{{.Resource | Pascal}}Request) (*v1.Delete{{.Resource | Pascal}}Response, error) {
	// TODO: implement {{.Resource | Pascal}} deletion logic
	return &v1.Delete{{.Resource | Pascal}}Response{}, nil
}
