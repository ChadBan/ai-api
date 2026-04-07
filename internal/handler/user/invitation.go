package user

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// InvitationHandler 邀请系统处理器
type InvitationHandler struct {
	db *gorm.DB
}

// NewInvitationHandler 创建 InvitationHandler
func NewInvitationHandler(db *gorm.DB) *InvitationHandler {
	return &InvitationHandler{db: db}
}

// GetInviteCode 获取我的邀请码
func (h *InvitationHandler) GetInviteCode(c *gin.Context) {
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

	// 生成或获取邀请码
	var user model.User
	if err := h.db.First(&user, realUserID).Error; err != nil {
		common.ErrorResponse(c, http.StatusNotFound, util.UserNotFound)
		return
	}

	// 使用用户 ID 生成邀请码（简单方式）
	inviteCode := fmt.Sprintf("INV%d", realUserID)

	common.SuccessResponse(c, util.Success, gin.H{
		"invite_code": inviteCode,
		"user_id":     realUserID,
	})
}

// BindInviteCode 绑定邀请码
func (h *InvitationHandler) BindInviteCode(c *gin.Context) {
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

	var req struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 检查是否已经绑定过
	var existing model.Invitation
	if err := h.db.Where("invitee_id = ?", realUserID).First(&existing).Error; err == nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "already bound to an inviter")
		return
	}

	// 解析邀请码，获取邀请人 ID
	var inviterID int64
	fmt.Sscanf(req.InviteCode, "INV%d", &inviterID)

	if inviterID <= 0 {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvitationCodeInvalid, "invalid invite code")
		return
	}

	// 检查邀请人是否存在
	var inviter model.User
	if err := h.db.First(&inviter, inviterID).Error; err != nil {
		common.ErrorResponse(c, http.StatusNotFound, util.UserNotFound, "inviter not found")
		return
	}

	// 不能邀请自己
	if inviterID == realUserID {
		common.ErrorResponse(c, http.StatusBadRequest, util.SelfInvitation)
		return
	}

	// 开启事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建邀请关系
	invitation := model.Invitation{
		InviterID:     inviterID,
		InviteeID:     realUserID,
		InviterCode:   req.InviteCode,
		Credit:        0,
		InviteeCredit: 0,
		CashbackRate:  0.2, // 20% 返现
		CreatedAt:     time.Now(),
	}

	if err := tx.Create(&invitation).Error; err != nil {
		tx.Rollback()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to bind invitation")
		return
	}

	// 给邀请人和被邀请人发放奖励积分
	rewardCredit := 1000 // 奖励 1000 积分

	// 给邀请人发奖励
	h.addBalance(tx, inviterID, rewardCredit, "邀请好友奖励")

	// 给被邀请人发奖励
	h.addBalance(tx, realUserID, rewardCredit, "新人奖励")

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "transaction commit failed")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"message":       "invitation bound successfully",
		"inviter_id":    inviterID,
		"reward_credit": rewardCredit,
	})
}

// GetInvitationStats 获取邀请统计
func (h *InvitationHandler) GetInvitationStats(c *gin.Context) {
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

	// 统计邀请数量
	var count int64
	h.db.Model(&model.Invitation{}).Where("inviter_id = ?", realUserID).Count(&count)

	// 统计获得的总积分
	var totalCredit int64
	h.db.Model(&model.Invitation{}).Where("inviter_id = ?", realUserID).Select("SUM(credit)").Scan(&totalCredit)

	common.SuccessResponse(c, util.Success, gin.H{
		"invite_count": count,
		"total_credit": totalCredit,
	})
}

// GetInvitees 获取我邀请的用户列表
func (h *InvitationHandler) GetInvitees(c *gin.Context) {
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

	var invitations []model.Invitation
	if err := h.db.Where("inviter_id = ?", realUserID).Preload("Invitee").Find(&invitations).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": invitations,
	})
}

// addBalance 增加用户余额（辅助函数）
func (h *InvitationHandler) addBalance(tx *gorm.DB, userID int64, credit int, remark string) error {
	var balance model.UserBalance
	result := tx.Where("user_id = ?", userID).First(&balance)

	now := time.Now()
	if result.Error == gorm.ErrRecordNotFound {
		balance = model.UserBalance{
			UserID:       userID,
			Quota:        credit,
			TotalQuota:   credit,
			TotalBalance: decimal.NewFromInt(int64(credit)),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		return tx.Create(&balance).Error
	} else if result.Error != nil {
		return result.Error
	}

	return tx.Model(&balance).Updates(map[string]interface{}{
		"quota":       gorm.Expr("quota + ?", credit),
		"total_quota": gorm.Expr("total_quota + ?", credit),
		"updated_at":  now,
	}).Error
}
