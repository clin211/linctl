// Package cache provides optional Redis connectivity (github.com/clin211/linhub/db).
package cache

import (
	"sync"

	"github.com/clin211/linhub/db"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

var (
	rdbMu sync.RWMutex
	rdb   *redis.Client
)

// RDB returns the initialized Redis client, or nil if InitRedis was not called.
func RDB() *redis.Client {
	rdbMu.RLock()
	defer rdbMu.RUnlock()
	return rdb
}

// InitRedis connects using viper keys: redis.addr, redis.password, redis.db, plus optional pool knobs.
func InitRedis() error {
	opts := &db.RedisOptions{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		Database:     viper.GetInt("redis.db"),
		MaxRetries:   viper.GetInt("redis.max-retries"),
		MinIdleConns: viper.GetInt("redis.min-idle-conns"),
		DialTimeout:  viper.GetDuration("redis.dial-timeout"),
		ReadTimeout:  viper.GetDuration("redis.read-timeout"),
		WriteTimeout: viper.GetDuration("redis.write-timeout"),
		PoolTimeout:  viper.GetDuration("redis.pool-timeout"),
		PoolSize:     viper.GetInt("redis.pool-size"),
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:6379"
	}
	c, err := db.NewRedis(opts)
	if err != nil {
		return err
	}
	rdbMu.Lock()
	defer rdbMu.Unlock()
	rdb = c
	return nil
}

// CloseRedis closes the global Redis client.
func CloseRedis() error {
	rdbMu.Lock()
	defer rdbMu.Unlock()
	if rdb == nil {
		return nil
	}
	err := rdb.Close()
	rdb = nil
	return err
}
