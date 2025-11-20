// Package services 提供翻译服务监控功能
package services

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"
)

// GoroutineInfo 协程信息
type GoroutineInfo struct {
	Type      string    `json:"type"`       // "room_service", "translation_reader", "reverse_translation", "reconnect"
	Status    string    `json:"status"`     // "running", "stopped"
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// TranslationConnectionInfo 翻译连接信息
type TranslationConnectionInfo struct {
	Connected        bool      `json:"connected"`          // 是否连接
	LastMessageTime  time.Time `json:"last_message_time"`  // 最后消息时间
	ConnectionStart  time.Time `json:"connection_start"`   // 连接开始时间
	ConnectionDuration string   `json:"connection_duration"` // 连接持续时间
	FromLanguage     string    `json:"from_language"`      // 源语言
	ToLanguage       string    `json:"to_language"`        // 目标语言
	ReconnectCount   int       `json:"reconnect_count"`    // 重连次数
	MessageCount     int64     `json:"message_count"`      // 消息计数
	AudioPacketCount int64     `json:"audio_packet_count"` // 音频包计数
}

// RoomStatus 房间状态信息
type RoomStatus struct {
	RoomID               string                     `json:"room_id"`               // 房间ID
	RoomType             string                     `json:"room_type"`             // 房间类型: "single" | "dual_terminal"
	ClientCount          int                        `json:"client_count"`          // 客户端数量
	TranslationConnection *TranslationConnectionInfo `json:"translation_connection"` // 翻译连接信息
	ActiveGoroutines     []GoroutineInfo            `json:"goroutines"`            // 活跃协程
	CreatedAt            time.Time                  `json:"created_at"`            // 房间创建时间
	LastActivity         time.Time                  `json:"last_activity"`         // 最后活动时间
}

// SystemMonitorResponse 系统监控响应
type SystemMonitorResponse struct {
	Timestamp                   time.Time     `json:"timestamp"`                      // 时间戳
	TotalRooms                  int           `json:"total_rooms"`                    // 总房间数
	TotalTranslationConnections int           `json:"total_translation_connections"`  // 总翻译连接数
	TotalGoroutines             int           `json:"total_goroutines"`               // 总协程数
	SystemGoroutines            int           `json:"system_goroutines"`              // 系统协程数
	Rooms                       []*RoomStatus `json:"rooms"`                          // 房间列表
	ZombieConnections           int           `json:"zombie_connections"`             // 僵尸连接数(超过5分钟无活动)
	LongRunningGoroutines       int           `json:"long_running_goroutines"`        // 长时间运行的协程数
}

// TranslationMonitor 翻译服务监控器
type TranslationMonitor struct {
	mu    sync.RWMutex
	rooms map[string]*RoomStatus
}

// 全局监控器实例
var GlobalTranslationMonitor *TranslationMonitor
var monitorOnce sync.Once

// GetTranslationMonitor 获取全局监控器实例（单例模式）
func GetTranslationMonitor() *TranslationMonitor {
	monitorOnce.Do(func() {
		GlobalTranslationMonitor = &TranslationMonitor{
			rooms: make(map[string]*RoomStatus),
		}
	})
	return GlobalTranslationMonitor
}

// RegisterRoom 注册房间
func (tm *TranslationMonitor) RegisterRoom(roomID, roomType string, clientCount int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	tm.rooms[roomID] = &RoomStatus{
		RoomID:               roomID,
		RoomType:             roomType,
		ClientCount:          clientCount,
		TranslationConnection: &TranslationConnectionInfo{
			Connected:        false,
			ReconnectCount:   0,
			MessageCount:     0,
			AudioPacketCount: 0,
		},
		ActiveGoroutines: []GoroutineInfo{},
		CreatedAt:        now,
		LastActivity:     now,
	}
}

// UnregisterRoom 注销房间
func (tm *TranslationMonitor) UnregisterRoom(roomID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.rooms, roomID)
}

// UpdateClientCount 更新客户端数量
func (tm *TranslationMonitor) UpdateClientCount(roomID string, count int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		room.ClientCount = count
		room.LastActivity = time.Now()
	}
}

// UpdateTranslationConnection 更新翻译连接状态
func (tm *TranslationMonitor) UpdateTranslationConnection(roomID string, connected bool, fromLang, toLang string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		room.TranslationConnection.Connected = connected
		room.TranslationConnection.FromLanguage = fromLang
		room.TranslationConnection.ToLanguage = toLang
		room.LastActivity = time.Now()

		if connected {
			room.TranslationConnection.ConnectionStart = time.Now()
		}
	}
}

// RecordReconnect 记录重连
func (tm *TranslationMonitor) RecordReconnect(roomID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		room.TranslationConnection.ReconnectCount++
		room.LastActivity = time.Now()
	}
}

// RecordMessage 记录收到消息
func (tm *TranslationMonitor) RecordMessage(roomID string, isAudio bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		room.TranslationConnection.LastMessageTime = time.Now()
		room.LastActivity = time.Now()

		if isAudio {
			room.TranslationConnection.AudioPacketCount++
		} else {
			room.TranslationConnection.MessageCount++
		}
	}
}

