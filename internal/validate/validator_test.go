package validate_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/validate"
)

type sample struct {
	Name   string `yaml:"name"   validate:"required,projectname"`
	Module string `yaml:"module" validate:"required,modulepath"`
	Email  string `yaml:"email"  validate:"omitempty,email"`
}

func TestNewValidator_NotNil(t *testing.T) {
	v := validate.NewValidator()
	require.NotNil(t, v)
	require.NotNil(t, v.Raw())
}

func TestDefault_Singleton(t *testing.T) {
	a := validate.Default()
	b := validate.Default()
	assert.Same(t, a, b, "Default() must return the same instance")
}

func TestStruct_HappyPath(t *testing.T) {
	v := validate.Default()
	err := v.Struct(&sample{
		Name:   "miniblog",
		Module: "github.com/foo/miniblog",
		Email:  "user@example.com",
	})
	assert.NoError(t, err)
}

func TestStruct_NilSafe(t *testing.T) {
	v := validate.Default()
	assert.NoError(t, v.Struct(nil))
}

func TestStruct_NilReceiverSafe(t *testing.T) {
	var v *validate.Validator
	assert.NoError(t, v.Struct(&sample{Name: "x", Module: "github.com/x/y"}))
}

func TestStruct_FailsOnRequired(t *testing.T) {
	v := validate.Default()
	err := v.Struct(&sample{}) // both required fields empty
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
	assert.Contains(t, lerr.Message, "name")
	assert.Contains(t, lerr.Message, "module")
}

func TestStruct_FieldPathUsesYAMLNames(t *testing.T) {
	v := validate.Default()
	err := v.Struct(&sample{Name: "Bad-Caps", Module: "no-dot"})
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "name", "should use yaml tag 'name', not Go field 'Name'")
	assert.Contains(t, lerr.Message, "module", "should use yaml tag 'module'")
}

func TestStruct_FailsOnEmail(t *testing.T) {
	v := validate.Default()
	err := v.Struct(&sample{
		Name:   "ok",
		Module: "github.com/x/y",
		Email:  "not-an-email",
	})
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Message, "email")
}

func TestVar_HappyPath(t *testing.T) {
	v := validate.Default()
	assert.NoError(t, v.Var("github.com/foo/bar", "modulepath"))
	assert.NoError(t, v.Var("miniblog", "projectname"))
}

func TestVar_Failure(t *testing.T) {
	v := validate.Default()
	err := v.Var("Bad-Caps", "projectname")
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Equal(t, linctlerr.ErrConfigInvalid, lerr.Code)
}

func TestRaw_NotNil(t *testing.T) {
	assert.NotNil(t, validate.NewValidator().Raw())
	assert.Nil(t, (*validate.Validator)(nil).Raw())
}

// ===== Concurrency =====

func TestValidator_Concurrent(t *testing.T) {
	v := validate.Default()
	var wg sync.WaitGroup
	const N = 64
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = v.Struct(&sample{Name: "ok", Module: "github.com/foo/bar"})
		}()
	}
	wg.Wait()
}

// ===== HintAttached on returned LinctlError =====

func TestStruct_AttachesHintFromTag(t *testing.T) {
	v := validate.Default()
	err := v.Struct(&sample{Name: "Bad-Caps", Module: "github.com/foo/bar"})
	require.Error(t, err)
	var lerr *linctlerr.LinctlError
	require.True(t, errors.As(err, &lerr))
	assert.Contains(t, lerr.Hint, "kebab-case", "hint should mention kebab-case for projectname")
}
