package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/clin211/lin/internal/cli"
	"github.com/clin211/lin/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand_Text(t *testing.T) {
	out := runCmd(t, []string{"version"})
	assert.Contains(t, out, "linctl")
	assert.Contains(t, out, "dev") // default Version
}

func TestVersionCommand_JSON(t *testing.T) {
	out := runCmd(t, []string{"version", "--output", "json"})
	var info version.Info
	require.NoError(t, json.Unmarshal([]byte(out), &info))
	assert.Equal(t, "dev", info.Version)
	assert.NotEmpty(t, info.GoVersion)
}

func TestVersionCommand_YAML(t *testing.T) {
	out := runCmd(t, []string{"version", "-o", "yaml"})
	assert.Contains(t, out, "version:")
	assert.Contains(t, out, "goVersion:")
}

func TestVersionCommand_InvalidOutput(t *testing.T) {
	root := cli.NewRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"version", "--output", "xml"})
	err := root.ExecuteContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config_invalid")
}

func runCmd(t *testing.T, args []string) string {
	t.Helper()
	root := cli.NewRootCommand()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	require.NoError(t, root.ExecuteContext(context.Background()))
	return strings.TrimSpace(buf.String())
}
