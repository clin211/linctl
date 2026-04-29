package store

import (
	"context"
{{- if ne .Component.Storage "mongo" }}

	"{{ .Project.Metadata.Module }}/pkg/store/where"
{{- end }}
)

// {{ .Custom.ResourcePascal }}M 是 {{ .Custom.ResourceLower }} 资源的持久化模型。
type {{ .Custom.ResourcePascal }}M struct {
	ID   int64  `gorm:"primaryKey"     json:"id"             bson:"_id,omitempty"`
	Name string `gorm:"size:255;index" json:"name"           bson:"name"`
}

{{- if ne .Component.Storage "mongo" }}

// TableName 覆写 GORM 默认表名。
func (m *{{ .Custom.ResourcePascal }}M) TableName() string { return "{{ snake (plural .Custom.ResourcePascal) }}" }
{{- end }}

// {{ .Custom.ResourcePascal }}Store 是 {{ .Custom.ResourceLower }} 资源的数据访问接口。
type {{ .Custom.ResourcePascal }}Store interface {
	List(ctx context.Context) (int64, []*{{ .Custom.ResourcePascal }}M, error)
	Get(ctx context.Context, id int64) (*{{ .Custom.ResourcePascal }}M, error)
	Create(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error
	Update(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error
	Delete(ctx context.Context, id int64) error
}

type {{ .Custom.ResourceLower }}Store struct {
	store *datastore
}

// new{{ .Custom.ResourcePascal }}Store 基于 datastore 构造一个默认 store 实例。
func new{{ .Custom.ResourcePascal }}Store(store *datastore) *{{ .Custom.ResourceLower }}Store {
	return &{{ .Custom.ResourceLower }}Store{store: store}
}

// {{ .Custom.ResourcePascal }} 由 `linctl add api {{ .Custom.ResourcePascal }}` 注入到 IStore 接口。
func (s *datastore) {{ .Custom.ResourcePascal }}() {{ .Custom.ResourcePascal }}Store {
	return new{{ .Custom.ResourcePascal }}Store(s)
}

{{- if eq .Component.Storage "mongo" }}

func (s *{{ .Custom.ResourceLower }}Store) List(ctx context.Context) (int64, []*{{ .Custom.ResourcePascal }}M, error) {
	return 0, nil, nil
}

func (s *{{ .Custom.ResourceLower }}Store) Get(ctx context.Context, id int64) (*{{ .Custom.ResourcePascal }}M, error) {
	return &{{ .Custom.ResourcePascal }}M{ID: id}, nil
}

func (s *{{ .Custom.ResourceLower }}Store) Create(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error {
	return nil
}

func (s *{{ .Custom.ResourceLower }}Store) Update(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error {
	return nil
}

func (s *{{ .Custom.ResourceLower }}Store) Delete(ctx context.Context, id int64) error {
	return nil
}

{{- else }}

func (s *{{ .Custom.ResourceLower }}Store) List(ctx context.Context) (count int64, ret []*{{ .Custom.ResourcePascal }}M, err error) {
	err = s.store.DB(ctx).Find(&ret).Offset(-1).Limit(-1).Count(&count).Error
	return
}

func (s *{{ .Custom.ResourceLower }}Store) Get(ctx context.Context, id int64) (*{{ .Custom.ResourcePascal }}M, error) {
	var m {{ .Custom.ResourcePascal }}M
	if err := s.store.DB(ctx, where.F("id", id)).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *{{ .Custom.ResourceLower }}Store) Create(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error {
	return s.store.DB(ctx).Create(m).Error
}

func (s *{{ .Custom.ResourceLower }}Store) Update(ctx context.Context, m *{{ .Custom.ResourcePascal }}M) error {
	return s.store.DB(ctx).Save(m).Error
}

func (s *{{ .Custom.ResourceLower }}Store) Delete(ctx context.Context, id int64) error {
	return s.store.DB(ctx, where.F("id", id)).Delete(&{{ .Custom.ResourcePascal }}M{}).Error
}

{{- end }}
