package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiufeng-chen278/developer-platform-backend/middlewares"
	"github.com/xiufeng-chen278/developer-platform-backend/models"
	"github.com/xiufeng-chen278/developer-platform-backend/services"
	"gorm.io/gorm"
)

// APIKeyController 管理用户 API 密钥。
type APIKeyController struct {
	authService   *services.GoogleAuthService
	apiKeyService *services.APIKeyService
}

func NewAPIKeyController(authService *services.GoogleAuthService, apiKeyService *services.APIKeyService) *APIKeyController {
	return &APIKeyController{
		authService:   authService,
		apiKeyService: apiKeyService,
	}
}

// List 返回当前用户的所有密钥。
func (a *APIKeyController) List(ctx *gin.Context) {
	claims, ok := currentClaims(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户上下文异常"})
		return
	}

	keys, err := a.apiKeyService.List(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"keys": sanitizeKeys(keys)})
}

// Create 生成新的密钥。
func (a *APIKeyController) Create(ctx *gin.Context) {
	claims, ok := currentClaims(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户上下文异常"})
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}

	user, err := a.authService.GetUserByID(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	key, err := a.apiKeyService.Create(user, req.Label)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"key": sanitizeKey(*key)})
}

// Update 修改密钥属性。
func (a *APIKeyController) Update(ctx *gin.Context) {
	claims, ok := currentClaims(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户上下文异常"})
		return
	}

	id, err := parseUintParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id 非法"})
		return
	}

	var req struct {
		Label      *string `json:"label"`
		Regenerate bool    `json:"regenerate"`
		MarkUsed   bool    `json:"mark_used"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求体格式错误"})
		return
	}

	if req.Label == nil && !req.Regenerate && !req.MarkUsed {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "至少指定一个更新字段"})
		return
	}

	user, err := a.authService.GetUserByID(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	key, err := a.apiKeyService.Update(claims.UserID, id, services.UpdateInput{
		Label:      req.Label,
		Regenerate: req.Regenerate,
		MarkUsed:   req.MarkUsed,
		NewLevel:   user.Level,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"key": sanitizeKey(*key)})
}

// Delete 移除密钥。
func (a *APIKeyController) Delete(ctx *gin.Context) {
	claims, ok := currentClaims(ctx)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "用户上下文异常"})
		return
	}

	id, err := parseUintParam(ctx, "id")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id 非法"})
		return
	}

	if err := a.apiKeyService.Delete(claims.UserID, id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func sanitizeKeys(keys []models.APIKey) []gin.H {
	result := make([]gin.H, 0, len(keys))
	for _, key := range keys {
		result = append(result, sanitizeKey(key))
	}
	return result
}

func sanitizeKey(key models.APIKey) gin.H {
	return gin.H{
		"id":             key.ID,
		"label":          key.Label,
		"key":            key.Key,
		"level_snapshot": key.LevelSnapshot,
		"created_at":     key.CreatedAt,
		"last_used_at":   key.LastUsedAt,
	}
}

func parseUintParam(ctx *gin.Context, name string) (uint, error) {
	val := ctx.Param(name)
	id64, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id64), nil
}

func currentClaims(ctx *gin.Context) (*services.AuthClaims, bool) {
	value, exists := ctx.Get(middlewares.CurrentUserContextKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*services.AuthClaims)
	return claims, ok
}
