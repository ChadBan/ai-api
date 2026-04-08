package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ai-api/app/internal/common"
	"ai-api/app/internal/logger"
	"ai-api/app/internal/util"

	"ai-api/app/internal/model"
	"ai-api/app/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LogHandler 日志处理器
type LogHandler struct {
	db              *gorm.DB
	logger          *logger.Logger
	auditService    *service.AuditService
	errorLogService *service.ErrorLogService
}

// NewLogHandler 创建 LogHandler
func NewLogHandler(db *gorm.DB, logger *logger.Logger, auditService *service.AuditService, errorLogService *service.ErrorLogService) *LogHandler {
	return &LogHandler{
		db:              db,
		logger:          logger,
		auditService:    auditService,
		errorLogService: errorLogService,
	}
}

// ListAuditLogs 获取审计日志列表（管理员）
func (h *LogHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")
	resource := c.Query("resource")
	userID := c.Query("user_id")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.AuditLog{})

	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	var logs []model.AuditLog

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetAuditLogDetail 获取审计日志详情（管理员）
func (h *LogHandler) GetAuditLogDetail(c *gin.Context) {
	id := c.Param("id")

	var log model.AuditLog
	if err := h.db.First(&log, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": log,
	})
}

// ListRequestLogs 获取请求日志列表（管理员）
func (h *LogHandler) ListRequestLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	path := c.Query("path")
	modelName := c.Query("model_name")
	statusCode := c.Query("status_code")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.RequestLog{})

	if path != "" {
		query = query.Where("path LIKE ?", "%"+path+"%")
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if statusCode != "" {
		query = query.Where("status_code = ?", statusCode)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	var logs []model.RequestLog

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListErrorLogs 获取错误日志列表（管理员）
func (h *LogHandler) ListErrorLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	level := c.Query("level")
	requestID := c.Query("request_id")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.ErrorLog{})

	if level != "" {
		query = query.Where("level = ?", level)
	}
	if requestID != "" {
		query = query.Where("request_id = ?", requestID)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	var logs []model.ErrorLog

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListLoginLogs 获取登录日志列表（管理员）
func (h *LogHandler) ListLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	username := c.Query("username")
	status := c.Query("status")
	ipAddress := c.Query("ip_address")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	offset := (page - 1) * pageSize

	query := h.db.Model(&model.LoginLog{})

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	var logs []model.LoginLog

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUserAuditLogs 获取当前用户的审计日志
func (h *LogHandler) GetUserAuditLogs(c *gin.Context) {
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	offset := (page - 1) * pageSize

	var total int64
	var logs []model.AuditLog

	query := h.db.Model(&model.AuditLog{}).Where("user_id = ?", realUserID)

	query.Count(&total)
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ExportLogs 导出日志（管理员）
func (h *LogHandler) ExportLogs(c *gin.Context) {
	logType := c.Query("type") // audit/request/error/login
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	var data interface{}
	var err error

	switch logType {
	case "audit":
		var logs []model.AuditLog
		query := h.db.Model(&model.AuditLog{})
		if startTime != "" {
			query = query.Where("created_at >= ?", startTime)
		}
		if endTime != "" {
			query = query.Where("created_at <= ?", endTime)
		}
		err = query.Order("created_at DESC").Limit(10000).Find(&logs).Error
		data = logs

	case "request":
		var logs []model.RequestLog
		query := h.db.Model(&model.RequestLog{})
		if startTime != "" {
			query = query.Where("created_at >= ?", startTime)
		}
		if endTime != "" {
			query = query.Where("created_at <= ?", endTime)
		}
		err = query.Order("created_at DESC").Limit(10000).Find(&logs).Error
		data = logs

	case "error":
		var logs []model.ErrorLog
		query := h.db.Model(&model.ErrorLog{})
		if startTime != "" {
			query = query.Where("created_at >= ?", startTime)
		}
		if endTime != "" {
			query = query.Where("created_at <= ?", endTime)
		}
		err = query.Order("created_at DESC").Limit(10000).Find(&logs).Error
		data = logs

	case "login":
		var logs []model.LoginLog
		query := h.db.Model(&model.LoginLog{})
		if startTime != "" {
			query = query.Where("created_at >= ?", startTime)
		}
		if endTime != "" {
			query = query.Where("created_at <= ?", endTime)
		}
		err = query.Order("created_at DESC").Limit(10000).Find(&logs).Error
		data = logs

	default:
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid log type")
		return
	}

	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "database error")
		return
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data":  data,
		"count": len(data.([]interface{})),
	})
}

// GetLogStatistics 获取日志统计（管理员）
func (h *LogHandler) GetLogStatistics(c *gin.Context) {
	hours := 24
	if h := c.Query("hours"); h != "" {
		fmt.Sscanf(h, "%d", &hours)
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	type Stats struct {
		TotalAuditLogs   int64   `json:"total_audit_logs"`
		TotalRequestLogs int64   `json:"total_request_logs"`
		TotalErrorLogs   int64   `json:"total_error_logs"`
		TotalLoginLogs   int64   `json:"total_login_logs"`
		ErrorRate        float64 `json:"error_rate"`
		AvgResponseTime  float64 `json:"avg_response_time"`
	}

	var stats Stats

	h.db.Model(&model.AuditLog{}).Where("created_at >= ?", since).Count(&stats.TotalAuditLogs)
	h.db.Model(&model.RequestLog{}).Where("created_at >= ?", since).Count(&stats.TotalRequestLogs)
	h.db.Model(&model.ErrorLog{}).Where("created_at >= ?", since).Count(&stats.TotalErrorLogs)
	h.db.Model(&model.LoginLog{}).Where("created_at >= ?", since).Count(&stats.TotalLoginLogs)

	// 计算错误率
	if stats.TotalRequestLogs > 0 {
		var errorCount int64
		h.db.Model(&model.RequestLog{}).
			Where("created_at >= ?", since).
			Where("status_code >= ?", 400).
			Count(&errorCount)
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalRequestLogs) * 100
	}

	// 计算平均响应时间
	h.db.Model(&model.RequestLog{}).
		Where("created_at >= ?", since).
		Select("AVG(duration)").
		Scan(&stats.AvgResponseTime)

	common.SuccessResponse(c, util.Success, gin.H{
		"data": stats,
	})
}
