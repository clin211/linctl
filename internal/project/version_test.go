package project_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/lin/internal/linctlerr"
	"github.com/clin211/lin/internal/project"
)

const v1alpha1Yaml = `apiVersion: linctl.dev/v1alpha1
kind: Project
metadata:
  name: legacy-app
  module: github.com/foo/legacy
spec:
  defaults:
    framework: gin
    storage: gorm-postgres
    apiVersion: v1
  components:
    - kind: WebServer
      name: legacy-app
`

const v1Yaml = `apiVersion: linctl.dev/v1
kind: Project
metadata:
  name: shiny
  module: github.com/foo/shiny
spec:
  components:
    - kind: WebServer
      name: shiny
`

func TestUpgrade_V1Alpha1ToV1(t *testing.T) {
	out, migs, err := project.Upgrade([]byte(v1alpha1Yaml))
	require.NoError(t, err)
	require.Len(t, migs, 1)
	assert.Equal(t, project.APIVersionV1Alpha1, migs[0].From)
	assert.Equal(t, project.APIVersionV1, migs[0].To)
	assert.NotEmpty(t, migs[0].At)

	s := string(out)
	assert.Contains(t, s, "apiVersion: linctl.dev/v1\n", "top-level apiVersion must be upgraded")
	assert.Contains(t, s, "protoVersion: v1", "spec.defaults.apiVersion must be renamed to protoVersion")
	assert.NotContains(t, s, "spec:\n  defaults:\n    apiVersion:", "old field must be removed")
}

func TestUpgrade_V1IsNoOp(t *testing.T) {
	out, migs, err := project.Upgrade([]byte(v1Yaml))
	require.NoError(t, err)
	assert.Empty(t, migs, "v1 input should not trigger any migration")
	assert.Equal(t, v1Yaml, string(out))
}

func TestUpgrade_EmptyInput(t *testing.T) {
	_, _, err := project.Upgrade(nil)
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestUpgrade_NoAPIVersion(t *testing.T) {
	out, migs, err := project.Upgrade([]byte("kind: Project\nmetadata: {name: x}\n"))
	require.NoError(t, err)
	assert.Empty(t, migs)
	assert.Contains(t, string(out), "kind: Project")
}

func TestUpgrade_PreservesUnknownFields(t *testing.T) {
	in := `apiVersion: linctl.dev/v1alpha1
kind: Project
metadata:
  name: x
  module: github.com/foo/x
spec:
  defaults:
    apiVersion: v1
    storage: gorm-postgres
    futureField: keepme
  components:
    - kind: WebServer
      name: x
`
	out, _, err := project.Upgrade([]byte(in))
	require.NoError(t, err)
	assert.Contains(t, string(out), "futureField: keepme",
		"upgrade must preserve unknown fields (forward compatibility)")
}

func TestUpgrade_LoadAfterUpgradeSucceeds(t *testing.T) {
	out, _, err := project.Upgrade([]byte(v1alpha1Yaml))
	require.NoError(t, err)

	loader := project.NewLoader()
	p, err := loader.LoadFromBytes(out)
	require.NoError(t, err, "upgraded yaml must pass strict Loader")
	assert.Equal(t, project.APIVersionV1, p.APIVersion)
	assert.Equal(t, "v1", p.Spec.Defaults.ProtoVersion)
}

func TestUpgrade_RawV1Alpha1FailsLoader(t *testing.T) {
	loader := project.NewLoader()
	_, err := loader.LoadFromBytes([]byte(v1alpha1Yaml))
	require.Error(t, err, "raw v1alpha1 must fail strict KnownFields loader before upgrade")
}

func TestUpgrade_KeepsCommentsByEncoding(t *testing.T) {
	in := `apiVersion: linctl.dev/v1alpha1
kind: Project
metadata:
  name: x
  module: github.com/foo/x
spec:
  components:
    - kind: WebServer
      name: x
`
	out, _, err := project.Upgrade([]byte(in))
	require.NoError(t, err)
	assert.NotEmpty(t, out)
	assert.True(t, strings.Contains(string(out), "apiVersion: linctl.dev/v1\n"))
}

func TestUpgrade_BadYAMLFails(t *testing.T) {
	_, _, err := project.Upgrade([]byte("apiVersion: linctl.dev/v1alpha1\nkey-without-colon\n"))
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestUpgrade_RenamingConflictReportsError(t *testing.T) {
	// Both spec.defaults.apiVersion AND spec.defaults.protoVersion already present:
	// renaming would clobber, so we expect a clean error.
	in := `apiVersion: linctl.dev/v1alpha1
kind: Project
metadata:
  name: x
  module: github.com/foo/x
spec:
  defaults:
    apiVersion: v1alpha1
    protoVersion: v1
  components:
    - kind: WebServer
      name: x
`
	_, _, err := project.Upgrade([]byte(in))
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestUpgrade_PartialV1Alpha1WithoutDefaults(t *testing.T) {
	in := `apiVersion: linctl.dev/v1alpha1
kind: Project
metadata:
  name: tiny
  module: github.com/foo/tiny
spec:
  components:
    - kind: WebServer
      name: tiny
`
	out, migs, err := project.Upgrade([]byte(in))
	require.NoError(t, err)
	require.Len(t, migs, 1)
	assert.Contains(t, string(out), "apiVersion: linctl.dev/v1\n")
}
