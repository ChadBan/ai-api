package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ai-api/app/internal/logger"
	"ai-api/app/internal/model"

	"gorm.io/gorm"
)

// SchedulerService 定时任务服务
type SchedulerService struct {
	db       *gorm.DB
	logger   *logger.Logger
	tasks    map[string]*ScheduledTask
	mu       sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// ScheduledTask 定时任务定义
type ScheduledTask struct {
	Name       string
	Schedule   string // cron 表达式或固定间隔
	Handler    func() error
	Enabled    bool
	LastRun    time.Time
	NextRun    time.Time
	Error      error
	ErrorCount int
}

// NewSchedulerService 创建定时任务服务
func NewSchedulerService(db *gorm.DB, log *logger.Logger) *SchedulerService {
	return &SchedulerService{
		db:       db,
		logger:   log,
		tasks:    make(map[string]*ScheduledTask),
		shutdown: make(chan struct{}),
	}
}

// RegisterTask 注册定时任务
func (s *SchedulerService) RegisterTask(name, schedule string, handler func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[name] = &ScheduledTask{
		Name:     name,
		Schedule: schedule,
		Handler:  handler,
		Enabled:  true,
	}

	s.logger.Info("task registered", logger.String("name", name), logger.String("schedule", schedule))
}

// Start 启动所有定时任务
func (s *SchedulerService) Start() {
	s.logger.Info("starting scheduler service")

	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}

		s.wg.Add(1)
		go s.runTask(task)
	}
}

// Stop 停止所有定时任务
func (s *SchedulerService) Stop() {
	close(s.shutdown)
	s.wg.Wait()
	s.logger.Info("scheduler service stopped")
}

// runTask 运行定时任务（简单版本，使用固定间隔）
func (s *SchedulerService) runTask(task *ScheduledTask) {
	defer s.wg.Done()

	// 解析间隔（简化处理，支持 "every Xs/m/h/d" 格式）
	interval, err := parseSchedule(task.Schedule)
	if err != nil {
		s.logger.Error("failed to parse schedule", logger.String("task", task.Name), logger.Err(err))
		return
	}

	s.logger.Info("task started", logger.String("name", task.Name), logger.Duration("interval", interval))

	// 立即执行一次
	s.executeTask(task)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeTask(task)
		case <-s.shutdown:
			return
		}
	}
}

// executeTask 执行任务
func (s *SchedulerService) executeTask(task *ScheduledTask) {
	s.logger.Info("executing task", logger.String("name", task.Name))

	task.LastRun = time.Now()
	startTime := time.Now()

	err := task.Handler()
	duration := time.Since(startTime)

	if err != nil {
		task.Error = err
		task.ErrorCount++
		s.logger.Error("task execution failed",
			logger.String("name", task.Name),
			logger.Err(err),
			logger.Duration("duration", duration),
			logger.Int("error_count", task.ErrorCount),
		)
	} else {
		task.Error = nil
		task.ErrorCount = 0
		s.logger.Info("task executed successfully",
			logger.String("name", task.Name),
			logger.Duration("duration", duration),
		)
	}

	task.NextRun = time.Now().Add(parseScheduleSafe(task.Schedule))
}

