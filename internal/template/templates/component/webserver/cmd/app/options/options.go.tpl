// nolint: err113
package options

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}"
	genericoptions "{{ .Project.Metadata.Module }}/pkg/options"
)

// ServerOptions 是服务器的命令行 / 配置文件选项集合。
//
// 字段顺序与 configs/{{ .Component.Name }}.yaml 中的 key 一一对应；
// 通过 Validate() 校验值，Config() 转换为内部使用的 *{{ .Component.Name }}.Config。
type ServerOptions struct {
	// JWTKey 是 JWT 的签名密钥（长度 >= 6）。
	JWTKey string `json:"jwt-key" mapstructure:"jwt-key"`
	// Expiration 是 JWT Token 的过期时长。
	Expiration time.Duration `json:"expiration" mapstructure:"expiration"`
	// TLSOptions 是 HTTPS 相关的 TLS 配置。
	TLSOptions *genericoptions.TLSOptions `json:"tls" mapstructure:"tls"`
	// HTTPOptions 是 HTTP 服务的监听配置。
	HTTPOptions *genericoptions.HTTPOptions `json:"http" mapstructure:"http"`
{{- if eq .Component.Storage "gorm-mysql" }}
	// MySQLOptions 是 MySQL 数据库的连接配置。
	MySQLOptions *genericoptions.MySQLOptions `json:"mysql" mapstructure:"mysql"`
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
	// PostgreSQLOptions 是 PostgreSQL 数据库的连接配置。
	PostgreSQLOptions *genericoptions.PostgreSQLOptions `json:"postgresql" mapstructure:"postgresql"`
{{- end }}
{{- if eq .Component.Storage "mongo" }}
	// MongoOptions 是 MongoDB 的连接配置。
	MongoOptions *genericoptions.MongoOptions `json:"mongo" mapstructure:"mongo"`
{{- end }}
	// OTelOptions 是 OpenTelemetry 的导出配置（trace / metric / log）。
	OTelOptions *genericoptions.OTelOptions `json:"otel" mapstructure:"otel"`
	// SlogOptions 是结构化日志的格式 / 级别 / 来源开关；与 OTel.Slog 互不冲突，
	// 这里独立存在便于在不接 OTel 时也能调日志选项。
	SlogOptions *genericoptions.SlogOptions `json:"slog" mapstructure:"slog"`
}

// NewServerOptions 用合理的默认值构造 *ServerOptions。
func NewServerOptions() *ServerOptions {
	opts := &ServerOptions{
		JWTKey:      "Rtg8BPKNEf2mB4mgvKONGPZZQSaJWNLijxR42qRgq0iBb5",
		Expiration:  2 * time.Hour,
		TLSOptions:  genericoptions.NewTLSOptions(),
		HTTPOptions: genericoptions.NewHTTPOptions(),
{{- if eq .Component.Storage "gorm-mysql" }}
		MySQLOptions: genericoptions.NewMySQLOptions(),
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
		PostgreSQLOptions: genericoptions.NewPostgreSQLOptions(),
{{- end }}
{{- if eq .Component.Storage "mongo" }}
		MongoOptions: genericoptions.NewMongoOptions(),
{{- end }}
		OTelOptions: genericoptions.NewOTelOptions(),
		SlogOptions: genericoptions.NewSlogOptions(),
	}
	opts.HTTPOptions.Addr = ":{{ default 8080 .Component.Port }}"

	return opts
}

// AddFlags 把 ServerOptions 上的字段绑定为命令行 flag。
func (o *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.JWTKey, "jwt-key", o.JWTKey, "JWT 签名密钥（长度 >= 6）")
	fs.DurationVar(&o.Expiration, "expiration", o.Expiration, "JWT Token 的过期时长")
	o.TLSOptions.AddFlags(fs, "tls")
	o.HTTPOptions.AddFlags(fs, "http")
{{- if eq .Component.Storage "gorm-mysql" }}
	o.MySQLOptions.AddFlags(fs, "mysql")
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
	o.PostgreSQLOptions.AddFlags(fs, "postgresql")
{{- end }}
{{- if eq .Component.Storage "mongo" }}
	o.MongoOptions.AddFlags(fs, "mongo")
{{- end }}
	o.OTelOptions.AddFlags(fs, "otel")
	o.SlogOptions.AddFlags(fs, "slog")
}

// Complete 在所有 flag / 配置加载完之后做必要的补全；当前为空。
func (o *ServerOptions) Complete() error {
	return nil
}

// Validate 校验 ServerOptions 是否合法，校验失败时返回聚合错误。
func (o *ServerOptions) Validate() error {
	errs := []error{}
	if len(o.JWTKey) < 6 {
		errs = append(errs, errors.New("JWTKey must be at least 6 characters long"))
	}

	errs = append(errs, o.TLSOptions.Validate()...)
	errs = append(errs, o.HTTPOptions.Validate()...)
{{- if eq .Component.Storage "gorm-mysql" }}
	errs = append(errs, o.MySQLOptions.Validate()...)
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
	errs = append(errs, o.PostgreSQLOptions.Validate()...)
{{- end }}
{{- if eq .Component.Storage "mongo" }}
	errs = append(errs, o.MongoOptions.Validate()...)
{{- end }}
	errs = append(errs, o.OTelOptions.Validate()...)
	errs = append(errs, o.SlogOptions.Validate()...)

	return utilerrors.NewAggregate(errs)
}

// Config 把 ServerOptions 转换为内部使用的 *{{ .Component.Name }}.Config。
func (o *ServerOptions) Config() (*{{ .Component.Name }}.Config, error) {
	return &{{ .Component.Name }}.Config{
		JWTKey:      o.JWTKey,
		Expiration:  o.Expiration,
		TLSOptions:  o.TLSOptions,
		HTTPOptions: o.HTTPOptions,
{{- if eq .Component.Storage "gorm-mysql" }}
		MySQLOptions: o.MySQLOptions,
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
		PostgreSQLOptions: o.PostgreSQLOptions,
{{- end }}
{{- if eq .Component.Storage "mongo" }}
		MongoOptions: o.MongoOptions,
{{- end }}
	}, nil
}
