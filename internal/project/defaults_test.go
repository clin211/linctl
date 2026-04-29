package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/linctl/internal/project"
)

func TestApplyDefaults_NilSafe(t *testing.T) {
	assert.NotPanics(t, func() {
		project.ApplyDefaults(nil)
	})
}

func TestApplyDefaults_FillsTopLevelMeta(t *testing.T) {
	p := &project.Project{}
	project.ApplyDefaults(p)
	assert.Equal(t, project.APIVersionV1, p.APIVersion)
	assert.Equal(t, project.KindProject, p.Kind)
}

func TestApplyDefaults_KeepsExistingValues(t *testing.T) {
	p := &project.Project{
		APIVersion: project.APIVersionV1Alpha1,
		Kind:       "Project",
		Spec: project.Spec{
			Defaults: project.Defaults{
				Framework:    "grpc",
				Storage:      "gorm-mysql",
				Deploy:       "kubernetes",
				Makefile:     "structured",
				ProtoVersion: "v1beta1",
			},
		},
	}
	project.ApplyDefaults(p)

	assert.Equal(t, project.APIVersionV1Alpha1, p.APIVersion, "should keep existing apiVersion")
	assert.Equal(t, "grpc", p.Spec.Defaults.Framework)
	assert.Equal(t, "gorm-mysql", p.Spec.Defaults.Storage)
	assert.Equal(t, "kubernetes", p.Spec.Defaults.Deploy)
	assert.Equal(t, "structured", p.Spec.Defaults.Makefile)
	assert.Equal(t, "v1beta1", p.Spec.Defaults.ProtoVersion)
}

func TestApplyDefaults_FillsDefaultsBlock(t *testing.T) {
	p := &project.Project{}
	project.ApplyDefaults(p)

	d := p.Spec.Defaults
	assert.Equal(t, "gin", d.Framework)
	assert.Equal(t, "memory", d.Storage)
	assert.Equal(t, "docker", d.Deploy)
	assert.Equal(t, "unstructured", d.Makefile)
	assert.Equal(t, "v1", d.ProtoVersion)
	require.NotNil(t, d.Image)
	assert.Equal(t, "multi-stage", d.Image.DockerfileMode)
	assert.Equal(t, "always", d.Image.DistrolessMode)
	require.NotNil(t, d.Docs)
	assert.Equal(t, []string{"zh-CN"}, d.Docs.Languages)
	require.NotNil(t, d.Telemetry)
	assert.Equal(t, "slog", d.Telemetry.Logging)
	assert.Equal(t, "prometheus", d.Telemetry.Metrics)
	assert.Equal(t, "otlp", d.Telemetry.Tracing)
}

func TestApplyDefaults_KeepsImagePartials(t *testing.T) {
	p := &project.Project{
		Spec: project.Spec{
			Defaults: project.Defaults{
				Image: &project.ImageDefaults{
					RegistryPrefix: "ghcr.io/foo",
				},
			},
		},
	}
	project.ApplyDefaults(p)

	require.NotNil(t, p.Spec.Defaults.Image)
	assert.Equal(t, "ghcr.io/foo", p.Spec.Defaults.Image.RegistryPrefix)
	assert.Equal(t, "multi-stage", p.Spec.Defaults.Image.DockerfileMode)
	assert.Equal(t, "always", p.Spec.Defaults.Image.DistrolessMode)
}

func TestApplyDefaults_KeepsDocsLanguages(t *testing.T) {
	p := &project.Project{
		Spec: project.Spec{
			Defaults: project.Defaults{
				Docs: &project.DocsDefaults{Languages: []string{"en-US"}},
			},
		},
	}
	project.ApplyDefaults(p)
	assert.Equal(t, []string{"en-US"}, p.Spec.Defaults.Docs.Languages)
}

func TestApplyDefaults_TelemetryPartial(t *testing.T) {
	p := &project.Project{
		Spec: project.Spec{
			Defaults: project.Defaults{
				Telemetry: &project.TelemetryDefaults{Logging: "zap"},
			},
		},
	}
	project.ApplyDefaults(p)
	assert.Equal(t, "zap", p.Spec.Defaults.Telemetry.Logging)
	assert.Equal(t, "prometheus", p.Spec.Defaults.Telemetry.Metrics)
	assert.Equal(t, "otlp", p.Spec.Defaults.Telemetry.Tracing)
}

func TestApplyDefaults_InheritsToComponents(t *testing.T) {
	p := &project.Project{
		Spec: project.Spec{
			Defaults: project.Defaults{
				Framework: "grpc",
				Storage:   "mongo",
			},
			Components: []project.Component{
				{Kind: "WebServer", Name: "a"},                    // 应继承
				{Kind: "WebServer", Name: "b", Framework: "gin"},  // framework 覆盖
				{Kind: "WebServer", Name: "c", Storage: "memory"}, // storage 覆盖
			},
		},
	}
	project.ApplyDefaults(p)

	assert.Equal(t, "grpc", p.Spec.Components[0].Framework)
	assert.Equal(t, "mongo", p.Spec.Components[0].Storage)
	assert.Equal(t, "gin", p.Spec.Components[1].Framework)
	assert.Equal(t, "mongo", p.Spec.Components[1].Storage)
	assert.Equal(t, "grpc", p.Spec.Components[2].Framework)
	assert.Equal(t, "memory", p.Spec.Components[2].Storage)
}

func TestApplyDefaults_Idempotent(t *testing.T) {
	p := &project.Project{
		Spec: project.Spec{
			Components: []project.Component{
				{Kind: "WebServer", Name: "a"},
			},
		},
	}
	project.ApplyDefaults(p)
	first := *p

	project.ApplyDefaults(p)
	second := *p

	assert.Equal(t, first.APIVersion, second.APIVersion)
	assert.Equal(t, first.Spec.Defaults, second.Spec.Defaults)
	assert.Equal(t, first.Spec.Components[0].Framework, second.Spec.Components[0].Framework)
	assert.Equal(t, first.Spec.Components[0].Storage, second.Spec.Components[0].Storage)
}
