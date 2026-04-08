package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-api/app/internal/logger"
	"ai-api/app/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TokenService Token 服务
type TokenService struct {
	db             *gorm.DB
	logger         *logger.Logger
	billingService *BillingService
}

// NewTokenService 创建 Token 服务
func NewTokenService(db *gorm.DB, logger *logger.Logger, billingService *BillingService) *TokenService {
	return &TokenService{
		db:             db,
		logger:         logger,
		billingService: billingService,
	}
}

// ValidateToken 验证 Token
func (s *TokenService) ValidateToken(ctx context.Context, tokenKey string) (*model.Token, error) {
	// 查询 Token
	token := &model.Token{}
	if err := s.db.Where("key = ?", tokenKey).First(token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("failed to query token: %w", err)
	}

	// 检查状态
	if !token.IsEnabled() {
		return nil, fmt.Errorf("token is disabled")
	}

	// 检查过期时间
	if token.IsExpired() {
		return nil, fmt.Errorf("token has expired")
	}

	// 检查配额
	if !token.HasQuota(0) {
		return nil, fmt.Errorf("token has insufficient quota")
	}

	// 更新最后访问时间
	now := time.Now()
	s.db.Model(token).Update("accessed_time", &now)

	return token, nil
}

// CheckModelPermission 检查模型权限
func (s *TokenService) CheckModelPermission(token *model.Token, modelName string) error {
	// 如果没有设置限制，允许所有模型
	if token.ModelLimit == "" || token.ModelLimit == "[]" || token.ModelLimit == "null" {
		return nil
	}

	// 解析 JSON
	var modelLimit []string
	if err := json.Unmarshal([]byte(token.ModelLimit), &modelLimit); err != nil {
		s.logger.Warn("failed to parse model_limit", logger.String("model_limit", token.ModelLimit), logger.Err(err))
		return nil // 解析失败时允许访问（向后兼容）
	}

	// 如果没有配置模型列表，允许所有模型
	if len(modelLimit) == 0 {
		return nil
	}

	// 检查是否在允许列表中
	for _, allowedModel := range modelLimit {
		// 完全匹配
		if allowedModel == modelName {
			return nil
		}
		// 通配符匹配（如 gpt-* 匹配 gpt-3.5-turbo, gpt-4）
		if strings.HasSuffix(allowedModel, "*") {
			prefix := strings.TrimSuffix(allowedModel, "*")
			if strings.HasPrefix(modelName, prefix) {
				return nil
			}
		}
	}

	return fmt.Errorf("model %s is not allowed by this token", modelName)
}

// CalculateQuota 计算配额
func (s *TokenService) CalculateQuota(modelName string, inputTokens, outputTokens int, ratio float64) (int, error) {
	// 获取模型价格
	modelPrice, err := s.billingService.GetModelPrice(modelName)
	if err != nil {
		// 如果找不到价格，使用默认价格
		modelPrice = decimal.NewFromFloat(0.002) // $0.002 / 1K tokens
	}

	// 计算总 tokens
	totalTokens := inputTokens + outputTokens

	// 计算美元成本
	costUSD := decimal.NewFromInt(int64(totalTokens)).
		Div(decimal.NewFromInt(1000)).
		Mul(modelPrice).
		Mul(decimal.NewFromFloat(ratio))

	// 转换为配额（美元 * 汇率 * 1000000）
	priceRatio := s.billingService.GetPriceRatio()
	quota := costUSD.Mul(priceRatio).Mul(decimal.NewFromInt(1000000))

	// 转换为 int
	quotaInt := quota.IntPart()
	if quotaInt > int64(^uint(0)>>1) || quotaInt < 0 {
		return 0, fmt.Errorf("quota overflow")
	}

	return int(quotaInt), nil
}

// DeductTokenQuota 扣减 Token 配额
func (s *TokenService) DeductTokenQuota(ctx context.Context, token *model.Token, quota int) error {
	if quota <= 0 {
		return nil
	}

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 再次检查配额（防止并发问题）
		var currentToken model.Token
		if err := tx.Where("id = ?", token.ID).First(&currentToken).Error; err != nil {
			return fmt.Errorf("failed to query token: %w", err)
		}

		if !currentToken.HasQuota(quota) {
			return fmt.Errorf("insufficient quota")
		}

		// 扣减配额
		if err := tx.Model(&currentToken).Updates(map[string]interface{}{
			"remain_quota": gorm.Expr("remain_quota - ?", quota),
			"used_quota":   gorm.Expr("used_quota + ?", quota),
		}).Error; err != nil {
			return fmt.Errorf("failed to deduct quota: %w", err)
		}

		return nil
	})
}

// RecordTokenUsage 记录 Token 使用
func (s *TokenService) RecordTokenUsage(ctx context.Context, log *model.TokenUsageLog) error {
	return s.db.Create(log).Error
}

// GetTokenUsageStats 获取 Token 使用统计
func (s *TokenService) GetTokenUsageStats(ctx context.Context, tokenKey string, startTime, endTime time.Time) (*TokenUsageStats, error) {
	var stats TokenUsageStats

	query := s.db.Model(&model.TokenUsageLog{}).
		Where("token_key = ? AND request_time BETWEEN ? AND ?", tokenKey, startTime, endTime)

	var result struct {
		TotalRequests   int64
		TotalTokens     int64
		TotalQuota      int64
		SuccessRequests int64
	}

	if err := query.Select(
		"COUNT(*) as total_requests",
		"SUM(tokens_used) as total_tokens",
		"SUM(quota_deducted) as total_quota",
		"COUNT(CASE WHEN success THEN 1 END) as success_requests",
	).Scan(&result).Error; err != nil {
		return nil, err
	}

	stats.TotalRequests = result.TotalRequests
	stats.TotalTokens = result.TotalTokens
	stats.TotalQuota = result.TotalQuota
	stats.SuccessRequests = result.SuccessRequests

	return &stats, nil
}

