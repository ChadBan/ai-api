package user

import (
	"net/http"
	"strconv"
	"time"

	"ai-api/app/internal/common"
	"ai-api/app/internal/util"

	"ai-api/app/internal/model"

	"github.com/gin-gonic/gin"
)

// UserDashboardResponse 用户仪表板响应
type UserDashboardResponse struct {
	TotalRequests   int64   `json:"total_requests"`    // 总请求数
	TodayRequests   int64   `json:"today_requests"`    // 今日请求数
	TotalTokens     int64   `json:"total_tokens"`      // 总 Token 消耗
	ActiveTokens    int64   `json:"active_tokens"`     // 活跃 Token 数
	AvgResponseTime float64 `json:"avg_response_time"` // 平均响应时间
}

// GetUserDashboard 获取用户仪表板数据
func (h *StatisticsHandler) GetUserDashboard(c *gin.Context) {
	// 获取用户 ID
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

	var response UserDashboardResponse
	today := time.Now().Truncate(24 * time.Hour)

	// 总请求数
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Count(&response.TotalRequests)

	// 今日请求数
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Where("created_at >= ?", today).
		Count(&response.TodayRequests)

	// 总 Token 消耗
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Select("SUM(total_tokens)").
		Scan(&response.TotalTokens)

	// 活跃 Token 数（最近 30 天有使用记录的 Token）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	h.db.Model(&model.Token{}).
		Where("user_id = ?", realUserID).
		Where("status = ?", 1).
		Where("last_used_at >= ?", thirtyDaysAgo).
		Count(&response.ActiveTokens)

	// 平均响应时间
	var avgTime float64
	h.db.Model(&model.UsageRecord{}).
		Where("user_id = ?", realUserID).
		Where("created_at >= ?", thirtyDaysAgo).
		Select("AVG(duration)").
		Scan(&avgTime)
	response.AvgResponseTime = avgTime / 1000.0 // 转换为秒

	common.SuccessResponse(c, util.Success, response)
}

// UserChartDataResponse 用户图表数据响应
type UserChartDataResponse struct {
	RequestTrend []TimeValueItem  `json:"request_trend"` // 请求趋势
	ModelUsage   []ModelUsageItem `json:"model_usage"`   // 模型使用分布
}

// TimeValueItem 时间值项
type TimeValueItem struct {
	Time  string `json:"time"`  // 时间
	Value int64  `json:"value"` // 值
}

// ModelUsageItem 模型使用项
type ModelUsageItem struct {
	Model string `json:"model"` // 模型名称
	Count int64  `json:"count"` // 使用次数
}

// GetUserChartData 获取用户图表数据
func (h *StatisticsHandler) GetUserChartData(c *gin.Context) {
	// 获取用户 ID
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

	// 获取时间范围
	timeRange := c.Query("time_range")
	if timeRange == "" {
		timeRange = "today"
	}

	var startTime time.Time
	switch timeRange {
	case "today":
		startTime = time.Now().Truncate(24 * time.Hour)
	case "7days":
		startTime = time.Now().AddDate(0, 0, -7)
	case "30days":
		startTime = time.Now().AddDate(0, 0, -30)
	default:
		startTime = time.Now().Truncate(24 * time.Hour)
	}

	var response UserChartDataResponse

	// 获取请求趋势
	if timeRange == "today" {
		// 按小时统计
		for i := 0; i < 24; i++ {
			hourStart := startTime.Add(time.Duration(i) * time.Hour)
			hourEnd := hourStart.Add(time.Hour)
			var count int64
			h.db.Model(&model.UsageRecord{}).
				Where("user_id = ?", realUserID).
				Where("created_at >= ? AND created_at < ?", hourStart, hourEnd).
				Count(&count)
			response.RequestTrend = append(response.RequestTrend, TimeValueItem{
				Time:  hourStart.Format("15:00"),
				Value: count,
			})
		}
	} else if timeRange == "7days" {
		// 按天统计
		for i := 0; i < 7; i++ {
			dayStart := startTime.AddDate(0, 0, i)
			dayEnd := dayStart.AddDate(0, 0, 1)
			var count int64
			h.db.Model(&model.UsageRecord{}).
				Where("user_id = ?", realUserID).
				Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
				Count(&count)
			response.RequestTrend = append(response.RequestTrend, TimeValueItem{
				Time:  dayStart.Format("01-02"),
				Value: count,
			})
		}
	} else {
		// 按周统计
		for i := 0; i < 4; i++ {
			weekStart := startTime.AddDate(0, 0, i*7)
			weekEnd := weekStart.AddDate(0, 0, 7)
			var count int64
			h.db.Model(&model.UsageRecord{}).
				Where("user_id = ?", realUserID).
				Where("created_at >= ? AND created_at < ?", weekStart, weekEnd).
				Count(&count)
			response.RequestTrend = append(response.RequestTrend, TimeValueItem{
				Time:  "Week " + strconv.Itoa(i+1),
				Value: count,
			})
		}
	}

	// 获取模型使用分布
	var modelStats []struct {
		ModelName string
		Count     int64
	}
	h.db.Model(&model.UsageRecord{}).
		Select("model_name, COUNT(*) as count").
		Where("user_id = ?", realUserID).
		Where("created_at >= ?", startTime).
		Group("model_name").
		Order("count DESC").
		Scan(&modelStats)

	for _, ms := range modelStats {
		response.ModelUsage = append(response.ModelUsage, ModelUsageItem{
			Model: ms.ModelName,
			Count: ms.Count,
		})
	}

	common.SuccessResponse(c, util.Success, response)
}

// UserRecentRequestsResponse 用户最近请求响应
type UserRecentRequestsResponse struct {
	Requests []UsageRecordItem `json:"requests"` // 最近请求列表
}

// UsageRecordItem 使用记录项
type UsageRecordItem struct {
	ID          int64  `json:"id"`           // ID
	ModelName   string `json:"model_name"`   // 模型名称
	Path        string `json:"path"`         // 路径
	StatusCode  int    `json:"status_code"`  // 状态码
	Duration    int    `json:"duration"`     // 响应时间(ms)
	CreatedAt   string `json:"created_at"`   // 创建时间
	TotalTokens int    `json:"total_tokens"` // 总 token 数
}

// GetUserRecentRequests 获取用户最近请求
func (h *StatisticsHandler) GetUserRecentRequests(c *gin.Context) {
	// 获取用户 ID
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

	var records []model.UsageRecord
	h.db.Where("user_id = ?", realUserID).
		Order("created_at DESC").
		Limit(10).
		Find(&records)

	var response UserRecentRequestsResponse
	for _, record := range records {
		response.Requests = append(response.Requests, UsageRecordItem{
			ID:          record.ID,
			ModelName:   record.ModelName,
			StatusCode:  record.StatusCode,
			CreatedAt:   record.CreatedAt.Format("2006-01-02 15:04:05"),
			TotalTokens: record.TotalTokens,
		})
	}

	common.SuccessResponse(c, util.Success, response)
}
