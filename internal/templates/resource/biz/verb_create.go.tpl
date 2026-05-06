package {{.Resource | Lower}}

import (
	"context"

	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// Create 创建一个新的 {{.Resource | Pascal}}。
func (b *{{.Resource | LowerCamel}}Biz) Create(ctx context.Context, req *v1.Create{{.Resource | Pascal}}Request) (*v1.Create{{.Resource | Pascal}}Response, error) {
	// TODO：实现 {{.Resource | Pascal}} 的创建逻辑
	// 1. 将请求转换为 model
	// 2. 调用 store
	// 3. 将 model 转换为响应
	return &v1.Create{{.Resource | Pascal}}Response{}, nil
}
