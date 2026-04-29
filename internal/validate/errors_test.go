package validate_test

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clin211/linctl/internal/linctlerr"
	"github.com/clin211/linctl/internal/validate"
)

type errSample struct {
	Name string `yaml:"name" validate:"required,projectname"`
	Port int    `yaml:"port" validate:"required,min=1024,max=65535"`
}

func TestToLinctlError_NilReturnsNil(t *testing.T) {
	assert.Nil(t, validate.ToLinctlError(nil))
}

func TestToLinctlError_NonValidatorWraps(t *testing.T) {
	cause := errors.New("io broke")
	out := validate.ToLinctlError(cause)
	require.NotNil(t, out)
	assert.Equal(t, linctlerr.ErrConfigInvalid, out.Code)
	assert.Contains(t, out.Error(), "io broke")
}

func TestToLinctlError_ValidatorErrorsAggregated(t *testing.T) {
	v := validate.NewValidator()
	err := v.Raw().Struct(&errSample{Name: "Bad", Port: 0})
	require.Error(t, err)
	out := validate.ToLinctlError(err)
	require.NotNil(t, out)
	assert.Equal(t, linctlerr.ErrConfigInvalid, out.Code)
	assert.Contains(t, out.Message, "name")
	assert.Contains(t, out.Message, "port")
}

func TestIssuesFromError_NotValidator(t *testing.T) {
	issues := validate.IssuesFromError(errors.New("plain"))
	assert.Nil(t, issues)
}

func TestIssuesFromError_NilReturnsNil(t *testing.T) {
	assert.Nil(t, validate.IssuesFromError(nil))
}

func TestIssuesFromError_FromValidator(t *testing.T) {
	v := validate.NewValidator()
	err := v.Raw().Struct(&errSample{Name: "BadCaps", Port: 0})
	require.Error(t, err)

	issues := validate.IssuesFromError(err)
	require.GreaterOrEqual(t, len(issues), 2)

	fields := make(map[string]validate.FieldIssue)
	for _, is := range issues {
		fields[is.Field] = is
	}

	require.Contains(t, fields, "name")
	require.Contains(t, fields, "port")
	assert.Equal(t, "projectname", fields["name"].Tag)
	assert.NotEmpty(t, fields["name"].Message)
	assert.NotEmpty(t, fields["name"].Hint)
}

func TestFieldIssue_String(t *testing.T) {
	fi := validate.FieldIssue{
		Field:   "spec.components[0].name",
		Tag:     "required",
		Message: "field is required",
	}
	got := fi.String()
	assert.Contains(t, got, "spec.components[0].name")
	assert.Contains(t, got, "field is required")

	fi2 := fi
	fi2.Value = "Bad"
	got2 := fi2.String()
	assert.Contains(t, got2, "got \"Bad\"")
}

func TestToLinctlError_WrapsValidationErrorsCause(t *testing.T) {
	v := validate.NewValidator()
	err := v.Raw().Struct(&errSample{Name: "x", Port: 0}) // single failure
	require.Error(t, err)
	out := validate.ToLinctlError(err)
	require.NotNil(t, out)

	var ves validator.ValidationErrors
	require.True(t, errors.As(out, &ves), "Cause must remain validator.ValidationErrors")
	require.NotEmpty(t, ves)
}

func TestToLinctlError_HintListsOneFieldPerLine(t *testing.T) {
	v := validate.NewValidator()
	err := v.Raw().Struct(&errSample{Name: "BadCaps", Port: 0})
	require.Error(t, err)
	out := validate.ToLinctlError(err)
	require.NotNil(t, out)
	assert.Contains(t, out.Hint, "- name:")
	assert.Contains(t, out.Hint, "- port:")
}

// ===== exhaustive tag coverage to exercise messageForTag branches =====

type tagSweep struct {
	OneOf      string `yaml:"oneOf"      validate:"omitempty,oneof=a b c"`
	Email      string `yaml:"email"      validate:"omitempty,email"`
	Hostport   string `yaml:"hostport"   validate:"omitempty,hostname_port"`
	StartsWith string `yaml:"startsWith" validate:"omitempty,startswith=v"`
	Min3       string `yaml:"min3"       validate:"omitempty,min=3"`
	Max3       string `yaml:"max3"       validate:"omitempty,max=3"`
	Len4       string `yaml:"len4"       validate:"omitempty,len=4"`
	Module     string `yaml:"module"     validate:"omitempty,modulepath"`
	Comp       string `yaml:"comp"       validate:"omitempty,componentname"`
	Feat       string `yaml:"feat"       validate:"omitempty,featurename"`
	Kind       string `yaml:"kind"       validate:"omitempty,kindname"`
	Project    string `yaml:"project"    validate:"omitempty,projectname"`
}

func TestMessageForTag_AllSupportedTagsRendered(t *testing.T) {
	v := validate.NewValidator()
	// Provide values that fail each tag.
	err := v.Raw().Struct(&tagSweep{
		OneOf:      "z",
		Email:      "no-at-sign",
		Hostport:   "no-colon",
		StartsWith: "x",
		Min3:       "ab",
		Max3:       "abcd",
		Len4:       "abc",
		Module:     "no-dot",
		Comp:       "Bad",
		Feat:       "Bad",
		Kind:       "with.dot",
		Project:    "Bad",
	})
	require.Error(t, err)

	issues := validate.IssuesFromError(err)
	require.NotEmpty(t, issues)

	tagsSeen := make(map[string]bool)
	for _, is := range issues {
		tagsSeen[is.Tag] = true
		assert.NotEmpty(t, is.Message, "tag %q should have a rendered message", is.Tag)
	}
	expected := []string{"oneof", "email", "hostname_port", "startswith", "min", "max", "len", "modulepath", "componentname", "featurename", "kindname", "projectname"}
	for _, e := range expected {
		assert.True(t, tagsSeen[e], "expected tag %q to be seen in issues", e)
	}
}

// ===== normalizeFieldPath edge cases (via real validator + IssuesFromError) =====

type lowerStruct struct {
	Already string `yaml:"already" validate:"required"`
	Bracket string `yaml:"bracket" validate:"required"`
}

func TestNormalizeFieldPath_AlreadyLowercase(t *testing.T) {
	v := validate.NewValidator()
	err := v.Raw().Struct(&lowerStruct{})
	require.Error(t, err)
	issues := validate.IssuesFromError(err)
	require.Len(t, issues, 2)
	for _, is := range issues {
		assert.Regexp(t, "^[a-z]", is.Field, "first char must be lowercase: %s", is.Field)
	}
}
