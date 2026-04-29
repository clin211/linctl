package where

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// defaultLimit 是分页查询时的默认 Limit 值，-1 表示不限制。
	defaultLimit = -1
)

// Tenant 描述一个租户：用 Key 标识，用 ValueFunc 从 context 中取值。
type Tenant struct {
	Key       string                           // 租户字段名（例如 "userID"）
	ValueFunc func(ctx context.Context) string // 从 context 中取出租户值的函数
}

// Where 是任何能修改 GORM 查询的类型应实现的接口。
type Where interface {
	Where(db *gorm.DB) *gorm.DB
}

// Query 表示一条 GORM 风格的 Where 查询：条件 + 参数。
type Query struct {
	// Query 是 GORM Where 子句支持的任意类型条件（string / map / struct ...）。
	Query interface{}

	// Args 是对应 Query 中占位符的参数列表。
	Args []interface{}
}

// Option 是 Options 的函数选项类型。
type Option func(*Options)

// Options 收集 GORM Where 查询的所有可调参数。
type Options struct {
	// Offset 分页起点（从 0 开始）。
	// +optional
	Offset int `json:"offset"`
	// Limit 单页最大记录数；-1 表示不限制。
	// +optional
	Limit int `json:"limit"`
	// Filters 是 key-value 形式的过滤条件。
	Filters map[any]any
	// Clauses 是要追加到查询的自定义 Clause。
	Clauses []clause.Expression
	// Queries 是要执行的若干 Query。
	Queries []Query
}

// registeredTenant 保存已注册的全局租户。
var registeredTenant Tenant

// WithOffset 在 Options 上设置 Offset。
func WithOffset(offset int64) Option {
	return func(whr *Options) {
		if offset < 0 {
			offset = 0
		}
		whr.Offset = int(offset)
	}
}

// WithLimit 在 Options 上设置 Limit。
func WithLimit(limit int64) Option {
	return func(whr *Options) {
		if limit <= 0 {
			limit = defaultLimit
		}
		whr.Limit = int(limit)
	}
}

// WithPage 是分页便利函数：把 page / pageSize 转换为 Offset / Limit。
func WithPage(page int, pageSize int) Option {
	return func(whr *Options) {
		if page == 0 {
			page = 1
		}
		if pageSize == 0 {
			pageSize = defaultLimit
		}

		whr.Offset = (page - 1) * pageSize
		whr.Limit = pageSize
	}
}

// WithFilter 在 Options 上设置过滤条件。
func WithFilter(filter map[any]any) Option {
	return func(whr *Options) {
		whr.Filters = filter
	}
}

// WithClauses 在 Options 上追加自定义 Clause。
func WithClauses(conds ...clause.Expression) Option {
	return func(whr *Options) {
		whr.Clauses = append(whr.Clauses, conds...)
	}
}

// WithQuery 在 Options 上追加一条带参数的 Query。
//
// query 可以是 string / map / struct 等 GORM Where 支持的所有类型；
// args 是替换 query 中占位符的实参。
func WithQuery(query interface{}, args ...interface{}) Option {
	return func(whr *Options) {
		whr.Queries = append(whr.Queries, Query{Query: query, Args: args})
	}
}

// NewWhere 构造一个 *Options 并应用全部 Option。
func NewWhere(opts ...Option) *Options {
	whr := &Options{
		Offset:  0,
		Limit:   defaultLimit,
		Filters: map[any]any{},
		Clauses: make([]clause.Expression, 0),
	}

	for _, opt := range opts {
		opt(whr)
	}

	return whr
}

// O 设置 Offset 并返回自身（链式调用）。
func (whr *Options) O(offset int) *Options {
	if offset < 0 {
		offset = 0
	}
	whr.Offset = offset
	return whr
}

// L 设置 Limit 并返回自身（链式调用）。
func (whr *Options) L(limit int) *Options {
	if limit <= 0 {
		limit = defaultLimit
	}
	whr.Limit = limit
	return whr
}

// P 按 page / pageSize 设置分页并返回自身（链式调用）。
func (whr *Options) P(page int, pageSize int) *Options {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultLimit
	}
	whr.Offset = (page - 1) * pageSize
	whr.Limit = pageSize
	return whr
}

// C 追加自定义 Clause 并返回自身（链式调用）。
func (whr *Options) C(conds ...clause.Expression) *Options {
	whr.Clauses = append(whr.Clauses, conds...)
	return whr
}

// Q 追加一条带参数的 Query 并返回自身（链式调用）。
func (whr *Options) Q(query interface{}, args ...interface{}) *Options {
	whr.Queries = append(whr.Queries, Query{Query: query, Args: args})
	return whr
}

// T 自动追加注册租户的过滤条件（如果有），并返回自身。
func (whr *Options) T(ctx context.Context) *Options {
	if registeredTenant.Key != "" && registeredTenant.ValueFunc != nil {
		whr.F(registeredTenant.Key, registeredTenant.ValueFunc(ctx))
	}
	return whr
}

// F 把 key-value 序列追加到 Filters，并返回自身。
//
// kvs 必须成对出现；个数为奇数时直接返回，不做改动。
func (whr *Options) F(kvs ...any) *Options {
	if len(kvs)%2 != 0 {
		// 出现奇数个 key-value 直接放弃本次调用。
		return whr
	}

	for i := 0; i < len(kvs); i += 2 {
		key := kvs[i]
		value := kvs[i+1]
		whr.Filters[key] = value
	}

	return whr
}

// Where 把 Filters / Clauses / Queries / Offset / Limit 作用到给定的 *gorm.DB。
func (whr *Options) Where(db *gorm.DB) *gorm.DB {
	for _, query := range whr.Queries {
		conds := db.Statement.BuildCondition(query.Query, query.Args...)
		whr.Clauses = append(whr.Clauses, conds...)
	}
	return db.Where(whr.Filters).Clauses(whr.Clauses...).Offset(whr.Offset).Limit(whr.Limit)
}

// O 是构造带 Offset 的 *Options 的便利函数。
func O(offset int) *Options {
	return NewWhere().O(offset)
}

// L 是构造带 Limit 的 *Options 的便利函数。
func L(limit int) *Options {
	return NewWhere().L(limit)
}

// P 是构造分页 *Options 的便利函数。
func P(page int, pageSize int) *Options {
	return NewWhere().P(page, pageSize)
}

// C 是构造带 Clauses 的 *Options 的便利函数。
func C(conds ...clause.Expression) *Options {
	return NewWhere().C(conds...)
}

// T 是构造带租户过滤的 *Options 的便利函数。
func T(ctx context.Context) *Options {
	return NewWhere().F(registeredTenant.Key, registeredTenant.ValueFunc(ctx))
}

// F 是构造带 Filters 的 *Options 的便利函数。
func F(kvs ...any) *Options {
	return NewWhere().F(kvs...)
}

// RegisterTenant 注册一个全局租户：T(ctx) 之后所有 *Options 都会附带该租户过滤。
func RegisterTenant(key string, valueFunc func(context.Context) string) {
	registeredTenant = Tenant{
		Key:       key,
		ValueFunc: valueFunc,
	}
}
