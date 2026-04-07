package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"gorm.io/gorm"
)

// ChannelService 渠道服务
type ChannelService struct {
	db            *gorm.DB
	channels      []model.Channel
	channelMu     sync.RWMutex
	lastLoadTime  time.Time
	cacheDuration time.Duration
}

// NewChannelService 创建渠道服务
func NewChannelService(db *gorm.DB) *ChannelService {
	cs := &ChannelService{
		db:            db,
		cacheDuration: 5 * time.Minute, // 缓存 5 分钟
	}

	// 启动时加载渠道
	go cs.loadChannels()

	return cs
}

// loadChannels 从数据库加载渠道
func (s *ChannelService) loadChannels() error {
	s.channelMu.Lock()
	defer s.channelMu.Unlock()

	var channels []model.Channel
	if err := s.db.Where("status = ?", 1).Find(&channels).Error; err != nil {
		return err
	}

	s.channels = channels
	s.lastLoadTime = time.Now()
	return nil
}

// getChannels 获取可用的渠道列表（带缓存）
func (s *ChannelService) getChannels() ([]model.Channel, error) {
	s.channelMu.RLock()
	defer s.channelMu.RUnlock()

	// 检查缓存是否过期
	if time.Since(s.lastLoadTime) > s.cacheDuration {
		// 异步重新加载
		go s.loadChannels()
	}

	return s.channels, nil
}

// SelectChannel 根据模型名称选择最优渠道
func (s *ChannelService) SelectChannel(modelName string) (*model.Channel, error) {
	channels, err := s.getChannels()
	if err != nil {
		return nil, err
	}

	if len(channels) == 0 {
		return nil, errors.New("no available channels")
	}

	// 过滤支持该模型的渠道
	var availableChannels []model.Channel
	for _, ch := range channels {
		// 跳过 API Key 无效的渠道
		if ch.APIKey == "" || ch.APIKey == "NEED_SETUP" || strings.TrimSpace(ch.APIKey) == "" {
			continue
		}
		if s.supportModel(ch, modelName) {
			availableChannels = append(availableChannels, ch)
		}
	}

	if len(availableChannels) == 0 {
		return nil, fmt.Errorf("no channel supports model: %s", modelName)
	}

	// 按优先级和权重选择渠道
	return s.selectByPriorityAndWeight(availableChannels), nil
}

// supportModel 检查渠道是否支持指定模型
func (s *ChannelService) supportModel(channel model.Channel, modelName string) bool {
	// 如果没有配置模型列表，默认支持所有模型
	if channel.Models == "" {
		return true
	}

	// 解析 Models JSON 字段
	var models []string
	if err := json.Unmarshal([]byte(channel.Models), &models); err != nil {
		// 如果解析失败，默认支持所有模型
		return true
	}

	// 如果没有配置模型列表，默认支持所有模型
	if len(models) == 0 {
		return true
	}

	// 检查是否在模型列表中
	for _, m := range models {
		if m == modelName || m == "*" {
			return true
		}
		// 支持通配符匹配（如 gpt-* 匹配 gpt-4, gpt-3.5-turbo）
		if len(m) > 0 && m[len(m)-1] == '*' {
			prefix := m[:len(m)-1]
			if len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
				return true
			}
		}
	}

	return false
}

// selectByPriorityAndWeight 按优先级和权重选择渠道
func (s *ChannelService) selectByPriorityAndWeight(channels []model.Channel) *model.Channel {
	if len(channels) == 1 {
		return &channels[0]
	}

	// 找到最小优先级（数字越小优先级越高）
	minPriority := channels[0].Priority
	for _, ch := range channels {
		if ch.Priority < minPriority {
			minPriority = ch.Priority
		}
	}

	// 筛选出最高优先级的渠道
	var priorityChannels []model.Channel
	for _, ch := range channels {
		if ch.Priority == minPriority {
			priorityChannels = append(priorityChannels, ch)
		}
	}

	// 如果只有一个，直接返回
	if len(priorityChannels) == 1 {
		return &priorityChannels[0]
	}

	// 计算总权重
	totalWeight := 0
	for _, ch := range priorityChannels {
		if ch.Weight > 0 {
			totalWeight += ch.Weight
		}
	}

	// 如果都没有设置权重，随机选择
	if totalWeight == 0 {
		return &priorityChannels[rand.Intn(len(priorityChannels))]
	}

	// 按权重随机选择
	r := rand.Intn(totalWeight)
	current := 0
	for _, ch := range priorityChannels {
		current += ch.Weight
		if r < current {
			return &ch
		}
	}

	return &priorityChannels[len(priorityChannels)-1]
}

