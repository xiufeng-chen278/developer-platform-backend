package services

import (
	"errors"
	"time"

	"github.com/xiufeng-chen278/developer-platform-backend/models"
	"gorm.io/gorm"
)

// APIKeyService 负责操作用户 API 密钥。
type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// Create 生成新的 API Key。
func (s *APIKeyService) Create(user *models.User, label string) (*models.APIKey, error) {
	if user == nil {
		return nil, errors.New("用户为空")
	}

	key := &models.APIKey{
		UserID:        user.ID,
		Label:         label,
		Key:           models.GenerateKeyWithLevel(user.Level),
		LevelSnapshot: user.Level,
	}

	if err := s.db.Create(key).Error; err != nil {
		return nil, err
	}
	return key, nil
}

// List 返回用户的全部密钥。
func (s *APIKeyService) List(userID uint) ([]models.APIKey, error) {
	var keys []models.APIKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// UpdateInput 表示更新请求。
type UpdateInput struct {
	Label      *string
	Regenerate bool
	MarkUsed   bool
	NewLevel   int
}

// Update 修改密钥（重命名/重置/打标使用时间）。
func (s *APIKeyService) Update(userID, keyID uint, input UpdateInput) (*models.APIKey, error) {
	var key models.APIKey
	if err := s.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}

	if input.Label != nil {
		updates["label"] = *input.Label
	}

	if input.Regenerate {
		updates["key"] = models.GenerateKeyWithLevel(input.NewLevel)
		updates["level_snapshot"] = input.NewLevel
	}

	if input.MarkUsed {
		now := time.Now()
		updates["last_used_at"] = &now
	}

	if len(updates) == 0 {
		return &key, nil
	}

	if err := s.db.Model(&key).Updates(updates).Error; err != nil {
		return nil, err
	}

	if err := s.db.First(&key, key.ID).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

// Delete 删除密钥。
func (s *APIKeyService) Delete(userID, keyID uint) error {
	res := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.APIKey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
