package feature

import (
	"sort"
	"sync"

	"github.com/clin211/linctl/internal/linctlerr"
)

// Registry 是 Feature 的全局注册中心。
//
// 提供按 Kind 过滤 + 拓扑排序的能力。
type Registry struct {
	mu       sync.RWMutex
	features map[string]Feature
}

// NewRegistry 构造一个空 Registry。
func NewRegistry() *Registry {
	return &Registry{features: make(map[string]Feature, 16)}
}

// Register 注册 Feature。重复注册同名 Feature 返回错误。
func (r *Registry) Register(f Feature) error {
	if f == nil {
		return linctlerr.New(linctlerr.ErrInternal, "Register: feature is nil")
	}
	if f.Name() == "" {
		return linctlerr.New(linctlerr.ErrInternal, "Register: feature name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.features[f.Name()]; exists {
		return linctlerr.Newf(linctlerr.ErrInternal,
			"feature already registered: %s", f.Name())
	}
	r.features[f.Name()] = f
	return nil
}

// Get 按名称返回 Feature。不存在时返回 nil + error。
func (r *Registry) Get(name string) (Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.features[name]
	if !ok {
		return nil, linctlerr.Newf(linctlerr.ErrFeatureDependency,
			"unknown feature: %s", name).
			WithHint("Registered features: " + joinFeatureNames(r.features))
	}
	return f, nil
}

// ResolveOrder 根据 selected feature 名列表 + 注册中心，
// 计算执行顺序（拓扑排序 + Order tie-break）。
//
// 失败场景：
//   - 选中的 feature 未注册 → ErrFeatureDependency
//   - Requires() 中引用了未在 selected 中的 feature → 自动加入（隐式扩展）
//   - 拓扑环 → ErrFeatureDependency 错误中含具体环路
func (r *Registry) ResolveOrder(selected []string) ([]Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 收集所有相关 Feature（含传递依赖）
	included := make(map[string]Feature, len(selected))
	var stack []string
	for _, name := range selected {
		stack = append(stack, name)
	}
	for len(stack) > 0 {
		name := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := included[name]; ok {
			continue
		}
		f, ok := r.features[name]
		if !ok {
			return nil, linctlerr.Newf(linctlerr.ErrFeatureDependency,
				"feature %q is not registered", name)
		}
		included[name] = f
		for _, dep := range f.Requires() {
			if _, ok := included[dep]; !ok {
				stack = append(stack, dep)
			}
		}
	}

	// Kahn 拓扑排序
	indeg := make(map[string]int, len(included))
	deps := make(map[string][]string, len(included))
	rev := make(map[string][]string, len(included))
	for name, f := range included {
		indeg[name] = 0
		deps[name] = f.Requires()
	}
	for name, ds := range deps {
		for _, d := range ds {
			rev[d] = append(rev[d], name)
			indeg[name]++
		}
	}

	// 初始队列：所有 indeg=0 的，按 (Order, Name) 排序保证稳定性
	var ready []string
	for name, deg := range indeg {
		if deg == 0 {
			ready = append(ready, name)
		}
	}
	sortByOrderName(ready, included)

	out := make([]Feature, 0, len(included))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		out = append(out, included[name])

		var next []string
		for _, child := range rev[name] {
			indeg[child]--
			if indeg[child] == 0 {
				next = append(next, child)
			}
		}
		sortByOrderName(next, included)
		// 把 next 合并到 ready 头部并重新排序
		ready = append(next, ready...)
		sortByOrderName(ready, included)
	}

	if len(out) != len(included) {
		// 存在环。找一条具体环路便于诊断
		remaining := make(map[string]struct{}, len(included)-len(out))
		for name := range included {
			seen := false
			for _, f := range out {
				if f.Name() == name {
					seen = true
					break
				}
			}
			if !seen {
				remaining[name] = struct{}{}
			}
		}
		var anyName string
		for n := range remaining {
			anyName = n
			break
		}
		cycle := findCycle(anyName, deps, remaining)
		return nil, linctlerr.Newf(linctlerr.ErrFeatureDependency,
			"feature dependency cycle detected: %v", cycle).
			WithHint("Remove one of the Requires() edges in the cycle.")
	}

	return out, nil
}

// findCycle 从 start 出发用 DFS 沿 Requires 找一条 start → ... → start 的环路。
func findCycle(start string, deps map[string][]string, remaining map[string]struct{}) []string {
	visited := map[string]bool{}
	var path []string
	var dfs func(node string) bool
	dfs = func(node string) bool {
		if node == start && len(path) > 0 {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		path = append(path, node)
		for _, d := range deps[node] {
			if _, in := remaining[d]; !in {
				continue
			}
			if dfs(d) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	path = append(path, start)
	visited[start] = true
	for _, d := range deps[start] {
		if _, in := remaining[d]; !in {
			continue
		}
		if dfs(d) {
			return append(path, start)
		}
	}
	// fallback：返回所有 remaining 节点
	out := make([]string, 0, len(remaining))
	for n := range remaining {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func sortByOrderName(names []string, m map[string]Feature) {
	sort.Slice(names, func(i, j int) bool {
		fi, fj := m[names[i]], m[names[j]]
		if fi.Order() != fj.Order() {
			return fi.Order() < fj.Order()
		}
		return fi.Name() < fj.Name()
	})
}

func joinFeatureNames(m map[string]Feature) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += ", "
		}
		out += k
	}
	return out
}
