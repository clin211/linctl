package store

import (
	"context"

	"{{.Module}}/internal/{{.AppName}}/model"
{{- if ne .Storage "memory"}}
	genericstore "{{.Module}}/pkg/store"
	"{{.Module}}/pkg/store/where"
{{- end}}
)

{{- if eq .Storage "memory"}}

// {{.Resource | Pascal}}Store defines the store methods for {{.Resource | Pascal}} (in-memory).
type {{.Resource | Pascal}}Store interface {
	Create(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Update(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*model.{{.Resource | Pascal}}M, error)
	List(ctx context.Context, offset, limit int64) (int64, []*model.{{.Resource | Pascal}}M, error)
}

// {{.Resource | LowerCamel}}Store is the in-memory implementation of {{.Resource | Pascal}}Store.
type {{.Resource | LowerCamel}}Store struct {
	items map[string]*model.{{.Resource | Pascal}}M
}

// Ensure {{.Resource | LowerCamel}}Store implements {{.Resource | Pascal}}Store.
var _ {{.Resource | Pascal}}Store = (*{{.Resource | LowerCamel}}Store)(nil)

// new{{.Resource | Pascal}}Store creates a new in-memory {{.Resource | Pascal}}Store.
func new{{.Resource | Pascal}}Store(_ *memoryStore) *{{.Resource | LowerCamel}}Store {
	return &{{.Resource | LowerCamel}}Store{items: make(map[string]*model.{{.Resource | Pascal}}M)}
}

// Create stores a {{.Resource | Pascal}} in memory.
func (s *{{.Resource | LowerCamel}}Store) Create(_ context.Context, obj *model.{{.Resource | Pascal}}M) error {
	s.items[obj.{{.Resource | Pascal}}ID] = obj
	return nil
}

// Update replaces a {{.Resource | Pascal}} in memory.
func (s *{{.Resource | LowerCamel}}Store) Update(_ context.Context, obj *model.{{.Resource | Pascal}}M) error {
	s.items[obj.{{.Resource | Pascal}}ID] = obj
	return nil
}

// Delete removes a {{.Resource | Pascal}} from memory by ID.
func (s *{{.Resource | LowerCamel}}Store) Delete(_ context.Context, id string) error {
	delete(s.items, id)
	return nil
}

// Get retrieves a single {{.Resource | Pascal}} from memory by ID.
func (s *{{.Resource | LowerCamel}}Store) Get(_ context.Context, id string) (*model.{{.Resource | Pascal}}M, error) {
	v, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	return v, nil
}

// List retrieves all {{.Resource | Pascal}} entries with simple pagination.
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

// {{.Resource | Pascal}}Store defines the store methods for {{.Resource | Pascal}}.
type {{.Resource | Pascal}}Store interface {
	Create(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Update(ctx context.Context, obj *model.{{.Resource | Pascal}}M) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.{{.Resource | Pascal}}M, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.{{.Resource | Pascal}}M, error)
}

// {{.Resource | LowerCamel}}Store is the gorm-backed implementation of {{.Resource | Pascal}}Store.
type {{.Resource | LowerCamel}}Store struct {
	*genericstore.Store[model.{{.Resource | Pascal}}M]
}

// Ensure {{.Resource | LowerCamel}}Store implements {{.Resource | Pascal}}Store.
var _ {{.Resource | Pascal}}Store = (*{{.Resource | LowerCamel}}Store)(nil)

// new{{.Resource | Pascal}}Store creates a new gorm-backed {{.Resource | Pascal}}Store.
func new{{.Resource | Pascal}}Store(store *datastore) *{{.Resource | LowerCamel}}Store {
	return &{{.Resource | LowerCamel}}Store{
		Store: genericstore.NewStore[model.{{.Resource | Pascal}}M](store, nil),
	}
}

{{- end}}
