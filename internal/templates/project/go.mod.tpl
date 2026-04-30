module {{.Module}}

go {{.GoVersion}}

require (
	github.com/gin-gonic/gin v1.10.1
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
{{- if ne .Storage "memory"}}
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
	go.mongodb.org/mongo-driver/v2 v2.2.0
{{- end}}
)
