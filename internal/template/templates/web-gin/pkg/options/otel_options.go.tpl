package options

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	emptyexporter "{{ .Project.Metadata.Module }}/pkg/otel/exporter/empty"
	"{{ .Project.Metadata.Module }}/pkg/otelslog"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// 编译期断言：确保 *OTelOptions 实现了 IOptions 接口。
var _ IOptions = (*OTelOptions)(nil)

// OutputMode 表示 OpenTelemetry 数据的导出模式。
type OutputMode string

const (
	OutputModeOTLP    OutputMode = "otel"    // 通过 OTLP gRPC 发送给 OpenTelemetry Collector
	OutputModeFile    OutputMode = "file"    // 写到文件
	OutputModeConsole OutputMode = "console" // 写到标准输出
	OutputModeClassic OutputMode = "classic" // 走传统 metrics + logging（不启用 OTel）

	// 混合模式：log → stdout、metric → prometheus、trace → otel collector
	OutputModeHybrid OutputMode = "hybrid"
)

// String 实现 fmt.Stringer。
func (o OutputMode) String() string {
	return string(o)
}

// IsValid 校验 OutputMode 是否合法。
func (o OutputMode) IsValid() bool {
	switch o {
	case OutputModeConsole, OutputModeFile, OutputModeOTLP, OutputModeClassic, OutputModeHybrid:
		return true
	default:
		return false
	}
}

// outputModeFlag 让 OutputMode 实现 pflag.Value。
type outputModeFlag OutputMode

func (f *outputModeFlag) String() string { return string(*f) }
func (f *outputModeFlag) Type() string   { return "string" }
func (f *outputModeFlag) Set(s string) error {
	mode := OutputMode(s)
	if !mode.IsValid() {
		return fmt.Errorf("invalid output mode: %s, valid options: otlp, console, file, classic, hybrid", s)
	}
	*f = outputModeFlag(mode)
	return nil
}

// Provider 是带 Shutdown 能力的 OTel provider 抽象。
type Provider interface {
	Shutdown(context.Context) error
}

// OTelProviders 聚合 OpenTelemetry 三套 provider（trace / metric / log）。
type OTelProviders struct {
	tracer *trace.TracerProvider
	meter  *metric.MeterProvider
	logger *log.LoggerProvider
}

// OTelOptions 是 OpenTelemetry 的配置选项集合。
type OTelOptions struct {
	// 连接配置
	Endpoint string `json:"endpoint,omitempty" mapstructure:"endpoint"`
	Insecure bool   `json:"insecure,omitempty" mapstructure:"insecure"`

	// 服务标识
	ServiceName       string `json:"service-name,omitempty" mapstructure:"service-name"`
	ServiceVersion    string `json:"service-version,omitempty" mapstructure:"service-version"`
	ServiceInstanceID string `json:"service-instance-id,omitempty" mapstructure:"service-instance-id"`
	Environment       string `json:"environment,omitempty" mapstructure:"environment"`

	// 行为配置
	SamplingRatio float64 `json:"sampling-ratio,omitempty" mapstructure:"sampling-ratio"`
	WithResource  bool    `json:"with-resource,omitempty" mapstructure:"with-resource"`

	// 输出配置
	OutputMode OutputMode `json:"output-mode,omitempty" mapstructure:"output-mode"`
	OutputDir  string     `json:"output-dir,omitempty" mapstructure:"output-dir"`

	// 日志配置
	Level     string `json:"level,omitempty" mapstructure:"level"`
	AddSource bool   `json:"add-source,omitempty" mapstructure:"add-source"`

	Slog *SlogOptions `json:"slog,omitempty" mapstructure:"slog"`

	// 内部状态
	mu        sync.RWMutex
	providers *OTelProviders
	files     []io.Closer
	resource  *resource.Resource
}

var (
	resourceOnce sync.Once
	otelResource *resource.Resource
)

// NewOTelOptions 用合理默认值构造 *OTelOptions。
func NewOTelOptions() *OTelOptions {
	hostname, _ := os.Hostname()
	opts := &OTelOptions{
		ServiceName:       "unknown-service",
		ServiceVersion:    "1.0.0",
		ServiceInstanceID: hostname,
		Environment:       "development",
		Endpoint:          "localhost:4317",
		Insecure:          true,
		WithResource:      false,
		SamplingRatio:     1.0,
		OutputMode:        OutputModeClassic,
		OutputDir:         "./otel-output",
		Level:             "info",
		AddSource:         false,
		Slog:              NewSlogOptions(),
		providers:         &OTelProviders{},
		files:             make([]io.Closer, 0),
	}

	// 把 OTel 自身的 Level / AddSource 同步到 Slog。
	opts.syncSlogOptions()
	return opts
}

