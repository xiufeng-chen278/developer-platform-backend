package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"github.com/xiufeng-chen278/developer-platform-backend/middlewares"
	"github.com/xiufeng-chen278/developer-platform-backend/models"
	"github.com/xiufeng-chen278/developer-platform-backend/routes"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := models.InitDB(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	if err := models.RunMigrations(db); err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	if _, err := models.InitRedis(cfg); err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}

	router := gin.Default()
	router.Use(middlewares.CORSMiddleware(cfg))
	routes.RegisterRoutes(router, cfg, db)

	server := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("服务启动，监听 %s", cfg.ServerAddr())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP 服务异常: %v", err)
	}
}
