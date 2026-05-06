package db

import (
	"context"
	"sync"
	"time"

	"github.com/clin211/linhub/options"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	mongoMu     sync.RWMutex
	mongoClient *mongo.Client
)

// MongoClient 返回已初始化的 *mongo.Client；若未调用 InitMongo 则返回 nil。
func MongoClient() *mongo.Client {
	mongoMu.RLock()
	defer mongoMu.RUnlock()
	return mongoClient
}

// InitMongo 根据 viper 中 mongo.* 下的配置项建立 MongoDB 连接（详见生成的应用 YAML）。
func InitMongo() error {
	mo := options.NewMongoOptions()
	mo.URL = viper.GetString("mongo.url")
	if mo.URL == "" {
		mo.URL = "mongodb://127.0.0.1:27017"
	}
	mo.Database = viper.GetString("mongo.database")
	if mo.Database == "" {
		mo.Database = "{{.AppName}}"
	}
	mo.Username = viper.GetString("mongo.username")
	mo.Password = viper.GetString("mongo.password")
	mo.Collection = viper.GetString("mongo.collection")
	if mo.Collection == "" {
		mo.Collection = "app"
	}
	mo.Timeout = viper.GetDuration("mongo.timeout")
	if mo.Timeout == 0 {
		mo.Timeout = 10 * time.Second
	}

	c, err := mo.NewClient()
	if err != nil {
		return err
	}
	mongoMu.Lock()
	defer mongoMu.Unlock()
	mongoClient = c
	return nil
}

// CloseMongo 断开全局 MongoDB 客户端连接。
func CloseMongo() error {
	mongoMu.Lock()
	defer mongoMu.Unlock()
	if mongoClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := mongoClient.Disconnect(ctx)
	mongoClient = nil
	return err
}
