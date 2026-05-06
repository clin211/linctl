package {{.Resource | Lower}}

import (
	"context"

	"{{.Module}}/internal/{{.AppName}}/store"
	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// {{.Resource | Pascal}}Biz 定义 {{.Resource | Pascal}} 业务逻辑的方法集。
type {{.Resource | Pascal}}Biz interface {
	Create(ctx context.Context, req *v1.Create{{.Resource | Pascal}}Request) (*v1.Create{{.Resource | Pascal}}Response, error)
	Update(ctx context.Context, req *v1.Update{{.Resource | Pascal}}Request) (*v1.Update{{.Resource | Pascal}}Response, error)
	Delete(ctx context.Context, req *v1.Delete{{.Resource | Pascal}}Request) (*v1.Delete{{.Resource | Pascal}}Response, error)
	Get(ctx context.Context, req *v1.Get{{.Resource | Pascal}}Request) (*v1.Get{{.Resource | Pascal}}Response, error)
	List(ctx context.Context, req *v1.List{{.Resource | Pascal}}Request) (*v1.List{{.Resource | Pascal}}Response, error)
}

// {{.Resource | LowerCamel}}Biz 是 {{.Resource | Pascal}}Biz 的具体实现。
type {{.Resource | LowerCamel}}Biz struct {
	store store.IStore
}

// 确保 {{.Resource | LowerCamel}}Biz 实现了 {{.Resource | Pascal}}Biz 接口。
var _ {{.Resource | Pascal}}Biz = (*{{.Resource | LowerCamel}}Biz)(nil)

// New 创建一个 {{.Resource | Pascal}}Biz 实例。
func New(s store.IStore) *{{.Resource | LowerCamel}}Biz {
	return &{{.Resource | LowerCamel}}Biz{store: s}
}
