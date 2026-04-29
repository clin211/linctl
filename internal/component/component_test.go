package component_test

import (
	"testing"

	"github.com/clin211/lin/internal/component"
	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_RegisterGet(t *testing.T) {
	r := component.NewRegistry()
	require.NoError(t, r.Register("WebServer", component.WebServerFactory))

	got, err := r.Get("WebServer")
	require.NoError(t, err)
	require.NotNil(t, got)

	_, err = r.Get("Unknown")
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrComponentNotFound, code)
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := component.NewRegistry()
	require.NoError(t, r.Register("X", component.WebServerFactory))
	err := r.Register("X", component.WebServerFactory)
	require.Error(t, err)
	code, _ := linctlerr.CodeOf(err)
	assert.Equal(t, linctlerr.ErrComponentExists, code)
}

func TestRegistry_Kinds(t *testing.T) {
	r := component.NewRegistry()
	require.NoError(t, r.Register("B", component.WebServerFactory))
	require.NoError(t, r.Register("A", component.WebServerFactory))
	assert.Equal(t, []string{"A", "B"}, r.Kinds())
}

func TestWebServer_Validate(t *testing.T) {
	cases := []struct {
		name    string
		spec    project.Component
		wantErr linctlerr.Code
	}{
		{
			name:    "missing name",
			spec:    project.Component{Framework: "gin", Storage: "gorm-postgres"},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "missing framework",
			spec:    project.Component{Name: "api", Storage: "gorm-postgres"},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "unsupported framework",
			spec:    project.Component{Name: "api", Framework: "kratos", Storage: "gorm-postgres"},
			wantErr: linctlerr.ErrNotImplementedYet,
		},
		{
			name:    "missing storage",
			spec:    project.Component{Name: "api", Framework: "gin"},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "unsupported storage",
			spec:    project.Component{Name: "api", Framework: "gin", Storage: "memory"},
			wantErr: linctlerr.ErrNotImplementedYet,
		},
		{
			name: "valid gorm-mysql",
			spec: project.Component{Name: "api", Framework: "gin", Storage: "gorm-mysql"},
		},
		{
			name: "valid gorm-postgres",
			spec: project.Component{Name: "api", Framework: "gin", Storage: "gorm-postgres"},
		},
		{
			name: "valid mongo",
			spec: project.Component{Name: "api", Framework: "gin", Storage: "mongo"},
		},
		{
			name: "valid grpc + gorm-postgres",
			spec: project.Component{Name: "api", Framework: "grpc", Storage: "gorm-postgres"},
		},
		{
			name: "valid grpc + mongo + grpcGateway",
			spec: project.Component{Name: "api", Framework: "grpc", Storage: "mongo", GrpcGateway: true, Port: 8080, GRPCPort: 9090},
		},
		{
			name:    "grpcGateway without grpc framework",
			spec:    project.Component{Name: "api", Framework: "gin", Storage: "gorm-postgres", GrpcGateway: true},
			wantErr: linctlerr.ErrConfigInvalid,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := component.NewWebServer(tc.spec)
			err := ws.Validate(&project.Project{})
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			code, _ := linctlerr.CodeOf(err)
			assert.Equal(t, tc.wantErr, code)
		})
	}
}

