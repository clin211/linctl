package store

import (
	"sync"
{{- if and (ne .Storage "memory") (ne .Storage "mongo")}}
	"context"

	"github.com/clin211/linhub/store/where"
	"gorm.io/gorm"
{{- end}}
)

// IStore 定义 store 层需要实现的方法集。
//
// `linctl add <Resource>` 通过 AST 向该接口追加新方法。
type IStore interface {
{{- if and (ne .Storage "memory") (ne .Storage "mongo")}}
	// DB 返回底层的 *gorm.DB，用于在需要时直接访问。
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
	// TX 在数据库事务中执行 fn。
	TX(ctx context.Context, fn func(ctx context.Context) error) error
{{- end}}
}

var (
	once sync.Once
	// S 是包级别的 store 单例。
	S IStore
)

{{- if or (eq .Storage "memory") (eq .Storage "mongo")}}
// memoryStore 是 IStore 的内存实现。
type memoryStore struct {
	mu sync.RWMutex
}

// 确保 memoryStore 实现了 IStore 接口。
var _ IStore = (*memoryStore)(nil)

// NewStore 创建（或返回）单例的内存 store。
func NewStore() IStore {
	once.Do(func() {
		S = &memoryStore{}
	})
	return S
}
{{- else}}
// transactionKey 是用于在 context 中存储活动 *gorm.DB 事务的 key。
type transactionKey struct{}

// datastore 是基于 gorm 的 IStore 实现。
type datastore struct {
	core *gorm.DB
}

// 确保 datastore 实现了 IStore 接口。
var _ IStore = (*datastore)(nil)

// NewStore 创建（或返回）单例的 datastore。
func NewStore(db *gorm.DB) *datastore {
	once.Do(func() {
		S = &datastore{core: db}
	})
	return S.(*datastore)
}

// DB 返回与 context 绑定（事务感知）并应用 wheres 过滤的 *gorm.DB。
func (ds *datastore) DB(ctx context.Context, wheres ...where.Where) *gorm.DB {
	d := ds.core
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		d = tx
	}
	for _, w := range wheres {
		d = w.Where(d)
	}
	return d
}

// TX 在 gorm 事务中执行 fn。
func (ds *datastore) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	return ds.core.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionKey{}, tx))
	})
}
{{- end}}
