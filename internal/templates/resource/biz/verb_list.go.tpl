package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// List 获取 {{.Resource | Pascal}} 列表。
func (b *{{.Resource | LowerCamel}}Biz) List(ctx context.Context, req *v1.List{{.Resource | Pascal}}Request) (*v1.List{{.Resource | Pascal}}Response, error) {
	// TODO：实现 {{.Resource | Pascal}} 的列表查询逻辑
	return &v1.List{{.Resource | Pascal}}Response{}, nil
}
