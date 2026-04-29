package template_test

import (
	"strings"
	"testing"

	tpl "github.com/clin211/linctl/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigs_RenderAllNew exercises every templated configs/* file added in
// Stream B (multi-env yaml + SQL bootstrap) using realistic Project / Component
// shapes. Failures here usually mean a missing helper, a mistyped pipeline,
// or an unexpected snake/kebab transform.
func TestConfigs_RenderAllNew(t *testing.T) {
	e, err := tpl.New()
	require.NoError(t, err)

	type project struct {
		Metadata struct {
			Name   string
			Module string
		}
	}
	type component struct {
		Name    string
		Storage string
		Port    int
	}

	mkData := func(storage string) map[string]any {
		p := project{}
		p.Metadata.Name = "miniblog-v5"
		p.Metadata.Module = "github.com/clin211/miniblog-v5"
		c := component{Name: "mb-apiserver", Storage: storage, Port: 8080}
		return map[string]any{"Project": p, "Component": c}
	}

	cases := []struct {
		name    string
		tpl     string
		storage string
		want    []string // substrings that must appear
	}{
		{
			name:    "app.yaml gorm-postgres",
			tpl:     "templates/component/webserver/configs/app.yaml.tpl",
			storage: "gorm-postgres",
			want: []string{
				"addr: 0.0.0.0:8080",
				"postgresql:",
				"database: miniblog_v5",
				"redis:",
				"casbin:",
				"otel:",
				"service-name: mb-apiserver",
				"output: ./_output/mb-apiserver.log",
			},
		},
		{
			name:    "app.yaml gorm-mysql",
			tpl:     "templates/component/webserver/configs/app.yaml.tpl",
			storage: "gorm-mysql",
			want: []string{
				"mysql:",
				"database: miniblog_v5",
			},
		},
		{
			name:    "app.yaml mongo",
			tpl:     "templates/component/webserver/configs/app.yaml.tpl",
			storage: "mongo",
			want: []string{
				"mongo:",
				"url: mongodb://127.0.0.1:27017",
			},
		},
		{
			name:    "app.docker.yaml gorm-postgres uses service names",
			tpl:     "templates/component/webserver/configs/app.docker.yaml.tpl",
			storage: "gorm-postgres",
			want: []string{
				"addr: postgres:5432",
				"addr: redis:6379",
				"endpoint: otel-collector:4317",
				"output-mode: otel",
				"output: stdout",
			},
		},
		{
			name:    "app.docker.yaml mongo",
			tpl:     "templates/component/webserver/configs/app.docker.yaml.tpl",
			storage: "mongo",
			want: []string{
				"url: mongodb://mongo:27017",
				"addr: redis:6379",
			},
		},
		{
			name:    "init_database.sql",
			tpl:     "templates/component/webserver/configs/init_database.sql.tpl",
			storage: "gorm-postgres",
			want: []string{
				"CREATE DATABASE miniblog_v5",
				"DROP DATABASE IF EXISTS miniblog_v5",
				"COMMENT ON DATABASE miniblog_v5",
			},
		},
		{
			name:    "basic.sql",
			tpl:     "templates/component/webserver/configs/basic.sql.tpl",
			storage: "gorm-postgres",
			want: []string{
				"CREATE TABLE public.sys_user",
				"CREATE TABLE public.casbin_rule",
				"CREATE TABLE public.sys_login_log",
				"CREATE TABLE public.sys_user_config",
				"CREATE TABLE public.sys_monitor",
				"INSERT INTO sys_user",
				"INSERT INTO casbin_rule",
				"r:super_admin",
				"miniblog-v5",                    // unchanged metadata.name in header
				"admin@miniblog_v5.local",        // snake transform inside email
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := e.RenderToString(tc.tpl, mkData(tc.storage))
			require.NoError(t, err, "render %s", tc.tpl)
			for _, sub := range tc.want {
				assert.True(t, strings.Contains(out, sub),
					"%s: expected output to contain %q\n--- output ---\n%s",
					tc.tpl, sub, out)
			}
		})
	}
}
