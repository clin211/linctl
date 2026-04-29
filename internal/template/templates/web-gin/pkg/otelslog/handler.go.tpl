package otelslog // import "go.opentelemetry.io/contrib/bridges/otelslog"

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// NewLogger 返回一个由 [Handler] 支撑的 *slog.Logger。
//
// Handler 的构造细节见 [NewHandler]。
func NewLogger(name string, options ...Option) *slog.Logger {
	return slog.New(NewHandler(name, options...))
}

type config struct {
	provider   log.LoggerProvider
	version    string
	schemaURL  string
	attributes []attribute.KeyValue
	source     bool
	level      slog.Level
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	if c.attributes != nil {
		opts = append(opts, log.WithInstrumentationAttributes(c.attributes...))
	}
	return c.provider.Logger(name, opts...)
}

// Option 用于配置 [Handler]。
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion 配置 [Handler] 内部 [log.Logger] 使用的版本号。
//
// version 通常等于被记录代码包的版本。
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL 配置 [Handler] 内部 [log.Logger] 的语义约定 schema URL。
//
// schemaURL 应当是日志记录所使用的语义约定的 schema URL。
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithAttributes 配置 [Handler] 内部 [log.Logger] 的 instrumentation scope 属性。
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optFunc(func(c config) config {
		c.attributes = attributes
		return c
	})
}

// WithLoggerProvider 配置 [Handler] 创建 [log.Logger] 时使用的 [log.LoggerProvider]。
//
// 不传递此 Option 时，Handler 会使用全局 LoggerProvider。
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

// WithSource 配置 [Handler] 是否在日志属性中追加源码定位（file:line:func）。
func WithSource(source bool) Option {
	return optFunc(func(c config) config {
		c.source = source
		return c
	})
}

// WithLevel 配置 Handler 的最低日志级别，低于该级别的日志会被丢弃。
func WithLevel(level slog.Level) Option {
	return optFunc(func(c config) config {
		c.level = level
		return c
	})
}

// WithLevelString 通过字符串形式配置 Handler 的最低日志级别。
//
// 支持的取值："DEBUG"、"INFO"、"WARN"、"WARNING"、"ERROR"。
// 不在以上集合内的取值默认按 INFO 处理。
func WithLevelString(levelStr string) Option {
	level := parseLevelString(levelStr)
	return WithLevel(level)
}

// parseLevelString 把字符串日志级别转换为 slog.Level。
func parseLevelString(levelStr string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(levelStr)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// 未知级别回退到 INFO
		return slog.LevelInfo
	}
}

// Handler 是把 slog 日志转发给 OpenTelemetry 的 [slog.Handler] 实现。
// 转换规则参见包级文档。
type Handler struct {
	// noCmp 显式让 Handler 不可比较，方便后续扩展字段时保持向后兼容。
	noCmp [0]func() //nolint:unused  // 实际有用，见上注释。

	attrs  *kvBuffer
	group  *group
	logger log.Logger
	level  slog.Level // 最低日志级别字段

	source bool
}

// 编译期断言：*Handler 实现了 slog.Handler。
var _ slog.Handler = (*Handler)(nil)

// NewHandler 返回一个新的 [Handler]，用作 [slog.Handler]。
//
// 不传递 [WithLoggerProvider] 时，返回的 Handler 会使用全局 LoggerProvider。
//
// name 用来唯一标识被记录代码（通常是包名）；为空时由底层 [log.Logger] 决定默认值。
func NewHandler(name string, options ...Option) *Handler {
	cfg := newConfig(options)
	return &Handler{
		logger: cfg.logger(name),
		source: cfg.source,
		level:  cfg.level, // 设置最低日志级别
	}
}

// Handle 处理一条传入的 slog.Record。
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	h.logger.Emit(ctx, h.convertRecord(record))
	return nil
}