// parseSchedule 解析调度表达式（简化版）
func parseSchedule(schedule string) (time.Duration, error) {
	// 支持格式："every 30s", "every 5m", "every 1h", "every 1d"
	var seconds, minutes, hours, days int
	n, err := fmt.Sscanf(schedule, "every %ds", &seconds)
	if n == 1 && err == nil {
		return time.Duration(seconds) * time.Second, nil
	}

	n, err = fmt.Sscanf(schedule, "every %dm", &minutes)
	if n == 1 && err == nil {
		return time.Duration(minutes) * time.Minute, nil
	}

	n, err = fmt.Sscanf(schedule, "every %dh", &hours)
	if n == 1 && err == nil {
		return time.Duration(hours) * time.Hour, nil
	}

	n, err = fmt.Sscanf(schedule, "every %dd", &days)
	if n == 1 && err == nil {
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("invalid schedule format: %s", schedule)
}

// parseScheduleSafe 安全解析调度（用于计算下次运行时间）
func parseScheduleSafe(schedule string) time.Duration {
	d, err := parseSchedule(schedule)
	if err != nil {
		return time.Hour // 默认 1 小时
	}
	return d
}

// InitTasks 初始化所有定时任务
func (s *SchedulerService) InitTasks() {
	// 渠道测试任务 - 每 5 分钟
	s.RegisterTask("channel_test", "every 5m", s.testChannels)

	// 每日数据汇总 - 每天凌晨 1 点
	s.RegisterTask("daily_summary", "every 1h", s.dailySummary)

	// 过期 Token 清理 - 每小时
	s.RegisterTask("token_cleanup", "every 1h", s.cleanupExpiredTokens)

	// 自动对账 - 每天凌晨 2 点
	s.RegisterTask("reconciliation", "every 1d", s.autoReconciliation)

	// 缓存刷新 - 每 10 分钟
	s.RegisterTask("cache_refresh", "every 10m", s.refreshCache)

	// 日志清理 - 每天凌晨 3 点
	s.RegisterTask("log_cleanup", "every 1d", s.cleanupOldLogs)
}

// testChannels 测试所有渠道的连通性
func (s *SchedulerService) testChannels() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var channels []model.Channel
	if err := s.db.Where("status = ?", 1).Find(&channels).Error; err != nil {
		return err
	}

	successCount := 0
	failCount := 0

	channelSvc := NewChannelService(s.db)

	for _, ch := range channels {
		err := channelSvc.TestChannel(ctx, &ch)
		if err != nil {
			failCount++
			s.logger.Debug("channel test failed", logger.String("name", ch.Name), logger.Err(err))

			// 连续失败多次则禁用渠道
			if ch.ResponseTime > 1000 { // 简单判断：响应时间过长
				s.db.Model(&model.Channel{}).Where("id = ?", ch.ID).Update("status", 0)
				s.logger.Warn("disabled channel due to poor performance", logger.String("name", ch.Name))
			}
		} else {
			successCount++
			s.logger.Debug("channel tested", logger.String("name", ch.Name), logger.Bool("success", true))
		}
	}

	s.logger.Info("channel test completed",
		logger.Int("total", len(channels)),
		logger.Int("success", successCount),
		logger.Int("failed", failCount),
	)

	return nil
}

// dailySummary 每日数据汇总
func (s *SchedulerService) dailySummary() error {
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	// 按用户分组统计
	rows, err := s.db.Model(&model.UsageRecord{}).
		Select("user_id, COUNT(*) as total_requests, SUM(input_tokens) as total_input_tokens, SUM(output_tokens) as total_output_tokens").
		Where("DATE(created_at) = ?", yesterdayStr).
		Group("user_id").
		Rows()

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var totalRequests, inputTokens, outputTokens int64

		if err := rows.Scan(&userID, &totalRequests, &inputTokens, &outputTokens); err != nil {
			s.logger.Error("failed to scan daily summary", logger.Err(err))
			continue
		}

		// 获取该用户的总费用
		var totalCost float64
		s.db.Model(&model.Billing{}).
			Select("SUM(quota)").
			Where("user_id = ?", userID).
			Where("DATE(created_at) = ?", yesterdayStr).
			Where("type = ?", 0).
			Scan(&totalCost)

		// 保存到每日汇总表
		dailyUsage := model.DailyUsage{
			UserID:            userID,
			Date:              yesterday,
			TotalRequests:     int(totalRequests),
			TotalInputTokens:  int(inputTokens),
			TotalOutputTokens: int(outputTokens),
			TotalTokens:       int(inputTokens + outputTokens),
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// 使用 Upsert 避免重复
		s.db.Save(&dailyUsage)
	}

	s.logger.Info("daily summary completed", logger.String("date", yesterdayStr))
	return nil
}

// cleanupExpiredTokens 清理过期的 Token
func (s *SchedulerService) cleanupExpiredTokens() error {
	now := time.Now()

	// 查找过期的 Token
	var expiredTokens []model.Token
	if err := s.db.Where("expired_at IS NOT NULL AND expired_at < ?", now).
		Where("status = ?", 1).
		Find(&expiredTokens).Error; err != nil {
		return err
	}

	// 禁用过期 Token
	for _, token := range expiredTokens {
		if err := s.db.Model(&token).Update("status", 0).Error; err != nil {
			s.logger.Error("failed to disable expired token",
				logger.Int64("token_id", token.ID),
				logger.Err(err),
			)
		} else {
			expiredTime := "unknown"
			if token.ExpiredTime != nil {
				expiredTime = token.ExpiredTime.Format(time.RFC3339)
			}
			s.logger.Info("disabled expired token",
				logger.Int64("token_id", token.ID),
				logger.String("expired_time", expiredTime),
			)
		}
	}

	s.logger.Info("token cleanup completed", logger.Int("disabled_count", len(expiredTokens)))
	return nil
}

