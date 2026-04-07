package handler

import (
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/handler/admin"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/handler/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/handler/relay"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/handler/user"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 导出 common handlers
var (
	HealthHandler = common.HealthHandler
	ReadyHandler  = common.ReadyHandler
)

// 导出 ModelHandler
type (
	ModelHandler = common.ModelHandler
)

func NewModelHandler(db *gorm.DB) *ModelHandler {
	return common.NewModelHandler(db)
}

// 导出 user handlers
type (
	AuthHandler        = user.AuthHandler
	TokenHandler       = user.TokenHandler
	InvitationHandler  = user.InvitationHandler
	RedemptionHandler  = user.RedemptionHandler
	StatisticsHandler  = user.StatisticsHandler
	UserHandler        = user.UserHandler
	RegisterRequest    = user.RegisterRequest
	LoginRequest       = user.LoginRequest
	CreateTokenRequest = user.CreateTokenRequest
)

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return user.NewAuthHandler(db)
}

func NewTokenHandler(db *gorm.DB) *TokenHandler {
	return user.NewTokenHandler(db)
}

func NewInvitationHandler(db *gorm.DB) *InvitationHandler {
	return user.NewInvitationHandler(db)
}

func NewRedemptionHandler(db *gorm.DB) *RedemptionHandler {
	return user.NewRedemptionHandler(db)
}

func NewStatisticsHandler(db *gorm.DB) *StatisticsHandler {
	return user.NewStatisticsHandler(db)
}

func NewUserHandler(db *gorm.DB, billingService *service.BillingService) *UserHandler {
	return user.NewUserHandler(db, billingService)
}

// 导出 admin handlers
type (
	AdminHandler = admin.AdminHandler
	LogHandler   = admin.LogHandler
)

func NewAdminHandler(
	db *gorm.DB,
	channelService *service.ChannelService,
	optionService *service.OptionService,
	pricingService *service.PricingService,
	groupService *service.GroupService,
) *AdminHandler {
	return admin.NewAdminHandler(db, channelService, optionService, pricingService, groupService)
}

func NewLogHandler(db *gorm.DB, logger *zap.Logger, auditService *service.AuditService, errorLogService *service.ErrorLogService) *LogHandler {
	return admin.NewLogHandler(db, logger, auditService, errorLogService)
}

// 导出 relay handlers
type (
	RelayHandler = relay.RelayHandler
)

func NewRelayHandler(db *gorm.DB, channelService *service.ChannelService, billingService *service.BillingService, tokenService *service.TokenService, logger *zap.Logger) *RelayHandler {
	return relay.NewRelayHandler(db, channelService, billingService, tokenService, logger)
}
