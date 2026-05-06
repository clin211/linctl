module {{.Module}}

go {{.GoVersion}}

require (
	github.com/clin211/linhub v0.0.1
	github.com/gin-gonic/gin v1.10.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
{{- if and (ne .Storage "memory") (ne .Storage "mongo")}}
	gorm.io/gorm v1.25.12
{{- end}}
{{- if eq .Storage "gorm-postgres"}}
	gorm.io/driver/postgres v1.5.11
{{- end}}
{{- if eq .Storage "gorm-mysql"}}
	gorm.io/driver/mysql v1.5.7
{{- end}}
{{- if eq .Storage "gorm-sqlite"}}
	gorm.io/driver/sqlite v1.5.7
{{- end}}
{{- if eq .Storage "mongo"}}
	go.mongodb.org/mongo-driver v1.17.2
{{- end}}
{{- if eq .Cache "redis"}}
	github.com/redis/go-redis/v9 v9.7.0
{{- end}}
{{- if eq .Cache "bigcache"}}
	github.com/allegro/bigcache/v3 v3.1.0
{{- end}}
)
