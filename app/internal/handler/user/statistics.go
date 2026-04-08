package user

import (
	"net/http"
	"strconv"
	"time"

	"ai-api/app/internal/common"
	"ai-api/app/internal/util"

	"ai-api/app/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatisticsHandler 统计处理器
type StatisticsHandler struct {
	db *gorm.DB
}

// NewStatisticsHandler 创建 StatisticsHandler
func NewStatisticsHandler(db *gorm.DB) *StatisticsHandler {
	return &StatisticsHandler{db: db}
}

// DashboardResponse 仪表板响应
type DashboardResponse struct {
	UserCount       int64   `json:"user_count"`        // 用户总数
	ActiveUsers     int64   `json:"active_users"`      // 活跃用户数
	TotalRevenue    float64 `json:"total_revenue"`     // 总收入（积分）
	TodayRevenue    float64 `json:"today_revenue"`     // 今日收入
	TotalTokens     int64   `json:"total_tokens"`      // 总 token 消耗
	TodayTokens     int64   `json:"today_tokens"`      // 今日 token 消耗
	ChannelCount    int64   `json:"channel_count"`     // 渠道数量
	ModelCount      int64   `json:"model_count"`       // 模型数量
	RequestCount    int64   `json:"request_count"`     // 总请求数
	TodayRequests   int64   `json:"today_requests"`    // 今日请求数
	AvgResponseTime float64 `json:"avg_response_time"` // 平均响应时间
}

// ModelStatsResponse 模型统计响应
type ModelStatsResponse struct {
	Models []ModelStatDetail `json:"models"`
}

// ModelStatDetail 模型统计详情
type ModelStatDetail struct {
	ModelName     string  `json:"model_name"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	UniqueUsers   int64   `json:"unique_users"`
	AvgTokens     float64 `json:"avg_tokens"`
}

// GetDashboard 获取仪表板数据
func (h *StatisticsHandler) GetDashboard(c *gin.Context) {
	var stats DashboardResponse

	// 用户统计
	h.db.Model(&model.User{}).Where("status = ?", 1).Count(&stats.UserCount)

	// 活跃用户（最近 7 天有使用记录）
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	h.db.Model(&model.UsageRecord{}).
		Where("created_at >= ?", sevenDaysAgo).
		Distinct("user_id").
		Count(&stats.ActiveUsers)

	// 收入统计
	var totalQuota int64
	h.db.Model(&model.Billing{}).Where("type = ?", 0).Select("SUM(quota)").Scan(&totalQuota)
	stats.TotalRevenue = float64(totalQuota)

	var todayQuota int64
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&model.Billing{}).
		Where("type = ?", 0).
		Where("created_at >= ?", today).
		Select("SUM(quota)").
		Scan(&todayQuota)
	stats.TodayRevenue = float64(todayQuota)

	// Token 统计
	h.db.Model(&model.UsageRecord{}).Select("SUM(total_tokens)").Scan(&stats.TotalTokens)
	h.db.Model(&model.UsageRecord{}).
		Where("created_at >= ?", today).
		Select("SUM(total_tokens)").
		Scan(&stats.TodayTokens)

	// 渠道和模型统计
	h.db.Model(&model.Channel{}).Where("status = ?", 1).Count(&stats.ChannelCount)
	h.db.Model(&model.Model{}).Where("status = ?", 1).Count(&stats.ModelCount)

	// 请求统计
	h.db.Model(&model.UsageRecord{}).Count(&stats.RequestCount)
	h.db.Model(&model.UsageRecord{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayRequests)

	// 平均响应时间
	var avgTime float64
	h.db.Model(&model.UsageRecord{}).
		Where("created_at >= ?", sevenDaysAgo).
		Select("AVG(duration)").
		Scan(&avgTime)
	stats.AvgResponseTime = avgTime / 1000.0 // 转换为秒

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// UserStatsRequest 用户统计请求
type UserStatsRequest struct {
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	UserID    int64  `form:"user_id"`
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	UserID         int64           `json:"user_id"`
	TotalRequests  int64           `json:"total_requests"`
	TotalTokens    int64           `json:"total_tokens"`
	TotalCost      float64         `json:"total_cost"`
	ModelBreakdown []ModelStatItem `json:"model_breakdown"`
	DailyStats     []DailyStatItem `json:"daily_stats"`
}

// ModelStatItem 模型统计项
type ModelStatItem struct {
	ModelName  string  `json:"model_name"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	Cost       float64 `json:"cost"`
	Percentage float64 `json:"percentage"`
}

// DailyStatItem 每日统计项
type DailyStatItem struct {
	Date     string  `json:"date"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
}

// GetUserStats 获取用户统计
func (h *StatisticsHandler) GetUserStats(c *gin.Context) {
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

	// 解析时间范围
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var startTime, endTime time.Time

	if startTimeStr != "" {
		startTime, err = time.Parse("2006-01-02", startTimeStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid start_time format")
			return
		}
	} else {
		startTime = time.Now().AddDate(0, -1, 0) // 默认最近 30 天
	}

	if endTimeStr != "" {
		endTime, err = time.Parse("2006-01-02", endTimeStr)
		if err != nil {
			common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid end_time format")
			return
		}
		endTime = endTime.Add(24 * time.Hour) // 包含结束日期
	} else {
		endTime = time.Now()
	}

	var response UserStatsResponse
	response.UserID = realUserID

	// 总请求数
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Count(&response.TotalRequests)

	// 总 token 数
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Select("SUM(total_tokens)").
		Scan(&response.TotalTokens)

	// 总费用
	var totalCost int64
	h.db.Model(&model.Billing{}).
		Where("user_id = ?", realUserID).
		Where("type = ?", 0).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Select("SUM(quota)").
		Scan(&totalCost)
	response.TotalCost = float64(totalCost)

	// 模型分类统计
	var modelStats []struct {
		ModelName string
		Requests  int64
		Tokens    int64
		Cost      int64
	}

	h.db.Model(&model.UsageRecord{}).
		Select("model_name, COUNT(*) as requests, SUM(total_tokens) as tokens, 0 as cost").
		Where("user_id = ?", realUserID).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("model_name").
		Scan(&modelStats)

	// 计算百分比
	for _, ms := range modelStats {
		percentage := 0.0
		if response.TotalRequests > 0 {
			percentage = float64(ms.Requests) / float64(response.TotalRequests) * 100
		}

		// 获取该模型的费用
		var modelCost int64
		h.db.Model(&model.Billing{}).
			Where("user_id = ?", realUserID).
			Where("model_name = ?", ms.ModelName).
			Where("type = ?", 0).
			Where("created_at BETWEEN ? AND ?", startTime, endTime).
			Select("SUM(quota)").
			Scan(&modelCost)

		response.ModelBreakdown = append(response.ModelBreakdown, ModelStatItem{
			ModelName:  ms.ModelName,
			Requests:   ms.Requests,
			Tokens:     ms.Tokens,
			Cost:       float64(modelCost),
			Percentage: percentage,
		})
	}

	// 每日统计
	var dailyStats []struct {
		Date     time.Time
		Requests int64
		Tokens   int64
	}

	h.db.Model(&model.UsageRecord{}).
		Select("DATE(created_at) as date, COUNT(*) as requests, SUM(total_tokens) as tokens").
		Where("user_id = ?", realUserID).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyStats)

	for _, ds := range dailyStats {
		// 获取该日的费用
		var dayCost int64
		h.db.Model(&model.Billing{}).
			Where("user_id = ?", realUserID).
			Where("type = ?", 0).
			Where("created_at >= ? AND created_at < ?", ds.Date, ds.Date.Add(24*time.Hour)).
			Select("SUM(quota)").
			Scan(&dayCost)

		response.DailyStats = append(response.DailyStats, DailyStatItem{
			Date:     ds.Date.Format("2006-01-02"),
			Requests: ds.Requests,
			Tokens:   ds.Tokens,
			Cost:     float64(dayCost),
		})
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": response,
	})
}

// ChannelStatsResponse 渠道统计响应
type ChannelStatsResponse struct {
	Channels []ChannelStatItem `json:"channels"`
}

// ChannelStatItem 渠道统计项
type ChannelStatItem struct {
	ChannelID       int64   `json:"channel_id"`
	ChannelName     string  `json:"channel_name"`
	TotalRequests   int64   `json:"total_requests"`
	TotalTokens     int64   `json:"total_tokens"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time"`
	Priority        int     `json:"priority"`
	Status          int     `json:"status"`
}

// GetChannelStats 获取渠道统计
func (h *StatisticsHandler) GetChannelStats(c *gin.Context) {
	var channels []model.Channel
	h.db.Where("status IN ?", []int{0, 1}).Find(&channels)

	var response ChannelStatsResponse

	for _, ch := range channels {
		var stat ChannelStatItem
		stat.ChannelID = ch.ID
		stat.ChannelName = ch.Name
		stat.Priority = ch.Priority
		stat.Status = ch.Status
		stat.TotalTokens = ch.UsedTokens
		stat.AvgResponseTime = float64(ch.ResponseTime) / 1000.0

		// 统计请求数和成功率
		var totalRequests, successRequests int64
		h.db.Model(&model.UsageRecord{}).
			Where("provider_name = ?", ch.Name).
			Count(&totalRequests)

		h.db.Model(&model.UsageRecord{}).
			Where("provider_name = ?", ch.Name).
			Where("status_code BETWEEN ? AND ?", 200, 299).
			Count(&successRequests)

		stat.TotalRequests = totalRequests
		if totalRequests > 0 {
			stat.SuccessRate = float64(successRequests) / float64(totalRequests) * 100
		}

		response.Channels = append(response.Channels, stat)
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": response,
	})
}

