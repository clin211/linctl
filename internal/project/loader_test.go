package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/project"
)

const fixturesDir = "testdata/loader"

func loadFixture(t *testing.T, name string) (*project.Project, error) {
	t.Helper()
	loader := project.NewLoader()
	return loader.LoadFromFile(filepath.Join(fixturesDir, name))
}

func mustLoadFixture(t *testing.T, name string) *project.Project {
	t.Helper()
	p, err := loadFixture(t, name)
	require.NoError(t, err)
	require.NotNil(t, p)
	return p
}

// ===== happy paths =====

func TestLoadFromFile_Minimal(t *testing.T) {
	p := mustLoadFixture(t, "minimal.yaml")

	assert.Equal(t, project.APIVersionV1, p.APIVersion)
	assert.Equal(t, project.KindProject, p.Kind)
	assert.Equal(t, "hello", p.Metadata.Name)
	assert.Equal(t, "github.com/foo/hello", p.Metadata.Module)
	require.Len(t, p.Spec.Components, 1)
	assert.Equal(t, "WebServer", p.Spec.Components[0].Kind)
	assert.Equal(t, "hello", p.Spec.Components[0].Name)
	assert.Equal(t, "gin", p.Spec.Components[0].Framework, "default framework should be inherited")
	assert.Equal(t, "memory", p.Spec.Components[0].Storage, "default storage should be inherited")

	assert.Equal(t, "v1", p.Spec.Defaults.ProtoVersion)
	require.NotNil(t, p.Spec.Defaults.Image)
	assert.Equal(t, "multi-stage", p.Spec.Defaults.Image.DockerfileMode)
	require.NotNil(t, p.Spec.Defaults.Docs)
	assert.Equal(t, []string{"zh-CN"}, p.Spec.Defaults.Docs.Languages)
	require.NotNil(t, p.Spec.Defaults.Telemetry)
	assert.Equal(t, "slog", p.Spec.Defaults.Telemetry.Logging)
}

func TestLoadFromFile_Full(t *testing.T) {
	p := mustLoadFixture(t, "full.yaml")

	assert.Equal(t, "miniblog", p.Metadata.Name)
	assert.Equal(t, "github.com/clin211/miniblog", p.Metadata.Module)
	assert.Equal(t, "767425412lin@gmail.com", p.Metadata.Author.Email)
	assert.Equal(t, "platform", p.Metadata.Labels["team"])

	assert.Equal(t, "gorm-postgres", p.Spec.Defaults.Storage)
	assert.Equal(t, "kubernetes", p.Spec.Defaults.Deploy)
	assert.Equal(t, "structured", p.Spec.Defaults.Makefile)
	assert.Equal(t, "v1", p.Spec.Defaults.ProtoVersion)
	require.NotNil(t, p.Spec.Defaults.Image)
	assert.Equal(t, "ghcr.io/clin211", p.Spec.Defaults.Image.RegistryPrefix)
	assert.Equal(t, []string{"zh-CN", "en-US"}, p.Spec.Defaults.Docs.Languages)

	require.Len(t, p.Spec.Components, 4)
	apiserver := p.Spec.Components[0]
	assert.Equal(t, "mb-apiserver", apiserver.Name)
	assert.Equal(t, 5555, apiserver.Port)
	assert.Equal(t, "nacos", apiserver.Registry)
	assert.Contains(t, apiserver.Features, "healthz")
	require.Len(t, apiserver.Resources, 4)
	assert.Equal(t, "post", apiserver.Resources[0].Name)

	admin := p.Spec.Components[1]
	assert.Equal(t, "grpc", admin.Framework)
	assert.Equal(t, 6666, admin.GRPCPort)

	worker := p.Spec.Components[2]
	assert.Equal(t, "Worker", worker.Kind)
	assert.ElementsMatch(t, []string{"cron", "kafka", "customized"}, worker.Variants)
	require.NotNil(t, worker.Cron)
	assert.Len(t, worker.Cron.Jobs, 2)
	require.NotNil(t, worker.Kafka)
	assert.Equal(t, []string{"kafka:9092"}, worker.Kafka.Brokers)
	assert.Len(t, worker.Kafka.Topics, 2)

	cli := p.Spec.Components[3]
	assert.Equal(t, "CLI", cli.Kind)
	assert.Len(t, cli.Commands, 4)

	require.Len(t, p.Spec.Hooks.PreApply, 1)
	assert.Equal(t, "restricted", p.Spec.Hooks.PreApply[0].Policy)
	require.Len(t, p.Spec.Hooks.PostApply, 2)
	assert.Equal(t, "tidy", p.Spec.Hooks.PostApply[0].Name)
}

// ===== happy path: LoadFromBytes =====

func TestLoadFromBytes_Minimal(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixturesDir, "minimal.yaml"))
	require.NoError(t, err)

	loader := project.NewLoader()
	p, err := loader.LoadFromBytes(data)
	require.NoError(t, err)
	assert.Equal(t, "hello", p.Metadata.Name)
}

// ===== Load alias =====

