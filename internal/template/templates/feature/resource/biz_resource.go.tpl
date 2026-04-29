package biz

import (
	"context"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
)

// {{ .Custom.ResourcePascal }}Biz 暴露 {{ .Custom.ResourceLower }} 资源的业务方法。
type {{ .Custom.ResourcePascal }}Biz interface {
	List(ctx context.Context) ([]*{{ .Custom.ResourcePascal }}, error)
	Get(ctx context.Context, id int64) (*{{ .Custom.ResourcePascal }}, error)
	Create(ctx context.Context, in *Create{{ .Custom.ResourcePascal }}Request) error
	Update(ctx context.Context, id int64, in *Update{{ .Custom.ResourcePascal }}Request) error
	Delete(ctx context.Context, id int64) error
}

// {{ .Custom.ResourcePascal }} 是 {{ .Custom.ResourceLower }} 的业务模型。
type {{ .Custom.ResourcePascal }} struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Create{{ .Custom.ResourcePascal }}Request 是创建 {{ .Custom.ResourceLower }} 的请求体。
type Create{{ .Custom.ResourcePascal }}Request struct {
	Name string `json:"name" binding:"required"`
}

// Update{{ .Custom.ResourcePascal }}Request 是更新 {{ .Custom.ResourceLower }} 的请求体。
type Update{{ .Custom.ResourcePascal }}Request struct {
	Name string `json:"name"`
}

// {{ .Custom.ResourceLower }}Biz 是 {{ .Custom.ResourcePascal }}Biz 的默认实现。
type {{ .Custom.ResourceLower }}Biz struct {
	store store.IStore
}

// New{{ .Custom.ResourcePascal }}Biz 构造一个默认实现 *{{ .Custom.ResourceLower }}Biz。
func New{{ .Custom.ResourcePascal }}Biz(store store.IStore) *{{ .Custom.ResourceLower }}Biz {
	return &{{ .Custom.ResourceLower }}Biz{store: store}
}

// {{ .Custom.ResourcePascal }}V1 由 `linctl add api {{ .Custom.ResourcePascal }}` 注入到 IBiz 接口。
func (b *biz) {{ .Custom.ResourcePascal }}V1() {{ .Custom.ResourcePascal }}Biz {
	return New{{ .Custom.ResourcePascal }}Biz(b.store)
}

func (b *{{ .Custom.ResourceLower }}Biz) List(ctx context.Context) ([]*{{ .Custom.ResourcePascal }}, error) {
	return nil, nil
}

func (b *{{ .Custom.ResourceLower }}Biz) Get(ctx context.Context, id int64) (*{{ .Custom.ResourcePascal }}, error) {
	return &{{ .Custom.ResourcePascal }}{ID: id}, nil
}

func (b *{{ .Custom.ResourceLower }}Biz) Create(ctx context.Context, _ *Create{{ .Custom.ResourcePascal }}Request) error {
	return nil
}

func (b *{{ .Custom.ResourceLower }}Biz) Update(ctx context.Context, _ int64, _ *Update{{ .Custom.ResourcePascal }}Request) error {
	return nil
}

func (b *{{ .Custom.ResourceLower }}Biz) Delete(ctx context.Context, _ int64) error {
	return nil
}
