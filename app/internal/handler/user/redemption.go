package user

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-api/app/internal/common"
	"ai-api/app/internal/util"

	"ai-api/app/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RedemptionHandler 兑换码处理器
type RedemptionHandler struct {
	db *gorm.DB
}

// NewRedemptionHandler 创建 RedemptionHandler
func NewRedemptionHandler(db *gorm.DB) *RedemptionHandler {
	return &RedemptionHandler{db: db}
}

// CreateRedemptionRequest 创建兑换码请求
type CreateRedemptionRequest struct {
	Quota        int    `json:"quota" binding:"required,min=1"` // 额度（积分）
	Count        int    `json:"count" binding:"min=1,max=100"`  // 生成数量
	Group        string `json:"group"`                          // 可用用户组
	Side         int    `json:"side"`                           // 0=所有人，1=邀请人，2=被邀请人
	IsPublic     bool   `json:"is_public"`                      // 是否公开
	CustomCredit int    `json:"custom_credit"`                  // 自定义积分
}

// CreateRedemptionResponse 创建兑换码响应
type CreateRedemptionResponse struct {
	Keys []string `json:"keys"`
}

// CreateRedemption 创建兑换码
func (h *RedemptionHandler) CreateRedemption(c *gin.Context) {
	var req CreateRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 获取当前用户 ID（管理员才能创建）
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid user id")
		return
	}

	// 批量生成兑换码
	var keys []string
	var redemptions []model.Redemption

	for i := 0; i < req.Count; i++ {
		key, err := generateRedemptionKey()
		if err != nil {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to generate key")
			return
		}

		keys = append(keys, key)
		redemptions = append(redemptions, model.Redemption{
			UserId:       userID,
			Key:          key,
			Status:       1, // 未使用
			Quota:        req.Quota,
			NominalQuota: req.Quota,
			Count:        0,
			Group:        req.Group,
			Side:         req.Side,
			IsPublic:     req.IsPublic,
			Verified:     false,
			CustomCredit: req.CustomCredit,
			CreatedAt:    time.Now(),
		})
	}

	// 批量插入
	if err := h.db.Create(&redemptions).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to save redemptions")
		return
	}

	common.SuccessResponse(c, util.Success, CreateRedemptionResponse{
		Keys: keys,
	})
}

// RedeemRequest 兑换请求
type RedeemRequest struct {
	Key string `json:"key" binding:"required"`
}

// Redeem 兑换兑换码
func (h *RedemptionHandler) Redeem(c *gin.Context) {
	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 获取当前用户 ID
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	realUserID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid user id")
		return
	}

	// 查找兑换码
	var redemption model.Redemption
	if err := h.db.Where("key = ?", req.Key).First(&redemption).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.NotFound, "invalid redemption code")
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	// 验证兑换码状态
	if redemption.Status != 1 {
		common.ErrorResponse(c, http.StatusBadRequest, util.TopupAlreadyUsed, "redemption code already used or disabled")
		return
	}

	// 检查用户组权限
	if redemption.Group != "" && redemption.Group != "default" {
		var user model.User
		if err := h.db.Where("id = ?", realUserID).First(&user).Error; err == nil {
			if user.Tier != redemption.Group {
				common.ErrorResponse(c, http.StatusForbidden, util.Forbidden, "insufficient permissions")
				return
			}
		}
	}

	// 开启事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 更新兑换码状态
	now := time.Now()
	if err := tx.Model(&redemption).Updates(map[string]interface{}{
		"status":        0, // 已使用
		"redeemed_time": now,
		"count":         gorm.Expr("count + 1"),
	}).Error; err != nil {
		tx.Rollback()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update redemption")
		return
	}

	// 增加用户余额
	var balance model.UserBalance
	result := tx.Where("user_id = ?", realUserID).First(&balance)

	credit := redemption.Quota
	if redemption.CustomCredit > 0 {
		credit = redemption.CustomCredit
	}

	if result.Error == gorm.ErrRecordNotFound {
		// 创建余额记录
		balance = model.UserBalance{
			UserID:       realUserID,
			Quota:        credit,
			TotalQuota:   credit,
			TotalBalance: decimal.NewFromInt(int64(credit)),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&balance).Error; err != nil {
			tx.Rollback()
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to create balance")
			return
		}
	} else if result.Error != nil {
		tx.Rollback()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	} else {
		// 更新余额
		if err := tx.Model(&balance).Updates(map[string]interface{}{
			"quota":       gorm.Expr("quota + ?", credit),
			"total_quota": gorm.Expr("total_quota + ?", credit),
			"updated_at":  now,
		}).Error; err != nil {
			tx.Rollback()
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update balance")
			return
		}
	}

	// 记录账单
	billing := model.Billing{
		UserID:    realUserID,
		Amount:    decimal.NewFromInt(int64(credit)),
		Quota:     credit,
		Type:      3, // 3=兑换
		CreatedAt: now,
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to record billing")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "transaction commit failed")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"message": "redemption successful",
		"credit":  credit,
	})
}

// ListRedemptions 获取兑换码列表（管理员）
func (h *RedemptionHandler) ListRedemptions(c *gin.Context) {
	page, _ := c.GetQuery("page")
	pageSize, _ := c.GetQuery("page_size")
	status, _ := c.GetQuery("status")

	offset := 0
	limit := 20

	if page != "" {
		p := 0
		fmt.Sscanf(page, "%d", &p)
		offset = (p - 1) * limit
	}

	if pageSize != "" {
		ps := 0
		fmt.Sscanf(pageSize, "%d", &ps)
		limit = ps
	}

	query := h.db.Model(&model.Redemption{})

	if status != "" {
		s := 0
		fmt.Sscanf(status, "%d", &s)
		query = query.Where("status = ?", s)
	}

	var total int64
	var redemptions []model.Redemption

	if err := query.Count(&total).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&redemptions).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data":      redemptions,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetRedemption 获取兑换码详情
func (h *RedemptionHandler) GetRedemption(c *gin.Context) {
	id := c.Param("id")

	var redemption model.Redemption
	if err := h.db.First(&redemption, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.NotFound, "redemption not found")
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": redemption,
	})
}

// DeleteRedemption 删除/禁用兑换码
func (h *RedemptionHandler) DeleteRedemption(c *gin.Context) {
	id := c.Param("id")

	// 软删除或禁用
	result := h.db.Model(&model.Redemption{}).Where("id = ?", id).Update("status", -1)

	if result.Error != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to delete redemption")
		return
	}

	if result.RowsAffected == 0 {
		common.ErrorResponse(c, http.StatusNotFound, util.NotFound, "redemption not found")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"message": "redemption deleted",
	})
}

// GetPublicRedemptions 获取公开兑换码
func (h *RedemptionHandler) GetPublicRedemptions(c *gin.Context) {
	var redemptions []model.Redemption
	if err := h.db.Where("is_public = ? AND status = ?", true, 1).Find(&redemptions).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": redemptions,
	})
}

// generateRedemptionKey 生成兑换码
func generateRedemptionKey() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	key := hex.EncodeToString(bytes)
	// 格式：RED-XXXX-XXXX-XXXX
	return fmt.Sprintf("RED-%s-%s-%s",
		key[0:4],
		key[4:8],
		key[8:12]), nil
}
