package user

import (
	"context"
	"log/slog"
	"sync"

	"golang.org/x/sync/errgroup"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/conversion"
	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/known"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// List 实现 UserBiz 接口中的 List 方法.
//
// 用 errgroup + sync.Map 并发组装 v1 视图，单次最多并发 known.MaxErrGroupConcurrency 条。
// 当业务上要在每条 user 上额外读取关联资源（例如 post 数量）时，把 IO 放到 eg.Go 内即可。
func (b *userBiz) List(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	whr := where.P(int(rq.GetOffset()), int(rq.GetLimit()))
	if contextx.Username(ctx) != known.AdminUsername {
		whr.T(ctx)
	}

	count, userList, err := b.store.User().List(ctx, whr)
	if err != nil {
		return nil, err
	}

	var m sync.Map
	eg, ctx := errgroup.WithContext(ctx)

	eg.SetLimit(known.MaxErrGroupConcurrency)

	for _, user := range userList {
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return nil
			default:
				userv1 := conversion.UserModelToUserV1(user)
				m.Store(user.ID, userv1)

				return nil
			}
		})
	}

	if err := eg.Wait(); err != nil {
		slog.ErrorContext(ctx, "Failed to wait all function calls returned", "error", err)
		return nil, err
	}

	users := make([]*v1.User, 0, len(userList))
	for _, item := range userList {
		user, _ := m.Load(item.ID)
		users = append(users, user.(*v1.User))
	}

	slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

	return &v1.ListUserResponse{TotalCount: count, Users: users}, nil
}
