package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgreSQLOptions 定义 PostgreSQL 数据库连接的低层配置项。
type PostgreSQLOptions struct {
	Addr                  string
	Username              string
	Password              string
	Database              string
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
	// +optional
	Logger logger.Interface
}

// DSN 根据 PostgreSQLOptions 拼出连接串。
func (o *PostgreSQLOptions) DSN() string {
	splited := strings.Split(o.Addr, ":")
	host, port := splited[0], "5432"
	if len(splited) > 1 {
		port = splited[1]
	}

	return fmt.Sprintf(`user=%s password=%s host=%s port=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai`,
		o.Username,
		o.Password,
		host,
		port,
		o.Database,
	)
}

// NewPostgreSQL 用给定配置创建一个 *gorm.DB 实例。
func NewPostgreSQL(opts *PostgreSQLOptions) (*gorm.DB, error) {
	// 给所有可选字段填充默认值。
	setPostgreSQLDefaults(opts)

	db, err := gorm.Open(postgres.Open(opts.DSN()), &gorm.Config{
		// PrepareStmt 缓存每个 SQL 的 prepare 语句，提升性能。
		PrepareStmt: true,
		Logger:      opts.Logger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxOpenConns 设置数据库连接池的最大打开连接数。
	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)

	// SetConnMaxLifetime 设置连接可被重用的最长时间。
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	// SetMaxIdleConns 设置连接池中的最大空闲连接数。
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}

// setPostgreSQLDefaults 给可选字段填充默认值。
func setPostgreSQLDefaults(opts *PostgreSQLOptions) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:5432"
	}
	if opts.MaxIdleConnections == 0 {
		opts.MaxIdleConnections = 100
	}
	if opts.MaxOpenConnections == 0 {
		opts.MaxOpenConnections = 100
	}
	if opts.MaxConnectionLifeTime == 0 {
		opts.MaxConnectionLifeTime = time.Duration(10) * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
}