// AddGoroutine 添加协程
func (tm *TranslationMonitor) AddGoroutine(roomID, goroutineType string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		goroutineInfo := GoroutineInfo{
			Type:      goroutineType,
			Status:    "running",
			CreatedAt: time.Now(),
		}
		room.ActiveGoroutines = append(room.ActiveGoroutines, goroutineInfo)
		room.LastActivity = time.Now()
	}
}

// RemoveGoroutine 移除协程（标记为stopped）
func (tm *TranslationMonitor) RemoveGoroutine(roomID, goroutineType string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		for i := range room.ActiveGoroutines {
			if room.ActiveGoroutines[i].Type == goroutineType && room.ActiveGoroutines[i].Status == "running" {
				room.ActiveGoroutines[i].Status = "stopped"
				room.LastActivity = time.Now()
				break
			}
		}
	}
}

// GetSystemStatus 获取系统状态
func (tm *TranslationMonitor) GetSystemStatus() *SystemMonitorResponse {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	now := time.Now()
	response := &SystemMonitorResponse{
		Timestamp:                   now,
		TotalRooms:                  len(tm.rooms),
		TotalTranslationConnections: 0,
		TotalGoroutines:             0,
		SystemGoroutines:            runtime.NumGoroutine(),
		Rooms:                       make([]*RoomStatus, 0, len(tm.rooms)),
		ZombieConnections:           0,
		LongRunningGoroutines:       0,
	}

	for _, room := range tm.rooms {
		// 计算连接持续时间
		if room.TranslationConnection.Connected {
			response.TotalTranslationConnections++
			duration := now.Sub(room.TranslationConnection.ConnectionStart)
			room.TranslationConnection.ConnectionDuration = duration.String()
		} else {
			room.TranslationConnection.ConnectionDuration = "0s"
		}

		// 统计活跃协程
		runningGoroutines := 0
		for _, g := range room.ActiveGoroutines {
			if g.Status == "running" {
				runningGoroutines++
				// 检查长时间运行的协程（超过30分钟）
				if now.Sub(g.CreatedAt) > 30*time.Minute {
					response.LongRunningGoroutines++
				}
			}
		}
		response.TotalGoroutines += runningGoroutines

		// 检查僵尸连接（超过5分钟无活动）
		if room.TranslationConnection.Connected && 
		   !room.TranslationConnection.LastMessageTime.IsZero() &&
		   now.Sub(room.TranslationConnection.LastMessageTime) > 5*time.Minute {
			response.ZombieConnections++
		}

		// 复制房间状态（避免并发修改）
		roomCopy := &RoomStatus{
			RoomID:      room.RoomID,
			RoomType:    room.RoomType,
			ClientCount: room.ClientCount,
			TranslationConnection: &TranslationConnectionInfo{
				Connected:          room.TranslationConnection.Connected,
				LastMessageTime:    room.TranslationConnection.LastMessageTime,
				ConnectionStart:    room.TranslationConnection.ConnectionStart,
				ConnectionDuration: room.TranslationConnection.ConnectionDuration,
				FromLanguage:       room.TranslationConnection.FromLanguage,
				ToLanguage:         room.TranslationConnection.ToLanguage,
				ReconnectCount:     room.TranslationConnection.ReconnectCount,
				MessageCount:       room.TranslationConnection.MessageCount,
				AudioPacketCount:   room.TranslationConnection.AudioPacketCount,
			},
			ActiveGoroutines: make([]GoroutineInfo, len(room.ActiveGoroutines)),
			CreatedAt:        room.CreatedAt,
			LastActivity:     room.LastActivity,
		}
		copy(roomCopy.ActiveGoroutines, room.ActiveGoroutines)
		response.Rooms = append(response.Rooms, roomCopy)
	}

	return response
}

// GetRoomStatus 获取特定房间状态
func (tm *TranslationMonitor) GetRoomStatus(roomID string) *RoomStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if room, exists := tm.rooms[roomID]; exists {
		// 复制房间状态以避免并发访问问题
		roomCopy := *room
		roomCopy.TranslationConnection = &TranslationConnectionInfo{}
		*roomCopy.TranslationConnection = *room.TranslationConnection
		roomCopy.ActiveGoroutines = make([]GoroutineInfo, len(room.ActiveGoroutines))
		copy(roomCopy.ActiveGoroutines, room.ActiveGoroutines)
		return &roomCopy
	}
	return nil
}

// ForceCloseRoom 强制关闭房间的翻译连接
func (tm *TranslationMonitor) ForceCloseRoom(roomID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if room, exists := tm.rooms[roomID]; exists {
		room.TranslationConnection.Connected = false
		room.LastActivity = time.Now()
		return true
	}
	return false
}

// ToJSON 转换为JSON字符串
func (tm *TranslationMonitor) ToJSON() (string, error) {
	status := tm.GetSystemStatus()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}