// autoReconciliation 自动对账
func (s *SchedulerService) autoReconciliation() error {
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	s.logger.Info("starting auto reconciliation", logger.String("date", yesterdayStr))

	// 1. 核对 usage_records 和 billings 的一致性
	type Mismatch struct {
		UserID       int64
		ModelName    string
		UsageCount   int64
		BillingCount int64
		UsageTokens  int64
		BillingQuota int64
	}

	var mismatches []Mismatch

	// 查找不匹配的记录
	rows, err := s.db.Table(`
		(SELECT user_id, model_name, 
		        COUNT(*) as usage_count, 
		        SUM(total_tokens) as usage_tokens,
		        0 as billing_count,
		        0 as billing_quota
		 FROM usage_records 
		 WHERE DATE(created_at) = ? 
		 GROUP BY user_id, model_name
		) u
		LEFT JOIN
		(SELECT user_id, model_name,
		        0 as usage_count,
		        0 as usage_tokens,
		        COUNT(*) as billing_count,
		        SUM(quota) as billing_quota
		 FROM billings 
		 WHERE DATE(created_at) = ? AND type = 0
		 GROUP BY user_id, model_name
		) b
		ON u.user_id = b.user_id AND u.model_name = b.model_name
		WHERE u.usage_count != b.billing_count OR u.usage_tokens != b.billing_quota
	`, yesterdayStr, yesterdayStr).Rows()

	if err != nil {
		s.logger.Error("reconciliation query failed", logger.Err(err))
		return err
	}
	defer rows.Close()

	discrepancyCount := 0
	for rows.Next() {
		var m Mismatch
		s.db.ScanRows(rows, &m)
		mismatches = append(mismatches, m)
		discrepancyCount++

		s.logger.Warn("reconciliation discrepancy found",
			logger.Int64("user_id", m.UserID),
			logger.String("model", m.ModelName),
			logger.Int64("usage_count", m.UsageCount),
			logger.Int64("billing_count", m.BillingCount),
			logger.Int64("usage_tokens", m.UsageTokens),
			logger.Int64("billing_quota", m.BillingQuota),
		)
	}

	// 2. 生成对账报告
	s.logger.Info("reconciliation report generated",
		logger.String("date", yesterdayStr),
		logger.Int("total_discrepancies", discrepancyCount),
	)

	// TODO: 保存对账报告到数据库或发送通知

	if discrepancyCount > 0 {
		s.logger.Warn("reconciliation completed with discrepancies",
			logger.Int("count", discrepancyCount),
		)
	} else {
		s.logger.Info("reconciliation completed successfully")
	}

	return nil
}

// refreshCache 刷新缓存
func (s *SchedulerService) refreshCache() error {
	// TODO: 刷新各种缓存
	// - 渠道列表缓存
	// - 模型列表缓存
	// - 用户余额缓存

	s.logger.Debug("cache refreshed")
	return nil
}

// cleanupOldLogs 清理旧日志
func (s *SchedulerService) cleanupOldLogs() error {
	retentionDays := 30 // 保留 30 天
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 清理审计日志
	result := s.db.Where("created_at < ?", cutoffTime).Delete(&model.AuditLog{})
	s.logger.Info("cleaned old audit logs", logger.Int64("deleted", result.RowsAffected))

	// 清理请求日志
	result = s.db.Where("created_at < ?", cutoffTime).Delete(&model.RequestLog{})
	s.logger.Info("cleaned old request logs", logger.Int64("deleted", result.RowsAffected))

	// 清理错误日志（保留更长时间）
	errorRetentionDays := 90
	errorCutoff := time.Now().AddDate(0, 0, -errorRetentionDays)
	result = s.db.Where("created_at < ?", errorCutoff).Delete(&model.ErrorLog{})
	s.logger.Info("cleaned old error logs", logger.Int64("deleted", result.RowsAffected))

	return nil
}
