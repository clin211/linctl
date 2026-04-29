package registry

import (
	"sync"

	"gorm.io/gorm"
)

// Registry 用于在进程内管理 GORM 模型的注册列表。
type Registry struct {
	models []interface{}
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// NewRegistry 创建并返回一个新的 *Registry。
func NewRegistry() *Registry {
	return &Registry{
		models: make([]interface{}, 0),
	}
}

// Register 把 model 加入到全局 registry。通常在 model 包的 init() 中调用。
func Register(model interface{}) {
	once.Do(func() {
		globalRegistry = NewRegistry()
	})
	globalRegistry.Register(model)
}

// Register 把 model 加入到当前 *Registry。
func (r *Registry) Register(model interface{}) {
	r.models = append(r.models, model)
}

// Migrate 触发全局 registry 中所有模型的 AutoMigrate；尚未注册任何模型时直接返回 nil。
func Migrate(db *gorm.DB) error {
	if globalRegistry == nil {
		return nil
	}

	return globalRegistry.Migrate(db)
}

// Migrate 顺序执行注册过的模型的 AutoMigrate。
func (r *Registry) Migrate(db *gorm.DB) error {
	for _, model := range r.models {
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}
