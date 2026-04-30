package store

import (
	"sync"
{{- if ne .Storage "memory"}}
	"context"

	"{{.Module}}/pkg/store/where"
	"gorm.io/gorm"
{{- end}}
)

// IStore defines the methods that the store layer needs to implement.
//
// `lin add <Resource>` appends new methods to this interface via AST.
type IStore interface {
{{- if ne .Storage "memory"}}
	// DB returns the underlying *gorm.DB for direct access when needed.
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
	// TX executes fn inside a database transaction.
	TX(ctx context.Context, fn func(ctx context.Context) error) error
{{- end}}
}

var (
	once sync.Once
	// S is the package-level store singleton.
	S IStore
)

{{- if eq .Storage "memory"}}
// memoryStore is an in-memory implementation of IStore.
type memoryStore struct {
	mu sync.RWMutex
}

// Ensure memoryStore implements IStore.
var _ IStore = (*memoryStore)(nil)

// NewStore creates (or returns) the singleton in-memory store.
func NewStore() IStore {
	once.Do(func() {
		S = &memoryStore{}
	})
	return S
}
{{- else}}
// transactionKey is the context key for storing an active *gorm.DB transaction.
type transactionKey struct{}

// datastore is the gorm-backed implementation of IStore.
type datastore struct {
	core *gorm.DB
}

// Ensure datastore implements IStore.
var _ IStore = (*datastore)(nil)

// NewStore creates (or returns) the singleton datastore.
func NewStore(db *gorm.DB) *datastore {
	once.Do(func() {
		S = &datastore{core: db}
	})
	return S.(*datastore)
}

// DB returns a *gorm.DB scoped to the context (transaction-aware) and filtered by wheres.
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

// TX runs fn inside a gorm transaction.
func (ds *datastore) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	return ds.core.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, transactionKey{}, tx))
	})
}
{{- end}}
