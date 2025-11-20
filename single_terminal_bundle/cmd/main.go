package main

import (
	"go-backEnd/internal/config"
	"go-backEnd/internal/handlers"
	"go-backEnd/internal/models"
	"go-backEnd/internal/services"
	"go-backEnd/internal/utils"
	"log"
	"net/http"
)

func main() {
	config.Init()
	services.InitRedis()
	services.InitAuthService(services.RDB, config.AppConfig.AuthCode, config.AppConfig.CodeVersion)

	roomManager := models.NewRoomManager()

	excludePaths := []string{"/auth", "/auth-status", "/logout", "/", "/index.html", "/standard_time"}
	authMiddleware := utils.NewAuthMiddleware(excludePaths)

	http.Handle("/auth", utils.WithCORS(http.HandlerFunc(handlers.HandleAuth)))
	http.Handle("/auth-status", utils.WithCORS(http.HandlerFunc(handlers.HandleAuthStatus)))
	http.Handle("/logout", utils.WithCORS(http.HandlerFunc(handlers.HandleLogout)))
	http.Handle("/standard_time", utils.WithCORS(http.HandlerFunc(handlers.HandleStandardTime)))

	http.HandleFunc("/ws", handlers.ServeWS(roomManager))

	http.Handle("/audios", utils.WithCORS(authMiddleware.RequireAuth(handlers.ListAudio)))
	http.Handle("/delete-audio", utils.WithCORS(authMiddleware.RequireAuth(handlers.DeleteAudio)))
	http.Handle("/audio/", utils.WithCORS(authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/audio/", http.FileServer(http.Dir("audio"))).ServeHTTP(w, r)
	})))

	http.Handle("/system/translation-status", utils.WithCORS(http.HandlerFunc(handlers.GetSystemTranslationStatus)))
	http.Handle("/system/health", utils.WithCORS(http.HandlerFunc(handlers.GetSystemHealth)))
	http.Handle("/system/goroutines", utils.WithCORS(http.HandlerFunc(handlers.GetGoroutineStats)))
	http.HandleFunc("/system/room-status/", func(w http.ResponseWriter, r *http.Request) {
		utils.WithCORS(http.HandlerFunc(handlers.GetRoomStatus)).ServeHTTP(w, r)
	})
	http.HandleFunc("/system/close-translation/", func(w http.ResponseWriter, r *http.Request) {
		utils.WithCORS(http.HandlerFunc(handlers.ForceCloseTranslationConnection)).ServeHTTP(w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Printf("✅ 单终端服务已启动，监听 :%s", config.AppConfig.ServerPort)
	log.Fatal(http.ListenAndServe(":"+config.AppConfig.ServerPort, nil))
}
