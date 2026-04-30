package tpl_test

import (
	"testing"

	"github.com/clin211/lin/internal/pkg/tpl"
)

func TestDefaultFuncs_CaseConversions(t *testing.T) {
	funcs := tpl.DefaultFuncs()

	tests := []struct {
		name     string
		fn       string
		input    string
		expected string
	}{
		{"Pascal snake", "Pascal", "post_item", "PostItem"},
		{"Pascal kebab", "Pascal", "post-item", "PostItem"},
		{"Camel", "Camel", "post_item", "postItem"},
		{"LowerCamel", "LowerCamel", "post_item", "postItem"},
		{"Snake pascal", "Snake", "PostItem", "post_item"},
		{"Snake camel", "Snake", "postItem", "post_item"},
		{"Kebab", "Kebab", "PostItem", "post-item"},
		{"Lower", "Lower", "HELLO", "hello"},
		{"Upper", "Upper", "hello", "HELLO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := funcs[tt.fn]
			if !ok {
				t.Fatalf("func %q not found in DefaultFuncs", tt.fn)
			}
			result := fn.(func(string) string)(tt.input)
			if result != tt.expected {
				t.Errorf("%s(%q) = %q, want %q", tt.fn, tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultFuncs_Plural(t *testing.T) {
	funcs := tpl.DefaultFuncs()
	plural := funcs["Plural"].(func(string) string)
	singular := funcs["Singular"].(func(string) string)

	if got := plural("user"); got != "users" {
		t.Errorf("Plural(user) = %q, want %q", got, "users")
	}
	if got := plural("post"); got != "posts" {
		t.Errorf("Plural(post) = %q, want %q", got, "posts")
	}
	if got := singular("users"); got != "user" {
		t.Errorf("Singular(users) = %q, want %q", got, "user")
	}
}

func TestDefaultFuncs_Has(t *testing.T) {
	funcs := tpl.DefaultFuncs()
	has := funcs["Has"].(func(string, []string) bool)

	if !has("healthz", []string{"healthz", "otel"}) {
		t.Error("Has('healthz', [healthz otel]) should be true")
	}
	if has("user", []string{"healthz", "otel"}) {
		t.Error("Has('user', [healthz otel]) should be false")
	}
	if has("anything", nil) {
		t.Error("Has('anything', nil) should be false")
	}
}

func TestDefaultFuncs_Default(t *testing.T) {
	funcs := tpl.DefaultFuncs()
	def := funcs["Default"].(func(string, string) string)

	if got := def("fallback", ""); got != "fallback" {
		t.Errorf("Default(fallback, '') = %q, want 'fallback'", got)
	}
	if got := def("fallback", "actual"); got != "actual" {
		t.Errorf("Default(fallback, actual) = %q, want 'actual'", got)
	}
}

func TestDefaultFuncs_Join(t *testing.T) {
	funcs := tpl.DefaultFuncs()
	join := funcs["Join"].(func([]string, string) string)

	if got := join([]string{"a", "b", "c"}, ","); got != "a,b,c" {
		t.Errorf("Join([a b c], ,) = %q, want 'a,b,c'", got)
	}
}
