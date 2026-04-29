package user

import (
	"context"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/authz"
)

// UserBiz 定义处理用户请求所需的方法.
//
// 当你执行 `linctl add api <Resource>` 时，新方法（例如 PostV1() PostBiz）
// 会被自动注入到这里。
type UserBiz interface {
	Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error)
	Update(ctx context.Context, rq *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error)
	Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error)
	Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error)
	List(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error)

	UserExpansion
}

// UserExpansion 定义用户操作的扩展方法.
type UserExpansion interface {
	Login(ctx context.Context, rq *v1.LoginRequest) (*v1.LoginResponse, error)
	RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.RefreshTokenResponse, error)
	ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error)
	ListWithBadPerformance(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error)
}

// userBiz 是 UserBiz 接口的实现.
type userBiz struct {
	store store.IStore
	authz *authz.Authz
}

// 编译期断言：确保 *userBiz 实现了 UserBiz.
var _ UserBiz = (*userBiz)(nil)

// New 构造一个携带 store / authz 依赖的 *userBiz 实例.
func New(store store.IStore, authz *authz.Authz) *userBiz {
	return &userBiz{store: store, authz: authz}
}
