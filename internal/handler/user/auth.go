package user

import (
	"net/http"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/security"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthHandler 认证 Handler
type AuthHandler struct {
	db *gorm.DB
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		db: db,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 检查邮箱是否已存在
	var existingUser model.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		common.ErrorResponse(c, http.StatusConflict, util.DuplicateEntry, "email already registered")
		return
	}

	// 密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to hash password")
		return
	}

	// 创建用户
	user := model.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Status:       1,
		Role:         "user",
		Tier:         "free",
	}

	if err := h.db.Create(&user).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to create user")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"user_id": user.ID,
	})
}

// GetToken 获取 Token（兼容接口）
func (h *AuthHandler) GetToken(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 查找用户
	var user model.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.InvalidCredentials)
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.InvalidCredentials)
		return
	}

	// 生成 JWT Token
	token, err := security.EncodeJwtToken(int64(user.ID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to generate token")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"token": token,
	})
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 查找用户
	var user model.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.InvalidCredentials)
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.InvalidCredentials)
		return
	}

	// 生成 JWT Token
	token, err := security.EncodeJwtToken(int64(user.ID))
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to generate token")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"tier":  user.Tier,
		},
	})
}
