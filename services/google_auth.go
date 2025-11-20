package services

import (
	"context"
	"fmt"

	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"github.com/xiufeng-chen278/developer-platform-backend/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

// GoogleAuthService 封装 OAuth 逻辑与用户落库。
type GoogleAuthService struct {
	cfg         *config.Config
	db          *gorm.DB
	oauthConfig *oauth2.Config
}

func NewGoogleAuthService(cfg *config.Config, db *gorm.DB) *GoogleAuthService {
	return &GoogleAuthService{
		cfg: cfg,
		db:  db,
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleSecret,
			RedirectURL:  cfg.GoogleRedirect,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

// AuthCodeURL 返回带 state 的授权链接。
func (g *GoogleAuthService) AuthCodeURL(state string) string {
	return g.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// HandleCallback 交换 code 并同步用户。
func (g *GoogleAuthService) HandleCallback(ctx context.Context, code string) (*models.User, error) {
	token, err := g.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange token 失败: %w", err)
	}

	httpClient := g.oauthConfig.Client(ctx, token)
	oauthService, err := oauth2api.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("创建 oauth2 service 失败: %w", err)
	}

	info, err := oauthService.Userinfo.Get().Do()
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	input := models.GoogleUserInput{
		GoogleID:  info.Id,
		Email:     info.Email,
		Name:      info.Name,
		AvatarURL: info.Picture,
	}

	return models.UpsertGoogleUser(g.db, input)
}

// GetUserByID 返回最新的用户数据。
func (g *GoogleAuthService) GetUserByID(id uint) (*models.User, error) {
	if g.db == nil {
		return nil, gorm.ErrInvalidDB
	}

	var user models.User
	if err := g.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
