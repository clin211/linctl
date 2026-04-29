package store

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"{{ .Project.Metadata.Module }}/pkg/store/logger/empty"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// DBProvider 是用于提供数据库连接的接口。
type DBProvider interface {
	// DB 根据给定的 context 返回数据库实例（可选地按 wheres 条件过滤）。
	DB(ctx context.Context, wheres ...where.Where) *gorm.DB
}

// Option 是用于配置 *Store[T] 的函数选项类型。
type Option[T any] func(*Store[T])

// Store 是带日志能力的泛型数据访问结构。
type Store[T any] struct {
	logger  Logger
	storage DBProvider
}

// WithLogger 设置 *Store[T] 使用的 Logger。
func WithLogger[T any](logger Logger) Option[T] {
	return func(s *Store[T]) {
		s.logger = logger
	}
}

// NewStore 用给定的 DBProvider 构造 *Store[T]；logger 为 nil 时使用空实现。
func NewStore[T any](storage DBProvider, logger Logger) *Store[T] {
	if logger == nil {
		logger = empty.NewLogger()
	}

	return &Store[T]{
		logger:  logger,
		storage: storage,
	}
}

// db 取出底层数据库实例并应用所有提供的 where 条件。
func (s *Store[T]) db(ctx context.Context, wheres ...where.Where) *gorm.DB {
	dbInstance := s.storage.DB(ctx)
	for _, whr := range wheres {
		if whr != nil {
			dbInstance = whr.Where(dbInstance)
		}
	}
	return dbInstance
}

// Create 把 obj 写入数据库。
func (s *Store[T]) Create(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Create(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to insert object into database", "object", obj)
		return err
	}
	return nil
}

// Update 把 obj 的修改写回数据库。
func (s *Store[T]) Update(ctx context.Context, obj *T) error {
	if err := s.db(ctx).Save(obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to update object in database", "object", obj)
		return err
	}
	return nil
}

// Delete 根据 where 条件从数据库中删除对象；记录不存在不视为错误。
func (s *Store[T]) Delete(ctx context.Context, opts *where.Options) error {
	err := s.db(ctx, opts).Delete(new(T)).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error(ctx, err, "Failed to delete object from database", "conditions", opts)
		return err
	}
	return nil
}

// Get 根据 where 条件从数据库取出单个对象。
func (s *Store[T]) Get(ctx context.Context, opts *where.Options) (*T, error) {
	var obj T
	if err := s.db(ctx, opts).First(&obj).Error; err != nil {
		s.logger.Error(ctx, err, "Failed to retrieve object from database", "conditions", opts)
		return nil, err
	}
	return &obj, nil
}

// List 根据 where 条件查询满足条件的对象列表与总条数。
func (s *Store[T]) List(ctx context.Context, opts *where.Options) (count int64, ret []*T, err error) {
	err = s.db(ctx, opts).Order("id desc").Find(&ret).Offset(-1).Limit(-1).Count(&count).Error
	if err != nil {
		s.logger.Error(ctx, err, "Failed to list objects from database", "conditions", opts)
	}
	return
}
