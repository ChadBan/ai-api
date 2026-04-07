package user

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/service"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserHandler 用户管理 Handler
type UserHandler struct {
	db             *gorm.DB
	billingService *service.BillingService
}

// NewUserHandler 创建 UserHandler
func NewUserHandler(db *gorm.DB, billingService *service.BillingService) *UserHandler {
	return &UserHandler{
		db:             db,
		billingService: billingService,
	}
}

// GetUserInfo 获取当前用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.UserNotFound)
		} else {
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		}
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"avatar":     user.Avatar,
		"role":       user.Role,
		"tier":       user.Tier,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	})
}

// GetBillings 获取用户消费记录
func (h *UserHandler) GetBillings(c *gin.Context) {
	userIDStr, exists := c.Get("userid")
	if !exists || userIDStr == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid user id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid start date")
			return
		}
	}
	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid end date")
			return
		}
		// 设置为当天结束时间
		end = end.Add(24*time.Hour - time.Second)
	}

	billings, total, err := h.billingService.GetUserBillings(userID, start, end, pageSize, (page-1)*pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to get billings")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data":      billings,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
