// Package db provides MongoDB connectivity via github.com/clin211/linhub/options.
//
// The default store layer for mongo-backed projects is still the in-memory scaffold
// (same as --storage memory) until linctl resource templates support MongoDB natively.
// Use MongoClient() for custom persistence against the configured cluster.
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

// MongoClient returns the initialized *mongo.Client, or nil if InitMongo was not called.
func MongoClient() *mongo.Client {
	mongoMu.RLock()
	defer mongoMu.RUnlock()
	return mongoClient
}

// InitMongo connects using viper keys under mongo.* (see the generated app YAML).
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

// CloseMongo disconnects the global MongoDB client.
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