// syncSlogOptions 把 Level / AddSource 等字段同步到 Slog 子配置。
func (o *OTelOptions) syncSlogOptions() {
	if o.Slog != nil {
		o.Slog.Level = o.Level
		o.Slog.AddSource = o.AddSource
	}
}

// Validate 校验 OTelOptions 的所有字段。
func (o *OTelOptions) Validate() []error {
	var errs []error

	// 校验前先同步 Slog 子配置
	o.syncSlogOptions()

	// 校验 Slog 子配置
	if o.Slog != nil {
		errs = append(errs, o.Slog.Validate()...)
	}

	// 校验 OutputMode
	if !o.OutputMode.IsValid() {
		errs = append(errs, fmt.Errorf("invalid output mode: %s", o.OutputMode))
	}

	// 校验必填字段
	if o.ServiceName == "" {
		errs = append(errs, fmt.Errorf("service name is required"))
	}
	if o.ServiceInstanceID == "" {
		errs = append(errs, fmt.Errorf("service instance ID is required"))
	}

	// OTLP 模式下必须有 endpoint
	if o.OutputMode == OutputModeOTLP && o.Endpoint == "" {
		errs = append(errs, fmt.Errorf("endpoint is required for OTLP output mode"))
	}

	// 采样率必须在 [0, 1] 区间
	if o.SamplingRatio < 0 || o.SamplingRatio > 1 {
		errs = append(errs, fmt.Errorf("sampling ratio must be between 0 and 1, got: %f", o.SamplingRatio))
	}

	// 文件模式下必须指定输出目录
	if o.OutputMode == OutputModeFile && o.OutputDir == "" {
		errs = append(errs, fmt.Errorf("output directory is required for file output mode"))
	}

	return errs
}

// AddFlags 把 OTelOptions 上的字段注册为命令行 flag。
func (o *OTelOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "服务名")
	fs.StringVar(&o.ServiceVersion, fullPrefix+".service-version", o.ServiceVersion, "服务版本")
	fs.StringVar(&o.ServiceInstanceID, fullPrefix+".service-instance-id", o.ServiceInstanceID, "服务实例 ID（留空则自动取 hostname）")
	fs.StringVar(&o.Environment, fullPrefix+".environment", o.Environment, "运行环境（development / production / ...）")
	fs.StringVar(&o.Endpoint, fullPrefix+".endpoint", o.Endpoint, "OTLP 接收端地址")
	fs.BoolVar(&o.Insecure, fullPrefix+".insecure", o.Insecure, "是否使用明文连接 OTLP")
	fs.Float64Var(&o.SamplingRatio, fullPrefix+".sampling-ratio", o.SamplingRatio, "采样率（0.0 ~ 1.0）")
	fs.BoolVar(&o.WithResource, fullPrefix+".with-resource", o.WithResource, "是否在 Resource 中包含系统信息")
	fs.Var((*outputModeFlag)(&o.OutputMode), fullPrefix+".output-mode", "导出模式：otel / console / file / classic / hybrid")
	fs.StringVar(&o.OutputDir, fullPrefix+".output-dir", o.OutputDir, "文件模式下的输出目录")
	fs.StringVar(&o.Level, fullPrefix+".level", o.Level, "日志级别：debug / info / warn / error")
	fs.BoolVar(&o.AddSource, fullPrefix+".add-source", o.AddSource, "是否在日志中追加 file:line 的源码定位")
	if o.Slog != nil {
		o.Slog.AddFlags(fs, fullPrefix+".slog")
	}
}

// GetResource 创建并缓存 OTel resource 配置（一次构造、复用）。
func (o *OTelOptions) GetResource() *resource.Resource {
	resourceOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 基础属性
		attrs := []resource.Option{
			resource.WithAttributes(
				semconv.ServiceName(o.ServiceName),
				semconv.ServiceVersion(o.ServiceVersion),
				semconv.ServiceInstanceID(o.ServiceInstanceID),
				semconv.DeploymentEnvironment(o.Environment),
			),
		}

		// 启用 WithResource 时附加系统信息
		if o.WithResource {
			attrs = append(attrs,
				resource.WithOS(),
				resource.WithProcess(),
				resource.WithContainer(),
				resource.WithHost(),
			)
		}

		var err error
		otelResource, err = resource.New(ctx, attrs...)
		if err != nil {
			// 失败时回退到默认 resource
			otelResource = resource.Default()
		}
	})
	return otelResource
}

// createFileWriter 创建一个面向文件的 io.Writer，并把句柄登记到 o.files。
func (o *OTelOptions) createFileWriter(name string) (io.Writer, error) {
	if err := os.MkdirAll(o.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := filepath.Join(o.OutputDir, name+".json")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", filename, err)
	}

	o.files = append(o.files, file)
	return file, nil
}

