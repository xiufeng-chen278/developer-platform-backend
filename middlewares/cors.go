package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
)

// CORSMiddleware 处理跨域请求。
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for _, origin := range cfg.AllowedOrigins {
		allowed[origin] = struct{}{}
	}
	allowAll := len(allowed) == 0

	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		if origin != "" && (allowAll || isOriginAllowed(origin, allowed)) {
			headers := ctx.Writer.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Access-Control-Allow-Credentials", "true")
			headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			headers.Set("Access-Control-Expose-Headers", "Content-Length")
			headers.Add("Vary", "Origin")
		}

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

func isOriginAllowed(origin string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	origin = strings.TrimSpace(origin)
	_, ok := allowed[origin]
	return ok
}
