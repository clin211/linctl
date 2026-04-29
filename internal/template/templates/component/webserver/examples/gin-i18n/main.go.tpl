package main

import (
	"log"

	"github.com/gin-gonic/gin"
	miniblogI18n "{{ .Project.Metadata.Module }}/pkg/i18n"
	"{{ .Project.Metadata.Module }}/examples/gin-i18n/middleware"
	"{{ .Project.Metadata.Module }}/examples/gin-i18n/handlers"
	"golang.org/x/text/language"
)

func main() {
	// Create i18n instance
	i18nInstance := miniblogI18n.New(
		miniblogI18n.WithLanguage(language.English),
		miniblogI18n.WithFormat("json"),
		miniblogI18n.WithFile("./locales"),
	)

	// Create Gin router
	router := gin.Default()

	// Add i18n middleware
	router.Use(middleware.I18nMiddleware(i18nInstance))

	// Create handlers
	handler := handlers.New()
	userHandler := handlers.NewUserHandler()

	// Define routes
	api := router.Group("/api/v1")
	{
		// Basic greeting endpoint
		api.GET("/hello", handler.HelloHandler)

		// User-related endpoints
		api.GET("/user", handler.UserHandler)
		api.GET("/error", handler.ErrorHandler)

		// New user management endpoints (展示跨层 i18n)
		api.GET("/users/:id", userHandler.GetUser)
		api.POST("/users", userHandler.CreateUser)
		api.GET("/users/:id/greeting", userHandler.GetUserGreeting)

		// Demonstration endpoints
		api.GET("/items", handler.ItemCountHandler)
		api.GET("/language", handler.LanguageHandler)

		// Language management
		router.POST("/set-language/:lang", handler.SetLanguageHandler)
	}

	// Health check
	router.GET("/healthz", func(c *gin.Context) {
		i18nInstance := middleware.GetI18n(c)
		c.JSON(200, gin.H{
			"status": "ok",
			"message": i18nInstance.T("common.success"),
			"service": "gin-i18n-example",
		})
	})

	// Add CORS middleware (optional)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Start server
	log.Println("Starting server on :8080")
	log.Println("Try these endpoints:")
	log.Println("  GET /api/v1/hello")
	log.Println("  GET /api/v1/hello?lang=zh")
	log.Println("  GET /api/v1/hello?lang=ja")
	log.Println("  GET /api/v1/user")
	log.Println("  GET /api/v1/error?type=login_required")
	log.Println("  GET /api/v1/items?count=5")
	log.Println("  GET /api/v1/language")
	log.Println("  POST /set-language/zh")
	log.Println("")
	log.Println("New cross-layer i18n examples:")
	log.Println("  GET /api/v1/users/123")
	log.Println("  GET /api/v1/users/123?lang=zh")
	log.Println("  POST -H 'Content-Type: application/json' -d '{\"name\":\"John\"}' /api/v1/users")
	log.Println("  GET /api/v1/users/123/greeting")
	log.Println("  GET /healthz")

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
