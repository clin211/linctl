package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"{{ .Project.Metadata.Module }}/examples/gin-i18n/middleware"
	"{{ .Project.Metadata.Module }}/examples/gin-i18n/service"
)

// UserHandler 用户处理器，展示跨层 i18n 使用
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: service.NewUserService(),
	}
}

// GetUser 获取用户信息 - 展示从 Handler -> Service -> Repository 的 i18n 传递
func (h *UserHandler) GetUser(c *gin.Context) {
	// 从 URL 参数获取用户 ID
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
		})
		return
	}

	// 获取 Gin Context 中的 i18n 实例
	i18nInstance := middleware.GetI18n(c)

	// 通过 context 传递给 Service 层
	// 这里 c.Request.Context() 已经通过 i18n 中间件注入了 i18n 实例
	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": err.Error(),
			"current_lang": i18nInstance.Language().String(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18nInstance.T("user.login_success"),
		"data":    user,
		"current_lang": i18nInstance.Language().String(),
	})
}

// CreateUser 创建用户 - 展示输入验证和错误处理的 i18n
func (h *UserHandler) CreateUser(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		i18nInstance := middleware.GetI18n(c)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": i18nInstance.T("user.invalid_credentials"),
			"details": err.Error(),
		})
		return
	}

	// 验证输入
	if err := h.userService.ValidateUserInput(c.Request.Context(), map[string]interface{}{
		"name": input.Name,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// 创建用户
	if err := h.userService.CreateUser(c.Request.Context(), input.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	i18nInstance := middleware.GetI18n(c)
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": i18nInstance.T("user.login_success"),
		"current_lang": i18nInstance.Language().String(),
	})
}

// GetUserGreeting 获取用户问候信息 - 展示复合翻译消息
func (h *UserHandler) GetUserGreeting(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid user ID",
		})
		return
	}

	// Service 层会处理所有的翻译逻辑
	greeting, err := h.userService.GetUserGreeting(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	i18nInstance := middleware.GetI18n(c)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": i18nInstance.T("common.success"),
		"data":    greeting,
		"current_lang": i18nInstance.Language().String(),
	})
}
