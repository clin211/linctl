package biz

import (
	"{{.Module}}/internal/{{.AppName}}/store"
)

// IBiz 定义业务层必须实现的方法集。
//
// `linctl add <Resource>` 通过 AST 向该接口追加新方法。
type IBiz interface {
}

// biz 是 IBiz 的具体实现。
type biz struct {
	store store.IStore
}

// 确保 biz 实现了 IBiz 接口。
var _ IBiz = (*biz)(nil)

// NewBiz 创建一个 IBiz 实例。
func NewBiz(s store.IStore) *biz {
	return &biz{store: s}
}
