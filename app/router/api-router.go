package router

import (
	"strconv"

	"ai-api/app/internal/handler"
	"ai-api/app/internal/handler/middleware"
	"ai-api/app/internal/logger"
	"ai-api/app/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// APIRouter 路由注册器
type APIRouter struct {
	db             *gorm.DB
	logger         *logger.Logger
	channelService *service.ChannelService
	billingService *service.BillingService
	optionService  *service.OptionService
	pricingService *service.PricingService
	groupService   *service.GroupService
	jwtSecret      string
}

// NewAPIRouter 创建路由注册器
func NewAPIRouter(
	db *gorm.DB,
	logger *logger.Logger,
	channelService *service.ChannelService,
	billingService *service.BillingService,
	optionService *service.OptionService,
	pricingService *service.PricingService,
	groupService *service.GroupService,
	jwtSecret string,
) *APIRouter {
	return &APIRouter{
		db:             db,
		logger:         logger,
		channelService: channelService,
		billingService: billingService,
		optionService:  optionService,
		pricingService: pricingService,
		groupService:   groupService,
		jwtSecret:      jwtSecret,
	}
}

// RegisterRoutes 注册所有 API 路由
func (r *APIRouter) RegisterRoutes(engine *gin.Engine) {
	v1 := engine.Group("/v1")

	// 健康检查（不需要认证）
	r.registerHealthRoutes(engine)

	// 认证路由（不需要认证）
	r.registerAuthRoutes(v1)

	// 模型路由（不需要认证）
	r.registerModelRoutes(v1)

	// 用户信息路由（需要认证）
	r.registerUserRoutes(v1)

	// OpenAI 兼容 API（需要认证）
	r.registerRelayRoutes(v1)

	// 余额查询（需要认证）
	r.registerBalanceRoutes(v1)

	// 兑换码路由
	r.registerRedemptionRoutes(v1)

	// 邀请系统路由
	r.registerInvitationRoutes(v1)

	// 统计数据路由
	r.registerStatisticsRoutes(v1)

	// Token 管理路由
	r.registerTokenRoutes(v1)

	// 后台管理路由（需要管理员权限）
	r.registerAdminRoutes(v1)

	// 日志管理路由（需要管理员权限）
	r.registerLogRoutes(v1)
}

// registerHealthRoutes 注册健康检查路由
func (r *APIRouter) registerHealthRoutes(engine *gin.Engine) {
	engine.GET("/health", handler.HealthHandler)
	engine.GET("/ready", handler.ReadyHandler)
}

// registerAuthRoutes 注册认证路由
func (r *APIRouter) registerAuthRoutes(v1 *gin.RouterGroup) {
	authHandler := handler.NewAuthHandler(r.db)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/token", authHandler.GetToken)
	}
}

// registerUserRoutes 注册用户信息路由
func (r *APIRouter) registerUserRoutes(v1 *gin.RouterGroup) {
	userHandler := handler.NewUserHandler(r.db, r.billingService)
	user := v1.Group("/user")
	user.Use(middleware.AuthMiddleware())
	{
		user.GET("/self", userHandler.GetUserInfo)
	}

	// 消费记录路由
	billings := v1.Group("/billings")
	billings.Use(middleware.AuthMiddleware())
	{
		billings.GET("", userHandler.GetBillings)
	}
}

// registerModelRoutes 注册模型路由
func (r *APIRouter) registerModelRoutes(v1 *gin.RouterGroup) {
	modelHandler := handler.NewModelHandler(r.db)
	models := v1.Group("/models")
	{
		models.GET("", modelHandler.ListModels)
		models.GET("/:id", modelHandler.GetModel)
		models.GET("/available", modelHandler.ListAvailableModels)
		models.GET("/channel", modelHandler.GetChannelModels)
	}
}

