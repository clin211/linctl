// Package validation provides request validation utilities.
package validation

import (
	"{{.Module}}/internal/{{.AppName}}/store"
)

// Validator holds dependencies for request validation.
type Validator struct {
	store store.IStore
}

// New creates a new Validator instance.
func New(s store.IStore) *Validator {
	return &Validator{store: s}
}
