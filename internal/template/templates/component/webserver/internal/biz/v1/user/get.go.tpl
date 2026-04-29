package user

import (
	"context"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/pkg/conversion"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	"{{ .Project.Metadata.Module }}/pkg/store/where"
)

// Get 实现 UserBiz 接口中的 Get 方法.
func (b *userBiz) Get(ctx context.Context, rq *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	userM, err := b.store.User().Get(ctx, where.T(ctx))
	if err != nil {
		return nil, err
	}

	return &v1.GetUserResponse{User: conversion.UserModelToUserV1(userM)}, nil
}
