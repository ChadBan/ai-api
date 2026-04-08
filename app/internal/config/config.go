package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 完整配置结构
type Config struct {
	Server         ServerConfig         `mapstructure:"server"`
	JWT            JWTConfig            `mapstructure:"jwt"`
	Database       DatabaseConfig       `mapstructure:"database"`
	Redis          RedisConfig          `mapstructure:"redis"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
	ModelProviders []ProviderConfig     `mapstructure:"model_providers"`
	Billing        BillingConfig        `mapstructure:"billing"`
	Subscription   SubscriptionConfig   `mapstructure:"subscription"`
	Log            LogConfig            `mapstructure:"log"`
	Monitoring     MonitoringConfig     `mapstructure:"monitoring"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           int           `mapstructure:"port"`
	Mode           string        `mapstructure:"mode"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret       string        `mapstructure:"secret"`
	Expire       time.Duration `mapstructure:"expire"`
	RefreshExpire time.Duration `mapstructure:"refresh_expire"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	GlobalQPS         int  `mapstructure:"global_qps"`
	UserQPS           int  `mapstructure:"user_qps"`
	UserDailyRequests int  `mapstructure:"user_daily_requests"`
	APIKeyQPS         int  `mapstructure:"api_key_qps"`
}

// ProviderConfig 模型提供商配置
type ProviderConfig struct {
	Name        string `mapstructure:"name"`
	DisplayName string `mapstructure:"display_name"`
	BaseURL     string `mapstructure:"base_url"`
	APIKey      string `mapstructure:"api_key"`
	Enabled     bool   `mapstructure:"enabled"`
	Priority    int    `mapstructure:"priority"`
	Weight      int    `mapstructure:"weight"`
}

// BillingConfig 计费配置
type BillingConfig struct {
	Currency          string  `mapstructure:"currency"`
	DefaultInputPrice float64 `mapstructure:"default_input_price"`
	DefaultOutputPrice float64 `mapstructure:"default_output_price"`
}

// SubscriptionConfig 订阅配置
type SubscriptionConfig struct {
	FreeTier TierConfig `mapstructure:"free_tier"`
	ProTier  TierConfig `mapstructure:"pro_tier"`
}

// TierConfig 套餐配置
type TierConfig struct {
	DailyTokenLimit int     `mapstructure:"daily_token_limit"`
	MonthlyBudget   float64 `mapstructure:"monthly_budget"`
	MaxQPS          int     `mapstructure:"max_qps"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	PrometheusPort  int    `mapstructure:"prometheus_port"`
	MetricsPath     string `mapstructure:"metrics_path"`
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	GlobalConfig = &config
	return &config, nil
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	if GlobalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return GlobalConfig
}
