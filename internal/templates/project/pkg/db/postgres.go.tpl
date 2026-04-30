// Package db provides database connection utilities.
package db

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgreSQLOptions defines connection options for PostgreSQL.
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

// DSN returns the PostgreSQL DSN string from the options.
func (o *PostgreSQLOptions) DSN() string {
	parts := strings.SplitN(o.Addr, ":", 2)
	host, port := parts[0], "5432"
	if len(parts) > 1 {
		port = parts[1]
	}
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		o.Username, o.Password, host, port, o.Database,
	)
}

// NewPostgreSQL creates a new gorm.DB instance connected to PostgreSQL.
func NewPostgreSQL(opts *PostgreSQLOptions) (*gorm.DB, error) {
	setPostgreSQLDefaults(opts)

	db, err := gorm.Open(postgres.Open(opts.DSN()), &gorm.Config{
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

	sqlDB.SetMaxOpenConns(opts.MaxOpenConnections)
	sqlDB.SetConnMaxLifetime(opts.MaxConnectionLifeTime)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConnections)

	return db, nil
}

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
		opts.MaxConnectionLifeTime = 10 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = logger.Default
	}
}
