// Package handlers 提供系统监控HTTP接口
package handlers

import (
	"encoding/json"
	"fmt"
	"go-backEnd/internal/services"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status                string    `json:"status"`                   // "healthy" | "warning" | "critical"
	Timestamp             time.Time `json:"timestamp"`                // 检查时间戳
	ZombieConnections     int       `json:"zombie_connections"`       // 僵尸连接数
	LongRunningGoroutines int       `json:"long_running_goroutines"`  // 长时间运行协程数
	SystemGoroutines      int       `json:"system_goroutines"`        // 系统协程总数
	MemoryUsage           string    `json:"memory_usage"`             // 内存使用情况
	Uptime                string    `json:"uptime"`                   // 系统运行时间
}

// 服务启动时间
var serviceStartTime = time.Now()

// GetSystemTranslationStatus 获取系统翻译状态
func GetSystemTranslationStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// 注意: CORS头部将由WithCORS中间件处理，这里不需要重复设置

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	monitor := services.GetTranslationMonitor()
	status := monitor.GetSystemStatus()

	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		log.Printf("❌ [SYSTEM_MONITOR] JSON序列化失败: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
	
	log.Printf("✅ [SYSTEM_MONITOR] 系统状态查询成功 - 房间数: %d, 连接数: %d", 
		status.TotalRooms, status.TotalTranslationConnections)
}

// GetRoomStatus 获取特定房间状态
func GetRoomStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 从URL路径中提取roomID
	path := strings.TrimPrefix(r.URL.Path, "/system/room-status/")
	roomID := path

	if roomID == "" || roomID == "system/room-status/" {
		http.Error(w, "Missing roomId parameter", http.StatusBadRequest)
		return
	}

	monitor := services.GetTranslationMonitor()
	roomStatus := monitor.GetRoomStatus(roomID)

	if roomStatus == nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		log.Printf("⚠️ [SYSTEM_MONITOR] 房间未找到: %s", roomID)
		return
	}

	jsonData, err := json.MarshalIndent(roomStatus, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		log.Printf("❌ [SYSTEM_MONITOR] 房间状态JSON序列化失败: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
	
	log.Printf("✅ [SYSTEM_MONITOR] 房间状态查询成功: %s", roomID)
}

// ForceCloseTranslationConnection 强制关闭特定房间的翻译连接
func ForceCloseTranslationConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从URL路径中提取roomID
	path := strings.TrimPrefix(r.URL.Path, "/system/close-translation/")
	roomID := path

	if roomID == "" || roomID == "system/close-translation/" {
		http.Error(w, "Missing roomId parameter", http.StatusBadRequest)
		return
	}

	monitor := services.GetTranslationMonitor()
	success := monitor.ForceCloseRoom(roomID)

	response := map[string]interface{}{
		"success":   success,
		"room_id":   roomID,
		"timestamp": time.Now(),
	}

	if success {
		response["message"] = "翻译连接已强制关闭"
		log.Printf("✅ [SYSTEM_MONITOR] 强制关闭房间翻译连接: %s", roomID)
	} else {
		response["message"] = "房间未找到或连接已关闭"
		log.Printf("⚠️ [SYSTEM_MONITOR] 强制关闭失败，房间未找到: %s", roomID)
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// GetGoroutineStats 获取协程统计信息
func GetGoroutineStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"timestamp":           time.Now(),
		"total_goroutines":    runtime.NumGoroutine(),
		"memory_stats": map[string]interface{}{
			"alloc_mb":      bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb":        bToMb(m.Sys),
			"num_gc":        m.NumGC,
		},
		"uptime": time.Since(serviceStartTime).String(),
	}

	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// GetSystemHealth 获取系统健康状态
func GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	monitor := services.GetTranslationMonitor()
	status := monitor.GetSystemStatus()

	var healthStatus string
	if status.ZombieConnections > 5 || status.LongRunningGoroutines > 50 {
		healthStatus = "critical"
	} else if status.ZombieConnections > 2 || status.LongRunningGoroutines > 20 {
		healthStatus = "warning"
	} else {
		healthStatus = "healthy"
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := bToMb(m.Alloc)

	response := &HealthCheckResponse{
		Status:                healthStatus,
		Timestamp:             time.Now(),
		ZombieConnections:     status.ZombieConnections,
		LongRunningGoroutines: status.LongRunningGoroutines,
		SystemGoroutines:      status.SystemGoroutines,
		MemoryUsage:           formatMemoryUsage(memoryUsage),
		Uptime:                time.Since(serviceStartTime).String(),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	var statusCode int
	switch healthStatus {
	case "critical":
		statusCode = http.StatusServiceUnavailable // 503
	case "warning":
		statusCode = http.StatusOK // 200 but with warning
	default:
		statusCode = http.StatusOK // 200
	}

	w.WriteHeader(statusCode)
	w.Write(jsonData)
	
	log.Printf("✅ [SYSTEM_MONITOR] 健康检查: %s - 僵尸连接: %d, 长期协程: %d", 
		healthStatus, status.ZombieConnections, status.LongRunningGoroutines)
}

// bToMb 字节转MB
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// formatMemoryUsage 格式化内存使用量
func formatMemoryUsage(mb uint64) string {
	if mb < 1024 {
		return fmt.Sprintf("%dMB", mb)
	}
	return fmt.Sprintf("%.2fGB", float64(mb)/1024)
}