// UpdateChannelUsedTokens 更新渠道已用 token 数
func (s *ChannelService) UpdateChannelUsedTokens(channelId int64, tokens int64, responseTime int64) error {
	return s.db.Model(&model.Channel{}).Where("id = ?", channelId).UpdateColumns(map[string]interface{}{
		"used_tokens":   gorm.Expr("used_tokens + ?", tokens),
		"response_time": gorm.Expr("(response_time * 9 + ?) / 10", responseTime), // 移动平均
	}).Error
}

// TestChannel 测试渠道连通性
func (s *ChannelService) TestChannel(ctx context.Context, channel *model.Channel) error {
	if channel == nil {
		return fmt.Errorf("channel is nil")
	}

	// 根据渠道类型选择不同的测试方式
	switch channel.Type {
	case 1: // OpenAI
		return s.testOpenAIChannel(ctx, channel)
	case 2: // Anthropic
		return s.testAnthropicChannel(ctx, channel)
	case 3: // Azure
		return s.testAzureChannel(ctx, channel)
	case 4: // Google Gemini
		return s.testGeminiChannel(ctx, channel)
	default:
		// 默认使用 OpenAI 格式测试
		return s.testOpenAIChannel(ctx, channel)
	}
}

// testOpenAIChannel 测试 OpenAI 格式渠道
func (s *ChannelService) testOpenAIChannel(ctx context.Context, channel *model.Channel) error {
	// 构建测试请求
	testReq := map[string]interface{}{
		"model": channel.TestModel,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 1,
	}

	if testReq["model"] == "" {
		testReq["model"] = "gpt-3.5-turbo"
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return err
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		channel.BaseURL+"/chat/completions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	// 发送请求（超时 10 秒）
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response failed: %w", err)
	}

	// 验证响应格式
	if choices, ok := result["choices"].([]interface{}); !ok || len(choices) == 0 {
		return fmt.Errorf("invalid response format: no choices")
	}

	// 更新渠道状态为正常
	s.db.Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]interface{}{
		"status":         1,
		"last_test_time": time.Now(),
	})

	return nil
}

// testAnthropicChannel 测试 Anthropic 渠道
func (s *ChannelService) testAnthropicChannel(ctx context.Context, channel *model.Channel) error {
	// 构建测试请求
	testReq := map[string]interface{}{
		"model": channel.TestModel,
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 1,
	}

	if testReq["model"] == "" {
		testReq["model"] = "claude-3-haiku-20240307"
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		channel.BaseURL+"/v1/messages",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", channel.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// 更新渠道状态
	s.db.Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]interface{}{
		"status":         1,
		"last_test_time": time.Now(),
	})

	return nil
}

// testAzureChannel 测试 Azure 渠道
func (s *ChannelService) testAzureChannel(ctx context.Context, channel *model.Channel) error {
	// Azure 的 BaseURL 通常已经包含 deployment 信息
	testURL := channel.BaseURL
	if !strings.Contains(testURL, "/chat/completions") {
		testURL = testURL + "/chat/completions"
	}

	testReq := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hi"},
		},
		"max_tokens": 1,
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		testURL,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", channel.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// 更新渠道状态
	s.db.Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]interface{}{
		"status":         1,
		"last_test_time": time.Now(),
	})

	return nil
}

// testGeminiChannel 测试 Google Gemini 渠道
func (s *ChannelService) testGeminiChannel(ctx context.Context, channel *model.Channel) error {
	// 构建测试请求
	testReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": "Hi"},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 1,
		},
	}

	testModel := channel.TestModel
	if testModel == "" {
		testModel = "gemini-pro"
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		channel.BaseURL+"/v1/models/"+testModel+":generateContent",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// 更新渠道状态
	s.db.Model(&model.Channel{}).Where("id = ?", channel.ID).Updates(map[string]interface{}{
		"status":         1,
		"last_test_time": time.Now(),
	})

	return nil
}

// RefreshChannels 手动刷新渠道缓存
func (s *ChannelService) RefreshChannels() error {
	return s.loadChannels()
}

// ChannelHealthStatus 渠道健康状态
type ChannelHealthStatus struct {
	ChannelID    int64   `json:"channel_id"`
	Name         string  `json:"name"`
	Status       int     `json:"status"`
	ResponseTime int64   `json:"response_time"`
	LastTestTime string  `json:"last_test_time"`
	SuccessRate  float64 `json:"success_rate"`
}