func TestWebServer_BasePairs(t *testing.T) {
	cases := []struct {
		name        string
		framework   string
		mustHave    []string
		mustNotHave []string
	}{
		{
			name:      "gin set",
			framework: "gin",
			mustHave: []string{
				"cmd/api/main.go",
				"cmd/api/app/server.go",
				"cmd/api/app/options/options.go",
				"internal/api/server.go",
				"internal/api/httpserver.go",
				"internal/api/wire.go",
				"internal/api/biz/biz.go",
				"internal/api/store/store.go",
				"internal/api/handler/handler.go",
				"internal/api/handler/healthz.go",
				"internal/api/pkg/validation/validation.go",
				"internal/api/pkg/metrics/metrics.go",
				"configs/api.yaml",
				"configs/casbin/model.conf",
				"configs/casbin/policy.csv",
				"go.mod",
				// web-gin 风格的项目级骨架（v0.2.3+ framework=gin 默认带）
				"internal/pkg/contextx/contextx.go",
				"internal/pkg/known/known.go",
				"internal/pkg/errno/code.go",
				"pkg/errorsx/errorsx.go",
				"pkg/id/sonyflake.go",
				"internal/pkg/rid/rid.go",
				// v0.3.0 起追加的 web-gin 运行时基础设施
				"pkg/core/core.go",
				"pkg/db/postgresql.go",
				"pkg/server/http_server.go",
				"pkg/options/http_options.go",
				"pkg/store/store.go",
				"pkg/store/where/where.go",
				"pkg/authz/authz.go",
				"pkg/token/token.go",
				"pkg/middleware/gin/observability.go",
				"pkg/binding/binding.go",
				"pkg/version/version.go",
				"internal/pkg/middleware/gin/header.go",
				"internal/pkg/middleware/gin/authn.go",
			},
			mustNotHave: []string{
				"internal/api/router.go", // 已废弃，路由由 httpserver.go + handler init 接管
				"api/api/v1/api.proto",
			},
		},
		{
			name:      "grpc set",
			framework: "grpc",
			mustHave: []string{
				"cmd/api/main.go",
				"internal/api/server.go",
				"internal/api/interceptor/interceptor.go",
				"internal/api/handler/handler.go",
				"api/api/v1/api.proto",
				"go.mod",
			},
			mustNotHave: []string{
				"internal/api/router.go",
				// grpc 不附带 web-gin 风格骨架与 biz/store（仅 framework=gin 默认开启）
				"internal/api/biz/biz.go",
				"internal/api/store/store.go",
				"internal/pkg/contextx/contextx.go",
				"pkg/errorsx/errorsx.go",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := component.NewWebServer(project.Component{
				Name:      "api",
				Framework: tc.framework,
				Storage:   "gorm-postgres",
			})
			pairs := ws.BasePairs(&project.Project{})
			require.NotEmpty(t, pairs)

			dsts := make(map[string]bool, len(pairs))
			for _, p := range pairs {
				dsts[p.Dst] = true
				assert.Contains(t, p.Owner, "WebServer:api")
				assert.NotEmpty(t, p.TemplateID, "TemplateID must be set for %s", p.Dst)
			}
			for _, w := range tc.mustHave {
				assert.True(t, dsts[w], "BasePairs(%s) must include %s", tc.framework, w)
			}
			for _, w := range tc.mustNotHave {
				assert.False(t, dsts[w], "BasePairs(%s) must NOT include %s", tc.framework, w)
			}
		})
	}
}

func TestWebServer_PostProcess_Noop(t *testing.T) {
	ws := component.NewWebServer(project.Component{Name: "x", Framework: "gin", Storage: "memory"})
	require.NoError(t, ws.PostProcess(&project.Project{}, nil))
}

// ===== Worker (Phase 3 Story 3.2) =====

