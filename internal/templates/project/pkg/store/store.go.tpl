// Package store provides a generic, reusable GORM-backed data store.
package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"{{.Module}}/pkg/store/where"
)

// DBProvider defines an interface for providing a database connection.
type DBProvider interface {
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
}

// Logger defines the logging interface used by Store.
type Logger interface {
	Error(ctx context.Context, err error, msg string, keysAndValues ...any)
}

// Store is a generic data access layer backed by GORM.
type Store[T any] struct {
	logger  Logger
	storage DBProvider
}

// NewStore creates a new Store instance.
func NewStore[T any](storage DBProvider, logger Logger) *Store[T] {
	if logger == nil {
		logger = &noopLogger{}
	}
	return &Store[T]{logger: logger, storage: storage}
}

func (s *Store[T]) db(ctx context.Context, wheres ...where.Where) *gorm.DB {
	d := s.storage.DB(ctx)
	for _, w := range wheres {
		if w != nil {
			d = w.Where(d)
		}
	}
	return d
}

// Create inserts obj into the database.
func (s *Store[T]) Create(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Create(obj).Error; err != nil {
		s.logger.Error(ctx, err, "failed to create object")
		return err
	}
	return nil
}

// Update saves obj into the database.
func (s *Store[T]) Update(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Save(obj).Error; err != nil {
		s.logger.Error(ctx, err, "failed to update object")
		return err
	}
	return nil
}

// Delete removes records matching opts.
func (s *Store[T]) Delete(ctx context.Context, opts *where.Options) error {
	err := s.db(ctx, opts).Delete(new(T)).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error(ctx, err, "failed to delete object")
		return err
	}
	return nil
}

// Get retrieves a single record matching opts.
func (s *Store[T]) Get(ctx context.Context, opts *where.Options) (*T, error) {
	var obj T
	if err := s.db(ctx, opts).First(&obj).Error; err != nil {
		s.logger.Error(ctx, err, "failed to get object")
		return nil, err
	}
	return &obj, nil
}

// List retrieves records matching opts and returns total count.
func (s *Store[T]) List(ctx context.Context, opts *where.Options) (int64, []*T, error) {
	var (
		count int64
		ret   []*T
	)
	err := s.db(ctx, opts).Order("id desc").Find(&ret).Offset(-1).Limit(-1).Count(&count).Error
	if err != nil {
		s.logger.Error(ctx, err, "failed to list objects")
	}
	return count, ret, err
}

// noopLogger is a Logger that does nothing.
type noopLogger struct{}

func (n *noopLogger) Error(_ context.Context, _ error, _ string, _ ...any) {}