func (h *Handler) convertRecord(r slog.Record) log.Record {
	var record log.Record
	record.SetTimestamp(r.Time)
	record.SetBody(log.StringValue(r.Message))

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	record.SetSeverity(log.Severity(r.Level + sevOffset))
	record.SetSeverityText(r.Level.String())

	if h.source {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		record.AddAttributes(
			log.String(string(semconv.CodeFilePathKey), f.File),
			log.String(string(semconv.CodeFunctionNameKey), f.Function),
			log.Int(string(semconv.CodeLineNumberKey), f.Line),
		)
	}

	if h.attrs.Len() > 0 {
		record.AddAttributes(h.attrs.KeyValues()...)
	}

	n := r.NumAttrs()
	if h.group != nil {
		if n > 0 {
			buf := newKVBuffer(n)
			r.Attrs(buf.AddAttr)
			record.AddAttributes(h.group.KeyValue(buf.KeyValues()...))
		} else {
			// Handler 在没有属性时不应该输出 group。
			g := h.group.NextNonEmpty()
			if g != nil {
				record.AddAttributes(g.KeyValue())
			}
		}
	} else if n > 0 {
		buf := newKVBuffer(n)
		r.Attrs(buf.AddAttr)
		record.AddAttributes(buf.KeyValues()...)
	}

	return record
}

// Enabled 返回 Handler 是否对给定 ctx + level 启用日志记录。
func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool {
	// 首先检查本地级别过滤
	if l < h.level {
		return false
	}

	const sevOffset = slog.Level(log.SeverityDebug) - slog.LevelDebug
	param := log.EnabledParameters{Severity: log.Severity(l + sevOffset)}
	return h.logger.Enabled(ctx, param)
}

// WithAttrs 返回一个新 [slog.Handler]，在原 h 的基础上附带给定 attrs。
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	if h2.group != nil {
		h2.group = h2.group.Clone()
		h2.group.AddAttrs(attrs)
	} else {
		if h2.attrs == nil {
			h2.attrs = newKVBuffer(len(attrs))
		} else {
			h2.attrs = h2.attrs.Clone()
		}
		h2.attrs.AddAttrs(attrs)
	}
	return &h2
}

// WithGroup 返回一个新 [slog.Handler]，在原 h 的基础上把后续日志放到指定 group 名下。
func (h *Handler) WithGroup(name string) slog.Handler {
	h2 := *h
	h2.group = &group{name: name, next: h2.group}
	return &h2
}

// group 表示从 slog 接收到的一个分组。
type group struct {
	// name 是 group 名。
	name string
	// attrs 是关联到该 group 的属性列表。
	attrs *kvBuffer
	// next 指向包含本 group 的上级 group。
	//
	// 在 OpenTelemetry 中 group 体现为 map 类型的 value。对于以下 slog 调用：
	//
	//   WithGroup("G").WithGroup("H").WithGroup("I")
	//
	// 对应的 OTel log value 层级如下：
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{ "{{" }}
	//       Key: "H",
	//       Value: []KeyValue{{ "{{" }}
	//         Key: "I",
	//         Value: []KeyValue{},
	//       {{ "}}" }},
	//     {{ "}}" }},
	//   }
	//
	// 当 Info("msg", "key", "value") 或 WithAttrs("key", "value") 记录属性时，
	// 这些属性需要被加到「叶子」group 上。沿用上面的例子，叶子是 "I"：
	//
	//   KeyValue{
	//     Key: "G",
	//     Value: []KeyValue{{ "{{" }}
	//       Key: "H",
	//       Value: []KeyValue{{ "{{" }}
	//         Key: "I",
	//         Value: []KeyValue{
	//           String("key", "value"),
	//         },
	//       {{ "}}" }},
	//     {{ "}}" }},
	//   }
	//
	// 因此 group 之间用链表组织，链表的「头」是叶子 group。沿用上面的例子，
	// 内存中的 group 链表是：
	//
	//   *group{"I", next: *group{"H", next: *group{"G"{{ "}}" }}{{ "}}" }}{{ "}}" }}
	next *group
}

// NextNonEmpty 沿 g 的链表向上找到第一个含属性的 group（包含 g 自身）。
// 找不到时返回 nil。
func (g *group) NextNonEmpty() *group {
	if g == nil || g.attrs.Len() > 0 {
		return g
	}
	return g.next.NextNonEmpty()
}

// KeyValue 把 g 与 kvs 一起渲染为 [log.KindMap] 类型的 [log.KeyValue]。
//
// kvs 只在返回值中出现，不会被加到 group 自身的 attrs 上。
//
// 该方法不检查 g 是否为空，调用方需自行保证 g 非空或 kvs 非空，
// 才能得到符合 slog 语义的 group 结果。
func (g *group) KeyValue(kvs ...log.KeyValue) log.KeyValue {
	// 调用方已确认 group g 非空。
	out := log.Map(g.name, g.attrs.KeyValues(kvs...)...)
	g = g.next
	for g != nil {
		// Handler 在没有属性时不应该输出 group。
		if g.attrs.Len() > 0 {
			out = log.Map(g.name, g.attrs.KeyValues(out)...)
		}
		g = g.next
	}
	return out
}

