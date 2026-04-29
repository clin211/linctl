package biz

import (
	"github.com/google/wire"

	userv1 "{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/biz/v1/user"
	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
	"{{ .Project.Metadata.Module }}/pkg/authz"
)

// ProviderSet 声明 biz 层的 Wire DI 规则：构造 *biz、并把 IBiz 绑定到具体实现。
var ProviderSet = wire.NewSet(NewBiz, wire.Bind(new(IBiz), new(*biz)))

// IBiz 是业务层对外暴露的统一入口接口。
//
// 当你执行 `linctl add api <Resource>` 时，新方法（例如 PostV1() PostBiz）
// 会被自动注入到这个接口里。
type IBiz interface {
	// UserV1 获取用户业务接口.
	UserV1() userv1.UserBiz
}

// biz 是 IBiz 的默认实现，持有数据访问与权限组件。
type biz struct {
	store store.IStore
	authz *authz.Authz
}

// 编译期断言：确保 *biz 实现了 IBiz。
var _ IBiz = (*biz)(nil)

// NewBiz 构造一个携带 store / authz 依赖的 *biz 实例。
func NewBiz(store store.IStore, az *authz.Authz) *biz {
	return &biz{store: store, authz: az}
}

// UserV1 返回一个实现了 UserBiz 接口的实例.
func (b *biz) UserV1() userv1.UserBiz {
	return userv1.New(b.store, b.authz)
}
