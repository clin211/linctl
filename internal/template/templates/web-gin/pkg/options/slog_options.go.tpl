package options

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

var _ IOptions = (*SlogOptions)(nil)

// SlogOptions 配置 log/slog 的运行参数。
type SlogOptions struct {
	// Level 是允许输出的最低日志级别。
	// 可选值：debug / info / warn / error
	Level string `json:"level,omitempty" mapstructure:"level"`
	// AddSource 是否在日志中追加 file:line 的源码定位。
	AddSource bool `json:"add-source,omitempty" mapstructure:"add-source"`
	// Format 是日志的结构格式。
	// 可选值：json / text
	Format string `json:"format,omitempty" mapstructure:"format"`
	// TimeFormat 是 text 模式下的时间格式（Go layout）；为空时使用 RFC3339。
	TimeFormat string `json:"time-format,omitempty" mapstructure:"time-format"`
	// Output 指定日志写出位置。
	// 可选值：stdout / stderr / 文件路径
	Output string `json:"output,omitempty" mapstructure:"output"`
}

// NewSlogOptions 用默认值构造 *SlogOptions。
func NewSlogOptions() *SlogOptions {
	return &SlogOptions{
		Level:      "info",
		AddSource:  false,
		Format:     "text",
		TimeFormat: "",
		Output:     "stdout",
	}
}

// Validate 校验 SlogOptions 的参数。
func (o *SlogOptions) Validate() []error {
	var errs []error

	// 校验日志级别
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG", "INFO", "WARN", "WARNING", "ERROR":
	default:
		errs = append(errs, fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", o.Level))
	}

	// 校验日志格式
	switch o.Format {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("invalid log format: %s (must be json or text)", o.Format))
	}

	// 校验输出路径
	if o.Output != "stdout" && o.Output != "stderr" && o.Output != "" {
		// 简单判断是否为合法文件路径
		if !filepath.IsAbs(o.Output) && !strings.Contains(o.Output, "/") {
			errs = append(errs, fmt.Errorf("invalid output path: %s", o.Output))
		}
	}

	return errs
}

// AddFlags 把 SlogOptions 上的字段注册为命令行 flag。
func (o *SlogOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Level, fullPrefix+".level", o.Level, "日志级别。可选：debug / info / warn / error")
	fs.StringVar(&o.Format, fullPrefix+".format", o.Format, "日志格式。可选：json / text")
	fs.BoolVar(&o.AddSource, fullPrefix+".add-source", o.AddSource, "是否在日志中追加 file:line 的源码定位")
	fs.StringVar(&o.TimeFormat, fullPrefix+".time-format", o.TimeFormat,
		"text 模式下的时间格式（Go layout）；留空使用 RFC3339。例如 '2006-01-02 15:04:05'")
	fs.StringVar(&o.Output, fullPrefix+".output", o.Output, "日志输出位置（stdout / stderr / 文件路径）")
}

// ToSlogLevel 把字符串级别转换为 slog.Level。
func (o *SlogOptions) ToSlogLevel() slog.Level {
	switch strings.ToUpper(strings.TrimSpace(o.Level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		// 未识别的级别回退到 INFO。
		return slog.LevelInfo
	}
}

// GetWriter 根据 Output 配置返回对应的 io.Writer。
func (o *SlogOptions) GetWriter() (io.Writer, error) {
	switch o.Output {
	case "stdout", "":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// 文件输出（追加模式）
		file, err := os.OpenFile(o.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", o.Output, err)
		}
		return file, nil
	}
}

// BuildHandler 根据当前配置构造 slog.Handler。
func (o *SlogOptions) BuildHandler() (slog.Handler, error) {
	writer, err := o.GetWriter()
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level:     o.ToSlogLevel(),
		AddSource: o.AddSource,
	}

	// text 模式下若指定了自定义时间格式，注入 ReplaceAttr 替换 time。
	if o.Format == "text" && o.TimeFormat != "" {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(o.TimeFormat))
			}
			return a
		}
	}

	var handler slog.Handler
	switch o.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	return handler, nil
}

// BuildLogger 根据当前配置构造 *slog.Logger，但不影响全局 logger。
func (o *SlogOptions) BuildLogger() (*slog.Logger, error) {
	handler, err := o.BuildHandler()
	if err != nil {
		return nil, err
	}
	return slog.New(handler), nil
}

// Apply 把当前配置应用到全局 slog 默认 logger。
func (o *SlogOptions) Apply() error {
	logger, err := o.BuildLogger()
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}
