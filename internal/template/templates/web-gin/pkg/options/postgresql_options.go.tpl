package options

import (
	"time"

	"github.com/spf13/pflag"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"{{ .Project.Metadata.Module }}/pkg/db"
)

var _ IOptions = (*PostgreSQLOptions)(nil)

// PostgreSQLOptions 定义 PostgreSQL 数据库连接的配置项。
type PostgreSQLOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewPostgreSQLOptions 构造一个带默认值的 *PostgreSQLOptions。
func NewPostgreSQLOptions() *PostgreSQLOptions {
	return &PostgreSQLOptions{
		Addr:                  "127.0.0.1:5432",
		Username:              "postgres",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate 校验 PostgreSQLOptions 的参数。
func (o *PostgreSQLOptions) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags 把 PostgreSQLOptions 上的字段注册为命令行 flag。
func (o *PostgreSQLOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr,
		"PostgreSQL 服务地址；留空时其余 PG 配置项被忽略")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "PostgreSQL 服务用户名")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "PostgreSQL 服务密码（与 username 配对使用）")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, "应用使用的数据库名")
	fs.IntVar(&o.MaxIdleConnections, fullPrefix+".max-idle-connections", o.MaxOpenConnections, "连接池允许的最大空闲连接数")
	fs.IntVar(&o.MaxOpenConnections, fullPrefix+".max-open-connections", o.MaxOpenConnections, "连接池允许的最大打开连接数")
	fs.DurationVar(&o.MaxConnectionLifeTime, fullPrefix+".max-connection-life-time", o.MaxConnectionLifeTime, "单个连接的最长生命周期")
	fs.IntVar(&o.LogLevel, fullPrefix+".log-mode", o.LogLevel, "GORM 日志级别")
}

// NewDB 用当前配置创建一个 GORM PostgreSQL 实例。
func (o *PostgreSQLOptions) NewDB() (*gorm.DB, error) {
	opts := &db.PostgreSQLOptions{
		Addr:                  o.Addr,
		Username:              o.Username,
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		Logger:                gormlogger.Default,
	}

	return db.NewPostgreSQL(opts)
}
