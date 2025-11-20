package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"github.com/xiufeng-chen278/developer-platform-backend/controllers"
	"github.com/xiufeng-chen278/developer-platform-backend/middlewares"
	"github.com/xiufeng-chen278/developer-platform-backend/services"
	"gorm.io/gorm"
)

// RegisterRoutes 初始化所有 HTTP 路由。
func RegisterRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tokenService := services.NewTokenService(cfg)

	api := router.Group("/api")
	{
		authService := services.NewGoogleAuthService(cfg, db)
		apiKeyService := services.NewAPIKeyService(db)
		authController := controllers.NewAuthController(cfg, authService, tokenService)
		apiKeyController := controllers.NewAPIKeyController(authService, apiKeyService)

		auth := api.Group("/auth")
		{
			google := auth.Group("/google")
			{
				google.GET("/login", authController.GoogleLogin)
				google.GET("/callback", authController.GoogleCallback)
			}

			auth.GET("/me", middlewares.JWTAuthMiddleware(tokenService), authController.CurrentUser)
		}

		protected := api.Group("/protected")
		protected.Use(middlewares.JWTAuthMiddleware(tokenService))
		{
			protected.GET("/ping", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"message": "认证通过"})
			})
		}

		apiKeys := api.Group("/api-keys")
		apiKeys.Use(middlewares.JWTAuthMiddleware(tokenService))
		{
			apiKeys.GET("", apiKeyController.List)
			apiKeys.POST("", apiKeyController.Create)
			apiKeys.PUT("/:id", apiKeyController.Update)
			apiKeys.DELETE("/:id", apiKeyController.Delete)
		}
	}
}
