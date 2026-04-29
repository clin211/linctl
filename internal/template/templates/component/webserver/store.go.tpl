package store

import (
	"context"
	"sync"

	"github.com/google/wire"
{{- if eq .Component.Storage "mongo" }}
	"go.mongodb.org/mongo-driver/mongo"
{{- else }}
	"gorm.io/gorm"

	"{{ .Project.Metadata.Module }}/pkg/store/where"
{{- end }}
)

// ProviderSet 声明 store 层的 Wire DI 规则。
var ProviderSet = wire.NewSet(NewStore, wire.Bind(new(IStore), new(*datastore)))

var (
	once sync.Once
	// S 是 *datastore 的全局单例句柄，方便包外直接引用。
	S *datastore
)

{{- if eq .Component.Storage "mongo" }}

// IStore 是 store 层对外暴露的接口（mongo 版本）。
//
// 当你执行 `linctl add api <Resource>` 时，新方法（例如 Posts() PostStore）
// 会被自动注入到这个接口里。
type IStore interface {
	// Client 返回底层 *mongo.Client，业务层可基于此自行实现事务等高级特性。
	Client(ctx context.Context) *mongo.Client
	// User 返回 user 资源的存储接口。
	User() UserStore
}

// datastore 是 IStore 的 Mongo 实现。
type datastore struct {
	client *mongo.Client
}

var _ IStore = (*datastore)(nil)

// NewStore 通过 sync.Once 构造单例 *datastore。
func NewStore(client *mongo.Client) *datastore {
	once.Do(func() {
		S = &datastore{client: client}
	})
	return S
}

// Client 返回底层 mongo 客户端；事务等扩展可在自定义实现里通过 ctx 携带。
func (s *datastore) Client(ctx context.Context) *mongo.Client {
	return s.client
}

// User 返回一个实现了 UserStore 接口的实例。
func (s *datastore) User() UserStore {
	return newUserStore(s)
}

{{- else }}

// IStore 是 store 层对外暴露的接口。
//
// 当你执行 `linctl add api <Resource>` 时，新方法（例如 Posts() PostStore）
// 会被自动注入到这个接口里。
type IStore interface {
	// DB 返回 *gorm.DB，可选地按提供的 where 条件过滤。
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
	// TX 在事务中执行 fn，事务通过 context 透传。
	TX(ctx context.Context, fn func(ctx context.Context) error) error
	// User 返回 user 资源的存储接口。
	User() UserStore
}

// transactionKey 用于在 context.Context 中携带活跃的事务实例。
type transactionKey struct{}

// datastore 是 IStore 的 GORM 实现。
type datastore struct {
	core *gorm.DB
}

var _ IStore = (*datastore)(nil)

// NewStore 通过 sync.Once 构造单例 *datastore。
func NewStore(db *gorm.DB) *datastore {
	once.Do(func() {
		S = &datastore{core: db}
	})
	return S
}

// DB 返回数据库实例：优先使用 ctx 中的事务句柄；并应用提供的 where 条件链。
func (s *datastore) DB(ctx context.Context, wheres ...where.Where) *gorm.DB {
	db := s.core
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		db = tx
	}
	for _, whr := range wheres {
		db = whr.Where(db)
	}
	return db
}

// TX 在事务中执行 fn，把活跃事务放进 ctx 后传递给业务函数。
//
// nolint: fatcontext
func (s *datastore) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.core.WithContext(ctx).Transaction(
		func(tx *gorm.DB) error {
			ctx = context.WithValue(ctx, transactionKey{}, tx)
			return fn(ctx)
		},
	)
}

// User 返回一个实现了 UserStore 接口的实例。
func (s *datastore) User() UserStore {
	return newUserStore(s)
}

{{- end }}