// GetAllHealthStatus 获取所有渠道的健康状态
func (s *ChannelService) GetAllHealthStatus() []ChannelHealthStatus {
	var channels []model.Channel
	s.db.Order("priority ASC").Find(&channels)

	result := make([]ChannelHealthStatus, 0, len(channels))
	for _, ch := range channels {
		h := ChannelHealthStatus{
			ChannelID:    ch.ID,
			Name:         ch.Name,
			Status:       ch.Status,
			ResponseTime: ch.ResponseTime,
		}
		if !ch.LastTestTime.IsZero() {
			h.LastTestTime = ch.LastTestTime.Format(time.RFC3339)
		}
		// 简单健康度：status=1 且 response_time < 5000ms 视为健康
		if ch.Status == 1 && ch.ResponseTime > 0 && ch.ResponseTime < 5000 {
			h.SuccessRate = 1.0
		} else if ch.Status == 1 {
			h.SuccessRate = 0.5
		} else {
			h.SuccessRate = 0.0
		}
		result = append(result, h)
	}

	return result
}

// GetLatestModels 获取最新模型列表
func (s *ChannelService) GetLatestModels(channelType string) ([]string, error) {
	// 这里可以根据channelType从官方API获取最新模型列表
	// 目前使用预定义的模型列表，实际项目中可以替换为真实的API调用

	models := map[string][]string{
		"1": { // OpenAI
			"gpt-3.5-turbo",
			"gpt-3.5-turbo-0125",
			"gpt-3.5-turbo-16k",
			"gpt-4",
			"gpt-4-0125-preview",
			"gpt-4-turbo",
			"gpt-4-turbo-2024-04-09",
			"gpt-4o",
			"gpt-4o-2024-05-13",
			"gpt-4o-mini",
			"gpt-4o-mini-2024-07-18",
		},
		"2": { // Anthropic
			"claude-3-haiku-20240307",
			"claude-3-sonnet-20240229",
			"claude-3-opus-20240229",
			"claude-3.5-sonnet-20240620",
		},
		"3": { // Azure
			"gpt-35-turbo",
			"gpt-35-turbo-16k",
			"gpt-4",
			"gpt-4-32k",
		},
		"4": { // Google Gemini
			"gemini-pro",
			"gemini-1.0-pro",
			"gemini-1.5-pro",
			"gemini-1.5-pro-001",
			"gemini-1.5-flash",
			"gemini-1.5-flash-001",
			"gemini-2.0-pro",
		},
		"14": { // 豆包
			"Doubao-pro-128k",
			"Doubao-pro-32k",
			"Doubao-pro-4k",
			"Doubao-lite-128k",
			"Doubao-lite-32k",
			"Doubao-lite-4k",
			"Doubao-embedding",
			"doubao-seedream-4-0-250828",
			"seedream-4-0-250828",
			"doubao-seedance-1-0-pro-250528",
			"seedance-1-0-pro-250528",
			"doubao-seed-1-6-thinking-250715",
			"seed-1-6-thinking-250715",
			"doubao-seed-2-0-code-preview-260215",
		},
		"15": { // 阿里通义
			"qwen-turbo",
			"qwen-plus",
			"qwen-max",
			"qwen-2.5-turbo",
			"qwen-2.5-plus",
			"qwen-2.5-max",
		},
		"16": { // DeepSeek
			"deepseek-chat",
			"deepseek-llm",
			"DeepSeek-V3.1",
		},
		"17": { // MiniMax
			"abab5.5-chat",
			"abab6-chat",
			"abab6.5-chat",
		},
		"18": { // 智谱 AI
			"chatglm3-6b",
			"chatglm3-6b-32k",
			"glm-4",
			"glm-4-flash",
			"glm-4-plus",
		},
	}

	if modelsList, ok := models[channelType]; ok {
		return modelsList, nil
	}

	// 默认返回所有模型
	allModels := []string{}
	for _, m := range models {
		allModels = append(allModels, m...)
	}

	return allModels, nil
}

// SearchModels 搜索模型
func (s *ChannelService) SearchModels(query, channelType string) ([]string, error) {
	// 获取所有模型
	allModels, err := s.GetLatestModels(channelType)
	if err != nil {
		return nil, err
	}

	// 过滤包含查询字符串的模型
	var results []string
	for _, model := range allModels {
		if strings.Contains(strings.ToLower(model), strings.ToLower(query)) {
			results = append(results, model)
		}
	}

	return results, nil
}
