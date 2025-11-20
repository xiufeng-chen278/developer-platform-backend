package handlers

import (
	"encoding/json"
	"net/http"

	"go-backEnd/internal/services"
)

type AuthRequest struct {
	AuthCode string `json:"auth_code"`
}

type AuthResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	SessionID   string `json:"session_id,omitempty"`
	CodeVersion string `json:"code_version,omitempty"`
}

// createAuthCookie 创建认证Cookie，根据环境配置设置安全属性
func createAuthCookie(sessionID string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "auth_session",
		Value:    sessionID,
		Path:     "/",
		Domain:   "session.glot.world", // 只针对当前域名
		HttpOnly: true,
		Secure:   true, // SameSite=None时必须为true
		SameSite: http.SameSiteNoneMode,
		MaxAge:   maxAge,
	}
}

// HandleAuth 处理授权码验证
func HandleAuth(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		response := AuthResponse{
			Success: false,
			Message: "仅支持POST请求",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response := AuthResponse{
			Success: false,
			Message: "请求数据格式错误",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.AuthCode == "" {
		response := AuthResponse{
			Success: false,
			Message: "授权码不能为空",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证授权码
	sessionData, err := services.Auth.VerifyAuthCode(req.AuthCode)
	if err != nil {
		response := AuthResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 设置Cookie，根据环境配置设置安全属性
	cookie := createAuthCookie(sessionData.SessionID, 30*24*60*60) // 30天
	http.SetCookie(w, cookie)

	response := AuthResponse{
		Success:     true,
		Message:     "认证成功",
		SessionID:   sessionData.SessionID,
		CodeVersion: sessionData.CodeVersion,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleLogout 处理登出
func HandleLogout(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		response := AuthResponse{
			Success: false,
			Message: "仅支持POST请求",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 从Cookie中获取session ID
	cookie, err := r.Cookie("auth_session")
	if err == nil && cookie.Value != "" {
		// 删除Redis中的session
		services.Auth.DeleteSession(cookie.Value)
	}

	// 删除Cookie
	deleteCookie := createAuthCookie("", -1) // MaxAge: -1 立即过期
	http.SetCookie(w, deleteCookie)

	response := AuthResponse{
		Success: true,
		Message: "登出成功",
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleAuthStatus 检查认证状态
func HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	// 处理预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		response := AuthResponse{
			Success: false,
			Message: "仅支持GET请求",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 从Cookie中获取session ID
	cookie, err := r.Cookie("auth_session")
	if err != nil {
		response := AuthResponse{
			Success: false,
			Message: "未找到认证信息",
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证session
	sessionData, err := services.Auth.ValidateSession(cookie.Value)
	if err != nil {
		response := AuthResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := AuthResponse{
		Success:     true,
		Message:     "认证有效",
		SessionID:   sessionData.SessionID,
		CodeVersion: sessionData.CodeVersion,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}