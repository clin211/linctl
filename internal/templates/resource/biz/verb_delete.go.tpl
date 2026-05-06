package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Delete 删除指定的 {{.Resource | Pascal}}。
func (b *{{.Resource | LowerCamel}}Biz) Delete(ctx context.Context, req *v1.Delete{{.Resource | Pascal}}Request) (*v1.Delete{{.Resource | Pascal}}Response, error) {
	// TODO：实现 {{.Resource | Pascal}} 的删除逻辑
	return &v1.Delete{{.Resource | Pascal}}Response{}, nil
}
