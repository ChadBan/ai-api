package service

import (
	"encoding/json"

	"ai-api/app/internal/logger"
	"ai-api/app/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// InitDomesticModels 初始化国内大模型数据
func InitDomesticModels(db *gorm.DB, log *logger.Logger) {
	log.Info("Initializing domestic LLM models...")

	// 定义国内大模型渠道配置
	domesticChannels := []struct {
		channelType int
		name        string
		baseURL     string
		models      []string
	}{
		{
			channelType: model.ChannelTypeDoubao,
			name:        "豆包官方",
			baseURL:     "https://ark.cn-beijing.volces.com/api/v3",
			models:      []string{"doubao-lite-4k", "doubao-pro-4k", "doubao-lite-32k", "doubao-pro-32k"},
		},
		{
			channelType: model.ChannelTypeAli,
			name:        "阿里通义官方",
			baseURL:     "https://dashscope.aliyuncs.com/compatible-mode",
			models:      []string{"qwen-turbo", "qwen-plus", "qwen-max", "qwen-max-longcontext"},
		},
		{
			channelType: model.ChannelTypeDeepSeek,
			name:        "DeepSeek 官方",
			baseURL:     "https://api.deepseek.com",
			models:      []string{"deepseek-chat", "deepseek-coder"},
		},
		{
			channelType: model.ChannelTypeMiniMax,
			name:        "MiniMax 官方",
			baseURL:     "https://api.minimax.chat/v1",
			models:      []string{"abab6.5-chat", "abab6.5g-chat", "abab6.5t-chat"},
		},
		{
			channelType: model.ChannelTypeZhipu,
			name:        "智谱 AI 官方",
			baseURL:     "https://open.bigmodel.cn/api/paas/v4",
			models:      []string{"glm-4", "glm-4-flash", "glm-3-turbo"},
		},
	}

	// 添加渠道（仅当不存在时）
	for _, cfg := range domesticChannels {
		var count int64
		db.Model(&model.Channel{}).Where("type = ? AND name = ?", cfg.channelType, cfg.name).Count(&count)

		if count == 0 {
			modelsJSON, _ := json.Marshal(cfg.models)
			channel := model.Channel{
				Type:     cfg.channelType,
				Name:     cfg.name,
				BaseURL:  cfg.baseURL,
				APIKey:   "NEED_SETUP", // 需要管理员后续配置
				Status:   0,            // 默认禁用，等待配置 API Key
				Models:   string(modelsJSON),
				Priority: 1,
				Weight:   100,
			}

			if err := db.Create(&channel).Error; err != nil {
				log.Warn("Failed to create domestic channel", logger.String("name", cfg.name), logger.Err(err))
			} else {
				log.Info("Created domestic channel", logger.String("name", cfg.name), logger.Int("type", cfg.channelType))
			}
		}
	}

	// 定义模型数据
	domesticModels := []struct {
		name          string
		displayName   string
		contextWindow int
		maxTokens     int
		inputPrice    float64
		outputPrice   float64
	}{
		// 豆包
		{"doubao-lite-4k", "豆包 Lite 4K", 4096, 2048, 0.0003, 0.0006},
		{"doubao-pro-4k", "豆包 Pro 4K", 4096, 2048, 0.0008, 0.0020},
		{"doubao-lite-32k", "豆包 Lite 32K", 32768, 4096, 0.0006, 0.0012},
		{"doubao-pro-32k", "豆包 Pro 32K", 32768, 4096, 0.0015, 0.0030},
		// 阿里通义
		{"qwen-turbo", "通义千问 Turbo", 8192, 4096, 0.0003, 0.0006},
		{"qwen-plus", "通义千问 Plus", 32768, 8192, 0.0008, 0.0020},
		{"qwen-max", "通义千问 Max", 32768, 8192, 0.0015, 0.0030},
		{"qwen-max-longcontext", "通义千问 Max 长文本", 131072, 8192, 0.0020, 0.0040},
		// DeepSeek
		{"deepseek-chat", "DeepSeek Chat", 32768, 4096, 0.0002, 0.0004},
		{"deepseek-coder", "DeepSeek Coder", 16384, 2048, 0.0002, 0.0004},
		// MiniMax
		{"abab6.5-chat", "MiniMax Abab6.5 Chat", 8192, 4096, 0.0005, 0.0010},
		{"abab6.5g-chat", "MiniMax Abab6.5G Chat", 8192, 4096, 0.0005, 0.0010},
		{"abab6.5t-chat", "MiniMax Abab6.5T Chat", 8192, 4096, 0.0005, 0.0010},
		// 智谱 AI
		{"glm-4", "智谱 GLM-4", 128000, 4096, 0.0010, 0.0020},
		{"glm-4-flash", "智谱 GLM-4 Flash", 8192, 4096, 0.0001, 0.0002},
		{"glm-3-turbo", "智谱 GLM-3 Turbo", 128000, 4096, 0.0005, 0.0010},
	}

	// 添加模型到 models 表
	for _, m := range domesticModels {
		var count int64
		db.Model(&model.Model{}).Where("name = ?", m.name).Count(&count)

		if count == 0 {
			modelItem := model.Model{
				ProviderID:    1, // 默认提供商
				Name:          m.name,
				DisplayName:   m.displayName,
				Type:          "chat",
				ContextWindow: m.contextWindow,
				MaxTokens:     m.maxTokens,
				InputPrice:    decimal.NewFromFloat(m.inputPrice),
				OutputPrice:   decimal.NewFromFloat(m.outputPrice),
				Status:        1,
			}

			if err := db.Create(&modelItem).Error; err != nil {
				log.Warn("Failed to create domestic model", logger.String("name", m.name), logger.Err(err))
			} else {
				log.Info("Created domestic model", logger.String("name", m.name), logger.String("displayName", m.displayName))
			}
		}
	}

	log.Info("Domestic LLM models initialization completed")
}
