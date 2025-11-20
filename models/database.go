package models

import (
	"log"
	"time"

	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbInstance *gorm.DB

// InitDB 根据配置初始化数据库连接。
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, ErrMissingConfig
	}

	if dbInstance != nil {
		return dbInstance, nil
	}

	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		},
	)

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	dbInstance = db
	return dbInstance, nil
}

// GetDB 返回已经初始化的实例。
func GetDB() *gorm.DB {
	return dbInstance
}
