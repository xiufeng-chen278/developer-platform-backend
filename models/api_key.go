package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKey 表示用户申请的密钥。
type APIKey struct {
	ID            uint       `gorm:"primaryKey"`
	UserID        uint       `gorm:"index"`
	User          User       `gorm:"constraint:OnDelete:CASCADE"`
	Key           string     `gorm:"uniqueIndex"`
	Label         string     `gorm:"size:128"`
	LevelSnapshot int        `gorm:"not null"`
	LastUsedAt    *time.Time `gorm:"column:last_used_at"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GenerateKeyWithLevel 组合当前等级生成密钥。
func GenerateKeyWithLevel(level int) string {
	random := uuid.NewString()
	return fmt.Sprintf("KF-%d-%s", level, random)
}

// DefaultAPIKeyLabel 返回空 label 的默认值。
const DefaultAPIKeyLabel = "default"

// BeforeCreate 确保默认值存在。
func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.Label == "" {
		a.Label = DefaultAPIKeyLabel
	}
	return nil
}
