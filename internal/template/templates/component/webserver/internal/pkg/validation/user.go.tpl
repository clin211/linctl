package validation

import (
	"context"
	"fmt"

	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	v1 "{{ .Project.Metadata.Module }}/pkg/api/{{ .Component.Name }}/v1"
	genericvalidation "{{ .Project.Metadata.Module }}/pkg/validation"
)

// ValidateUserRules 返回 user 资源所有字段的通用校验规则.
//
// 单元测试 / 派生 Validator 也可以直接复用本函数；新增字段时只需在这里加一项即可。
func (v *Validator) ValidateUserRules() genericvalidation.Rules {
	validatePassword := func() genericvalidation.ValidatorFunc {
		return func(value any) error {
			return isValidPassword(value.(string))
		}
	}

	return genericvalidation.Rules{
		"Password":    validatePassword(),
		"OldPassword": validatePassword(),
		"NewPassword": validatePassword(),
		"UserID": func(value any) error {
			if value.(string) == "" {
				return errno.ErrInvalidArgument.WithMessage("userID cannot be empty")
			}
			return nil
		},
		"Username": func(value any) error {
			if !isValidUsername(value.(string)) {
				return errno.ErrUsernameInvalid
			}
			return nil
		},
		"Nickname": func(value any) error {
			if len(value.(string)) >= 30 {
				return errno.ErrInvalidArgument.WithMessage("nickname must be less than 30 characters")
			}
			return nil
		},
		"Email": func(value any) error {
			return isValidEmail(value.(string))
		},
		"Phone": func(value any) error {
			return isValidPhone(value.(string))
		},
		"Limit": func(value any) error {
			if value.(int64) <= 0 {
				return errno.ErrInvalidArgument.WithMessage("limit must be greater than 0")
			}
			return nil
		},
		"Offset": func(value any) error {
			return nil
		},
	}
}

// ValidateLoginRequest 校验登录请求.
func (v *Validator) ValidateLoginRequest(ctx context.Context, rq *v1.LoginRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateChangePasswordRequest 校验修改密码请求.
//
// 同时校验当前登录用户与请求 userID 是否一致——只允许改自己的密码。
func (v *Validator) ValidateChangePasswordRequest(ctx context.Context, rq *v1.ChangePasswordRequest) error {
	if rq.GetUserID() != contextx.UserID(ctx) {
		return errno.ErrPermissionDenied.WithMessage(fmt.Sprintf("The logged-in user `%s` does not match request user `%s`", contextx.UserID(ctx), rq.GetUserID()))
	}
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateCreateUserRequest 校验创建用户请求.
func (v *Validator) ValidateCreateUserRequest(ctx context.Context, rq *v1.CreateUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateUpdateUserRequest 校验更新用户请求.
//
// 仅校验非空字段，并强制要求当前登录用户和请求 userID 匹配。
func (v *Validator) ValidateUpdateUserRequest(ctx context.Context, rq *v1.UpdateUserRequest) error {
	if rq.GetUserID() != contextx.UserID(ctx) {
		return errno.ErrPermissionDenied.WithMessage(fmt.Sprintf("The logged-in user `%s` does not match request user `%s`", contextx.UserID(ctx), rq.GetUserID()))
	}
	return genericvalidation.ValidateSelectedFields(rq, v.ValidateUserRules(), "UserID")
}

// ValidateDeleteUserRequest 校验删除用户请求.
func (v *Validator) ValidateDeleteUserRequest(ctx context.Context, rq *v1.DeleteUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateGetUserRequest 校验获取用户详情请求.
func (v *Validator) ValidateGetUserRequest(ctx context.Context, rq *v1.GetUserRequest) error {
	if rq.GetUserID() != contextx.UserID(ctx) {
		return errno.ErrPermissionDenied.WithMessage(fmt.Sprintf("The logged-in user `%s` does not match request user `%s`", contextx.UserID(ctx), rq.GetUserID()))
	}
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}

// ValidateListUserRequest 校验列出用户请求.
func (v *Validator) ValidateListUserRequest(ctx context.Context, rq *v1.ListUserRequest) error {
	return genericvalidation.ValidateAllFields(rq, v.ValidateUserRules())
}
