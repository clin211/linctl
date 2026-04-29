// Package transform 提供 templatesync 使用的可组合变换规则集合。
//
// Transform 是声明式的：每个规则在 sync.yaml 中以 `kind: <name>` + 参数表达，
// 由本包的 Registry 解析为具体实现。新增 transform 只需：
//   1. 在本包加一个新文件实现 Transform 接口
//   2. 在 Registry 的 init() 中注册 Factory
//   3. 在 sync.yaml 引用 kind 名
//
// 所有 transform 都必须满足：纯函数（不做 IO，不依赖随机源），便于幂等性测试。
package transform

import (
	"context"
	"fmt"

	"github.com/clin211/lin/internal/linctlerr"
)

// RuntimeContext 是 transform 执行时的运行时上下文。
//
// 由 templatesync.Runner 在执行链路中填充；transform 自身不应改写它。
type RuntimeContext struct {
	// UpstreamRoot 是上游项目根的绝对路径（如 /abs/.../miniblog-v4）
	UpstreamRoot string
	// SrcPath 是当前文件相对 upstream root 的路径（如 internal/pkg/contextx/contextx.go）
	SrcPath string
	// DstPath 是当前文件相对 lin templates root 的路径（如 internal/pkg/contextx/contextx.go.tpl）
	DstPath string
	// Owner 是该文件的归属（upstream / shared / linSpecific）；为空时表示 transform 不关心。
	Owner string
	// Manifest 是 sync.yaml 解析对象，用于读取 Upstream.* 占位等。
	//
	// 用 any 是为了避免 transform 包反向依赖 templatesync 包（防止 import cycle）。
	// 使用方应通过 ManifestAccessor 接口访问。
	Manifest ManifestAccessor
}

// ManifestAccessor 是 transform 访问 manifest 字段的最小接口（解耦 import cycle）。
type ManifestAccessor interface {
	UpstreamModuleOld() string
	UpstreamBinaryNameOld() string
	UpstreamComponentNameOld() string
	UpstreamName() string
}

// Transform 是单个变换规则。
//
// Apply 接受当前内容 + 运行时上下文，返回变换后的字节流。
// 失败时返回 *linctlerr.LinctlError（外部可 errors.As 判别）。
type Transform interface {
	// Kind 返回该 transform 的稳定名（与 sync.yaml 中的 kind 字段对应）。
	Kind() string

	// Apply 执行变换。content 已经是上一步 transform 的产物或源文件的字节流。
	Apply(ctx context.Context, rc *RuntimeContext, content []byte) ([]byte, error)
}

// Factory 从 sync.yaml 中的 raw 配置块构造一个 Transform。
//
// rawCfg 是去掉 kind 字段后的剩余 mapping（例如 from / to / pairs / appliesTo 等）。
type Factory func(rawCfg map[string]any) (Transform, error)

// Registry 管理 transform kind → Factory 的映射。
//
// 默认 Registry 由本包 init() 时填充全部内置 transform。
type Registry struct {
	factories map[string]Factory
}

// NewRegistry 创建一个空 Registry。
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory, 16)}
}

// Register 注册一个 kind → Factory。重复注册返回错误。
func (r *Registry) Register(kind string, f Factory) error {
	if kind == "" {
		return linctlerr.New(linctlerr.ErrInternal, "transform.Register: kind is empty")
	}
	if f == nil {
		return linctlerr.Newf(linctlerr.ErrInternal,
			"transform.Register: factory for %s is nil", kind)
	}
	if _, ok := r.factories[kind]; ok {
		return linctlerr.Newf(linctlerr.ErrInternal,
			"transform.Register: kind %q already registered", kind)
	}
	r.factories[kind] = f
	return nil
}

// MustRegister 同 Register 但失败时 panic（用于 init()）。
func (r *Registry) MustRegister(kind string, f Factory) {
	if err := r.Register(kind, f); err != nil {
		panic(err)
	}
}

// Build 按 kind 构造 Transform 实例。
func (r *Registry) Build(kind string, rawCfg map[string]any) (Transform, error) {
	f, ok := r.factories[kind]
	if !ok {
		return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"unknown transform kind: %q (available: %v)", kind, r.kinds())
	}
	return f(rawCfg)
}

func (r *Registry) kinds() []string {
	out := make([]string, 0, len(r.factories))
	for k := range r.factories {
		out = append(out, k)
	}
	return out
}

// DefaultRegistry 是包级注册中心，包含全部内置 transform。
//
// 由各 transform 文件的 init() 注册。
var DefaultRegistry = NewRegistry()

// asString 是 rawCfg map 中读 string 字段的辅助函数。
func asString(rawCfg map[string]any, key string, required bool) (string, error) {
	v, ok := rawCfg[key]
	if !ok {
		if required {
			return "", linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"transform: missing required field %q", key)
		}
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"transform: field %q must be string, got %T", key, v)
	}
	return s, nil
}

// asStringSlice 读取一个字符串数组字段。
func asStringSlice(rawCfg map[string]any, key string) ([]string, error) {
	v, ok := rawCfg[key]
	if !ok {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"transform: field %q must be array, got %T", key, v)
	}
	out := make([]string, 0, len(arr))
	for i, e := range arr {
		s, ok := e.(string)
		if !ok {
			return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"transform: field %q[%d] must be string, got %T", key, i, e)
		}
		out = append(out, s)
	}
	return out, nil
}

// asPairs 读取 [{from, to}, ...] 风格的字段（用于 replaceLiteral.pairs）。
func asPairs(rawCfg map[string]any, key string) ([]LiteralPair, error) {
	v, ok := rawCfg[key]
	if !ok {
		return nil, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
			"transform: field %q must be array, got %T", key, v)
	}
	out := make([]LiteralPair, 0, len(arr))
	for i, e := range arr {
		obj, ok := e.(map[string]any)
		if !ok {
			return nil, linctlerr.Newf(linctlerr.ErrConfigInvalid,
				"transform: field %q[%d] must be object, got %T", key, i, e)
		}
		fromAny, hasFrom := obj["from"]
		toAny, hasTo := obj["to"]
		if !hasFrom || !hasTo {
			return nil, fmt.Errorf("transform: %s[%d] missing 'from'/'to'", key, i)
		}
		fromS, ok := fromAny.(string)
		if !ok {
			return nil, fmt.Errorf("transform: %s[%d].from must be string", key, i)
		}
		toS, ok := toAny.(string)
		if !ok {
			return nil, fmt.Errorf("transform: %s[%d].to must be string", key, i)
		}
		out = append(out, LiteralPair{From: fromS, To: toS})
	}
	return out, nil
}

// LiteralPair 是 replaceLiteral.pairs 单条记录。
type LiteralPair struct {
	From string
	To   string
}
