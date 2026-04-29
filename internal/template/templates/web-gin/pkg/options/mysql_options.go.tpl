package options

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"{{ .Project.Metadata.Module }}/pkg/db"
)

var _ IOptions = (*MySQLOptions)(nil)

// MySQLOptions 定义 MySQL 数据库连接的配置项。
type MySQLOptions struct {
	Addr                  string        `json:"addr,omitempty" mapstructure:"addr"`
	Username              string        `json:"username,omitempty" mapstructure:"username"`
	Password              string        `json:"-" mapstructure:"password"`
	Database              string        `json:"database" mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int           `json:"log-level" mapstructure:"log-level"`
}

// NewMySQLOptions 构造一个带默认值的 *MySQLOptions。
func NewMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Addr:                  "127.0.0.1:3306",
		Username:              "",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
	}
}

// Validate 校验 MySQLOptions 的参数。
func (o *MySQLOptions) Validate() []error {
	errs := []error{}

	return errs
}

// AddFlags 把 MySQLOptions 上的字段注册为命令行 flag。
func (o *MySQLOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".host", o.Addr, "MySQL 服务地址")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "MySQL 服务用户名")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "MySQL 服务密码（与 username 配对使用）")
	fs.StringVar(&o.Database, fullPrefix+".database", o.Database, "应用使用的数据库名")
	fs.IntVar(&o.MaxIdleConnections, fullPrefix+".max-idle-connections", o.MaxOpenConnections, "连接池允许的最大空闲连接数")
	fs.IntVar(&o.MaxOpenConnections, fullPrefix+".max-open-connections", o.MaxOpenConnections, "连接池允许的最大打开连接数")
	fs.DurationVar(&o.MaxConnectionLifeTime, fullPrefix+".max-connection-life-time", o.MaxConnectionLifeTime, "单个连接的最长生命周期")
	fs.IntVar(&o.LogLevel, fullPrefix+".log-mode", o.LogLevel, "GORM 日志级别")
}

// DSN 根据 MySQLOptions 拼出 MySQL 的连接串。
func (o *MySQLOptions) DSN() string {
	return fmt.Sprintf(`%s:%s@tcp(%s)/%s?charset=utf8&parseTime=%t&loc=%s`,
		o.Username,
		o.Password,
		o.Addr,
		o.Database,
		true,
		"Local")
}

// NewDB 用当前配置创建一个 GORM MySQL 实例。
func (o *MySQLOptions) NewDB() (*gorm.DB, error) {
	opts := &db.MySQLOptions{
		Addr:                  o.Addr,
		Username:              o.Username,
		Password:              o.Password,
		Database:              o.Database,
		MaxIdleConnections:    o.MaxIdleConnections,
		MaxOpenConnections:    o.MaxOpenConnections,
		MaxConnectionLifeTime: o.MaxConnectionLifeTime,
		Logger:                gormlogger.Default,
	}

	return db.NewMySQL(opts)
}
