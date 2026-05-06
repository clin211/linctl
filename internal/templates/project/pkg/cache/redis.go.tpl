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

// RDB 返回已初始化的 Redis 客户端；若未调用 InitRedis 则返回 nil。
func RDB() *redis.Client {
	rdbMu.RLock()
	defer rdbMu.RUnlock()
	return rdb
}

// InitRedis 根据 viper 中的配置项（redis.addr / redis.password / redis.db 以及可选的连接池参数）建立连接。
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

// CloseRedis 关闭全局 Redis 客户端。
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
