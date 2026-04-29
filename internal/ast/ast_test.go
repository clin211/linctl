package ast_test

import (
	"context"
	"testing"

	last "github.com/clin211/lin/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInterfaceMethodMutator_NewMethod(t *testing.T) {
	src := `package biz

type IBiz interface {
	Users() UserBiz
}
`
	m := &last.AddInterfaceMethodMutator{
		FilePath:      "biz.go",
		InterfaceName: "IBiz",
		MethodName:    "Posts",
		Params:        "",
		Returns:       "PostBiz",
	}
	out, modified, err := m.Apply(context.Background(), []byte(src))
	require.NoError(t, err)
	assert.True(t, modified)
	assert.Contains(t, string(out), "Posts()")
	assert.Contains(t, string(out), "PostBiz")
	assert.Contains(t, string(out), "Users()")
}

func TestAddInterfaceMethodMutator_Idempotent(t *testing.T) {
	src := `package biz

type IBiz interface {
	Posts() PostBiz
}
`
	m := &last.AddInterfaceMethodMutator{
		FilePath:      "biz.go",
		InterfaceName: "IBiz",
		MethodName:    "Posts",
		Params:        "",
		Returns:       "PostBiz",
	}
	_, modified, err := m.Apply(context.Background(), []byte(src))
	require.NoError(t, err)
	assert.False(t, modified)
}

func TestAddInterfaceMethodMutator_SignatureConflict(t *testing.T) {
	src := `package biz

type IBiz interface {
	Posts(id int64) PostBiz
}
`
	m := &last.AddInterfaceMethodMutator{
		FilePath:      "biz.go",
		InterfaceName: "IBiz",
		MethodName:    "Posts",
		Params:        "",
		Returns:       "PostBiz",
	}
	_, _, err := m.Apply(context.Background(), []byte(src))
	require.Error(t, err)
	conflict, ok := err.(*last.ConflictError)
	require.True(t, ok, "expected *ConflictError, got %T", err)
	assert.Equal(t, "interface_method_signature_mismatch", conflict.Kind)
	assert.Contains(t, conflict.Existing, "id int64")
}

func TestAddInterfaceMethodMutator_InterfaceNotFound(t *testing.T) {
	src := `package biz

type IUser interface {
	Get() string
}
`
	m := &last.AddInterfaceMethodMutator{
		FilePath:      "biz.go",
		InterfaceName: "IBiz",
		MethodName:    "Posts",
		Returns:       "PostBiz",
	}
	_, _, err := m.Apply(context.Background(), []byte(src))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interface IBiz not found")
}
