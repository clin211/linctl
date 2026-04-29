package user

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/token"
)

// RefreshToken 用于刷新用户的身份验证令牌.
//
// 调用前必须经过 AuthnMiddleware（从 Header 中拿到 userID 注入 ctx），
// 当 token 即将过期时由前端调用本接口生成新 token。
func (b *userBiz) RefreshToken(ctx context.Context, rq *v1.RefreshTokenRequest) (*v1.RefreshTokenResponse, error) {
	tokenStr, expireAt, err := token.Sign(contextx.UserID(ctx))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to sign token", "error", err)
		return nil, errno.ErrSignToken
	}

	return &v1.RefreshTokenResponse{Token: tokenStr, ExpireAt: timestamppb.New(expireAt)}, nil
}
