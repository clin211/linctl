// Package cache provides optional in-process BigCache (github.com/allegro/bigcache/v3).
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/spf13/viper"
)

var (
	bcMu sync.RWMutex
	bc   *bigcache.BigCache
)

// Local returns the BigCache instance, or nil if InitBigCache was not called.
func Local() *bigcache.BigCache {
	bcMu.RLock()
	defer bcMu.RUnlock()
	return bc
}

// InitBigCache builds BigCache from viper keys under bigcache.* (see the generated app YAML).
func InitBigCache() error {
	life := viper.GetDuration("bigcache.life-window")
	if life <= 0 {
		life = 10 * time.Minute
	}
	cfg := bigcache.DefaultConfig(life)

	if shards := viper.GetInt("bigcache.shards"); shards > 0 {
		cfg.Shards = shards
	}
	if cw := viper.GetDuration("bigcache.clean-window"); cw > 0 {
		cfg.CleanWindow = cw
	}
	if n := viper.GetInt("bigcache.max-entries-in-window"); n > 0 {
		cfg.MaxEntriesInWindow = n
	}
	if n := viper.GetInt("bigcache.max-entry-size"); n > 0 {
		cfg.MaxEntrySize = n
	}
	cfg.Verbose = viper.GetBool("bigcache.verbose")
	if mb := viper.GetInt("bigcache.hard-max-cache-size-mb"); mb > 0 {
		cfg.HardMaxCacheSize = mb
	}

	c, err := bigcache.New(context.Background(), cfg)
	if err != nil {
		return err
	}
	bcMu.Lock()
	defer bcMu.Unlock()
	bc = c
	return nil
}

// CloseBigCache resets the global BigCache.
func CloseBigCache() error {
	bcMu.Lock()
	defer bcMu.Unlock()
	if bc == nil {
		return nil
	}
	err := bc.Close()
	bc = nil
	return err
}
