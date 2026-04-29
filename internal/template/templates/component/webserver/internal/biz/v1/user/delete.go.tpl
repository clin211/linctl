package user

import (
	"context"
	"log/slog"

	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	"{{ .Project.Metadata.Module }}/internal/pkg/known"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// Delete 实现 UserBiz 接口中的 Delete 方法.
//
// 这里有意不用 where.T(ctx) —— 因为 root 用户允许删除其它用户，
// 用 where.T() 会把删除范围收窄到当前登录用户自己。
func (b *userBiz) Delete(ctx context.Context, rq *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	if err := b.store.User().Delete(ctx, where.F("user_id", rq.GetUserID())); err != nil {
		return nil, err
	}

	if _, err := b.authz.RemoveGroupingPolicy(rq.GetUserID(), known.RoleUser); err != nil {
		slog.ErrorContext(ctx, "Failed to remove grouping policy for user", "user", rq.GetUserID(), "role", known.RoleUser, "error", err)
		return nil, errno.ErrRemoveRole.WithMessage(err.Error())
	}

	return &v1.DeleteUserResponse{}, nil
}
