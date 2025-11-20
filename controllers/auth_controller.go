package controllers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xiufeng-chen278/developer-platform-backend/config"
	"github.com/xiufeng-chen278/developer-platform-backend/middlewares"
	"github.com/xiufeng-chen278/developer-platform-backend/services"
)

// AuthController 处理 Google 登录流程。
type AuthController struct {
	cfg          *config.Config
	service      *services.GoogleAuthService
	tokenService *services.TokenService
}

func NewAuthController(cfg *config.Config, service *services.GoogleAuthService, tokenService *services.TokenService) *AuthController {
	return &AuthController{
		cfg:          cfg,
		service:      service,
		tokenService: tokenService,
	}
}

// GoogleLogin 生成 state 并跳转到 Google OAuth。
func (a *AuthController) GoogleLogin(ctx *gin.Context) {
	state := uuid.NewString()
	maxAge := 300
	secure := a.cfg.AppEnv == "production"
	ctx.SetCookie(
		a.cfg.SessionStateName,
		state,
		maxAge,
		"/",
		a.cfg.CookieDomain,
		secure,
		true,
	)

	loginURL := a.service.AuthCodeURL(state)
	ctx.Redirect(http.StatusTemporaryRedirect, loginURL)
}

// GoogleCallback 处理重定向并返回用户资料。
func (a *AuthController) GoogleCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")
	if code == "" || state == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少 code 或 state"})
		return
	}

	cookieState, err := ctx.Cookie(a.cfg.SessionStateName)
	if err != nil || cookieState != state {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "state 不匹配"})
		return
	}

	user, err := a.service.HandleCallback(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token, err := a.tokenService.Generate(user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "生成 token 失败"})
		return
	}

	// 删除一次性 state，避免重复使用。
	ctx.SetCookie(
		a.cfg.SessionStateName,
		"",
		-1,
		"/",
		a.cfg.CookieDomain,
		a.cfg.AppEnv == "production",
		true,
	)

	payload := gin.H{
		"token": gin.H{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int(a.tokenService.ExpiresIn().Seconds()),
		},
		"user": gin.H{
			"email":      user.Email,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
			"level":      user.Level,
		},
		"synced_at": time.Now().UTC(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "序列化响应失败"})
		return
	}

	if a.cfg.FrontendRedirect == "" {
		ctx.JSON(http.StatusOK, payload)
		return
	}

	target, err := url.Parse(a.cfg.FrontendRedirect)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "FRONTEND_REDIRECT_URL 无效"})
		return
	}

	q := target.Query()
	q.Set("payload", base64.URLEncoding.EncodeToString(data))
	target.RawQuery = q.Encode()
	ctx.Redirect(http.StatusTemporaryRedirect, target.String())
}

// CurrentUser 返回当前登录用户。
func (a *AuthController) CurrentUser(ctx *gin.Context) {
	value, exists := ctx.Get(middlewares.CurrentUserContextKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	claims, ok := value.(*services.AuthClaims)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "上下文中的用户信息无效"})
		return
	}

	user, err := a.service.GetUserByID(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在或已删除"})
		return
	}

	var expiresAt interface{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"email":      user.Email,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
			"level":      user.Level,
		},
		"token_expires_at": expiresAt,
	})
}
