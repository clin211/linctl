package validation

import (
	"{{.Module}}/internal/{{.AppName}}/store"
)

// Validator 持有请求校验所需的依赖。
type Validator struct {
	store store.IStore
}

// New 创建一个 Validator 实例。
func New(s store.IStore) *Validator {
	return &Validator{store: s}
}
