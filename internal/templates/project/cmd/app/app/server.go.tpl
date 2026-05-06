package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/clin211/linhub/log"

	"{{.Module}}/internal/{{.AppName}}/biz"
	"{{.Module}}/internal/{{.AppName}}/handler"
	"{{.Module}}/internal/{{.AppName}}/store"
	"{{.Module}}/internal/pkg/middleware"
{{- if ne .Storage "memory"}}
	dbpkg "{{.Module}}/pkg/db"
{{- end}}
{{- if eq .Cache "redis"}}
	"{{.Module}}/pkg/cache"
{{- end}}
{{- if eq .Cache "bigcache"}}
	"{{.Module}}/pkg/cache"
{{- end}}
)

const (
	defaultHomeDir    = ".{{.AppName}}"
	defaultConfigName = "{{.AppName}}"
)

var configFile string

// NewWebServerCommand 创建应用程序的 cobra 根命令。
func NewWebServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "{{.AppName}}",
		Short:        "{{.AppName | Title}} API server",
		Long:         `{{.AppName | Title}} is a Go backend service.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context())
		},
		Args: cobra.NoArgs,
	}

	cobra.OnInitialize(initConfig)
	cmd.PersistentFlags().StringVarP(&configFile, "config", "c", filePath(), "Path to the configuration file.")

	return cmd
}

func run(ctx context.Context) error {
	addr := viper.GetString("server.http.addr")
	if addr == "" {
		addr = ":8080"
	}

	// 初始化 store
{{- if eq .Storage "memory"}}
	s := store.NewStore()
{{- else if eq .Storage "mongo"}}
	if err := dbpkg.InitMongo(); err != nil {
		return fmt.Errorf("mongodb init: %w", err)
	}
	defer func() { _ = dbpkg.CloseMongo() }()
	// 业务 store 暂为内存实现；在资源层支持 MongoDB 之前，可使用 dbpkg.MongoClient() 直连 MongoDB。
	s := store.NewStore()
{{- else}}
	dbInstance, err := dbpkg.OpenGORM("{{.Storage}}")
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s := store.NewStore(dbInstance)
{{- end}}

{{- if eq .Cache "redis"}}
	if err := cache.InitRedis(); err != nil {
		return fmt.Errorf("redis init: %w", err)
	}
	defer func() { _ = cache.CloseRedis() }()
{{- else if eq .Cache "bigcache"}}
	if err := cache.InitBigCache(); err != nil {
		return fmt.Errorf("bigcache init: %w", err)
	}
	defer func() { _ = cache.CloseBigCache() }()
{{- end}}

	// 注入依赖
	b := biz.NewBiz(s)
	h := handler.NewHandler(b)

	// 配置 gin 路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	v1 := r.Group("/v1")
	h.InstallAll(v1)

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 后台启动服务器
	go func() {
		log.Infow("Starting server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorw(err, "Server error")
		}
	}()

	// 等待关停信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-quit:
	}

	log.Infow("Shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Infow("Server exited successfully.")
	return nil
}

func searchDirs() []string {
	homeDir, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return []string{filepath.Join(homeDir, defaultHomeDir), "."}
}

func filePath() string {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	return filepath.Join(home, defaultHomeDir, defaultConfigName+".yaml")
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		for _, dir := range searchDirs() {
			viper.AddConfigPath(dir)
		}
		viper.SetConfigName(defaultConfigName)
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.Warnw("Config file not found, using defaults", "error", err)
	}
}