// initTraces 初始化 trace provider。
func (o *OTelOptions) initTraces(ctx context.Context) error {
	var (
		exporter trace.SpanExporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic:
		exporter = emptyexporter.NewEmptyExporter()
	case OutputModeOTLP, OutputModeHybrid:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("traces")
		if writerErr != nil {
			return fmt.Errorf("failed to create trace writer: %w", writerErr)
		}
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(writer))
	default: // OutputModeConsole 与未识别情况
		exporter, err = stdouttrace.New(stdouttrace.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(o.GetResource()),
		trace.WithSampler(trace.TraceIDRatioBased(o.SamplingRatio)),
	)

	o.providers.tracer = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// initMetrics 初始化 metric provider。
func (o *OTelOptions) initMetrics(ctx context.Context) error {
	var (
		reader   metric.Reader
		exporter metric.Exporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic, OutputModeHybrid:
		reader, err = prometheus.New()
	case OutputModeOTLP:
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(o.Endpoint)}
		if o.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, err := o.createFileWriter("metrics")
		if err != nil {
			return fmt.Errorf("failed to create metrics writer: %w", err)
		}
		exporter, err = stdoutmetric.New(stdoutmetric.WithWriter(writer))
	default: // OutputModeConsole 与未识别情况
		exporter, err = stdoutmetric.New(stdoutmetric.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	if exporter != nil {
		reader = metric.NewPeriodicReader(exporter)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(o.GetResource()),
	)

	o.providers.meter = mp
	otel.SetMeterProvider(mp)
	return nil
}

// initLogs 初始化 log provider 与 slog 集成。
func (o *OTelOptions) initLogs(ctx context.Context) error {
	var (
		exporter log.Exporter
		err      error
	)

	switch o.OutputMode {
	case OutputModeClassic, OutputModeHybrid:
		return o.Slog.Apply()
	case OutputModeOTLP:
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(o.Endpoint),
		}
		if o.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		exporter, err = otlploggrpc.New(ctx, opts...)
	case OutputModeFile:
		writer, writerErr := o.createFileWriter("logs")
		if writerErr != nil {
			return fmt.Errorf("failed to create logs writer: %w", writerErr)
		}
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(writer))
	default: // OutputModeConsole 与未识别情况
		exporter, err = stdoutlog.New(stdoutlog.WithWriter(os.Stdout))
	}

	if err != nil {
		return fmt.Errorf("failed to create log exporter: %w", err)
	}

	var processor log.Processor
	if o.OutputMode == OutputModeConsole {
		processor = log.NewSimpleProcessor(exporter)
	} else {
		processor = log.NewBatchProcessor(exporter)
	}

	lp := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(o.GetResource()),
	)

	o.providers.logger = lp
	global.SetLoggerProvider(lp)

	// 把 slog 默认 logger 切换到 OTel bridge
	logger := otelslog.NewLogger(
		o.ServiceName,
		otelslog.WithLoggerProvider(global.GetLoggerProvider()),
		otelslog.WithSource(o.AddSource),
		otelslog.WithLevelString(o.Level),
	)
	slog.SetDefault(logger)

	return nil
}

// Apply 把当前配置应用到 OTel SDK：依次启用 trace / metric / log provider。
func (o *OTelOptions) Apply() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 同步 Slog 子配置
	o.syncSlogOptions()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 按需初始化各 provider
	if err := o.initTraces(ctx); err != nil {
		return fmt.Errorf("failed to initialize traces: %w", err)
	}

	if err := o.initMetrics(ctx); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	if err := o.initLogs(ctx); err != nil {
		return fmt.Errorf("failed to initialize logs: %w", err)
	}

	return nil
}

// Shutdown 关停所有 provider 与已打开的文件句柄；返回所有错误的聚合。
func (o *OTelOptions) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var errs []error

	// 关停 provider
	if o.providers.tracer != nil {
		if err := o.providers.tracer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if o.providers.meter != nil {
		if err := o.providers.meter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}

	if o.providers.logger != nil {
		if err := o.providers.logger.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
		}
	}

	// 关闭文件句柄
	for _, file := range o.files {
		if err := file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("file close: %w", err))
		}
	}
	o.files = o.files[:0] // 清空 slice

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// GetTracerProvider 返回当前 *trace.TracerProvider（可能为 nil）。
func (o *OTelOptions) GetTracerProvider() *trace.TracerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.tracer
}

// GetMeterProvider 返回当前 *metric.MeterProvider（可能为 nil）。
func (o *OTelOptions) GetMeterProvider() *metric.MeterProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.meter
}

// GetLoggerProvider 返回当前 *log.LoggerProvider（可能为 nil）。
func (o *OTelOptions) GetLoggerProvider() *log.LoggerProvider {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.providers.logger
}
