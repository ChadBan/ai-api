package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ai-api/app/internal/config"
	applogger "ai-api/app/internal/logger"

	"ai-api/app/internal/handler/middleware"
	"ai-api/app/internal/model"
	"ai-api/app/internal/repository"
	"ai-api/app/internal/service"
	"ai-api/app/router"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger, err := initLogger(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("starting AI Model Scheduler",
		applogger.String("version", "0.1.0"),
		applogger.String("mode", cfg.Server.Mode),
	)

	// 初始化数据库
	db, gormDB, err := initDatabase(logger, &cfg.Database)
	if err != nil {
		logger.Fatal("failed to initialize database", applogger.Err(err))
	}
	defer db.Close()

	// 初始化 Redis（可选）
	redisClient, err := initRedis(logger, &cfg.Redis)
	if err != nil {
		logger.Warn("failed to connect to redis, continuing without redis", applogger.Err(err))
	} else {
		defer redisClient.Close()
	}

	// 初始化服务
	services, err := initServices(logger, gormDB)
	if err != nil {
		logger.Fatal("failed to initialize services", applogger.Err(err))
	}

	// 初始化路由器
	r := initRouter(logger, redisClient, gormDB, services, cfg.JWT.Secret)

	// 启动定时任务服务
	services.schedulerService.Start()

	// 启动 Prometheus 指标端点
	if cfg.Monitoring.Enabled {
		startPrometheusServer(logger, &cfg.Monitoring)
	}

	// 启动 HTTP 服务器
	srv := startServer(logger, r, &cfg.Server)

	// 设置优雅关闭
	setupGracefulShutdown(logger, srv, db, redisClient, services.schedulerService)

	// 等待服务器关闭
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("failed to start server", applogger.Err(err))
	}
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// 获取当前工作目录
		workDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		// 尝试不同的配置文件路径
		possiblePaths := []string{
			filepath.Join(workDir, "configs/config.default.yaml"),
			filepath.Join(workDir, "../configs/config.default.yaml"),
			filepath.Join(workDir, "../../configs/config.default.yaml"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return nil, fmt.Errorf("could not find config file in any of the possible paths")
		}
	}

	return config.Load(configPath)
}

// initLogger 初始化日志
func initLogger(cfg config.LogConfig) (*applogger.Logger, error) {
	return applogger.NewLogger(cfg.Level, cfg.Format, cfg.Output, cfg.FilePath)
}

// initDatabase 初始化数据库
func initDatabase(logger *applogger.Logger, dbConfig *config.DatabaseConfig) (*repository.Database, *gorm.DB, error) {
	db, err := repository.NewDatabase(dbConfig)
	if err != nil {
		return nil, nil, err
	}

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.Provider{},
		&model.Model{},
		&model.UsageRecord{},
		&model.DailyUsage{},
		&model.Channel{},
		&model.Token{},
		&model.TokenUsageLog{}, // 新增：Token 使用记录
		&model.Redemption{},
		&model.Invitation{},
		&model.Billing{},
		&model.UserBalance{},
		// 日志表
		&model.AuditLog{},
		&model.RequestLog{},
		&model.ErrorLog{},
		&model.LoginLog{},
		// 系统配置
		&model.SystemOption{},
		// 定价与分组
		&model.ModelPrice{},
		&model.GroupPriceMultiplier{},
		&model.Group{},
		// 充值
		&model.TopUp{},
	); err != nil {
		return nil, nil, err
	}

	logger.Info("database migration completed")

	// 获取底层的 *gorm.DB
	gormDB := db.GetDB()

	return db, gormDB, nil
}

// initRedis 初始化 Redis
func initRedis(logger *applogger.Logger, redisConfig *config.RedisConfig) (*repository.RedisClient, error) {
	if redisConfig.Host == "" {
		return nil, nil
	}

	redisClient, err := repository.NewRedisClient(redisConfig)
	if err != nil {
		return nil, err
	}

	logger.Info("redis connected")
	return redisClient, nil
}

// Services 服务集合
type Services struct {
	channelService   *service.ChannelService
	optionService    *service.OptionService
	pricingService   *service.PricingService
	groupService     *service.GroupService
	billingService   *service.BillingService
	schedulerService *service.SchedulerService
}

// initServices 初始化服务
func initServices(logger *applogger.Logger, gormDB *gorm.DB) (*Services, error) {
	channelService := service.NewChannelService(gormDB)
	optionService := service.NewOptionService(gormDB)
	pricingService := service.NewPricingService(gormDB)
	groupService := service.NewGroupService(gormDB)
	billingService := service.NewBillingService(gormDB, pricingService)
	schedulerService := service.NewSchedulerService(gormDB, logger)

	// 初始化国内大模型数据
	service.InitDomesticModels(gormDB, logger)

	// 初始化定时任务
	schedulerService.InitTasks()

	return &Services{
		channelService:   channelService,
		optionService:    optionService,
		pricingService:   pricingService,
		groupService:     groupService,
		billingService:   billingService,
		schedulerService: schedulerService,
	}, nil
}

// initRouter 初始化路由器
func initRouter(logger *applogger.Logger, redisClient *repository.RedisClient, gormDB *gorm.DB, services *Services, jwtSecret string) *gin.Engine {
	// 设置 Gin 模式
	if gin.Mode() != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由器
	r := gin.New()
	r.Use(middleware.LoggingMiddleware(logger))
	r.Use(middleware.CORSMiddleware())

	if redisClient != nil {
		r.Use(middleware.RateLimitMiddleware(redisClient.Client))
	}

	// 注册 API 路由（中间件在路由级别单独配置）
	apiRouter := router.NewAPIRouter(gormDB, logger, services.channelService, services.billingService, services.optionService, services.pricingService, services.groupService, jwtSecret)
	apiRouter.RegisterRoutes(r)

	return r
}

// startPrometheusServer 启动 Prometheus 指标端点
func startPrometheusServer(logger *applogger.Logger, monitoringConfig *config.MonitoringConfig) {
	go func() {
		mux := http.NewServeMux()
		mux.Handle(monitoringConfig.MetricsPath, promhttp.Handler())
		addr := fmt.Sprintf(":%d", monitoringConfig.PrometheusPort)
		logger.Info("prometheus metrics server starting",
			applogger.String("address", addr),
			applogger.String("path", monitoringConfig.MetricsPath),
		)
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Error("prometheus server failed", applogger.Err(err))
		}
	}()
}

// startServer 启动 HTTP 服务器
func startServer(logger *applogger.Logger, router *gin.Engine, serverConfig *config.ServerConfig) *http.Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", serverConfig.Port),
		Handler:      router,
		ReadTimeout:  serverConfig.ReadTimeout,
		WriteTimeout: serverConfig.WriteTimeout,
		IdleTimeout:  serverConfig.IdleTimeout,
	}

	logger.Info("server starting",
		applogger.Int("port", serverConfig.Port),
		applogger.String("mode", serverConfig.Mode),
	)

	return srv
}

// setupGracefulShutdown 设置优雅关闭
func setupGracefulShutdown(logger *applogger.Logger, srv *http.Server, db *repository.Database, redisClient *repository.RedisClient, schedulerService *service.SchedulerService) {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 停止定时任务服务
		schedulerService.Stop()

		// 关闭 HTTP 服务器
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("server forced to shutdown", applogger.Err(err))
		}

		// 关闭数据库连接
		db.Close()

		// 关闭 Redis 连接
		if redisClient != nil {
			redisClient.Close()
		}

		logger.Info("server exited")
	}()
}
