package user

import (
	"context"
	"log/slog"

	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/authn"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// ChangePassword 实现 UserBiz 接口中的 ChangePassword 方法.
//
// 流程：
//  1. 按租户查出当前用户。
//  2. 用 authn.Compare 比对旧密码（输入的明文 vs DB 中的哈希）。
//  3. 用 authn.Encrypt 重新加密新密码并写回。
func (b *userBiz) ChangePassword(ctx context.Context, rq *v1.ChangePasswordRequest) (*v1.ChangePasswordResponse, error) {
	userM, err := b.store.User().Get(ctx, where.T(ctx))
	if err != nil {
		return nil, err
	}

	if err := authn.Compare(userM.Password, rq.GetOldPassword()); err != nil {
		slog.ErrorContext(ctx, "Failed to compare password", "error", err)
		return nil, errno.ErrPasswordInvalid
	}

	userM.Password, _ = authn.Encrypt(rq.GetNewPassword())
	if err := b.store.User().Update(ctx, userM); err != nil {
		return nil, err
	}

	return &v1.ChangePasswordResponse{}, nil
}
