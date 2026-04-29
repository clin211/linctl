package user

import (
	"context"
	"log/slog"

	"github.com/jinzhu/copier"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/model"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	"{{ .Project.Metadata.Module }}/internal/pkg/known"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
)

// Create 实现 UserBiz 接口中的 Create 方法.
func (b *userBiz) Create(ctx context.Context, rq *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	var userM model.UserM
	_ = copier.Copy(&userM, rq)

	if err := b.store.User().Create(ctx, &userM); err != nil {
		return nil, err
	}

	if _, err := b.authz.AddGroupingPolicy(userM.UserID, known.RoleUser); err != nil {
		slog.ErrorContext(ctx, "Failed to add grouping policy for user", "user", userM.UserID, "role", known.RoleUser, "error", err)
		return nil, errno.ErrAddRole.WithMessage(err.Error())
	}

	return &v1.CreateUserResponse{UserID: userM.UserID}, nil
}