// Clone 返回 g 的一个深拷贝。
func (g *group) Clone() *group {
	if g == nil {
		return nil
	}
	g2 := *g
	g2.attrs = g2.attrs.Clone()
	return &g2
}

// AddAttrs 把 attrs 追加到 g.attrs。
func (g *group) AddAttrs(attrs []slog.Attr) {
	if g.attrs == nil {
		g.attrs = newKVBuffer(len(attrs))
	}
	g.attrs.AddAttrs(attrs)
}

type kvBuffer struct {
	data []log.KeyValue
}

func newKVBuffer(n int) *kvBuffer {
	return &kvBuffer{data: make([]log.KeyValue, 0, n)}
}

// Len 返回 b 中保存的 [log.KeyValue] 数量。
func (b *kvBuffer) Len() int {
	if b == nil {
		return 0
	}
	return len(b.data)
}

// Clone 返回 b 的一个深拷贝。
func (b *kvBuffer) Clone() *kvBuffer {
	if b == nil {
		return nil
	}
	return &kvBuffer{data: slices.Clone(b.data)}
}

// KeyValues 把 kvs 追加到 b 已保存的 [log.KeyValue] 后返回。
func (b *kvBuffer) KeyValues(kvs ...log.KeyValue) []log.KeyValue {
	if b == nil {
		return kvs
	}
	return append(b.data, kvs...)
}

// AddAttrs 把 attrs 追加到 b。
func (b *kvBuffer) AddAttrs(attrs []slog.Attr) {
	b.data = slices.Grow(b.data, len(attrs))
	for _, a := range attrs {
		_ = b.AddAttr(a)
	}
}

// AddAttr 把 attr 追加到 b 并返回 true。
//
// 这个签名设计成可直接传给 [slog.Record].AddAttributes。
//
// 当 attr 是 key 为空的 group 时，其内部的 attrs 会被「展平」追加。
// 当 attr 自身为空时，会被丢弃。
func (b *kvBuffer) AddAttr(attr slog.Attr) bool {
	if attr.Key == "" {
		if attr.Value.Kind() == slog.KindGroup {
			// Handler 应该把 key 为空的 group 内嵌展开。
			for _, a := range attr.Value.Group() {
				b.data = append(b.data, log.KeyValue{
					Key:   a.Key,
					Value: convert(a.Value),
				})
			}
			return true
		}

		if attr.Value.Any() == nil {
			// Handler 应该忽略空 attr。
			return true
		}
	}
	b.data = append(b.data, log.KeyValue{
		Key:   attr.Key,
		Value: convert(attr.Value),
	})
	return true
}

func convert(v slog.Value) log.Value {
	switch v.Kind() {
	case slog.KindAny:
		return convertValue(v.Any())
	case slog.KindBool:
		return log.BoolValue(v.Bool())
	case slog.KindDuration:
		return log.Int64Value(v.Duration().Nanoseconds())
	case slog.KindFloat64:
		return log.Float64Value(v.Float64())
	case slog.KindInt64:
		return log.Int64Value(v.Int64())
	case slog.KindString:
		return log.StringValue(v.String())
	case slog.KindTime:
		return log.Int64Value(v.Time().UnixNano())
	case slog.KindUint64:
		const maxInt64 = ^uint64(0) >> 1
		u := v.Uint64()
		if u > maxInt64 {
			return log.Float64Value(float64(u))
		}
		return log.Int64Value(int64(u))
	case slog.KindGroup:
		g := v.Group()
		buf := newKVBuffer(len(g))
		buf.AddAttrs(g)
		return log.MapValue(buf.data...)
	case slog.KindLogValuer:
		return convert(v.Resolve())
	default:
		// 兜底处理：尽量优雅而不是 panic。
		// 对开发者来说，把意外类型加 "unhandled: " 前缀
		// 至少比让用户的代码 panic 友好得多。
		return log.StringValue(fmt.Sprintf("unhandled: (%s) %+v", v.Kind(), v.Any()))
	}
}