// registerRelayRoutes 注册 OpenAI 兼容 API 路由
func (r *APIRouter) registerRelayRoutes(v1 *gin.RouterGroup) {
	// 创建 Token 服务
	tokenService := service.NewTokenService(r.db, r.logger, r.billingService)

	// 创建 Relay Handler
	relayHandler := handler.NewRelayHandler(r.db, r.channelService, r.billingService, tokenService, r.logger)

	// OpenAI 兼容 API（使用 Token 认证 - 适用于外部客户端）
	relay := v1.Group("")
	{
		relay.POST("/chat/completions", relayHandler.TokenAuthMiddleware(), relayHandler.ChatCompletions)
		relay.POST("/completions", relayHandler.TokenAuthMiddleware(), relayHandler.ChatCompletions)
		relay.POST("/embeddings", relayHandler.TokenAuthMiddleware(), relayHandler.Embeddings)
		relay.POST("/images/generations", relayHandler.TokenAuthMiddleware(), relayHandler.ImagesGenerations)
	}

	// Playground API（使用 JWT 用户认证 - 适用于前端对话页面）
	playground := v1.Group("/playground")
	playground.Use(middleware.AuthMiddleware())
	{
		playground.POST("/chat/completions", relayHandler.PlaygroundChatCompletions)
	}
}

// registerBalanceRoutes 注册余额查询路由
func (r *APIRouter) registerBalanceRoutes(v1 *gin.RouterGroup) {
	balance := v1.Group("")
	balance.Use(middleware.AuthMiddleware())
	{
		balance.GET("/balance", func(c *gin.Context) {
			userID, _ := c.Get("userid")
			if userID == nil {
				c.JSON(401, gin.H{"error": "unauthorized"})
				return
			}
			// 将userID从字符串转换为int64
			userIDStr, ok := userID.(string)
			if !ok {
				c.JSON(401, gin.H{"error": "unauthorized"})
				return
			}
			userIDInt64, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				c.JSON(401, gin.H{"error": "unauthorized"})
				return
			}
			balance, err := r.billingService.GetBalance(userIDInt64)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"balance": balance})
		})
	}
}

// registerRedemptionRoutes 注册兑换码路由
func (r *APIRouter) registerRedemptionRoutes(v1 *gin.RouterGroup) {
	redemptionHandler := handler.NewRedemptionHandler(r.db)
	redemptions := v1.Group("/redemptions")
	redemptions.Use(middleware.AuthMiddleware())
	{
		redemptions.POST("", redemptionHandler.CreateRedemption)
		redemptions.POST("/redeem", redemptionHandler.Redeem)
		redemptions.GET("", redemptionHandler.ListRedemptions)
		redemptions.GET("/:id", redemptionHandler.GetRedemption)
		redemptions.DELETE("/:id", redemptionHandler.DeleteRedemption)
		redemptions.GET("/public/list", redemptionHandler.GetPublicRedemptions)
	}
}

// registerInvitationRoutes 注册邀请系统路由
func (r *APIRouter) registerInvitationRoutes(v1 *gin.RouterGroup) {
	invitationHandler := handler.NewInvitationHandler(r.db)
	invitations := v1.Group("/invitations")
	invitations.Use(middleware.AuthMiddleware())
	{
		invitations.GET("/code", invitationHandler.GetInviteCode)
		invitations.POST("/bind", invitationHandler.BindInviteCode)
		invitations.GET("/stats", invitationHandler.GetInvitationStats)
		invitations.GET("/invitees", invitationHandler.GetInvitees)
	}
}

// registerStatisticsRoutes 注册统计数据路由
func (r *APIRouter) registerStatisticsRoutes(v1 *gin.RouterGroup) {
	statisticsHandler := handler.NewStatisticsHandler(r.db)
	statistics := v1.Group("/statistics")
	statistics.Use(middleware.AuthMiddleware())
	{
		statistics.GET("/dashboard", statisticsHandler.GetDashboard)
		statistics.GET("/user", statisticsHandler.GetUserStats)
		statistics.GET("/channel", statisticsHandler.GetChannelStats)
		statistics.GET("/model", statisticsHandler.GetModelStats)
		statistics.GET("/revenue", statisticsHandler.GetRevenueStats)
	}
}

