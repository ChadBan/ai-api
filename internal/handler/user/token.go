package user

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TokenHandler Token 管理处理器
type TokenHandler struct {
	db *gorm.DB
}

// NewTokenHandler 创建 TokenHandler
func NewTokenHandler(db *gorm.DB) *TokenHandler {
	return &TokenHandler{db: db}
}

// CreateTokenRequest 创建 Token 请求
type CreateTokenRequest struct {
	Name           string             `json:"name"`
	RemainQuota    int                `json:"remain_quota"`
	UnlimitedQuota bool               `json:"unlimited_quota"`
	ExpiredTime    *time.Time         `json:"expired_time"` // -1 表示永不过期
	ModelLimit     []string           `json:"model_limit"`  // 允许的模型列表
	Ratio          float64            `json:"ratio"`        // 汇率倍率
	ModelRatio     map[string]float64 `json:"model_ratio"`
	Group          string             `json:"group"`
}

// CreateToken 创建 Token
func (h *TokenHandler) CreateToken(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 生成 Token Key
	tokenKey := "sk-" + uuid.New().String()

	// 处理过期时间
	var expiredTime *time.Time
	if req.ExpiredTime != nil {
		// 如果设置为 -1，表示永不过期
		if req.ExpiredTime.Unix() == -1 {
			expiredTime = nil
		} else {
			expiredTime = req.ExpiredTime
		}
	}

	// 序列化 model_limit
	modelLimitJSON := "[]"
	if len(req.ModelLimit) > 0 {
		data, err := json.Marshal(req.ModelLimit)
		if err != nil {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to serialize model_limit")
			return
		}
		modelLimitJSON = string(data)
	}

	// 设置默认 ratio
	ratio := req.Ratio
	if ratio <= 0 {
		ratio = 1.0 // 默认标准汇率
	}

	token := model.Token{
		UserID:         realUserID,
		Key:            tokenKey,
		Status:         1, // 启用
		Name:           req.Name,
		CreatedTime:    time.Now(),
		AccessedTime:   nil,
		ExpiredTime:    expiredTime,
		RemainQuota:    req.RemainQuota,
		UnlimitedQuota: req.UnlimitedQuota,
		UsedQuota:      0,
		ModelLimit:     modelLimitJSON,
		Ratio:          ratio,
		Group:          req.Group,
	}

	if req.Group == "" {
		token.Group = "default"
	}

	if err := h.db.Create(&token).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"id":  token.ID,
		"key": token.Key,
	})
}

// ListTokens 获取 Token 列表
func (h *TokenHandler) ListTokens(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询
	var tokens []model.Token
	var total int64

	query := h.db.Model(&model.Token{}).Where("user_id = ?", realUserID)

	// 状态筛选
	if status := c.Query("status"); status != "" {
		statusInt, _ := strconv.Atoi(status)
		query = query.Where("status = ?", statusInt)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	// 查询数据
	if err := query.Order("created_time DESC").Limit(pageSize).Offset(offset).Find(&tokens).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"items":     tokens,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetToken 获取 Token 详情
func (h *TokenHandler) GetToken(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid id")
		return
	}

	var token model.Token
	if err := h.db.Where("id = ? AND user_id = ?", id, realUserID).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	common.SuccessResponse(c, util.Success, token)
}

// UpdateToken 更新 Token
func (h *TokenHandler) UpdateToken(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid id")
		return
	}

	// 检查 Token 是否存在
	var token model.Token
	if err := h.db.Where("id = ? AND user_id = ?", id, realUserID).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["remain_quota"] = req.RemainQuota
	updates["unlimited_quota"] = req.UnlimitedQuota
	if req.ExpiredTime != nil {
		if req.ExpiredTime.Unix() == -1 {
			updates["expired_time"] = nil
		} else {
			updates["expired_time"] = req.ExpiredTime
		}
	}
	if req.Group != "" {
		updates["group"] = req.Group
	}

	if err := h.db.Model(&token).Updates(updates).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "Token updated successfully"})
}

// DeleteToken 删除 Token
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid id")
		return
	}

	// 软删除
	result := h.db.Where("id = ? AND user_id = ?", id, realUserID).Delete(&model.Token{})
	if result.Error != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "Token deleted successfully"})
}

