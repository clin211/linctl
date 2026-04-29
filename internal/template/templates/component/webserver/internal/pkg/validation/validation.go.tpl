package validation

import (
	"regexp"

	"github.com/google/wire"

	"{{ .Project.Metadata.Module }}/internal/{{ .Component.Name }}/store"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
)

// Validator 是请求参数自定义校验器.
//
// 同包中的 ValidateXxxRequest 方法被 pkg/validation 通过反射注册（约定：
// 方法名必须形如 "Validate" + 请求结构体名）；复杂校验需要查库时可调用 store。
type Validator struct {
	// store 用于校验时查询数据库；如不需要可不调用。
	store store.IStore
}

// 全局预编译的正则表达式，避免每次校验都重新编译。
var (
	lengthRegex = regexp.MustCompile(`^.{3,20}$`)                                        // 长度在 3~20 字符之间
	validRegex  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)                                  // 仅允许字母 / 数字 / 下划线
	letterRegex = regexp.MustCompile(`[A-Za-z]`)                                         // 至少包含一个字母
	numberRegex = regexp.MustCompile(`\d`)                                               // 至少包含一个数字
	emailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`) // 邮箱格式
	phoneRegex  = regexp.MustCompile(`^1[3-9]\d{9}$`)                                    // 中国大陆手机号
)

// ProviderSet 声明 Wire DI 规则。
//
// 同时把 *Validator 绑定到 any 类型，便于 pkg/validation 在 wire 中通过反射注入。
var ProviderSet = wire.NewSet(New, wire.Bind(new(any), new(*Validator)))

// New 构造一个 *Validator。
func New(store store.IStore) *Validator {
	return &Validator{store: store}
}

// isValidUsername 校验 username 是否符合长度与字符规则。
func isValidUsername(username string) bool {
	if !lengthRegex.MatchString(username) {
		return false
	}
	if !validRegex.MatchString(username) {
		return false
	}
	return true
}

// isValidPassword 校验 password 是否符合复杂度规则。
func isValidPassword(password string) error {
	switch {
	case password == "":
		return errno.ErrInvalidArgument.WithMessage("password cannot be empty")
	case len(password) < 6:
		return errno.ErrInvalidArgument.WithMessage("password must be at least 6 characters long")
	case !letterRegex.MatchString(password):
		return errno.ErrInvalidArgument.WithMessage("password must contain at least one letter")
	case !numberRegex.MatchString(password):
		return errno.ErrInvalidArgument.WithMessage("password must contain at least one number")
	}
	return nil
}

// isValidEmail 校验 email 是否合法。
func isValidEmail(email string) error {
	if email == "" {
		return errno.ErrInvalidArgument.WithMessage("email cannot be empty")
	}
	if !emailRegex.MatchString(email) {
		return errno.ErrInvalidArgument.WithMessage("invalid email format")
	}
	return nil
}

// isValidPhone 校验 phone 是否合法。
func isValidPhone(phone string) error {
	if phone == "" {
		return errno.ErrInvalidArgument.WithMessage("phone cannot be empty")
	}
	if !phoneRegex.MatchString(phone) {
		return errno.ErrInvalidArgument.WithMessage("invalid phone format")
	}
	return nil
}
