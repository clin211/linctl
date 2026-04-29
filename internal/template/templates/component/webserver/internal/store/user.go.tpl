// nolint: dupl
package store

import (
	"context"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/model"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
{{- if ne .Component.Storage "mongo" }}
	genericstore "{{ .Project.Metadata.Module }}/pkg/store"
	"{{ .Project.Metadata.Module }}/pkg/store/logger/empty"
{{- end }}
)

// UserStore 定义了 user 模块在 store 层所实现的方法.
type UserStore interface {
	Create(ctx context.Context, obj *model.UserM) error
	Update(ctx context.Context, obj *model.UserM) error
	Delete(ctx context.Context, opts *where.Options) error
	Get(ctx context.Context, opts *where.Options) (*model.UserM, error)
	List(ctx context.Context, opts *where.Options) (int64, []*model.UserM, error)

	UserExpansion
}

// UserExpansion 定义了用户操作的附加方法（占位接口，便于按需扩展）.
// nolint: iface
type UserExpansion interface{}

{{- if ne .Component.Storage "mongo" }}

// userStore 是 UserStore 接口的 GORM 实现.
//
// 通过组合泛型 *genericstore.Store[model.UserM] 复用 Create / Update / Delete /
// Get / List 五个标准方法；非标准查询写在本文件追加方法上即可。
type userStore struct {
	*genericstore.Store[model.UserM]
}

// 编译期断言：确保 *userStore 实现了 UserStore 接口.
var _ UserStore = (*userStore)(nil)

// newUserStore 创建 userStore 的实例.
func newUserStore(store *datastore) *userStore {
	return &userStore{
		Store: genericstore.NewStore[model.UserM](store, empty.NewLogger()),
	}
}
{{- else }}

// userStore 是 UserStore 接口的 Mongo 占位实现.
//
// TODO(linctl): 当 storage=mongo 时本骨架不绑定具体 collection 操作，请按业务自行替换：
//   - 用 store.client 拿到 *mongo.Client / *mongo.Collection
//   - 实现 Create / Update / Delete / Get / List 五个方法
type userStore struct {
	store *datastore
}

// 编译期断言：确保 *userStore 实现了 UserStore 接口.
var _ UserStore = (*userStore)(nil)

// newUserStore 创建 userStore 的实例.
func newUserStore(store *datastore) *userStore {
	return &userStore{store: store}
}

// Create TODO(linctl): 实现 mongo 版本的写入.
func (s *userStore) Create(ctx context.Context, obj *model.UserM) error {
	_ = s.store
	return nil
}

// Update TODO(linctl): 实现 mongo 版本的更新.
func (s *userStore) Update(ctx context.Context, obj *model.UserM) error {
	return nil
}

// Delete TODO(linctl): 实现 mongo 版本的删除.
func (s *userStore) Delete(ctx context.Context, opts *where.Options) error {
	return nil
}

// Get TODO(linctl): 实现 mongo 版本的查询.
func (s *userStore) Get(ctx context.Context, opts *where.Options) (*model.UserM, error) {
	return nil, nil
}

// List TODO(linctl): 实现 mongo 版本的列表.
func (s *userStore) List(ctx context.Context, opts *where.Options) (int64, []*model.UserM, error) {
	return 0, nil, nil
}
{{- end }}