// registerTokenRoutes 注册 Token 管理路由
func (r *APIRouter) registerTokenRoutes(v1 *gin.RouterGroup) {
	tokenHandler := handler.NewTokenHandler(r.db)
	tokens := v1.Group("/tokens")
	tokens.Use(middleware.AuthMiddleware())
	{
		tokens.POST("", tokenHandler.CreateToken)
		tokens.GET("", tokenHandler.ListTokens)
		tokens.GET("/:id", tokenHandler.GetToken)
		tokens.PUT("/:id", tokenHandler.UpdateToken)
		tokens.DELETE("/:id", tokenHandler.DeleteToken)
		tokens.POST("/:id/status", tokenHandler.ToggleTokenStatus)
		tokens.GET("/stats", tokenHandler.GetTokenStats)

		// 新增：Token 充值
		tokens.POST("/:id/topup", tokenHandler.TopupToken)

		// 新增：使用记录查询（按 Token ID）
		tokens.GET("/:id/usage", tokenHandler.GetTokenUsageLogs)

		// 新增：使用记录列表（支持多 Token 筛选）
		tokens.GET("/usage-logs", tokenHandler.ListTokenUsageLogs)
	}
}

// registerAdminRoutes 注册后台管理路由
func (r *APIRouter) registerAdminRoutes(v1 *gin.RouterGroup) {
	adminHandler := handler.NewAdminHandler(r.db, r.channelService, r.optionService, r.pricingService, r.groupService)
	admin := v1.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), adminHandler.AdminMiddleware())
	{
		// 用户管理
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/users/:id", adminHandler.GetUser)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/users/:id/ban", adminHandler.BanUser)
		admin.POST("/users/:id/balance", adminHandler.AddBalance)

		// 渠道管理
		admin.GET("/channels", adminHandler.ListChannels)
		admin.POST("/channels", adminHandler.CreateChannel)
		admin.PUT("/channels/:id", adminHandler.UpdateChannel)
		admin.DELETE("/channels/:id", adminHandler.DeleteChannel)
		admin.POST("/channels/:id/test", adminHandler.TestChannel)
		admin.POST("/channels/test-all", adminHandler.BatchTestChannels)
		admin.GET("/channels/health", adminHandler.GetChannelHealth)
		// 模型管理
		admin.GET("/models/latest", adminHandler.GetLatestModels)
		admin.GET("/models/search", adminHandler.SearchModels)

		// 系统配置
		admin.GET("/config", adminHandler.GetSystemConfig)
		admin.PUT("/config", adminHandler.UpdateSystemConfig)

		// 模型定价管理
		admin.GET("/pricing/models", adminHandler.ListModelPrices)
		admin.POST("/pricing/models", adminHandler.CreateModelPrice)
		admin.PUT("/pricing/models/:id", adminHandler.UpdateModelPrice)
		admin.DELETE("/pricing/models/:id", adminHandler.DeleteModelPrice)

		// 分组倍率管理
		admin.GET("/pricing/groups", adminHandler.ListGroupMultipliers)
		admin.POST("/pricing/groups", adminHandler.CreateGroupMultiplier)
		admin.PUT("/pricing/groups/:id", adminHandler.UpdateGroupMultiplier)
		admin.DELETE("/pricing/groups/:id", adminHandler.DeleteGroupMultiplier)

		// 分组管理
		admin.GET("/groups", adminHandler.ListGroups)
		admin.POST("/groups", adminHandler.CreateGroup)
		admin.PUT("/groups/:id", adminHandler.UpdateGroup)
		admin.DELETE("/groups/:id", adminHandler.DeleteGroup)

		// 日志查询
		admin.GET("/logs", adminHandler.GetLogs)
	}
}

// registerLogRoutes 注册日志管理路由
func (r *APIRouter) registerLogRoutes(v1 *gin.RouterGroup) {
	adminHandler := handler.NewAdminHandler(r.db, r.channelService, r.optionService, r.pricingService, r.groupService)
	logHandler := handler.NewLogHandler(r.db, r.logger,
		service.NewAuditService(r.db, r.logger),
		service.NewErrorLogService(r.db, r.logger),
	)
	logs := v1.Group("/logs")
	logs.Use(middleware.AuthMiddleware(), adminHandler.AdminMiddleware())
	{
		logs.GET("/audit", logHandler.ListAuditLogs)
		logs.GET("/audit/:id", logHandler.GetAuditLogDetail)
		logs.GET("/request", logHandler.ListRequestLogs)
		logs.GET("/error", logHandler.ListErrorLogs)
		logs.GET("/login", logHandler.ListLoginLogs)
		logs.GET("/user/audit", logHandler.GetUserAuditLogs)
		logs.GET("/export", logHandler.ExportLogs)
		logs.GET("/stats", logHandler.GetLogStatistics)
	}
}
