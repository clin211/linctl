package store

import (
	"context"

	"{{.Module}}/internal/{{.AppName}}/model"
{{- if ne .Storage "memory"}}
	genericstore "github.com/clin211/linhub/store"
	"github.com/clin211/linhub/store/where"
{{- end}}
)

{{- if eq .Storage "memory"}}

// {{.Resource | Pascal}}Store 定义 {{.Resource | Pascal}} 的 store 方法集（内存版）。
type {{.Resource | Pascal}}Store interface {
	Create(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Update(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*model.{{.Resource | Pascal}}M, error)
	List(ctx context.Context, offset, limit int64) (int64, []*model.{{.Resource | Pascal}}M, error)
}

// {{.Resource | LowerCamel}}Store 是 {{.Resource | Pascal}}Store 的内存实现。
type {{.Resource | LowerCamel}}Store struct {
	items map[string]*model.{{.Resource | Pascal}}M
}

// 确保 {{.Resource | LowerCamel}}Store 实现了 {{.Resource | Pascal}}Store 接口。
var _ {{.Resource | Pascal}}Store = (*{{.Resource | LowerCamel}}Store)(nil)

// new{{.Resource | Pascal}}Store 创建一个新的内存版 {{.Resource | Pascal}}Store。
func new{{.Resource | Pascal}}Store(_ *memoryStore) *{{.Resource | LowerCamel}}Store {
	return &{{.Resource | LowerCamel}}Store{items: make(map[string]*model.{{.Resource | Pascal}}M)}
}

// Create 将 {{.Resource | Pascal}} 写入内存。
func (s *{{.Resource | LowerCamel}}Store) Create(_ context.Context, obj *model.{{.Resource | Pascal}}M) error {
	s.items[obj.{{.Resource | Pascal}}ID] = obj
	return nil
}

// Update 替换内存中的 {{.Resource | Pascal}}。
func (s *{{.Resource | LowerCamel}}Store) Update(_ context.Context, obj *model.{{.Resource | Pascal}}M) error {
	s.items[obj.{{.Resource | Pascal}}ID] = obj
	return nil
}

// Delete 按 ID 从内存中移除 {{.Resource | Pascal}}。
func (s *{{.Resource | LowerCamel}}Store) Delete(_ context.Context, id string) error {
	delete(s.items, id)
	return nil
}

// Get 按 ID 从内存中读取单个 {{.Resource | Pascal}}。
func (s *{{.Resource | LowerCamel}}Store) Get(_ context.Context, id string) (*model.{{.Resource | Pascal}}M, error) {
	v, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	return v, nil
}

// List 以简单分页方式返回内存中所有 {{.Resource | Pascal}}。
func (s *{{.Resource | LowerCamel}}Store) List(_ context.Context, offset, limit int64) (int64, []*model.{{.Resource | Pascal}}M, error) {
	all := make([]*model.{{.Resource | Pascal}}M, 0, len(s.items))
	for _, v := range s.items {
		all = append(all, v)
	}
	total := int64(len(all))
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return total, all[start:end], nil
}

{{- else}}

// {{.Resource | Pascal}}Store 定义 {{.Resource | Pascal}} 的 store 方法集。
type {{.Resource | Pascal}}Store interface {
	Create(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Update(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Delete(ctx context.Context, opts *where.Options) (int64, error)
	Get(ctx context.Context, opts *where.Options) (*model.{{.Resource | Pascal}}M, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.{{.Resource | Pascal}}M, error)
}

// {{.Resource | LowerCamel}}Store 是基于 gorm 的 {{.Resource | Pascal}}Store 实现。
type {{.Resource | LowerCamel}}Store struct {
	*genericstore.Store[model.{{.Resource | Pascal}}M]
}

// 确保 {{.Resource | LowerCamel}}Store 实现了 {{.Resource | Pascal}}Store 接口。
var _ {{.Resource | Pascal}}Store = (*{{.Resource | LowerCamel}}Store)(nil)

// new{{.Resource | Pascal}}Store 创建一个新的 gorm 版 {{.Resource | Pascal}}Store。
func new{{.Resource | Pascal}}Store(store *datastore) *{{.Resource | LowerCamel}}Store {
	return &{{.Resource | LowerCamel}}Store{
		Store: genericstore.NewStore[model.{{.Resource | Pascal}}M](store, nil),
	}
}

{{- end}}
