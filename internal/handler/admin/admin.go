package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	db             *gorm.DB
	channelService *service.ChannelService
	optionService  *service.OptionService
	pricingService *service.PricingService
	groupService   *service.GroupService
}

// NewAdminHandler 创建 AdminHandler
func NewAdminHandler(
	db *gorm.DB,
	channelService *service.ChannelService,
	optionService *service.OptionService,
	pricingService *service.PricingService,
	groupService *service.GroupService,
) *AdminHandler {
	return &AdminHandler{
		db:             db,
		channelService: channelService,
		optionService:  optionService,
		pricingService: pricingService,
		groupService:   groupService,
	}
}

// AdminMiddleware 管理员权限验证中间件
func (h *AdminHandler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userid")
		if !exists || userID == nil {
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
			c.Abort()
			return
		}

		// Parse userid as string then convert to int64
		userIDStr, ok := userID.(string)
		if !ok {
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
			c.Abort()
			return
		}

		userIDInt64, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
			c.Abort()
			return
		}

		var user model.User
		if err := h.db.First(&user, userIDInt64).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				common.ErrorResponse(c, http.StatusNotFound, util.UserNotFound)
			} else {
				common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
			}
			c.Abort()
			return
		}

		if user.Role != "admin" {
			common.ErrorResponse(c, http.StatusForbidden, util.Forbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// ===================== 用户管理 =====================

// ListUsers 用户列表（管理员）
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	role := c.Query("role")
	search := c.Query("search")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.User{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		query = query.Where("email LIKE ? OR name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	var users []model.User

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// GetUser 获取用户详情（管理员）
func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.UserNotFound)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, user)
}

// UpdateUser 更新用户信息（管理员）
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
		Status int    `json:"status"`
		Role   string `json:"role"`
		Tier   string `json:"tier"`
		Group  string `json:"group"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Status != 0 {
		updates["status"] = req.Status
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Tier != "" {
		updates["tier"] = req.Tier
	}
	if req.Group != "" {
		updates["group"] = req.Group
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update user")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "user updated"})
}

// DeleteUser 删除用户（管理员）
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&model.User{}, id).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to delete user")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "user deleted"})
}

// BanUser 封禁/解封用户（管理员）
func (h *AdminHandler) BanUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Ban bool `json:"ban"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	status := 1
	if req.Ban {
		status = 0
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update user status")
		return
	}

	data := gin.H{
		"message": "user status updated",
		"banned":  req.Ban,
	}
	common.SuccessResponse(c, util.Success, data)
}

// AddBalance 给用户增加余额（管理员）
func (h *AdminHandler) AddBalance(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Quota  int    `json:"quota" binding:"required,min=1"`
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var balance model.UserBalance
	result := tx.Where("user_id = ?", id).First(&balance)

	now := time.Now()
	if result.Error == gorm.ErrRecordNotFound {
		balance = model.UserBalance{
			UserID:     getInt64(id),
			Quota:      req.Quota,
			TotalQuota: req.Quota,
			CreatedAt:  now,
			UpdatedAt:  now,
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
		if err := tx.Model(&balance).Updates(map[string]interface{}{
			"quota":       gorm.Expr("quota + ?", req.Quota),
			"total_quota": gorm.Expr("total_quota + ?", req.Quota),
			"updated_at":  now,
		}).Error; err != nil {
			tx.Rollback()
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update balance")
			return
		}
	}

	billing := model.Billing{
		UserID:        getInt64(id),
		Amount:        decimal.NewFromInt(int64(req.Quota)),
		Quota:         req.Quota,
		Type:          4, // 4=赠送
		BillingStatus: 1,
		CreatedAt:     now,
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to record billing")
		return
	}

	if err := tx.Commit().Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "transaction commit failed")
		return
	}

	data := gin.H{
		"message": "balance added",
		"quota":   req.Quota,
	}
	common.SuccessResponse(c, util.Success, data)
}

// ===================== 渠道管理 =====================

// ListChannels 渠道列表（管理员）
func (h *AdminHandler) ListChannels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.Channel{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	var channels []model.Channel

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("priority ASC").Find(&channels).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      channels,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// CreateChannel 创建渠道（管理员）
func (h *AdminHandler) CreateChannel(c *gin.Context) {
	var req model.Channel
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	if err := h.db.Create(&req).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to create channel")
		return
	}

	// 刷新渠道缓存
	h.channelService.RefreshChannels()

	data := gin.H{
		"data":    req,
		"message": "channel created",
	}
	common.SuccessResponse(c, util.Success, data)
}

// UpdateChannel 更新渠道（管理员）
func (h *AdminHandler) UpdateChannel(c *gin.Context) {
	id := c.Param("id")

	var req model.Channel
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	req.UpdatedAt = time.Now()

	if err := h.db.Model(&model.Channel{}).Where("id = ?", id).Updates(&req).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to update channel")
		return
	}

	// 刷新渠道缓存
	h.channelService.RefreshChannels()

	common.SuccessResponse(c, util.Success, gin.H{"message": "channel updated"})
}

// DeleteChannel 删除渠道（管理员）
func (h *AdminHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&model.Channel{}, id).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to delete channel")
		return
	}

	h.channelService.RefreshChannels()

	common.SuccessResponse(c, util.Success, gin.H{"message": "channel deleted"})
}

