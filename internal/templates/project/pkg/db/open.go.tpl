// Package db opens the primary SQL database via github.com/clin211/linhub/db (GORM).
package db

import (
	"fmt"

	"github.com/clin211/linhub/db"
	"github.com/clin211/linhub/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// OpenGORM returns *gorm.DB for the configured storage backend (see scaffold --storage).
func OpenGORM(storage string) (*gorm.DB, error) {
	switch storage {
	case "gorm-postgres":
		return db.NewPostgreSQL(&db.PostgreSQLOptions{
			Addr:                  viper.GetString("postgresql.addr"),
			Username:              viper.GetString("postgresql.username"),
			Password:              viper.GetString("postgresql.password"),
			Database:              viper.GetString("postgresql.database"),
			MaxIdleConnections:    viper.GetInt("postgresql.max-idle-connections"),
			MaxOpenConnections:    viper.GetInt("postgresql.max-open-connections"),
			MaxConnectionLifeTime: viper.GetDuration("postgresql.max-connection-life-time"),
			Logger:                log.Default(),
		})
	case "gorm-mysql":
		return db.NewMySQL(&db.MySQLOptions{
			Addr:                  viper.GetString("mysql.addr"),
			Username:              viper.GetString("mysql.username"),
			Password:              viper.GetString("mysql.password"),
			Database:              viper.GetString("mysql.database"),
			MaxIdleConnections:    viper.GetInt("mysql.max-idle-connections"),
			MaxOpenConnections:    viper.GetInt("mysql.max-open-connections"),
			MaxConnectionLifeTime: viper.GetDuration("mysql.max-connection-life-time"),
			Logger:                log.Default(),
		})
	case "gorm-sqlite":
		return db.NewSQLite(&db.SQLiteOptions{
			Addr:                  viper.GetString("sqlite.path"),
			Database:              viper.GetString("sqlite.database"),
			MaxIdleConnections:    viper.GetInt("sqlite.max-idle-connections"),
			MaxOpenConnections:    viper.GetInt("sqlite.max-open-connections"),
			MaxConnectionLifeTime: viper.GetDuration("sqlite.max-connection-life-time"),
			Logger:                log.Default(),
		})
	default:
		return nil, fmt.Errorf("OpenGORM: unsupported storage %q", storage)
	}
}
