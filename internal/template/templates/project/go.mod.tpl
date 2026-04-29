module {{ .Project.Metadata.Module }}

go 1.25

require (
{{- if eq (index .Project.Spec.Components 0).Framework "gin" }}
	// HTTP framework + middleware
	github.com/gin-gonic/gin v1.11.0
	github.com/gin-contrib/pprof v1.5.3

	// CLI / config
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.19.0
	github.com/spf13/pflag v1.0.6
	github.com/fsnotify/fsnotify v1.9.0
	github.com/fatih/color v1.18.0

	// Dependency injection
	github.com/google/wire v0.6.0
	github.com/google/uuid v1.6.0

	// k8s.io/{apiserver,apimachinery,klog,utils,client-go,component-base}
	// (signal handling, errors aggregator, logging, homedir, base flag set)
	k8s.io/apiserver v0.34.1
	k8s.io/apimachinery v0.34.1
	k8s.io/client-go v0.34.1
	k8s.io/component-base v0.34.1
	k8s.io/klog/v2 v2.130.1
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397

	// JWT
	github.com/golang-jwt/jwt/v4 v4.5.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2

	// Casbin (authz)
	github.com/casbin/casbin/v2 v2.103.0
	github.com/casbin/gorm-adapter/v3 v3.32.0

	// gRPC status (used by pkg/errorsx for HTTP↔gRPC error mapping) +
	// grpc-gateway runtime (用户 make protoc 后生成的 *.pb.gw.go 会引用)
	google.golang.org/grpc v1.65.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2

	// ID generators
	github.com/sony/sonyflake v1.2.0

	// Pretty printing for `<binary> --version`
	github.com/gosuri/uitable v0.0.4

	// Prometheus
	github.com/prometheus/client_golang v1.23.0

	// OpenTelemetry
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/log v0.14.0
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/sdk/log v0.14.0
	go.opentelemetry.io/otel/sdk/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.34.0
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.14.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.38.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.38.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.63.0

	// Logging (zap-based pkg/log + multi-bridge pkg/logger)
	go.uber.org/zap v1.27.0
	go.uber.org/automaxprocs v1.6.0

	// i18n (pkg/i18n)
	github.com/BurntSushi/toml v1.4.0
	github.com/nicksnyder/go-i18n/v2 v2.4.1
	golang.org/x/text v0.30.0

	// Validation / util helpers
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2
	github.com/h2non/filetype v1.1.3
	github.com/jinzhu/copier v0.4.0

	// Redis (pkg/db/redis, pkg/options/redis_options, pkg/authn/jwt/store/redis)
	github.com/redis/go-redis/v9 v9.7.0
	github.com/redis/go-redis/extra/rediscensus/v9 v9.7.0

	// Kafka (pkg/options/kafka_options)
	github.com/segmentio/kafka-go v0.4.47

	// Health (pkg/options/health_options)
	github.com/gorilla/mux v1.8.1

	// Cryptography (pkg/authn bcrypt)
	golang.org/x/crypto v0.43.0

	// Lint analyzer 集合（pkg/util/lint/*）
	github.com/kisielk/errcheck v1.8.0
	golang.org/x/tools v0.38.0

	// YAML 解析（pkg/i18n / pkg/util/reflect）
	gopkg.in/yaml.v3 v3.0.1
{{- end }}
{{- if eq (index .Project.Spec.Components 0).Framework "grpc" }}
	google.golang.org/grpc v1.65.0
	{{- if (index .Project.Spec.Components 0).GrpcGateway }}
	google.golang.org/protobuf v1.34.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2
	{{- end }}
{{- end }}
{{- if eq (index .Project.Spec.Components 0).Storage "gorm-mysql" }}
	gorm.io/gorm v1.25.10
	gorm.io/driver/mysql v1.5.7

	// cmd/gen-gorm-model 工具依赖
	gorm.io/gen v0.3.27
	github.com/samber/lo v1.52.0
{{- end }}
{{- if eq (index .Project.Spec.Components 0).Storage "gorm-postgres" }}
	gorm.io/gorm v1.25.10
	gorm.io/driver/postgres v1.5.7

	// cmd/gen-gorm-model 工具依赖
	gorm.io/gen v0.3.27
	github.com/samber/lo v1.52.0
{{- end }}
{{- if eq (index .Project.Spec.Components 0).Storage "mongo" }}
	go.mongodb.org/mongo-driver v1.16.1
{{- end }}
{{- if eq (index .Project.Spec.Components 0).Kind "CLI" }}
	github.com/spf13/cobra v1.9.1
{{- end }}
)
