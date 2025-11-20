package services

import (
	"go-backEnd/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT() (string, error) {
	claims := jwt.MapClaims{
		"sub": "e",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecretKey))
}
