package biz

import (
	"{{.Module}}/internal/{{.AppName}}/store"
)

// IBiz defines the methods that must be implemented by the business layer.
//
// `lin add <Resource>` appends new methods to this interface via AST.
type IBiz interface {
}

// biz is the concrete implementation of IBiz.
type biz struct {
	store store.IStore
}

// Ensure biz implements IBiz.
var _ IBiz = (*biz)(nil)

// NewBiz creates a new IBiz instance.
func NewBiz(s store.IStore) *biz {
	return &biz{store: s}
}
