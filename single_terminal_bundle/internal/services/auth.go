package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-backEnd/internal/config"

	"github.com/google/uuid"
)

type AuthService struct {
	rdbAuth     *redis.Client // 专门用于认证的Redis客户端（DB1）
	authCode    string
	codeVersion string
}

type SessionData struct {
	SessionID   string    `json:"session_id"`
	CodeVersion string    `json:"code_version"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

var Auth *AuthService

func InitAuthService(rdb *redis.Client, authCode, codeVersion string) {
	// 创建专门用于认证的Redis客户端（连接到DB1）
	rdbAuth := redis.NewClient(&redis.Options{
		Addr:     config.AppConfig.RedisAddr,
		Password: config.AppConfig.RedisPassword,
		DB:       1, // 使用DB1
	})

	Auth = &AuthService{
		rdbAuth:     rdbAuth,
		authCode:    authCode,
		codeVersion: codeVersion,
	}
	log.Printf("✅ 认证服务已初始化 - 版本: %s", codeVersion)
}

// VerifyAuthCode 验证授权码并生成session
func (a *AuthService) VerifyAuthCode(inputCode string) (*SessionData, error) {
	if inputCode != a.authCode {
		return nil, fmt.Errorf("授权码错误")
	}

	sessionID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour) // 30天过期

	sessionData := &SessionData{
		SessionID:   sessionID,
		CodeVersion: a.codeVersion,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	// 存储到Redis DB1
	ctx := context.Background()
	data, err := json.Marshal(sessionData)
	if err != nil {
		return nil, fmt.Errorf("序列化session数据失败: %v", err)
	}

	key := fmt.Sprintf("auth_session:%s", sessionID)
	err = a.rdbAuth.Set(ctx, key, data, 30*24*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("保存session失败: %v", err)
	}

	log.Printf("✅ 新会话已创建 - SessionID: %s, 过期时间: %s", sessionID, expiresAt.Format("2006-01-02 15:04:05"))
	return sessionData, nil
}

// ValidateSession 验证session
func (a *AuthService) ValidateSession(sessionID string) (*SessionData, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID为空")
	}

	ctx := context.Background()
	key := fmt.Sprintf("auth_session:%s", sessionID)

	data, err := a.rdbAuth.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("session不存在或已过期")
	}
	if err != nil {
		return nil, fmt.Errorf("获取session失败: %v", err)
	}

	var sessionData SessionData
	err = json.Unmarshal([]byte(data), &sessionData)
	if err != nil {
		return nil, fmt.Errorf("解析session数据失败: %v", err)
	}

	// 检查是否过期
	if time.Now().After(sessionData.ExpiresAt) {
		// 删除过期的session
		a.rdbAuth.Del(ctx, key)
		return nil, fmt.Errorf("session已过期")
	}

	// 检查版本是否匹配
	if sessionData.CodeVersion != a.codeVersion {
		return nil, fmt.Errorf("代码版本不匹配")
	}

	return &sessionData, nil
}

// DeleteSession 删除session
func (a *AuthService) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("auth_session:%s", sessionID)

	err := a.rdbAuth.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除session失败: %v", err)
	}

	log.Printf("✅ 会话已删除 - SessionID: %s", sessionID)
	return nil
}

// RefreshSession 刷新session过期时间
func (a *AuthService) RefreshSession(sessionID string) error {
	sessionData, err := a.ValidateSession(sessionID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	key := fmt.Sprintf("auth_session:%s", sessionID)

	// 更新过期时间
	sessionData.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("序列化session数据失败: %v", err)
	}

	err = a.rdbAuth.Set(ctx, key, data, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("刷新session失败: %v", err)
	}

	return nil
}
