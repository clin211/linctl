package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/clin211/linctl/internal/project"
)

func TestStatus_IsEmpty_TrueByDefault(t *testing.T) {
	var s project.Status
	assert.True(t, s.IsEmpty())
}

func TestStatus_IsEmpty_FalseWhenAnyFieldSet(t *testing.T) {
	cases := []project.Status{
		{GeneratedAt: "x"},
		{CLIVersion: "v1"},
		{LastApplyHash: "sha"},
		{SchemaMigrations: []project.SchemaMigration{{From: "a", To: "b", At: "now"}}},
	}
	for _, s := range cases {
		assert.False(t, s.IsEmpty())
	}
}

func TestStatus_AppendMigration_DoesNotMutateOriginal(t *testing.T) {
	s := project.Status{
		SchemaMigrations: []project.SchemaMigration{{From: "a", To: "b", At: "t1"}},
	}

	s2 := s.AppendMigration(project.SchemaMigration{From: "b", To: "c", At: "t2"})
	assert.Len(t, s.SchemaMigrations, 1, "original must remain untouched")
	assert.Len(t, s2.SchemaMigrations, 2)
	assert.Equal(t, "c", s2.SchemaMigrations[1].To)
}

func TestStatus_AppendMigration_NilSliceOK(t *testing.T) {
	var s project.Status
	s2 := s.AppendMigration(project.SchemaMigration{From: "a", To: "b", At: "t"})
	assert.Empty(t, s.SchemaMigrations)
	assert.Len(t, s2.SchemaMigrations, 1)
}

func TestProjectState_Defaults_AreEmpty(t *testing.T) {
	var st project.ProjectState
	assert.Empty(t, st.APIVersion)
	assert.Empty(t, st.Kind)
	assert.True(t, st.Status.IsEmpty())
}
