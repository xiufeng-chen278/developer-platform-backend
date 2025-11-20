package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiufeng-chen278/developer-platform-backend/services"
)

// CurrentUserContextKey 标记 Gin Context 存放鉴权信息的键。
const CurrentUserContextKey = "currentUser"

// JWTAuthMiddleware 验证 Authorization Bearer 令牌。
func JWTAuthMiddleware(tokenService *services.TokenService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Authorization Bearer token"})
			return
		}

		rawToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		if rawToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token 为空"})
			return
		}

		claims, err := tokenService.Parse(rawToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx.Set(CurrentUserContextKey, claims)
		ctx.Next()
	}
}
