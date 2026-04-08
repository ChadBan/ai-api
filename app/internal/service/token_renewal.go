package service

import (
	"context"
	"sync"
	"time"

	"ai-api/app/internal/logger"
	"ai-api/app/internal/model"
)

// TokenRenewalService Token 自动续期服务
type TokenRenewalService struct {
	tokenService *TokenService
	logger       *logger.Logger
	config       *AutoRenewalConfig
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewTokenRenewalService 创建 Token 自动续期服务
func NewTokenRenewalService(tokenService *TokenService, logger *logger.Logger, config *AutoRenewalConfig) *TokenRenewalService {
	return &TokenRenewalService{
		tokenService: tokenService,
		logger:       logger,
		config:       config,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动自动续期定时任务
func (s *TokenRenewalService) Start() {
	s.logger.Info("Starting token auto-renewal service",
		logger.Bool("enabled", s.config.Enabled),
		logger.Int("renewal_cycle_days", s.config.RenewalCycle),
		logger.Int("min_quota_threshold", s.config.MinQuota),
		logger.Int("renewal_quota", s.config.RenewalQuota))

	if !s.config.Enabled {
		s.logger.Info("Auto-renewal is disabled, skipping")
		return
	}

	// 每 24 小时检查一次
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ticker.C:
				s.checkAndRenewAllTokens()
			case <-s.stopChan:
				s.logger.Info("Stopping token auto-renewal service")
				return
			}
		}
	}()
}

// Stop 停止自动续期服务
func (s *TokenRenewalService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("Token auto-renewal service stopped")
}

// checkAndRenewAllTokens 检查并续期所有符合条件的 Token
func (s *TokenRenewalService) checkAndRenewAllTokens() {
	ctx := context.Background()

	// 获取低配额 Token
	lowQuotaTokens, err := s.tokenService.GetLowQuotaTokens(ctx, s.config.MinQuota, 100)
	if err != nil {
		s.logger.Error("Failed to get low quota tokens", logger.Err(err))
	} else {
		s.logger.Info("Found low quota tokens", logger.Int("count", len(lowQuotaTokens)))
		for i := range lowQuotaTokens {
			if err := s.tokenService.CheckAndAutoRenewToken(ctx, &lowQuotaTokens[i], s.config); err != nil {
				s.logger.Warn("Failed to auto-renew low quota token",
					logger.Int64("token_id", lowQuotaTokens[i].ID),
					logger.Err(err))
			}
		}
	}

	// 获取即将过期的 Token
	expiringTokens, err := s.tokenService.GetExpiringTokens(ctx, 7, 100)
	if err != nil {
		s.logger.Error("Failed to get expiring tokens", logger.Err(err))
	} else {
		s.logger.Info("Found expiring tokens", logger.Int("count", len(expiringTokens)))
		for i := range expiringTokens {
			if err := s.tokenService.CheckAndAutoRenewToken(ctx, &expiringTokens[i], s.config); err != nil {
				s.logger.Warn("Failed to auto-renew expiring token",
					logger.Int64("token_id", expiringTokens[i].ID),
					logger.Err(err))
			}
		}
	}
}

// ManualRenew 手动续期指定 Token
func (s *TokenRenewalService) ManualRenew(ctx context.Context, tokenID int64) error {
	// 查询 Token
	var token model.Token
	if err := s.tokenService.db.First(&token, tokenID).Error; err != nil {
		return err
	}

	return s.tokenService.CheckAndAutoRenewToken(ctx, &token, s.config)
}
