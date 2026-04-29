package user

import (
	"context"
	"log/slog"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/conversion"
	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/known"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// ListWithBadPerformance 是 List 的串行实现，仅作为对照范例保留.
//
// 真实场景下请优先用 List —— 当每条 user 还要读取若干关联资源时，
// 串行版本会让响应时间随条数线性增长。
func (b *userBiz) ListWithBadPerformance(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	whr := where.P(int(rq.GetOffset()), int(rq.GetLimit()))
	if contextx.Username(ctx) != known.AdminUsername {
		whr.T(ctx)
	}

	count, userList, err := b.store.User().List(ctx, whr)
	if err != nil {
		return nil, err
	}

	users := make([]*v1.User, 0, len(userList))
	for _, user := range userList {
		userv1 := conversion.UserModelToUserV1(user)
		users = append(users, userv1)
	}

	slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

	return &v1.ListUserResponse{TotalCount: count, Users: users}, nil
}