// ToggleTokenStatus 切换 Token 状态（启用/禁用）
func (h *TokenHandler) ToggleTokenStatus(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid id")
		return
	}

	var token model.Token
	if err := h.db.Where("id = ? AND user_id = ?", id, realUserID).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	// 切换状态
	newStatus := 1
	if token.Status == 1 {
		newStatus = 0
	}

	if err := h.db.Model(&token).Update("status", newStatus).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"message": "Token status toggled successfully",
		"status":  newStatus,
	})
}

// GetTokenStats 获取 Token 统计信息
func (h *TokenHandler) GetTokenStats(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	// 统计总数
	var total, enabled, disabled int64
	h.db.Model(&model.Token{}).Where("user_id = ?", realUserID).Count(&total)
	h.db.Model(&model.Token{}).Where("user_id = ? AND status = ?", realUserID, 1).Count(&enabled)
	h.db.Model(&model.Token{}).Where("user_id = ? AND status = ?", realUserID, 0).Count(&disabled)

	// 计算总配额和已用配额
	type QuotaSum struct {
		TotalRemain int64
		TotalUsed   int64
	}
	var quotaSum QuotaSum
	h.db.Model(&model.Token{}).
		Where("user_id = ?", realUserID).
		Select("SUM(remain_quota) as total_remain, SUM(used_quota) as total_used").
		Scan(&quotaSum)

	common.SuccessResponse(c, util.Success, gin.H{
		"total_tokens":       total,
		"enabled_tokens":     enabled,
		"disabled_tokens":    disabled,
		"total_remain_quota": quotaSum.TotalRemain,
		"total_used_quota":   quotaSum.TotalUsed,
	})
}

// TopupTokenRequest Token 充值请求
type TopupTokenRequest struct {
	Quota  int    `json:"quota" binding:"required,min=1"`
	Reason string `json:"reason"`
}

// TopupToken Token 充值
func (h *TokenHandler) TopupToken(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid id")
		return
	}

	var req TopupTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 检查 Token 是否存在且属于当前用户
	var token model.Token
	if err := h.db.Where("id = ? AND user_id = ?", id, realUserID).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	// 增加配额
	if err := h.db.Model(&token).Updates(map[string]interface{}{
		"remain_quota": token.RemainQuota + req.Quota,
	}).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"message":      "Token topped up successfully",
		"quota_added":  req.Quota,
		"remain_quota": token.RemainQuota + req.Quota,
	})
}

// GetTokenUsageLogs 获取 Token 使用记录
func (h *TokenHandler) GetTokenUsageLogs(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	tokenIDStr := c.Param("id")
	tokenID, err := strconv.ParseInt(tokenIDStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid token id")
		return
	}

	// 获取 Token Key
	var token model.Token
	if err := h.db.Where("id = ? AND user_id = ?", tokenID, realUserID).First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.TokenNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询使用记录
	var logs []model.TokenUsageLog
	var total int64

	query := h.db.Model(&model.TokenUsageLog{}).Where("token_key = ? AND user_id = ?", token.Key, realUserID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	// 查询数据
	if err := query.Order("request_time DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListTokenUsageLogs 获取 Token 使用记录列表（支持多 Token 筛选）
func (h *TokenHandler) ListTokenUsageLogs(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 构建查询
	query := h.db.Model(&model.TokenUsageLog{}).Where("user_id = ?", realUserID)

	// Token ID 筛选
	if tokenIDStr := c.Query("token_id"); tokenIDStr != "" {
		tokenID, err := strconv.ParseInt(tokenIDStr, 10, 64)
		if err == nil {
			// 获取 Token Key
			var token model.Token
			if err := h.db.Where("id = ? AND user_id = ?", tokenID, realUserID).First(&token).Error; err == nil {
				query = query.Where("token_key = ?", token.Key)
			}
		}
	}

	// 模型筛选
	if model := c.Query("model"); model != "" {
		query = query.Where("model LIKE ?", "%"+model+"%")
	}

	// 成功/失败筛选
	if success := c.Query("success"); success != "" {
		if success == "true" {
			query = query.Where("success = ?", true)
		} else if success == "false" {
			query = query.Where("success = ?", false)
		}
	}

	// 时间范围筛选
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("request_time >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("request_time <= ?", endTime)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	// 查询数据
	var logs []model.TokenUsageLog
	if err := query.Order("request_time DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
