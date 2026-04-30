package {{.Resource | Lower}}

import (
	"context"

	"{{.Module}}/internal/{{.AppName}}/store"
	v1 "{{.Module}}/pkg/api/{{.AppName}}/v1"
)

// {{.Resource | Pascal}}Biz defines the methods for {{.Resource | Pascal}} business logic.
type {{.Resource | Pascal}}Biz interface {
	Create(ctx context.Context, req *v1.Create{{.Resource | Pascal}}Request) (*v1.Create{{.Resource | Pascal}}Response, error)
	Update(ctx context.Context, req *v1.Update{{.Resource | Pascal}}Request) (*v1.Update{{.Resource | Pascal}}Response, error)
	Delete(ctx context.Context, req *v1.Delete{{.Resource | Pascal}}Request) (*v1.Delete{{.Resource | Pascal}}Response, error)
	Get(ctx context.Context, req *v1.Get{{.Resource | Pascal}}Request) (*v1.Get{{.Resource | Pascal}}Response, error)
	List(ctx context.Context, req *v1.List{{.Resource | Pascal}}Request) (*v1.List{{.Resource | Pascal}}Response, error)
}

// {{.Resource | LowerCamel}}Biz is the concrete implementation of {{.Resource | Pascal}}Biz.
type {{.Resource | LowerCamel}}Biz struct {
	store store.IStore
}

// Ensure {{.Resource | LowerCamel}}Biz implements {{.Resource | Pascal}}Biz.
var _ {{.Resource | Pascal}}Biz = (*{{.Resource | LowerCamel}}Biz)(nil)

// New creates a new {{.Resource | Pascal}}Biz instance.
func New(s store.IStore) *{{.Resource | LowerCamel}}Biz {
	return &{{.Resource | LowerCamel}}Biz{store: s}
}
