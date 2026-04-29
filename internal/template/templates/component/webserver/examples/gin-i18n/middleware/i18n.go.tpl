package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	miniblogI18n "{{ .Project.Metadata.Module }}/pkg/i18n"
	"golang.org/x/text/language"
)

// I18nMiddleware returns a Gin middleware for internationalization
func I18nMiddleware(i18nInstance *miniblogI18n.I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get language from different sources in order of preference:
		// 1. URL query parameter ?lang=en
		// 2. Cookie
		// 3. Accept-Language header
		// 4. Default language

		var lang language.Tag
		var found bool

		// 1. Check URL query parameter
		if langParam := c.Query("lang"); langParam != "" {
			if tag, err := language.Parse(langParam); err == nil {
				lang = tag
				found = true
			}
		}

		// 2. Check cookie
		if !found {
			if cookie, err := c.Cookie("lang"); err == nil {
				if tag, err := language.Parse(cookie); err == nil {
					lang = tag
					found = true
				}
			}
		}

		// 3. Check Accept-Language header
		if !found {
			if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
				// Parse Accept-Language header and get the first preferred language
				tags, _, err := language.ParseAcceptLanguage(acceptLang)
				if err == nil && len(tags) > 0 {
					lang = tags[0]
					found = true
				}
			}
		}

		// 4. Use default language from i18n instance
		if !found {
			lang = i18nInstance.Language()
		}

		// Create localizer with the detected language
		localizer := i18nInstance.Select(lang)

		// Store the localizer in the context
		c.Set("i18n", localizer)

		// Also store in context using i18n package for easy access
		c.Request = c.Request.WithContext(miniblogI18n.WithContext(c.Request.Context(), localizer))

		c.Next()
	}
}

// GetI18n returns the i18n instance from the Gin context
func GetI18n(c *gin.Context) *miniblogI18n.I18n {
	if i18nInstance, exists := c.Get("i18n"); exists {
		return i18nInstance.(*miniblogI18n.I18n)
	}

	// Fallback to context-based i18n
	return miniblogI18n.FromContext(c.Request.Context())
}

// T is a helper function to translate text using the i18n instance from context
func T(c *gin.Context, messageID string) string {
	return GetI18n(c).T(messageID)
}

// LocalizeMessage is a helper function to translate a message with template data
func LocalizeMessage(c *gin.Context, messageID string, templateData map[string]interface{}) string {
	i18nInstance := GetI18n(c)

	// Create a message with template data
	message := &i18n.Message{
		ID:    messageID,
		Other: messageID, // Fallback message
	}

	localizedStr := i18nInstance.LocalizeT(message)

	return localizedStr
}

// ToContext 是一个辅助函数，用于在非 Gin 上下文中传递 i18n
// 这个函数可以在 Service 层或其他层中使用
func ToContext(ctx context.Context, i18nInstance *miniblogI18n.I18n) context.Context {
	return miniblogI18n.WithContext(ctx, i18nInstance)
}
