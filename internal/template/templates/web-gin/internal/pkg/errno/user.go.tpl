package errno

import (
	"{{ .Project.Metadata.Module }}/pkg/errorsx"
)

var (
	// 用户相关错误
	ErrUsernameInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidUsername,
		"User.UsernameInvalid",
		"Invalid username: Username must consist of letters, digits, and underscores only, and its length must be between 3 and 20 characters.",
	)

	ErrPasswordInvalid = errorsx.NewBizError(
		errorsx.CodeUserInvalidPassword,
		"User.PasswordInvalid",
		"Password is incorrect.",
	)

	ErrUserAlreadyExists = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.AlreadyExists",
		"User already exists.",
	)

	ErrUserNotFound = errorsx.NewBizError(
		errorsx.CodeUserNotFound,
		"User.NotFound",
		"User not found.",
	)

	// 新增更多用户相关错误
	ErrUserDisabled = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Disabled",
		"User account has been disabled.",
	)

	ErrUserLocked = errorsx.NewBizError(
		errorsx.CodeUserPermissionDenied,
		"User.Locked",
		"User account has been locked due to multiple failed login attempts.",
	)

	ErrUserPasswordExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.PasswordExpired",
		"User password has expired, please reset your password.",
	)

	ErrUserInsufficientBalance = errorsx.NewBizError(
		errorsx.CodeUserInsufficientBalance,
		"User.InsufficientBalance",
		"User balance is insufficient for this operation.",
	)

	ErrUserEmailAlreadyVerified = errorsx.NewBizError(
		errorsx.CodeUserAlreadyExists,
		"User.EmailAlreadyVerified",
		"User email has already been verified.",
	)

	ErrUserEmailVerificationExpired = errorsx.NewBizError(
		errorsx.CodeUserInvalidCredentials,
		"User.EmailVerificationExpired",
		"Email verification token has expired.",
	)
)