func TestWorker_Validate(t *testing.T) {
	cases := []struct {
		name    string
		spec    project.Component
		wantErr linctlerr.Code
	}{
		{
			name:    "missing name",
			spec:    project.Component{Variants: []string{"cron"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "missing variants",
			spec:    project.Component{Name: "w"},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "unknown variant",
			spec:    project.Component{Name: "w", Variants: []string{"hyper"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "duplicate variant",
			spec:    project.Component{Name: "w", Variants: []string{"cron", "cron"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "cron without spec",
			spec:    project.Component{Name: "w", Variants: []string{"cron"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "cron with empty jobs",
			spec:    project.Component{Name: "w", Variants: []string{"cron"}, Cron: &project.CronSpec{}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "cron valid",
			spec: project.Component{
				Name:     "w",
				Variants: []string{"cron"},
				Cron:     &project.CronSpec{Jobs: []project.NamedSpec{{Name: "daily"}}},
			},
		},
		{
			name:    "kafka without spec",
			spec:    project.Component{Name: "w", Variants: []string{"kafka"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "kafka without brokers",
			spec: project.Component{
				Name:     "w",
				Variants: []string{"kafka"},
				Kafka:    &project.KafkaSpec{Topics: []project.NamedSpec{{Name: "events"}}},
			},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "kafka valid",
			spec: project.Component{
				Name:     "w",
				Variants: []string{"kafka"},
				Kafka: &project.KafkaSpec{
					Brokers: []string{"localhost:9092"},
					Topics:  []project.NamedSpec{{Name: "events"}},
				},
			},
		},
		{
			name:    "customized empty",
			spec:    project.Component{Name: "w", Variants: []string{"customized"}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "customized valid",
			spec: project.Component{
				Name:       "w",
				Variants:   []string{"customized"},
				Customized: []project.NamedSpec{{Name: "longpoll"}},
			},
		},
		{
			name: "multi variants valid",
			spec: project.Component{
				Name:     "w",
				Variants: []string{"cron", "kafka", "customized"},
				Cron:     &project.CronSpec{Jobs: []project.NamedSpec{{Name: "daily"}}},
				Kafka: &project.KafkaSpec{
					Brokers: []string{"localhost:9092"},
					Topics:  []project.NamedSpec{{Name: "events"}},
				},
				Customized: []project.NamedSpec{{Name: "longpoll"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := component.NewWorker(tc.spec)
			err := w.Validate(&project.Project{})
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			code, _ := linctlerr.CodeOf(err)
			assert.Equal(t, tc.wantErr, code)
		})
	}
}

func TestWorker_BasePairs_VariantOrdering(t *testing.T) {
	w := component.NewWorker(project.Component{
		Name:       "reporter",
		Variants:   []string{"kafka", "cron", "customized"}, // 故意乱序
		Cron:       &project.CronSpec{Jobs: []project.NamedSpec{{Name: "daily"}}},
		Kafka:      &project.KafkaSpec{Brokers: []string{"x:9092"}, Topics: []project.NamedSpec{{Name: "t"}}},
		Customized: []project.NamedSpec{{Name: "tail"}},
	})
	pairs := w.BasePairs(&project.Project{})

	dsts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		dsts = append(dsts, p.Dst)
		assert.Contains(t, p.Owner, "Worker:reporter")
		assert.NotEmpty(t, p.TemplateID)
	}

	for _, want := range []string{
		"cmd/reporter/main.go",
		"internal/reporter/runner.go",
		"internal/reporter/cron.go",
		"internal/reporter/kafka.go",
		"internal/reporter/customized.go",
		"go.mod",
		"Makefile",
	} {
		assert.Contains(t, dsts, want, "BasePairs missing %s", want)
	}
}

func TestWorker_BasePairs_OnlyCron(t *testing.T) {
	w := component.NewWorker(project.Component{
		Name:     "tick",
		Variants: []string{"cron"},
		Cron:     &project.CronSpec{Jobs: []project.NamedSpec{{Name: "minute"}}},
	})
	pairs := w.BasePairs(&project.Project{})
	dsts := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		dsts[p.Dst] = true
	}
	assert.True(t, dsts["internal/tick/cron.go"])
	assert.False(t, dsts["internal/tick/kafka.go"], "kafka.go must NOT appear when variant absent")
	assert.False(t, dsts["internal/tick/customized.go"], "customized.go must NOT appear when variant absent")
}

func TestWorker_PostProcess_Noop(t *testing.T) {
	w := component.NewWorker(project.Component{Name: "n", Variants: []string{"cron"}, Cron: &project.CronSpec{Jobs: []project.NamedSpec{{Name: "j"}}}})
	require.NoError(t, w.PostProcess(&project.Project{}, nil))
}

// ===== CLI (Phase 3 Story 3.4) =====

func TestCLI_Validate(t *testing.T) {
	cases := []struct {
		name    string
		spec    project.Component
		wantErr linctlerr.Code
	}{
		{
			name:    "missing name",
			spec:    project.Component{Commands: []project.NamedSpec{{Name: "x"}}},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name:    "missing commands",
			spec:    project.Component{Name: "admin"},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "command without name",
			spec: project.Component{
				Name:     "admin",
				Commands: []project.NamedSpec{{}},
			},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "duplicate commands",
			spec: project.Component{
				Name:     "admin",
				Commands: []project.NamedSpec{{Name: "dump"}, {Name: "dump"}},
			},
			wantErr: linctlerr.ErrConfigInvalid,
		},
		{
			name: "valid single command",
			spec: project.Component{
				Name:     "admin",
				Commands: []project.NamedSpec{{Name: "version"}},
			},
		},
		{
			name: "valid multiple commands",
			spec: project.Component{
				Name:     "admin",
				Commands: []project.NamedSpec{{Name: "dump"}, {Name: "migrate"}, {Name: "seed"}},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := component.NewCLI(tc.spec)
			err := c.Validate(&project.Project{})
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			code, _ := linctlerr.CodeOf(err)
			assert.Equal(t, tc.wantErr, code)
		})
	}
}

func TestCLI_BasePairs(t *testing.T) {
	c := component.NewCLI(project.Component{
		Name:     "admin",
		Commands: []project.NamedSpec{{Name: "migrate"}, {Name: "dump"}, {Name: "seed"}},
	})
	pairs := c.BasePairs(&project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   "myadmin",
			Module: "github.com/example/myadmin",
		},
	})

	dsts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		dsts = append(dsts, p.Dst)
		assert.Contains(t, p.Owner, "CLI:admin")
		assert.NotEmpty(t, p.TemplateID)
	}

	for _, want := range []string{
		"cmd/admin/main.go",
		"internal/admin/cmd/all.go",
		"internal/admin/cmd/dump.go",
		"internal/admin/cmd/migrate.go",
		"internal/admin/cmd/seed.go",
		"go.mod",
		"Makefile",
	} {
		assert.Contains(t, dsts, want, "BasePairs missing %s", want)
	}

	// command pair 必须带 Data，否则模板访问 .CommandName 会失败
	for _, p := range pairs {
		if p.Dst == "internal/admin/cmd/dump.go" {
			data, ok := p.Data.(map[string]any)
			require.True(t, ok, "command pair Data must be map[string]any")
			assert.Equal(t, "dump", data["CommandName"])
			assert.NotNil(t, data["Project"])
			assert.NotNil(t, data["Component"])
		}
	}
}

func TestCLI_PostProcess_Noop(t *testing.T) {
	c := component.NewCLI(project.Component{Name: "n", Commands: []project.NamedSpec{{Name: "v"}}})
	require.NoError(t, c.PostProcess(&project.Project{}, nil))
}
