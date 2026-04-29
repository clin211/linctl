package handlers

import (
	"fmt"
	"net/http"

	"{{ .Project.Metadata.Module }}/examples/gin-i18n/middleware"
	"github.com/gin-gonic/gin"
)

// Handler contains the example handlers
type Handler struct{}

// New creates a new handler instance
func New() *Handler {
	return &Handler{}
}

// HelloHandler returns a greeting message
func (h *Handler) HelloHandler(c *gin.Context) {
	// Get the i18n instance from context
	i18nInstance := middleware.GetI18n(c)

	// Translate different messages
	response := map[string]interface{}{
		"hello":        i18nInstance.T("greeting.hello"),
		"welcome":      i18nInstance.T("greeting.welcome"),
		"success":      i18nInstance.T("common.success"),
		"current_lang": i18nInstance.Language().String(),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Hello handler executed",
		"data":    response,
	})
}

// UserHandler demonstrates user-related translations
func (h *Handler) UserHandler(c *gin.Context) {
	i18nInstance := middleware.GetI18n(c)

	// Simulate different user scenarios
	username := c.Query("username")
	if username == "" {
		username = "Guest"
	}

	// Different user-related messages
	response := map[string]string{
		"welcome_message":     i18nInstance.T("messages.welcome_user"), // This would need template data
		"login_required":      i18nInstance.T("user.login_required"),
		"login_success":       i18nInstance.T("user.login_success"),
		"invalid_credentials": i18nInstance.T("user.invalid_credentials"),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "User handler executed",
		"data":     response,
		"username": username,
	})
}

// ErrorHandler demonstrates error message translations
func (h *Handler) ErrorHandler(c *gin.Context) {
	i18nInstance := middleware.GetI18n(c)

	// Simulate different error scenarios
	errorType := c.Query("type")
	var errorMessage string

	switch errorType {
	case "login_required":
		errorMessage = i18nInstance.T("user.login_required")
	case "invalid_credentials":
		errorMessage = i18nInstance.T("user.invalid_credentials")
	case "user_not_found":
		errorMessage = i18nInstance.T("user.user_not_found")
	default:
		errorMessage = i18nInstance.T("common.error")
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"status":       "error",
		"message":      errorMessage,
		"current_lang": i18nInstance.Language().String(),
	})
}

// ItemCountHandler demonstrates pluralization
func (h *Handler) ItemCountHandler(c *gin.Context) {
	i18nInstance := middleware.GetI18n(c)

	// Get count from query parameter, default to 0
	var count int
	if countStr := c.Query("count"); countStr != "" {
		if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
			count = 0
		}
	}

	var countMessage string
	switch {
	case count == 0:
		countMessage = i18nInstance.T("messages.item_count.zero")
	case count == 1:
		countMessage = i18nInstance.T("messages.item_count.one")
	default:
		// For pluralization, you might need more advanced templating
		countMessage = i18nInstance.T("messages.item_count.other")
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"count":        count,
		"message":      countMessage,
		"current_lang": i18nInstance.Language().String(),
	})
}

// SetLanguageHandler sets the language preference in a cookie
func (h *Handler) SetLanguageHandler(c *gin.Context) {
	lang := c.Param("lang")
	if lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Language parameter is required",
		})
		return
	}

	// Set language cookie with proper settings
	c.SetCookie("lang", lang, 3600*24*30, "/", "", false, false)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Language preference saved",
		"lang":    lang,
	})
}

// LanguageHandler shows current language and available languages
func (h *Handler) LanguageHandler(c *gin.Context) {
	i18nInstance := middleware.GetI18n(c)

	availableLanguages := map[string]string{
		"en": "English",
		"zh": "中文",
		"ja": "日本語",
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              "success",
		"current_language":    i18nInstance.Language().String(),
		"available_languages": availableLanguages,
		"usage_tips": map[string]string{
			"query_param": "Use ?lang=en in URL",
			"cookie":      "Set cookie via POST /set-language/{lang}",
			"header":      "Use Accept-Language header",
		},
	})
}
