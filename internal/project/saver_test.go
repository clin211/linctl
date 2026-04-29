package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/project"
)

func newSampleProject() *project.Project {
	p := &project.Project{
		APIVersion: project.APIVersionV1,
		Kind:       project.KindProject,
		Metadata: project.Metadata{
			Name:   "miniblog",
			Module: "github.com/foo/miniblog",
		},
		Spec: project.Spec{
			Components: []project.Component{
				{Kind: "WebServer", Name: "miniblog"},
			},
		},
	}
	project.ApplyDefaults(p)
	return p
}

func TestSaveState_WritesHeaderAndStatus(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")

	status := project.Status{
		LastApplyHash: "sha256:dead-beef",
		SchemaMigrations: []project.SchemaMigration{
			{From: project.APIVersionV1Alpha1, To: project.APIVersionV1, At: "2026-04-25T00:00:00Z"},
		},
	}
	err := project.SaveState(path, newSampleProject(), status, "v1.2.3")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.True(t, strings.HasPrefix(content, "# DO NOT EDIT MANUALLY."),
		"PROJECT file must start with 'DO NOT EDIT MANUALLY.' comment header")
	assert.Contains(t, content, "linctl plan")
	assert.Contains(t, content, "linctl apply")
	assert.Contains(t, content, "lastApplyHash: sha256:dead-beef")
	assert.Contains(t, content, "cliVersion: v1.2.3")
	assert.Contains(t, content, "name: miniblog")
	assert.Contains(t, content, "apiVersion: linctl.dev/v1")
	assert.Contains(t, content, "schemaMigrations:")
}

func TestSaveState_NilProject(t *testing.T) {
	err := project.SaveState("ignored", nil, project.Status{}, "v1")
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrInternal, lerr.Code)
}

func TestSaveState_EmptyPath(t *testing.T) {
	err := project.SaveState("", newSampleProject(), project.Status{}, "v1")
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestSaveState_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "PROJECT")
	err := project.SaveState(path, newSampleProject(), project.Status{}, "v1")
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestSaveState_AtomicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")

	require.NoError(t, os.WriteFile(path, []byte("OLD"), 0o644))
	require.NoError(t, project.SaveState(path, newSampleProject(), project.Status{}, "v2"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "OLD")
	assert.Contains(t, string(data), "cliVersion: v2")
}

func TestSaveState_SyncsMetadataNameOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")

	p := newSampleProject()
	p.Metadata.Description = "secret-description"
	p.Metadata.Author = project.Author{Name: "secret-name", Email: "secret@example.com"}
	p.Metadata.Module = "github.com/should-not-leak"

	require.NoError(t, project.SaveState(path, p, project.Status{}, "v1"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "name: miniblog")
	assert.NotContains(t, content, "secret-description")
	assert.NotContains(t, content, "secret-name")
	assert.NotContains(t, content, "secret@example.com")
	assert.NotContains(t, content, "should-not-leak")
}

func TestLoadState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")

	status := project.Status{
		LastApplyHash: "sha256:abc123",
	}
	require.NoError(t, project.SaveState(path, newSampleProject(), status, "v1.5.0"))

	state, err := project.LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, project.APIVersionV1, state.APIVersion)
	assert.Equal(t, "Project", state.Kind)
	assert.Equal(t, "miniblog", state.Metadata.Name)
	assert.Equal(t, "v1.5.0", state.Status.CLIVersion)
	assert.Equal(t, "sha256:abc123", state.Status.LastApplyHash)
	assert.NotEmpty(t, state.Status.GeneratedAt)
}

func TestLoadState_EmptyPath(t *testing.T) {
	_, err := project.LoadState("")
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestLoadState_NotFound(t *testing.T) {
	_, err := project.LoadState(filepath.Join(t.TempDir(), "no-such"))
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrEnvironment, lerr.Code)
}

func TestLoadState_TolerantToHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")
	body := `# DO NOT EDIT MANUALLY.
# (more comments)

apiVersion: linctl.dev/v1
kind: Project
metadata:
  name: hi
status:
  cliVersion: v9
`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	state, err := project.LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, "hi", state.Metadata.Name)
	assert.Equal(t, "v9", state.Status.CLIVersion)
}

func TestSaveState_KeepsExistingGeneratedAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PROJECT")

	status := project.Status{
		GeneratedAt: "2020-01-01T00:00:00Z",
	}
	require.NoError(t, project.SaveState(path, newSampleProject(), status, ""))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "generatedAt: \"2020-01-01T00:00:00Z\"")
}
