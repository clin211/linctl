package options

import (
	"errors"
{{- if .Features | Has "user" }}
	"time"
{{- end }}
	"strings"

	linhub_options "github.com/clin211/linhub/options"
	"github.com/spf13/pflag"
)

// ServerOptions 集中管理 Web 服务器的所有配置项。
//
// 各子选项均来自 linhub/options 共享库，若需要新增字段或子选项，
// 请直接在此结构体内追加并对应实现 AddFlags / Validate。
type ServerOptions struct {
	// HTTPOptions 控制 HTTP 服务器的监听与超时等参数。
	HTTPOptions *linhub_options.HTTPOptions `json:"http" mapstructure:"http"`
{{- if eq .Storage "gorm-postgres" }}

	// PostgreSQLOptions 控制 PostgreSQL 数据源。
	PostgreSQLOptions *linhub_options.PostgreSQLOptions `json:"postgresql" mapstructure:"postgresql"`
{{- else if eq .Storage "gorm-mysql" }}

	// MySQLOptions 控制 MySQL 数据源。
	MySQLOptions *linhub_options.MySQLOptions `json:"mysql" mapstructure:"mysql"`
{{- else if eq .Storage "gorm-sqlite" }}

	// SQLiteOptions 控制 SQLite 数据源。
	SQLiteOptions *linhub_options.SQLiteOptions `json:"sqlite" mapstructure:"sqlite"`
{{- else if eq .Storage "mongo" }}

	// MongoOptions 控制 MongoDB 数据源。
	MongoOptions *linhub_options.MongoOptions `json:"mongo" mapstructure:"mongo"`
{{- end }}
{{- if eq .Cache "redis" }}

	// RedisOptions 控制 Redis 缓存连接。
	RedisOptions *linhub_options.RedisOptions `json:"redis" mapstructure:"redis"`
{{- end }}
{{- if .Features | Has "otel" }}

	// OTelOptions 控制 OpenTelemetry 上报。
	OTelOptions *linhub_options.OTelOptions `json:"otel" mapstructure:"otel"`
{{- end }}
{{- if .Features | Has "user" }}

	// JWTKey 为 JWT 签名密钥，长度需 ≥ 6。
	JWTKey string `json:"jwt-key" mapstructure:"jwt-key"`
	// Expiration 为 JWT Token 的过期时长。
	Expiration time.Duration `json:"expiration" mapstructure:"expiration"`
{{- end }}
}

// NewServerOptions 创建带默认值的 ServerOptions。
func NewServerOptions() *ServerOptions {
	opts := &ServerOptions{
		HTTPOptions: linhub_options.NewHTTPOptions(),
{{- if eq .Storage "gorm-postgres" }}
		PostgreSQLOptions: linhub_options.NewPostgreSQLOptions(),
{{- else if eq .Storage "gorm-mysql" }}
		MySQLOptions: linhub_options.NewMySQLOptions(),
{{- else if eq .Storage "gorm-sqlite" }}
		SQLiteOptions: linhub_options.NewSQLiteOptions(),
{{- else if eq .Storage "mongo" }}
		MongoOptions: linhub_options.NewMongoOptions(),
{{- end }}
{{- if eq .Cache "redis" }}
		RedisOptions: linhub_options.NewRedisOptions(),
{{- end }}
{{- if .Features | Has "otel" }}
		OTelOptions: linhub_options.NewOTelOptions(),
{{- end }}
{{- if .Features | Has "user" }}
		JWTKey:     "",
		Expiration: 2 * time.Hour,
{{- end }}
	}
	opts.HTTPOptions.Addr = ":8080"
	return opts
}

// AddFlags 将 ServerOptions 中的字段绑定为命令行 flag。
func (o *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.HTTPOptions.AddFlags(fs, "http")
{{- if eq .Storage "gorm-postgres" }}
	o.PostgreSQLOptions.AddFlags(fs, "postgresql")
{{- else if eq .Storage "gorm-mysql" }}
	o.MySQLOptions.AddFlags(fs, "mysql")
{{- else if eq .Storage "gorm-sqlite" }}
	o.SQLiteOptions.AddFlags(fs, "sqlite")
{{- else if eq .Storage "mongo" }}
	o.MongoOptions.AddFlags(fs, "mongo")
{{- end }}
{{- if eq .Cache "redis" }}
	o.RedisOptions.AddFlags(fs, "redis")
{{- end }}
{{- if .Features | Has "otel" }}
	o.OTelOptions.AddFlags(fs, "otel")
{{- end }}
{{- if .Features | Has "user" }}
	fs.StringVar(&o.JWTKey, "jwt-key", o.JWTKey, "JWT 签名密钥（长度需 ≥ 6）。")
	fs.DurationVar(&o.Expiration, "expiration", o.Expiration, "JWT Token 的过期时长。")
{{- end }}
}

// Complete 补全那些未设置但需要有效数据的字段。
func (o *ServerOptions) Complete() error {
	return nil
}

// Validate 校验所有子选项是否合法。
func (o *ServerOptions) Validate() error {
	var msgs []string

{{- if .Features | Has "user" }}
	if len(o.JWTKey) < 6 {
		msgs = append(msgs, "JWTKey must be at least 6 characters long")
	}
{{- end }}

	for _, e := range o.HTTPOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- if eq .Storage "gorm-postgres" }}
	for _, e := range o.PostgreSQLOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- else if eq .Storage "gorm-mysql" }}
	for _, e := range o.MySQLOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- else if eq .Storage "gorm-sqlite" }}
	for _, e := range o.SQLiteOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- else if eq .Storage "mongo" }}
	for _, e := range o.MongoOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- end }}
{{- if eq .Cache "redis" }}
	for _, e := range o.RedisOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- end }}
{{- if .Features | Has "otel" }}
	for _, e := range o.OTelOptions.Validate() {
		msgs = append(msgs, e.Error())
	}
{{- end }}

	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "; "))
	}
	return nil
}