// GetModelStats 获取模型统计
func (h *StatisticsHandler) GetModelStats(c *gin.Context) {
	var models []model.Model
	h.db.Where("status = ?", 1).Find(&models)

	var response ModelStatsResponse

	for _, m := range models {
		var stat ModelStatDetail
		stat.ModelName = m.Name

		// 统计请求数
		h.db.Model(&model.UsageRecord{}).
			Where("model_name = ?", m.Name).
			Count(&stat.TotalRequests)

		// 统计 token 数
		h.db.Model(&model.UsageRecord{}).
			Where("model_name = ?", m.Name).
			Select("SUM(total_tokens)").
			Scan(&stat.TotalTokens)

		// 统计费用
		var totalCost int64
		h.db.Model(&model.Billing{}).
			Where("model_name = ?", m.Name).
			Where("type = ?", 0).
			Select("SUM(quota)").
			Scan(&totalCost)
		stat.TotalCost = float64(totalCost)

		// 统计独立用户数
		h.db.Model(&model.UsageRecord{}).
			Where("model_name = ?", m.Name).
			Distinct("user_id").
			Count(&stat.UniqueUsers)

		// 计算平均 token 数
		if stat.TotalRequests > 0 {
			stat.AvgTokens = float64(stat.TotalTokens) / float64(stat.TotalRequests)
		}

		response.Models = append(response.Models, stat)
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": response,
	})
}

