package models

import (
	"time"

	"gorm.io/gorm"
)

// User 表示注册在平台中的 Google 账号。
type User struct {
	ID        uint   `gorm:"primaryKey"`
	GoogleID  string `gorm:"uniqueIndex"`
	Email     string `gorm:"uniqueIndex"`
	Name      string
	AvatarURL string
	Level     int `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

const DefaultUserLevel = 1

// GoogleUserInput 用于落库 Google 返回的 Profile。
type GoogleUserInput struct {
	GoogleID  string
	Email     string
	Name      string
	AvatarURL string
}

// UpsertGoogleUser 根据 GoogleID/Email 插入或更新记录。
func UpsertGoogleUser(db *gorm.DB, input GoogleUserInput) (*User, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}

	user := User{}
	err := db.Where("google_id = ? OR email = ?", input.GoogleID, input.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			user = User{
				GoogleID:  input.GoogleID,
				Email:     input.Email,
				Name:      input.Name,
				AvatarURL: input.AvatarURL,
				Level:     DefaultUserLevel,
			}
			if err := db.Create(&user).Error; err != nil {
				return nil, err
			}
			return &user, nil
		}
		return nil, err
	}

	updates := map[string]interface{}{
		"name":       input.Name,
		"avatar_url": input.AvatarURL,
	}
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
