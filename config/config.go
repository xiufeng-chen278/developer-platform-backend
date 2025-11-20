package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 包含运行服务所需的关键配置。
type Config struct {
	AppEnv           string
	ServerHost       string
	ServerPort       string
	DatabaseURL      string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	GoogleClientID   string
	GoogleSecret     string
	GoogleRedirect   string
	AllowedOrigins   []string
	CookieDomain     string
	SessionStateName string
	JWTSecret        string
	JWTExpiresIn     time.Duration
	FrontendRedirect string
}

// LoadConfig 负责加载 .env 并组合最终配置。
func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRES_IN", "24h"))
	if err != nil || jwtExpiry <= 0 {
		return nil, fmt.Errorf("JWT_EXPIRES_IN 格式无效，示例：24h、15m")
	}

	cfg := &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		ServerHost:       getEnv("SERVER_HOST", "0.0.0.0"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "developer_platform"),
		DBSSLMode:        getEnv("DB_SSL_MODE", "disable"),
		GoogleClientID:   os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirect:   os.Getenv("GOOGLE_REDIRECT_URL"),
		AllowedOrigins:   splitAndTrim(os.Getenv("ALLOWED_ORIGINS")),
		CookieDomain:     os.Getenv("COOKIE_DOMAIN"),
		SessionStateName: getEnv("SESSION_STATE_NAME", "google_oauth_state"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTExpiresIn:     jwtExpiry,
		FrontendRedirect: os.Getenv("FRONTEND_REDIRECT_URL"),
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBName,
			cfg.DBSSLMode,
		)
	}

	return cfg, cfg.Validate()
}

// Validate 对关键字段做最小校验。
func (c *Config) Validate() error {
	required := map[string]string{
		"GOOGLE_CLIENT_ID":      c.GoogleClientID,
		"GOOGLE_CLIENT_SECRET":  c.GoogleSecret,
		"GOOGLE_REDIRECT_URL":   c.GoogleRedirect,
		"JWT_SECRET":            c.JWTSecret,
		"FRONTEND_REDIRECT_URL": c.FrontendRedirect,
	}

	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s 未配置", key)
		}
	}

	if c.JWTExpiresIn <= 0 {
		return fmt.Errorf("JWT_EXPIRES_IN 必须大于 0")
	}

	return nil
}

// ServerAddr 返回标准 host:port。
func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func splitAndTrim(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	var cleaned []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
