package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"{{.Module}}/internal/{{.AppName}}/biz"
	"{{.Module}}/internal/{{.AppName}}/handler"
	"{{.Module}}/internal/{{.AppName}}/store"
{{- if ne .Storage "memory"}}
	"{{.Module}}/pkg/db"
{{- end}}
)

const (
	defaultHomeDir    = ".{{.AppName}}"
	defaultConfigName = "{{.AppName}}"
)

var configFile string

// NewWebServerCommand creates the root cobra command for the application.
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

	// Initialize store
{{- if eq .Storage "memory"}}
	s := store.NewStore()
{{- else}}
	dbOpts := &db.PostgreSQLOptions{
		Addr:     viper.GetString("postgresql.addr"),
		Username: viper.GetString("postgresql.username"),
		Password: viper.GetString("postgresql.password"),
		Database: viper.GetString("postgresql.database"),
	}
	dbInstance, err := db.NewPostgreSQL(dbOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s := store.NewStore(dbInstance)
{{- end}}

	// Wire up dependencies
	b := biz.NewBiz(s)
	h := handler.NewHandler(b)

	// Set up gin router
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/v1")
	h.InstallAll(v1)

	// Create HTTP server
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in background
	go func() {
		slog.Info("Starting server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-quit:
	}

	slog.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	slog.Info("Server exited successfully.")
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
		slog.Warn("Config file not found, using defaults", "error", err)
	}
}
