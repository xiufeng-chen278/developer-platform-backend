package utils

import "net/http"

func WithCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取请求的Origin
		origin := r.Header.Get("Origin")
		
		// 允许的域名列表
		allowedOrigins := []string{
			"https://glot.world",
			"https://www.glot.world",
			"https://test.glot.world",
			"http://localhost:3000",  // 本地开发
			"http://localhost:8080",  // 本地开发
		}
		
		// 检查Origin是否在允许列表中
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}
		
		// 如果Origin在允许列表中，设置对应的CORS头
		if isAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// 默认允许glot.world（向后兼容）
			w.Header().Set("Access-Control-Allow-Origin", "https://glot.world")
		}
		
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE, PUT")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
