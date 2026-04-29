package template_test

import (
	"testing"
	"time"

	tpl "github.com/clin211/linctl/internal/template"
	"github.com/stretchr/testify/assert"
)

func TestToKebab(t *testing.T) {
	cases := map[string]string{
		"UserProfile":  "user-profile",
		"user_profile": "user-profile",
		"user-profile": "user-profile",
		"USER":         "user",
		"":             "",
		"myABCTest":    "my-abc-test",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.ToKebab(in), "input=%q", in)
	}
}

func TestToSnake(t *testing.T) {
	cases := map[string]string{
		"UserProfile":  "user_profile",
		"user-profile": "user_profile",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.ToSnake(in), "input=%q", in)
	}
}

func TestToUpperKebab(t *testing.T) {
	cases := map[string]string{
		"miniblog-v4":  "MINIBLOG-V4",
		"UserProfile":  "USER-PROFILE",
		"user_profile": "USER-PROFILE",
		"apiserver":    "APISERVER",
		"":             "",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.ToUpperKebab(in), "input=%q", in)
	}
}

func TestToCamel(t *testing.T) {
	cases := map[string]string{
		"user-profile": "userProfile",
		"user_profile": "userProfile",
		"UserProfile":  "userProfile",
		"":             "",
		"single":       "single",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.ToCamel(in), "input=%q", in)
	}
}

func TestToPascal(t *testing.T) {
	cases := map[string]string{
		"user-profile": "UserProfile",
		"user_profile": "UserProfile",
		"userProfile":  "UserProfile",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.ToPascal(in), "input=%q", in)
	}
}

func TestPlural(t *testing.T) {
	cases := map[string]string{
		"post":   "posts",
		"box":    "boxes",
		"city":   "cities",
		"buzz":   "buzzes",
		"life":   "lives",
		"calf":   "calves",
		"person": "people",
		"child":  "children",
		"":       "",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.Plural(in), "input=%q", in)
	}
}

func TestSingular(t *testing.T) {
	cases := map[string]string{
		"posts":    "post",
		"cities":   "city",
		"lives":    "life",
		"people":   "person",
		"":         "",
	}
	for in, want := range cases {
		assert.Equal(t, want, tpl.Singular(in), "input=%q", in)
	}
}

func TestContains(t *testing.T) {
	assert.True(t, tpl.Contains([]string{"a", "b"}, "a"))
	assert.False(t, tpl.Contains([]string{"a", "b"}, "c"))
	assert.False(t, tpl.Contains(nil, "a"))
}

func TestUnique(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b"}
	assert.Equal(t, []string{"a", "b", "c"}, tpl.Unique(in))
}

func TestFirstLast(t *testing.T) {
	items := []int{1, 2, 3}
	assert.Equal(t, 1, tpl.First(items))
	assert.Equal(t, 3, tpl.Last(items))
	assert.Equal(t, 0, tpl.First[int](nil))
	assert.Equal(t, 0, tpl.Last[int](nil))
}

func TestDefault(t *testing.T) {
	assert.Equal(t, "x", tpl.Default("x", ""))
	assert.Equal(t, "y", tpl.Default("x", "y"))
	assert.Equal(t, 8080, tpl.Default(8080, 0))
	assert.Equal(t, 9090, tpl.Default(8080, 9090))
}

func TestHasFeature(t *testing.T) {
	assert.True(t, tpl.HasFeature([]string{"healthz", "user"}, "user"))
	assert.False(t, tpl.HasFeature([]string{"healthz"}, "user"))
	assert.True(t, tpl.HasFeature([]any{"healthz", "user"}, "user"))
	assert.False(t, tpl.HasFeature(nil, "user"))
}

func TestHasComponent(t *testing.T) {
	components := []any{
		map[string]any{"Kind": "WebServer", "Name": "api"},
		map[string]any{"Kind": "Worker", "Name": "bg"},
	}
	assert.True(t, tpl.HasComponent(components, "WebServer"))
	assert.True(t, tpl.HasComponent(components, "api"))
	assert.False(t, tpl.HasComponent(components, "CLI"))
}

func TestLen(t *testing.T) {
	assert.Equal(t, 0, tpl.Len(nil))
	assert.Equal(t, 3, tpl.Len("abc"))
	assert.Equal(t, 2, tpl.Len([]string{"a", "b"}))
	assert.Equal(t, 1, tpl.Len(map[string]any{"k": 1}))
}

func TestNow_NotZero(t *testing.T) {
	got := tpl.Now()
	assert.False(t, got.IsZero())
}

func TestDate_Format(t *testing.T) {
	frozen := time.Date(2025, 11, 9, 18, 30, 0, 0, time.UTC)
	assert.Equal(t, "2025", tpl.Date("2006", frozen))
	assert.Equal(t, "2025-11-09", tpl.Date("2006-01-02", frozen))
	assert.Equal(t, "2025-11-09 18:30", tpl.Date("2006-01-02 15:04", frozen))
}

func TestYear_FrozenViaSetNowForTest(t *testing.T) {
	frozen := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	restore := tpl.SetNowForTest(func() time.Time { return frozen })
	t.Cleanup(restore)

	assert.Equal(t, "2030", tpl.Year())
	assert.Equal(t, "2030", tpl.Date("2006", tpl.Now()))
}