func TestLoad_Alias(t *testing.T) {
	loader := project.NewLoader()
	p, err := loader.Load(filepath.Join(fixturesDir, "minimal.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "hello", p.Metadata.Name)
}

// ===== error: file not found / empty path =====

func TestLoadFromFile_EmptyPath(t *testing.T) {
	loader := project.NewLoader()
	_, err := loader.LoadFromFile("")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

func TestLoadFromFile_NotFound(t *testing.T) {
	loader := project.NewLoader()
	_, err := loader.LoadFromFile(filepath.Join(fixturesDir, "no-such-file.yaml"))
	requireLinctlError(t, err, linctlerr.ErrEnvironment)
}

func TestLoadFromBytes_EmptyInput(t *testing.T) {
	loader := project.NewLoader()
	_, err := loader.LoadFromBytes(nil)
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

// ===== error: yaml syntax =====

func TestLoadFromFile_BadSyntax(t *testing.T) {
	_, err := loadFixture(t, "invalid_yaml_syntax.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

// ===== error: KnownFields strict mode =====

func TestLoadFromFile_UnknownField(t *testing.T) {
	_, err := loadFixture(t, "invalid_unknown_field.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)

	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Hint+lerr.Message, "unknownField")
}

// ===== error: schema validation failures =====

func TestLoadFromFile_BadAPIVersion(t *testing.T) {
	_, err := loadFixture(t, "invalid_apiversion.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

func TestLoadFromFile_NoComponents(t *testing.T) {
	_, err := loadFixture(t, "invalid_no_components.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

func TestLoadFromFile_BadModulePath(t *testing.T) {
	_, err := loadFixture(t, "invalid_module_path.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Hint+lerr.Message, "module")
}

// ===== error: cross-field validation =====

func TestLoadFromFile_DuplicateComponentName(t *testing.T) {
	_, err := loadFixture(t, "invalid_dup_component.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "duplicate component name")
}

func TestLoadFromFile_GRPCWithoutPort(t *testing.T) {
	_, err := loadFixture(t, "invalid_grpc_no_port.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "grpcPort")
}

func TestLoadFromFile_GinWithGRPCPort(t *testing.T) {
	_, err := loadFixture(t, "invalid_gin_with_grpcport.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "framework=gin")
}

func TestLoadFromFile_WorkerNoVariants(t *testing.T) {
	_, err := loadFixture(t, "invalid_worker_no_variants.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "variant")
}

func TestLoadFromFile_WorkerKafkaMissing(t *testing.T) {
	_, err := loadFixture(t, "invalid_worker_kafka_missing.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "kafka")
}

func TestLoadFromBytes_WorkerCustomizedMissing(t *testing.T) {
	loader := project.NewLoader()
	yaml := []byte(`apiVersion: linctl.dev/v1
kind: Project
metadata:
  name: hello
  module: github.com/foo/hello
spec:
  components:
    - kind: Worker
      name: w1
      variants: [customized]
`)
	_, err := loader.LoadFromBytes(yaml)
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "customized")
}

func TestLoadFromBytes_WorkerCronMissing(t *testing.T) {
	loader := project.NewLoader()
	yaml := []byte(`apiVersion: linctl.dev/v1
kind: Project
metadata:
  name: hello
  module: github.com/foo/hello
spec:
  components:
    - kind: Worker
      name: w1
      variants: [cron]
`)
	_, err := loader.LoadFromBytes(yaml)
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "cron")
}

func TestLoadFromFile_CLINoCommands(t *testing.T) {
	_, err := loadFixture(t, "invalid_cli_no_commands.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

// ===== error: multi-document =====

func TestLoadFromFile_MultiDocument(t *testing.T) {
	_, err := loadFixture(t, "invalid_two_documents.yaml")
	requireLinctlError(t, err, linctlerr.ErrConfigInvalid)
}

// ===== Hint contains source path =====

func TestLoadFromFile_HintContainsSourcePath(t *testing.T) {
	path := filepath.Join(fixturesDir, "invalid_unknown_field.yaml")
	loader := project.NewLoader()
	_, err := loader.LoadFromFile(path)
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Hint, path, "Hint should mention the source file path")
}

// ===== LoadFromBytes uses default validator if no validator passed =====

func TestNewLoaderWithValidator_NilFallsBackToDefault(t *testing.T) {
	loader := project.NewLoaderWithValidator(nil)
	require.NotNil(t, loader)
	p, err := loader.LoadFromFile(filepath.Join(fixturesDir, "minimal.yaml"))
	require.NoError(t, err)
	assert.NotNil(t, p)
}

// ===== Concurrency: Loader is safe to call concurrently =====

func TestLoader_ConcurrentLoads(t *testing.T) {
	loader := project.NewLoader()
	const N = 16
	type result struct {
		p   *project.Project
		err error
	}
	results := make(chan result, N)
	for i := 0; i < N; i++ {
		go func() {
			p, err := loader.LoadFromFile(filepath.Join(fixturesDir, "full.yaml"))
			results <- result{p: p, err: err}
		}()
	}
	for i := 0; i < N; i++ {
		r := <-results
		require.NoError(t, r.err)
		require.NotNil(t, r.p)
	}
}

// ===== requireLinctlError helper =====

func requireLinctlError(t *testing.T, err error, expectedCode linctlerr.Code) {
	t.Helper()
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr), "expected *linctlerr.LinctlError, got %T: %v", err, err)
	assert.Equal(t, expectedCode, lerr.Code, "unexpected error code")
}
