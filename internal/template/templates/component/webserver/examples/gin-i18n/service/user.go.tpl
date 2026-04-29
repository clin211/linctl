package service

import (
	"context"
	"fmt"

	"{{ .Project.Metadata.Module }}/examples/gin-i18n/repository"
	"{{ .Project.Metadata.Module }}/pkg/i18n"
)

// UserService 用户服务层
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(),
	}
}

// GetUser 获取用户信息
func (s *UserService) GetUser(ctx context.Context, userID int) (*repository.User, error) {
	// Service 层也可以使用 context 中的 i18n
	i18nInstance := i18n.FromContext(ctx)

	user, err := s.userRepo.FindUser(ctx, userID)
	if err != nil {
		// 在 Service 层处理错误翻译
		return nil, fmt.Errorf(i18nInstance.T("common.error") + ": " + err.Error())
	}

	return user, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, userName string) error {
	i18nInstance := i18n.FromContext(ctx)

	user := &repository.User{
		Name: userName,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return fmt.Errorf(i18nInstance.T("common.error") + ": " + err.Error())
	}

	return nil
}

// GetUserGreeting 获取用户问候信息
func (s *UserService) GetUserGreeting(ctx context.Context, userID int) (map[string]interface{}, error) {
	i18nInstance := i18n.FromContext(ctx)

	user, err := s.userRepo.FindUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 在 Service 层组合多个翻译
	welcomeMsg := s.userRepo.GetWelcomeMessage(ctx, user.Name)

	return map[string]interface{}{
		"greeting": i18nInstance.T("greeting.hello"),
		"welcome":  welcomeMsg,
		"success":  i18nInstance.T("common.success"),
		"user":     user,
	}, nil
}

// ValidateUserInput 验证用户输入
func (s *UserService) ValidateUserInput(ctx context.Context, input map[string]interface{}) error {
	i18nInstance := i18n.FromContext(ctx)

	// 验证逻辑
	if _, exists := input["name"]; !exists {
		return i18nInstance.E("user.invalid_credentials")
	}

	return nil
}
