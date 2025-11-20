package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// StandardTimeResponse 标准时间响应结构
type StandardTimeResponse struct {
	Timestamp int64 `json:"timestamp"`
}

// HandleStandardTime 处理标准时间请求，返回UTC毫秒时间戳
func HandleStandardTime(w http.ResponseWriter, r *http.Request) {
	// 只允许GET请求
	if r.Method != http.MethodGet {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 获取当前UTC时间的毫秒时间戳
	now := time.Now().UTC()
	timestamp := now.UnixNano() / int64(time.Millisecond)

	// 构造响应
	response := StandardTimeResponse{
		Timestamp: timestamp,
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 返回JSON响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
		return
	}
}