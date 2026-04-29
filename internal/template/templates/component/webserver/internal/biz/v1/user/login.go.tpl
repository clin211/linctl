package user

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/authn"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
	"{{ .Project.Metadata.Module }}/pkg/token"
)

// Login 实现 UserBiz 接口中的 Login 方法.
//
// 流程：用 username 查出用户 → 校验明文密码 → token.Sign 签发 JWT。
func (b *userBiz) Login(ctx context.Context, rq *v1.LoginRequest) (*v1.LoginResponse, error) {
	whr := where.F("username", rq.GetUsername())
	userM, err := b.store.User().Get(ctx, whr)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	if err := authn.Compare(userM.Password, rq.GetPassword()); err != nil {
		slog.ErrorContext(ctx, "Failed to compare password", "error", err)
		return nil, errno.ErrPasswordInvalid
	}

	tokenStr, expireAt, err := token.Sign(userM.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to sign token", "error", err)
		return nil, errno.ErrSignToken
	}

	return &v1.LoginResponse{Token: tokenStr, ExpireAt: timestamppb.New(expireAt)}, nil
}
