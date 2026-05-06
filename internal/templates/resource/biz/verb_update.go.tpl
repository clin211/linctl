package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Update 更新已存在的 {{.Resource | Pascal}}。
func (b *{{.Resource | LowerCamel}}Biz) Update(ctx context.Context, req *v1.Update{{.Resource | Pascal}}Request) (*v1.Update{{.Resource | Pascal}}Response, error) {
	// TODO：实现 {{.Resource | Pascal}} 的更新逻辑
	return &v1.Update{{.Resource | Pascal}}Response{}, nil
}
