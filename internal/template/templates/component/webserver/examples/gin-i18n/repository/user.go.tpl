package repository

import (
	"context"
	"fmt"

	"{{ .Project.Metadata.Module }}/pkg/i18n"
)

// UserRepository 用户仓储示例
type UserRepository struct {
	i18n *i18n.I18n
}

// NewUserRepository 创建用户仓储
func NewUserRepository() *UserRepository {
	return &UserRepository{
		i18n: i18n.New(),
	}
}

// FindUser 查找用户 - 这里使用 context 而不是 gin.Context
func (r *UserRepository) FindUser(ctx context.Context, userID int) (*User, error) {
	// 方法1: 从 context 中获取 i18n 实例 (推荐)
	i18nInstance := i18n.FromContext(ctx)

	// 方法2: 如果 context 中没有，使用实例的默认 i18n (回退方案)
	if i18nInstance == nil {
		i18nInstance = r.i18n
	}

	// 模拟数据库查询
	if userID <= 0 {
		// 使用 i18n 翻译错误消息
		return nil, i18nInstance.E("user.user_not_found")
	}

	// 模拟找到用户
	user := &User{
		ID:   userID,
		Name: "John Doe",
	}

	fmt.Printf("Repository: %s\n", i18nInstance.T("user.login_success"))
	return user, nil
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(ctx context.Context, user *User) error {
	i18nInstance := i18n.FromContext(ctx)
	if i18nInstance == nil {
		i18nInstance = r.i18n
	}

	if user.Name == "" {
		return fmt.Errorf(i18nInstance.T("user.invalid_credentials"))
	}

	// 模拟创建用户逻辑
	fmt.Printf("Creating user: %s\n", user.Name)
	fmt.Printf("Repository: %s\n", i18nInstance.T("common.success"))
	return nil
}

// GetWelcomeMessage 获取欢迎消息 - 展示模板翻译
func (r *UserRepository) GetWelcomeMessage(ctx context.Context, userName string) string {
	i18nInstance := i18n.FromContext(ctx)
	if i18nInstance == nil {
		i18nInstance = r.i18n
	}

	// 这里需要更复杂的模板处理
	// 目前简化为基本翻译
	welcomeMsg := i18nInstance.T("messages.welcome_user")
	return welcomeMsg
}

// User 用户模型
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
