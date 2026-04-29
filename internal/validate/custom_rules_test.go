package validate_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/linctl/internal/validate"
)

func newRawValidator(t *testing.T) *validator.Validate {
	t.Helper()
	v := validator.New()
	require.NoError(t, validate.RegisterCustomRules(v))
	return v
}

func TestRegisterCustomRules_Idempotent(t *testing.T) {
	v := validator.New()
	require.NoError(t, validate.RegisterCustomRules(v))
	// re-register: validator/v10 silently overrides the function (no panic, no error).
	require.NoError(t, validate.RegisterCustomRules(v))
}

func TestRegisterCustomRules_NilSafe(t *testing.T) {
	require.NoError(t, validate.RegisterCustomRules(nil))
}

// ===== modulepath =====

func TestModulePath_Valid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{
		"github.com/foo/bar",
		"github.com/clin211/linctl",
		"gitlab.example.org/group/sub/proj",
		"go.opentelemetry.io/otel",
		"k8s.io/api/core/v1",
	}
	for _, in := range cases {
		assert.NoError(t, v.Var(in, "modulepath"), "input: %q", in)
	}
}

func TestModulePath_Invalid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{
		"",
		"no-dot",
		"ends/with/slash/",
		"/leading/slash",
		"with space/foo",
	}
	for _, in := range cases {
		assert.Error(t, v.Var(in, "modulepath"), "input: %q should be invalid", in)
	}
}

// ===== projectname =====

func TestProjectName_Valid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"a", "abc", "abc-123", "miniblog", "x-y-z"}
	for _, in := range cases {
		assert.NoError(t, v.Var(in, "projectname"), "input: %q", in)
	}
}

func TestProjectName_Invalid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{
		"",        // empty
		"1abc",    // starts with digit
		"-abc",    // starts with dash
		"AbcD",    // uppercase
		"abc_def", // underscore
		"abcdefghijklmnopqrstuvwxyz0123456789012345", // > 40 chars
	}
	for _, in := range cases {
		assert.Error(t, v.Var(in, "projectname"), "input: %q should be invalid", in)
	}
}

// ===== kindname =====

func TestKindName_Valid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"Post", "post", "post_v2", "User-1", "job/cron", "X"}
	for _, in := range cases {
		assert.NoError(t, v.Var(in, "kindname"), "input: %q", in)
	}
}

func TestKindName_Invalid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"", "1abc", "-abc", "abc def", "with.dot"}
	for _, in := range cases {
		assert.Error(t, v.Var(in, "kindname"), "input: %q should be invalid", in)
	}
}

// ===== featurename =====

func TestFeatureName_Valid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"healthz", "open-telemetry", "user", "preloader-v2"}
	for _, in := range cases {
		assert.NoError(t, v.Var(in, "featurename"), "input: %q", in)
	}
}

func TestFeatureName_Invalid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"", "Healthz", "user_x", "1user"}
	for _, in := range cases {
		assert.Error(t, v.Var(in, "featurename"), "input: %q should be invalid", in)
	}
}

// ===== componentname =====

func TestComponentName_Valid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"mb-apiserver", "worker", "mbctl", "x-y-1"}
	for _, in := range cases {
		assert.NoError(t, v.Var(in, "componentname"), "input: %q", in)
	}
}

func TestComponentName_Invalid(t *testing.T) {
	v := newRawValidator(t)
	cases := []string{"", "Mb-API", "mb_under", "1abc"}
	for _, in := range cases {
		assert.Error(t, v.Var(in, "componentname"), "input: %q should be invalid", in)
	}
}

// ===== HintForTag =====

func TestHintForTag_KnownTags(t *testing.T) {
	tags := []string{
		"modulepath", "projectname", "kindname", "featurename", "componentname",
		"required", "oneof", "min", "max", "email", "hostname_port", "startswith",
	}
	for _, tag := range tags {
		assert.NotEmpty(t, validate.HintForTag(tag), "tag %q should have a hint", tag)
	}
}

func TestHintForTag_UnknownReturnsEmpty(t *testing.T) {
	assert.Empty(t, validate.HintForTag("no_such_tag"))
}
