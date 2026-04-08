package service

import (
	"context"
	"fmt"
	"time"

	"ai-api/app/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BillingService 计费服务
type BillingService struct {
	db             *gorm.DB
	pricingService *PricingService
}

// NewBillingService 创建计费服务
func NewBillingService(db *gorm.DB, pricingService *PricingService) *BillingService {
	return &BillingService{
		db:             db,
		pricingService: pricingService,
	}
}

// CalculateTokens 计算 token 使用量和费用 (兼容旧接口)
func (s *BillingService) CalculateTokens(modelName string, promptTokens, completionTokens int) (int, decimal.Decimal, error) {
	_, _, totalQuota := s.pricingService.CalculateQuota(modelName, "default", promptTokens, completionTokens)
	return promptTokens + completionTokens, decimal.NewFromInt(int64(totalQuota)), nil
}

// ConsumeQuota 消费配额（扣费）
func (s *BillingService) ConsumeQuota(ctx context.Context, userID, tokenID int64, modelName string, promptTokens, completionTokens int, channelID int64, channelName string) error {
	return s.ConsumeQuotaWithGroup(ctx, userID, tokenID, modelName, "default", promptTokens, completionTokens, channelID, channelName)
}

// ConsumeQuotaWithGroup 消费配额（带分组）
func (s *BillingService) ConsumeQuotaWithGroup(ctx context.Context, userID, tokenID int64, modelName, groupName string, promptTokens, completionTokens int, channelID int64, channelName string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 计算费用
	inputQuota, outputQuota, totalQuota := s.pricingService.CalculateQuota(modelName, groupName, promptTokens, completionTokens)
	quota := decimal.NewFromInt(int64(totalQuota))

	// 检查并扣减用户余额
	if err := s.deductUserBalance(tx, userID, quota); err != nil {
		tx.Rollback()
		return err
	}

	// 如果使用了 Token，也扣减 Token 配额
	if tokenID > 0 {
		if err := s.deductTokenQuota(tx, tokenID, quota); err != nil {
			tx.Rollback()
			return err
		}
	}

	// 获取模型倍率用于账单记录
	price := s.pricingService.GetModelPrice(modelName)
	multiplier := price.InputRatio

	// 记录账单
	billing := model.Billing{
		UserID:           userID,
		TokenID:          tokenID,
		Amount:           quota,
		Quota:            totalQuota,
		ModelName:        modelName,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		Multiplier:       multiplier,
		Type:             0, // 0=消费
		ChannelId:        channelID,
		ChannelName:      channelName,
		InputQuota:       inputQuota,
		OutputQuota:      outputQuota,
		GroupRatio:       s.pricingService.GetGroupMultiplier(groupName, modelName),
		BillingStatus:    1, // completed
		CreatedAt:        time.Now(),
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新渠道统计
	if err := tx.Model(&model.Channel{}).Where("id = ?", channelID).UpdateColumns(map[string]interface{}{
		"used_tokens": gorm.Expr("used_tokens + ?", promptTokens+completionTokens),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// PreConsumeQuota 预扣费
func (s *BillingService) PreConsumeQuota(ctx context.Context, userID, tokenID int64, modelName, groupName string, estimatedPromptTokens int) (int64, error) {
	// 估算 completion tokens (假设为 prompt 的 1.5 倍或 512，取较大值)
	estimatedCompletion := estimatedPromptTokens
	if estimatedCompletion < 512 {
		estimatedCompletion = 512
	}

	_, _, estimatedQuota := s.pricingService.CalculateQuota(modelName, groupName, estimatedPromptTokens, estimatedCompletion)
	if estimatedQuota < 1 {
		estimatedQuota = 1
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查余额
	var balance model.UserBalance
	if err := tx.Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			balance = model.UserBalance{
				UserID:     userID,
				Quota:      1000,
				TotalQuota: 1000,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := tx.Create(&balance).Error; err != nil {
				tx.Rollback()
				return 0, err
			}
		} else {
			tx.Rollback()
			return 0, err
		}
	}

	availableQuota := balance.Quota - balance.FrozenQuota
	if availableQuota < estimatedQuota {
		tx.Rollback()
		return 0, fmt.Errorf("insufficient balance: available %d, need %d", availableQuota, estimatedQuota)
	}

	// 冻结额度
	if err := tx.Model(&model.UserBalance{}).Where("user_id = ?", userID).UpdateColumns(map[string]interface{}{
		"frozen_quota": gorm.Expr("frozen_quota + ?", estimatedQuota),
		"updated_at":   time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	// 创建预扣费账单记录
	billing := model.Billing{
		UserID:        userID,
		TokenID:       tokenID,
		Amount:        decimal.NewFromInt(int64(estimatedQuota)),
		Quota:         estimatedQuota,
		ModelName:     modelName,
		Type:          7, // 7=预扣费
		BillingStatus: 0, // pending
		CreatedAt:     time.Now(),
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return billing.ID, nil
}

// ReconcileQuota 结算预扣费
func (s *BillingService) ReconcileQuota(ctx context.Context, preConsumeID int64, userID, tokenID int64, modelName, groupName string, actualPromptTokens, actualCompletionTokens int, channelID int64, channelName string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取预扣费记录
	var preBilling model.Billing
	if err := tx.First(&preBilling, preConsumeID).Error; err != nil {
		tx.Rollback()
		return err
	}

	preConsumedQuota := preBilling.Quota

	// 计算实际费用
	inputQuota, outputQuota, actualQuota := s.pricingService.CalculateQuota(modelName, groupName, actualPromptTokens, actualCompletionTokens)

	// 差额 = 预扣 - 实际
	diff := preConsumedQuota - actualQuota

	// 解冻预扣额度，扣除实际费用
	if err := tx.Model(&model.UserBalance{}).Where("user_id = ?", userID).UpdateColumns(map[string]interface{}{
		"frozen_quota": gorm.Expr("GREATEST(frozen_quota - ?, 0)", preConsumedQuota),
		"quota":        gorm.Expr("quota - ?", actualQuota),
		"used_quota":   gorm.Expr("used_quota + ?", actualQuota),
		"updated_at":   time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 扣减 Token 配额
	if tokenID > 0 {
		quota := decimal.NewFromInt(int64(actualQuota))
		if err := s.deductTokenQuota(tx, tokenID, quota); err != nil {
			tx.Rollback()
			return err
		}
	}

	// 标记预扣费记录为已结算
	if err := tx.Model(&model.Billing{}).Where("id = ?", preConsumeID).Updates(map[string]interface{}{
		"billing_status": 2, // reconciled
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 获取模型倍率
	price := s.pricingService.GetModelPrice(modelName)

	// 创建实际消费账单
	billing := model.Billing{
		UserID:           userID,
		TokenID:          tokenID,
		Amount:           decimal.NewFromInt(int64(actualQuota)),
		Quota:            actualQuota,
		ModelName:        modelName,
		PromptTokens:     actualPromptTokens,
		CompletionTokens: actualCompletionTokens,
		TotalTokens:      actualPromptTokens + actualCompletionTokens,
		Multiplier:       price.InputRatio,
		Type:             0, // 消费
		ChannelId:        channelID,
		ChannelName:      channelName,
		InputQuota:       inputQuota,
		OutputQuota:      outputQuota,
		GroupRatio:       s.pricingService.GetGroupMultiplier(groupName, modelName),
		PreConsumeID:     preConsumeID,
		BillingStatus:    1, // completed
		CreatedAt:        time.Now(),
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 如果有多退的，记录退款
	if diff > 0 {
		// 已在上面的 quota 更新中处理了差额 (解冻 preConsumed, 扣除 actual, 净效果就是退了差额)
		_ = diff
	}

	// 更新渠道统计
	if err := tx.Model(&model.Channel{}).Where("id = ?", channelID).UpdateColumns(map[string]interface{}{
		"used_tokens": gorm.Expr("used_tokens + ?", actualPromptTokens+actualCompletionTokens),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// deductUserBalance 扣减用户余额
func (s *BillingService) deductUserBalance(tx *gorm.DB, userID int64, quota decimal.Decimal) error {
	var balance model.UserBalance
	if err := tx.Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			balance = model.UserBalance{
				UserID:     userID,
				Balance:    decimal.Zero,
				Quota:      1000,
				TotalQuota: 1000,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := tx.Create(&balance).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if balance.Quota < int(quota.IntPart()) {
		return fmt.Errorf("insufficient balance: have %d, need %d", balance.Quota, int(quota.IntPart()))
	}

	return tx.Model(&model.UserBalance{}).Where("user_id = ?", userID).UpdateColumns(map[string]interface{}{
		"quota":      gorm.Expr("quota - ?", int(quota.IntPart())),
		"used_quota": gorm.Expr("used_quota + ?", int(quota.IntPart())),
		"updated_at": time.Now(),
	}).Error
}

// deductTokenQuota 扣减 Token 配额
func (s *BillingService) deductTokenQuota(tx *gorm.DB, tokenID int64, quota decimal.Decimal) error {
	var token model.Token
	if err := tx.Where("id = ?", tokenID).First(&token).Error; err != nil {
		return err
	}

	if !token.UnlimitedQuota {
		if token.RemainQuota < int(quota.IntPart()) {
			return fmt.Errorf("token insufficient balance: have %d, need %d", token.RemainQuota, int(quota.IntPart()))
		}
	}

	updates := map[string]interface{}{
		"used_quota":    gorm.Expr("used_quota + ?", int(quota.IntPart())),
		"accessed_time": time.Now(),
	}

	if !token.UnlimitedQuota {
		updates["remain_quota"] = gorm.Expr("remain_quota - ?", int(quota.IntPart()))
	}

	return tx.Model(&token).Updates(updates).Error
}

// AddBalance 增加用户余额（充值）
func (s *BillingService) AddBalance(userID int64, quota int, remark string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var balance model.UserBalance
	result := tx.Where("user_id = ?", userID).First(&balance)

	if result.Error == gorm.ErrRecordNotFound {
		balance = model.UserBalance{
			UserID:     userID,
			Quota:      quota,
			TotalQuota: quota,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := tx.Create(&balance).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else if result.Error != nil {
		tx.Rollback()
		return result.Error
	} else {
		if err := tx.Model(&balance).Updates(map[string]interface{}{
			"quota":       gorm.Expr("quota + ?", quota),
			"total_quota": gorm.Expr("total_quota + ?", quota),
			"updated_at":  time.Now(),
		}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	billing := model.Billing{
		UserID:        userID,
		Amount:        decimal.NewFromInt(int64(quota)),
		Quota:         quota,
		Type:          1, // 1=充值
		BillingStatus: 1,
		CreatedAt:     time.Now(),
	}

	if err := tx.Create(&billing).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetBalance 获取用户余额
func (s *BillingService) GetBalance(userID int64) (*model.UserBalance, error) {
	var balance model.UserBalance
	if err := s.db.Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			balance = model.UserBalance{
				UserID:     userID,
				Quota:      1000,
				TotalQuota: 1000,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if err := s.db.Create(&balance).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &balance, nil
}

// GetUserBillings 获取用户账单列表
func (s *BillingService) GetUserBillings(userID int64, start, end time.Time, limit, offset int) ([]model.Billing, int64, error) {
	var billings []model.Billing
	var total int64

	query := s.db.Model(&model.Billing{}).Where("user_id = ?", userID)

	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at <= ?", end)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&billings).Error; err != nil {
		return nil, 0, err
	}

	return billings, total, nil
}

// GetModelPrice 获取模型价格（每 1K tokens 的美元价格）
// 简化实现：返回默认价格 $0.002 / 1K tokens
func (s *BillingService) GetModelPrice(modelName string) (decimal.Decimal, error) {
	return decimal.NewFromFloat(0.002), nil // $0.002 / 1K tokens
}

// GetPriceRatio 获取汇率倍率（美元转配额）
func (s *BillingService) GetPriceRatio() decimal.Decimal {
	// 从 SystemOption 获取汇率，默认 7.2
	var option model.SystemOption
	if err := s.db.Where("key = ?", "price").First(&option).Error; err != nil {
		return decimal.NewFromFloat(7.2) // 默认汇率
	}

	ratio, err := decimal.NewFromString(option.Value)
	if err != nil {
		return decimal.NewFromFloat(7.2)
	}

	return ratio
}

// DeductUserBalance 扣减用户余额（简化版）
func (s *BillingService) DeductUserBalance(ctx context.Context, userID int64, quota int, reason string) error {
	if quota <= 0 {
		return nil
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查询用户余额
	var balance model.UserBalance
	if err := tx.Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user balance not found")
		}
		return err
	}

	// 检查余额是否充足
	if balance.Quota < quota {
		return fmt.Errorf("insufficient user balance")
	}

	// 扣减余额
	newQuota := balance.Quota - quota
	if err := tx.Model(&balance).Update("quota", newQuota).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}
