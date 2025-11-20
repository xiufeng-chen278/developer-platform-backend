package handlers

import (
	"go-backEnd/internal/models"
	"go-backEnd/internal/services"
	"go-backEnd/internal/utils"
	websocketPkg "go-backEnd/pkg/websocket"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func ServeWS(manager *models.RoomManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 验证认证状态
		if err := utils.ValidateWebSocketAuth(r); err != nil {
			log.Printf("❌ [WS] 认证失败: %v", err)
			http.Error(w, "认证失败", http.StatusUnauthorized)
			return
		}

		roomID := strings.TrimPrefix(r.URL.Query().Get("room_id"), "/")
		fromLang := strings.TrimPrefix(r.URL.Query().Get("from_language"), "/")
		toLang := r.URL.Query().Get("to_language")

		if roomID == "" || fromLang == "" || toLang == "" {
			http.Error(w, "缺少参数（room_id, from_language, to_language）", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket 升级失败:", err)
			return
		}
		client := &models.Client{Conn: conn, Send: make(chan []byte, 256)}
		room := manager.GetRoom(roomID, fromLang, toLang)

		// 创建房间服务并启动运行协程
		roomService := services.NewRoomService(room)
		go roomService.Run()

		room.Register <- client
		go websocketPkg.WritePump(client)
		go websocketPkg.ReadPump(client, room)
	}
}
