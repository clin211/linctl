package {{ .Component.Name }}

import (
	"context"
	"log/slog"
	"time"

{{- if ne .Component.Storage "mongo" }}
	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
{{- end }}
	"{{ .Project.Metadata.Module }}/internal/pkg/known"
	mw "{{ .Project.Metadata.Module }}/internal/pkg/middleware/gin"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/biz"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/validation"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
	"{{ .Project.Metadata.Module }}/pkg/authz"
	genericoptions "{{ .Project.Metadata.Module }}/pkg/options"
	"{{ .Project.Metadata.Module }}/pkg/server"
{{- if ne .Component.Storage "mongo" }}
	"{{ .Project.Metadata.Module }}/pkg/store/registry"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
{{- end }}
	"{{ .Project.Metadata.Module }}/pkg/token"
{{- if eq .Component.Storage "mongo" }}
	"go.mongodb.org/mongo-driver/mongo"
{{- else }}
	"gorm.io/gorm"
{{- end }}
)

// Config 是应用级配置，由命令行 / 配置文件解析后填充。
type Config struct {
	JWTKey      string
	Expiration  time.Duration
	TLSOptions  *genericoptions.TLSOptions
	HTTPOptions *genericoptions.HTTPOptions
{{- if eq .Component.Storage "gorm-mysql" }}
	MySQLOptions *genericoptions.MySQLOptions
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
	PostgreSQLOptions *genericoptions.PostgreSQLOptions
{{- end }}
{{- if eq .Component.Storage "mongo" }}
	MongoOptions *genericoptions.MongoOptions
{{- end }}
}

// Server 表示当前 Web 服务器进程。
type Server struct {
	cfg *ServerConfig
	srv server.Server
}

// ServerConfig 聚合服务器运行所需的核心依赖（biz / validator / authz / userRetriever 等）。
//
// 字段全部为非导出，由 Wire 通过 wire.Struct(new(ServerConfig), "*") 一次性注入。
type ServerConfig struct {
	*Config
	biz       biz.IBiz
	val       *validation.Validator
	retriever mw.UserRetriever
	authz     *authz.Authz
}

// NewServer 完成进程级一次性初始化（租户、token），然后调用 wire 生成的 NewServer 装配整棵依赖树。
func (cfg *Config) NewServer(ctx context.Context) (*Server, error) {
{{- if ne .Component.Storage "mongo" }}
	// 注册租户：让 pkg/store/where 在查询时自动按 userID 隔离数据。
	where.RegisterTenant("user_id", func(ctx context.Context) string {
		return contextx.UserID(ctx)
	})
{{- end }}

	// 初始化 token 包的签名密钥、身份字段和默认过期时间。
	token.Init(
		cfg.JWTKey,
		token.WithIdentityKey(known.XUserID),
		token.WithExpiration(cfg.Expiration),
	)

	return NewServer(cfg)
}

// Run 启动服务器并监听终止信号。收到信号后会触发优雅停机（10 秒超时）。
func (s *Server) Run(ctx context.Context) error {
	go s.srv.RunOrDie()

	<-ctx.Done()
	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s.srv.GracefulStop(ctx)

	slog.Info("Server exited successfully.")
	return nil
}

{{- if eq .Component.Storage "mongo" }}

// NewMongoClient 根据配置创建一个 *mongo.Client 实例。
func (cfg *Config) NewMongoClient() (*mongo.Client, error) {
	slog.Info("Initializing MongoDB connection")
	client, err := cfg.MongoOptions.NewClient()
	if err != nil {
		slog.Error("Failed to create MongoDB connection", "error", err)
		return nil, err
	}
	return client, nil
}

// ProvideMongo 是 Wire 注入入口：从 *Config 取出 MongoDB 客户端。
func ProvideMongo(cfg *Config) (*mongo.Client, error) {
	return cfg.NewMongoClient()
}
{{- else }}

// NewDB 根据配置创建一个 *gorm.DB 实例并完成数据库迁移。
func (cfg *Config) NewDB() (*gorm.DB, error) {
{{- if eq .Component.Storage "gorm-mysql" }}
	slog.Info("Initializing database connection", "type", "mysql")
	db, err := cfg.MySQLOptions.NewDB()
{{- end }}
{{- if eq .Component.Storage "gorm-postgres" }}
	slog.Info("Initializing database connection", "type", "postgresql")
	db, err := cfg.PostgreSQLOptions.NewDB()
{{- end }}
	if err != nil {
		slog.Error("Failed to create database connection", "error", err)
		return nil, err
	}

	// 触发所有通过 model.init() 注册到 registry 的 GORM 模型自动迁移。
	if err := registry.Migrate(db); err != nil {
		slog.Error("Failed to migrate database schema", "error", err)
		return nil, err
	}
	return db, nil
}

// ProvideDB 是 Wire 注入入口：从 *Config 取出 GORM 数据库实例。
func ProvideDB(cfg *Config) (*gorm.DB, error) {
	return cfg.NewDB()
}
{{- end }}

// UserRetriever 实现 internal/pkg/middleware/gin.UserRetriever 接口；
// AuthnMiddleware 拿到 JWT 中的 userID 后通过本结构加载用户对象，
// 用于把 username 等信息投影回 contextx 便于审计。
//
// 注意 mw.UserRetriever 的方法返回 (any, error)，*model.UserM 会自动装箱为 any。
type UserRetriever struct {
	store store.IStore
}

// GetUser 根据用户 ID 加载用户.
{{- if eq .Component.Storage "mongo" }}
//
// TODO(linctl): mongo 分支未实现 store.User().Get；当前返回 (nil, nil) 作为占位，
// 真实业务请按 store/user.go 中的 Get 完成查询后在这里调用并返回。
func (r *UserRetriever) GetUser(ctx context.Context, userID string) (any, error) {
	_ = r.store
	_ = ctx
	_ = userID
	return nil, nil
}
{{- else }}
func (r *UserRetriever) GetUser(ctx context.Context, userID string) (any, error) {
	return r.store.User().Get(ctx, where.F("user_id", userID))
}
{{- end }}

// NewWebServer 是 Wire 链路的最末端：当所有依赖都准备好之后，构造 Gin server。
func NewWebServer(serverConfig *ServerConfig) (server.Server, error) {
	return serverConfig.NewGinServer()
}