// TokenUsageStats Token 使用统计
type TokenUsageStats struct {
	TotalRequests   int64 `json:"total_requests"`
	TotalTokens     int64 `json:"total_tokens"`
	TotalQuota      int64 `json:"total_quota"`
	SuccessRequests int64 `json:"success_requests"`
	FailedRequests  int64 `json:"failed_requests"`
	AvgTokensPerReq int64 `json:"avg_tokens_per_req"`
	AvgQuotaPerReq  int64 `json:"avg_quota_per_req"`
}

// CalculateAvg 计算平均值
func (s *TokenUsageStats) CalculateAvg() {
	if s.TotalRequests > 0 {
		s.AvgTokensPerReq = s.TotalTokens / s.TotalRequests
		s.AvgQuotaPerReq = s.TotalQuota / s.TotalRequests
	}
	s.FailedRequests = s.TotalRequests - s.SuccessRequests
}

// AutoRenewalConfig 自动续期配置
type AutoRenewalConfig struct {
	Enabled      bool // 是否启用自动续期
	RenewalCycle int  // 续期周期（天数）
	MinQuota     int  // 触发续期的最低配额阈值
	RenewalQuota int  // 续期配额量
}

// CheckAndAutoRenewToken 检查并执行 Token 自动续期
func (s *TokenService) CheckAndAutoRenewToken(ctx context.Context, token *model.Token, config *AutoRenewalConfig) error {
	if !config.Enabled {
		return nil
	}

	// 检查 Token 是否已过期或即将过期（提前 7 天）
	now := time.Now()
	shouldRenew := false

	if token.ExpiredTime != nil {
		// 如果已过期或 7 天内过期
		if token.ExpiredTime.Before(now) || token.ExpiredTime.Before(now.Add(7*24*time.Hour)) {
			shouldRenew = true
		}
	}

	// 检查配额是否低于阈值
	if token.RemainQuota < config.MinQuota {
		shouldRenew = true
	}

	if !shouldRenew {
		return nil
	}

	// 获取用户信息以检查余额
	var user model.User
	if err := s.db.First(&user, token.UserID).Error; err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}

	// 检查用户余额是否足够
	var balance model.UserBalance
	if err := s.db.Where("user_id = ?", token.UserID).First(&balance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user balance not found")
		}
		return fmt.Errorf("failed to query user balance: %w", err)
	}

	// 简单策略：如果用户余额充足，自动增加配额
	// 实际应用中可能需要更复杂的计费逻辑
	renewalCost := config.RenewalQuota / 1000000 // 假设 1 元 = 1000000 配额

	if balance.Quota >= renewalCost*1000000 {
		// 开启事务执行续期
		return s.db.Transaction(func(tx *gorm.DB) error {
			// 扣减用户余额
			if err := tx.Model(&balance).Updates(map[string]interface{}{
				"quota": gorm.Expr("quota - ?", renewalCost*1000000),
			}).Error; err != nil {
				return fmt.Errorf("failed to deduct user balance: %w", err)
			}

			// 增加 Token 配额
			if err := tx.Model(token).Updates(map[string]interface{}{
				"remain_quota": gorm.Expr("remain_quota + ?", config.RenewalQuota),
			}).Error; err != nil {
				return fmt.Errorf("failed to renew token quota: %w", err)
			}

			// 如果 Token 已过期，延长过期时间
			if token.ExpiredTime != nil && token.ExpiredTime.Before(now) {
				newExpiredTime := now.Add(time.Duration(config.RenewalCycle) * 24 * time.Hour)
				if err := tx.Model(token).Update("expired_time", newExpiredTime).Error; err != nil {
					return fmt.Errorf("failed to extend token expiration: %w", err)
				}
			}

			s.logger.Info("Token auto-renewed successfully",
				logger.Int64("token_id", token.ID),
				logger.Int("renewal_quota", config.RenewalQuota),
				logger.Int("user_balance_deducted", renewalCost*1000000))

			return nil
		})
	}

	return fmt.Errorf("insufficient user balance for auto-renewal")
}

// GetLowQuotaTokens 获取低配额 Token 列表（用于定时任务检查）
func (s *TokenService) GetLowQuotaTokens(ctx context.Context, threshold int, limit int) ([]model.Token, error) {
	var tokens []model.Token

	query := s.db.Where("status = ? AND unlimited_quota = ?", 1, false).
		Where("remain_quota < ?", threshold)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&tokens).Error; err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetExpiringTokens 获取即将过期的 Token 列表（用于定时任务检查）
func (s *TokenService) GetExpiringTokens(ctx context.Context, daysUntilExpiry int, limit int) ([]model.Token, error) {
	var tokens []model.Token

	thresholdTime := time.Now().Add(time.Duration(daysUntilExpiry) * 24 * time.Hour)

	query := s.db.Where("status = ?", 1).
		Where("expired_time IS NOT NULL").
		Where("expired_time <= ?", thresholdTime)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&tokens).Error; err != nil {
		return nil, err
	}

	return tokens, nil
}
