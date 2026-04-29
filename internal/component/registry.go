package component

import (
	"sort"
	"sync"

	"github.com/clin211/lin/internal/linctlerr"
)

// Factory 创建一个 Component 实例（基于 YAML struct）。
type Factory func(c map[string]any) (Component, error)

// Registry 管理 Kind 到 Factory 的注册关系。
//
// 并发安全：内部用 sync.RWMutex 保护。
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry 创建一个空 Registry。
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory, 8)}
}

// Register 注册 Kind 对应的 Factory。重复注册同一 Kind 会返回错误。
func (r *Registry) Register(kind string, f Factory) error {
	if kind == "" {
		return linctlerr.New(linctlerr.ErrInternal, "Register: kind is empty")
	}
	if f == nil {
		return linctlerr.Newf(linctlerr.ErrInternal, "Register: factory for %s is nil", kind)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[kind]; exists {
		return linctlerr.Newf(linctlerr.ErrComponentExists,
			"component kind already registered: %s", kind)
	}
	r.factories[kind] = f
	return nil
}

// Get 返回 Kind 对应的 Factory。不存在时返回 ErrComponentNotFound。
func (r *Registry) Get(kind string) (Factory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.factories[kind]
	if !ok {
		return nil, linctlerr.Newf(linctlerr.ErrComponentNotFound,
			"unknown component kind: %s", kind).
			WithHint("Registered kinds: " + joinKinds(r.factories))
	}
	return f, nil
}

// Kinds 返回所有已注册的 Kind 列表（字典序）。
func (r *Registry) Kinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.factories))
	for k := range r.factories {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinKinds(m map[string]Factory) string {
	out := ""
	first := true
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !first {
			out += ", "
		}
		out += k
		first = false
	}
	return out
}
