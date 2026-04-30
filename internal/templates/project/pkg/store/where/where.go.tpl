// Package where provides GORM query condition builders.
package where

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultLimit = -1

// Where defines an interface for types that can apply conditions to a gorm.DB.
type Where interface {
	Where(db *gorm.DB) *gorm.DB
}

// Tenant represents a tenant with a key and context value function.
type Tenant struct {
	Key       string
	ValueFunc func(ctx context.Context) string
}

// Query holds a GORM query condition with arguments.
type Query struct {
	Query interface{}
	Args  []interface{}
}

// Option is a function that modifies Options.
type Option func(*Options)

// Options holds the query conditions.
type Options struct {
	Offset  int
	Limit   int
	Filters map[any]any
	Clauses []clause.Expression
	Queries []Query
}

var registeredTenant Tenant

// WithOffset sets the query offset.
func WithOffset(offset int64) Option {
	return func(whr *Options) {
		if offset < 0 {
			offset = 0
		}
		whr.Offset = int(offset)
	}
}

// WithLimit sets the query limit.
func WithLimit(limit int64) Option {
	return func(whr *Options) {
		if limit <= 0 {
			limit = defaultLimit
		}
		whr.Limit = int(limit)
	}
}

// WithPage converts page/pageSize to offset/limit.
func WithPage(page, pageSize int) Option {
	return func(whr *Options) {
		if page < 1 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = defaultLimit
		}
		whr.Offset = (page - 1) * pageSize
		whr.Limit = pageSize
	}
}

// WithFilter sets key-value filter conditions.
func WithFilter(filter map[any]any) Option {
	return func(whr *Options) { whr.Filters = filter }
}

// WithClauses appends GORM clause expressions.
func WithClauses(conds ...clause.Expression) Option {
	return func(whr *Options) {
		whr.Clauses = append(whr.Clauses, conds...)
	}
}

// WithQuery adds a raw query condition.
func WithQuery(query interface{}, args ...interface{}) Option {
	return func(whr *Options) {
		whr.Queries = append(whr.Queries, Query{Query: query, Args: args})
	}
}

// NewWhere constructs a new Options applying the given options.
func NewWhere(opts ...Option) *Options {
	whr := &Options{
		Filters: map[any]any{},
		Clauses: make([]clause.Expression, 0),
		Limit:   defaultLimit,
	}
	for _, opt := range opts {
		opt(whr)
	}
	return whr
}

// Where applies the conditions to the given gorm.DB.
func (whr *Options) Where(db *gorm.DB) *gorm.DB {
	for _, q := range whr.Queries {
		conds := db.Statement.BuildCondition(q.Query, q.Args...)
		whr.Clauses = append(whr.Clauses, conds...)
	}
	return db.Where(whr.Filters).Clauses(whr.Clauses...).Offset(whr.Offset).Limit(whr.Limit)
}

// F creates Options with the given key-value filters.
func F(kvs ...any) *Options { return NewWhere().F(kvs...) }

// F adds key-value filter pairs.
func (whr *Options) F(kvs ...any) *Options {
	if len(kvs)%2 != 0 {
		return whr
	}
	for i := 0; i < len(kvs); i += 2 {
		whr.Filters[kvs[i]] = kvs[i+1]
	}
	return whr
}

// P creates Options with the given page/pageSize.
func P(page, pageSize int) *Options { return NewWhere(WithPage(page, pageSize)) }

// RegisterTenant registers a tenant for multi-tenant queries.
func RegisterTenant(key string, valueFunc func(context.Context) string) {
	registeredTenant = Tenant{Key: key, ValueFunc: valueFunc}
}
