package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Get 获取单个 {{.Resource | Pascal}}。
func (b *{{.Resource | LowerCamel}}Biz) Get(ctx context.Context, req *v1.Get{{.Resource | Pascal}}Request) (*v1.Get{{.Resource | Pascal}}Response, error) {
	// TODO：实现 {{.Resource | Pascal}} 的查询逻辑
	return &v1.Get{{.Resource | Pascal}}Response{}, nil
}
