package utils

import (
	"encoding/json"
	"net/http"

	"go-backEnd/internal/services"
)

type AuthMiddleware struct {
	excludePaths []string
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(excludePaths []string) *AuthMiddleware {
	return &AuthMiddleware{
		excludePaths: excludePaths,
	}
}

// isExcludedPath 检查路径是否在排除列表中
func (am *AuthMiddleware) isExcludedPath(path string) bool {
	for _, excludePath := range am.excludePaths {
		if path == excludePath {
			return true
		}
	}
	return false
}

// RequireAuth HTTP认证中间件
func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取请求的Origin
		origin := r.Header.Get("Origin")

		// 允许的域名列表
		allowedOrigins := []string{
			"https://glot.world",
			"https://www.glot.world",
			"https://test.glot.world",
			"http://localhost:3000",
			"http://localhost:8080",
		}

		// 检查Origin是否在允许列表中
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		// 设置CORS头
		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// 默认允许glot.world（向后兼容）
			w.Header().Set("Access-Control-Allow-Origin", "https://glot.world")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Content-Type", "application/json")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 检查是否为排除路径
		if am.isExcludedPath(r.URL.Path) {
			next(w, r)
			return
		}

		// 从Cookie中获取session ID
		cookie, err := r.Cookie("auth_session")
		if err != nil {
			errorResponse := ErrorResponse{
				Success: false,
				Message: "未找到认证信息",
				Code:    401,
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// 验证session
		_, err = services.Auth.ValidateSession(cookie.Value)
		if err != nil {
			errorResponse := ErrorResponse{
				Success: false,
				Message: err.Error(),
				Code:    401,
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		// 认证成功，继续处理请求
		next(w, r)
	}
}

// ValidateWebSocketAuth WebSocket认证验证函数
func ValidateWebSocketAuth(r *http.Request) error {
	// 从Cookie中获取session ID
	cookie, err := r.Cookie("auth_session")
	if err != nil {
		return err
	}

	// 验证session
	_, err = services.Auth.ValidateSession(cookie.Value)
	return err
}