// RevenueStatsResponse 收入统计响应
type RevenueStatsResponse struct {
	TotalRevenue  float64        `json:"total_revenue"`
	TodayRevenue  float64        `json:"today_revenue"`
	MonthRevenue  float64        `json:"month_revenue"`
	DailyRevenue  []DailyRevItem `json:"daily_revenue"`
	RevenueByType []TypeRevItem  `json:"revenue_by_type"`
}

// DailyRevItem 每日收入项
type DailyRevItem struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

// TypeRevItem 类型收入项
type TypeRevItem struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
	Count  int64   `json:"count"`
}

// GetRevenueStats 获取收入统计
func (h *StatisticsHandler) GetRevenueStats(c *gin.Context) {
	var response RevenueStatsResponse

	// 总收入
	var totalQuota int64
	h.db.Model(&model.Billing{}).Where("type IN ?", []int{0, 1, 3}).Select("SUM(quota)").Scan(&totalQuota)
	response.TotalRevenue = float64(totalQuota)

	// 今日收入
	var todayQuota int64
	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&model.Billing{}).
		Where("type IN ?", []int{0, 1, 3}).
		Where("created_at >= ?", today).
		Select("SUM(quota)").
		Scan(&todayQuota)
	response.TodayRevenue = float64(todayQuota)

	// 本月收入
	var monthQuota int64
	monthStart := time.Now().AddDate(0, 0, -30)
	h.db.Model(&model.Billing{}).
		Where("type IN ?", []int{0, 1, 3}).
		Where("created_at >= ?", monthStart).
		Select("SUM(quota)").
		Scan(&monthQuota)
	response.MonthRevenue = float64(monthQuota)

	// 每日收入（最近 30 天）
	var dailyRev []struct {
		Date   time.Time
		Amount int64
	}
	h.db.Model(&model.Billing{}).
		Select("DATE(created_at) as date, SUM(quota) as amount").
		Where("type IN ?", []int{0, 1, 3}).
		Where("created_at >= ?", monthStart).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyRev)

	for _, dr := range dailyRev {
		response.DailyRevenue = append(response.DailyRevenue, DailyRevItem{
			Date:   dr.Date.Format("2006-01-02"),
			Amount: float64(dr.Amount),
		})
	}

	// 按类型统计
	typeStats := []struct {
		Type   int
		Name   string
		Count  int64
		Amount int64
	}{
		{0, "消费", 0, 0},
		{1, "充值", 0, 0},
		{3, "兑换", 0, 0},
	}

	for i := range typeStats {
		h.db.Model(&model.Billing{}).
			Where("type = ?", typeStats[i].Type).
			Count(&typeStats[i].Count)

		h.db.Model(&model.Billing{}).
			Where("type = ?", typeStats[i].Type).
			Select("SUM(quota)").
			Scan(&typeStats[i].Amount)
	}

	for _, ts := range typeStats {
		response.RevenueByType = append(response.RevenueByType, TypeRevItem{
			Type:   ts.Name,
			Amount: float64(ts.Amount),
			Count:  ts.Count,
		})
	}

	common.SuccessResponse(c, util.Success, gin.H{
		"data": response,
	})
}
