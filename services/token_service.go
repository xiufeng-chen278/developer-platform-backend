package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"github.com/xiufeng-chen278/developer-platform-backend/models"
)

// AuthClaims 描述写入 JWT 的信息。
type AuthClaims struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	jwt.RegisteredClaims
}

// TokenService 负责签发与解析 JWT。
type TokenService struct {
	secret     []byte
	expiresIn  time.Duration
	issuerName string
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{
		secret:     []byte(cfg.JWTSecret),
		expiresIn:  cfg.JWTExpiresIn,
		issuerName: "developer-platform-backend",
	}
}

// Generate 为指定用户生成访问令牌。
func (t *TokenService) Generate(user *models.User) (string, error) {
	if user == nil {
		return "", errors.New("用户信息为空")
	}

	now := time.Now()
	claims := AuthClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.GoogleID,
			Issuer:    t.issuerName,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.expiresIn)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

// Parse 验证并解析令牌。
func (t *TokenService) Parse(tokenString string) (*AuthClaims, error) {
	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("签名算法不被支持")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("无效的 token")
	}

	return claims, nil
}

// ExpiresIn 返回 token 有效期，用于前端提示。
func (t *TokenService) ExpiresIn() time.Duration {
	return t.expiresIn
}