// TestChannel 测试渠道（管理员）- 实际调用 ChannelService
func (h *AdminHandler) TestChannel(c *gin.Context) {
	id := c.Param("id")

	var channel model.Channel
	if err := h.db.First(&channel, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.ErrorResponse(c, http.StatusNotFound, util.ChannelNotFound)
			return
		}
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	startTime := time.Now()
	err := h.channelService.TestChannel(ctx, &channel)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		data := gin.H{
			"success":       false,
			"error":         err.Error(),
			"response_time": responseTime,
		}
		common.SuccessResponse(c, util.Success, data)
		return
	}

	// 更新渠道响应时间
	h.db.Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]interface{}{
		"response_time":  responseTime,
		"last_test_time": time.Now(),
	})

	data := gin.H{
		"success":       true,
		"response_time": responseTime,
	}
	common.SuccessResponse(c, util.Success, data)
}

// BatchTestChannels 批量测试渠道
func (h *AdminHandler) BatchTestChannels(c *gin.Context) {
	var channels []model.Channel
	if err := h.db.Where("status = ?", 1).Find(&channels).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	results := make([]gin.H, 0)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	for _, ch := range channels {
		startTime := time.Now()
		err := h.channelService.TestChannel(ctx, &ch)
		responseTime := time.Since(startTime).Milliseconds()

		result := gin.H{
			"id":            ch.ID,
			"name":          ch.Name,
			"response_time": responseTime,
			"success":       err == nil,
		}
		if err != nil {
			result["error"] = err.Error()
		}
		results = append(results, result)
	}

	common.SuccessResponse(c, util.Success, gin.H{"data": results})
}

// GetChannelHealth 获取渠道健康状态
func (h *AdminHandler) GetChannelHealth(c *gin.Context) {
	healthMap := h.channelService.GetAllHealthStatus()

	common.SuccessResponse(c, util.Success, gin.H{"data": healthMap})
}

// GetLatestModels 获取最新模型列表
func (h *AdminHandler) GetLatestModels(c *gin.Context) {
	channelType := c.Query("type")

	models, err := h.channelService.GetLatestModels(channelType)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"data": models})
}

// SearchModels 搜索模型
func (h *AdminHandler) SearchModels(c *gin.Context) {
	query := c.Query("q")
	channelType := c.Query("type")

	models, err := h.channelService.SearchModels(query, channelType)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"data": models})
}

// ===================== 系统配置 =====================

// GetSystemConfig 获取系统配置（管理员）
func (h *AdminHandler) GetSystemConfig(c *gin.Context) {
	allOptions, err := h.optionService.GetAll()
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to load config")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"data": allOptions})
}

// UpdateSystemConfig 更新系统配置（管理员）
func (h *AdminHandler) UpdateSystemConfig(c *gin.Context) {
	var config map[string]string
	if err := c.ShouldBindJSON(&config); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.optionService.SetBulk(config); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to save config")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "config updated"})
}

// ===================== 模型定价管理 =====================

// ListModelPrices 列出模型定价
func (h *AdminHandler) ListModelPrices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	search := c.Query("search")

	prices, total, err := h.pricingService.ListModelPrices(page, pageSize, search)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      prices,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// CreateModelPrice 创建模型定价
func (h *AdminHandler) CreateModelPrice(c *gin.Context) {
	var price model.ModelPrice
	if err := c.ShouldBindJSON(&price); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.pricingService.CreateModelPrice(&price); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	data := gin.H{"data": price, "message": "model price created"}
	common.SuccessResponse(c, util.Success, data)
}

// UpdateModelPrice 更新模型定价
func (h *AdminHandler) UpdateModelPrice(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.pricingService.UpdateModelPrice(id, updates); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "model price updated"})
}

// DeleteModelPrice 删除模型定价
func (h *AdminHandler) DeleteModelPrice(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.pricingService.DeleteModelPrice(id); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "model price deleted"})
}

// ===================== 分组倍率管理 =====================

// ListGroupMultipliers 列出分组倍率
func (h *AdminHandler) ListGroupMultipliers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	multipliers, total, err := h.pricingService.ListGroupMultipliers(page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      multipliers,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// CreateGroupMultiplier 创建分组倍率
func (h *AdminHandler) CreateGroupMultiplier(c *gin.Context) {
	var m model.GroupPriceMultiplier
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.pricingService.CreateGroupMultiplier(&m); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	data := gin.H{"data": m, "message": "group multiplier created"}
	common.SuccessResponse(c, util.Success, data)
}

// UpdateGroupMultiplier 更新分组倍率
func (h *AdminHandler) UpdateGroupMultiplier(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.pricingService.UpdateGroupMultiplier(id, updates); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "group multiplier updated"})
}

// DeleteGroupMultiplier 删除分组倍率
func (h *AdminHandler) DeleteGroupMultiplier(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.pricingService.DeleteGroupMultiplier(id); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "group multiplier deleted"})
}

// ===================== 分组管理 =====================

// ListGroups 列出分组
func (h *AdminHandler) ListGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	groups, total, err := h.groupService.ListGroups(page, pageSize)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      groups,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// CreateGroup 创建分组
func (h *AdminHandler) CreateGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.groupService.CreateGroup(&group); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	data := gin.H{"data": group, "message": "group created"}
	common.SuccessResponse(c, util.Success, data)
}

// UpdateGroup 更新分组
func (h *AdminHandler) UpdateGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	if err := h.groupService.UpdateGroup(id, updates); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "group updated"})
}

// DeleteGroup 删除分组
func (h *AdminHandler) DeleteGroup(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.groupService.DeleteGroup(id); err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{"message": "group deleted"})
}

// ===================== 日志查询 =====================

// GetLogs 获取操作日志（管理员）
func (h *AdminHandler) GetLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	key := c.Query("key")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.UsageRecord{})

	if key != "" {
		query = query.Where("model_name LIKE ? OR provider_name LIKE ?", "%"+key+"%", "%"+key+"%")
	}

	var total int64
	var logs []model.UsageRecord

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	data := gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}
	common.SuccessResponse(c, util.Success, data)
}

// 辅助函数
func getInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}
