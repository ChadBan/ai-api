package relay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RelayHandler 转发处理器
type RelayHandler struct {
	db             *gorm.DB
	channelService *service.ChannelService
	billingService *service.BillingService
	tokenService   *service.TokenService
	logger         *zap.Logger
}

// NewRelayHandler 创建 RelayHandler
func NewRelayHandler(db *gorm.DB, channelService *service.ChannelService, billingService *service.BillingService, tokenService *service.TokenService, logger *zap.Logger) *RelayHandler {
	return &RelayHandler{
		db:             db,
		channelService: channelService,
		billingService: billingService,
		tokenService:   tokenService,
		logger:         logger,
	}
}

// ChatCompletionsRequest 聊天补全请求（OpenAI 格式）
type ChatCompletionsRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	User        string    `json:"user,omitempty"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionsResponse 聊天补全响应
type ChatCompletionsResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice 流式选择项
type StreamChoice struct {
	Index        int     `json:"index"`
	Delta        Message `json:"delta"`
	FinishReason string  `json:"finish_reason"`
}

// ChatCompletions 处理聊天补全请求
func (h *RelayHandler) ChatCompletions(c *gin.Context) {
	startTime := time.Now()

	// 从上下文获取 Token（通过 TokenAuthMiddleware 设置）
	token, ok := h.GetTokenFromContext(c)
	if !ok {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	// 解析请求体
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 检查模型权限
	if err := h.CheckModelPermission(c, req.Model); err != nil {
		h.logger.Warn("model permission denied", zap.String("model", req.Model), zap.Int64("userid", token.UserID))
		common.ErrorResponse(c, http.StatusForbidden, util.Forbidden, err.Error())
		return
	}

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}

	// 转发请求到渠道
	response, _, err := h.forwardToChannelWithBody(c, channel, &req)
	durationMs := formatDuration(startTime)

	var inputTokens, outputTokens int
	success := true
	errorMsg := ""

	if err != nil {
		success = false
		errorMsg = err.Error()
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
	} else {
		// 计算 token 使用量
		inputTokens = estimatePromptTokens(req.Messages)
		outputTokens = estimateCompletionTokens(response.Choices)

		// 更新渠道统计
		responseTime := int64(durationMs)
		h.channelService.UpdateChannelUsedTokens(channel.ID, int64(inputTokens+outputTokens), responseTime)

		// 返回响应
		common.SuccessResponse(c, util.Success, response)
	}

	// 扣减配额并记录使用（异步）
	go func() {
		h.DeductQuotaAndRecord(c, token, req.Model, inputTokens, outputTokens, channel.ID, durationMs, success, errorMsg)
	}()
}

// PlaygroundChatCompletions 处理 Playground 聊天补全请求（使用 JWT 用户认证）
func (h *RelayHandler) PlaygroundChatCompletions(c *gin.Context) {
	// 从 JWT 认证中间件获取用户 ID
	userID, exists := c.Get("userid")
	if !exists {
		h.logger.Error("PlaygroundChatCompletions: userid not found in context")
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "userid not set by middleware")
		return
	}

	// Parse userid as string then convert to int64
	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("PlaygroundChatCompletions: userid is not a string")
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid userid format")
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to parse userid", zap.String("userid", userIDStr), zap.Error(err))
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid userid")
		return
	}
	h.logger.Info("PlaygroundChatCompletions: user authenticated", zap.Int64("userid", realUserID))

	// 解析请求体
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to bind JSON", zap.Error(err))
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}
	h.logger.Info("PlaygroundChatCompletions: request parsed", zap.String("model", req.Model), zap.Int("messages", len(req.Messages)), zap.Bool("stream", req.Stream))

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to select channel", zap.String("model", req.Model), zap.Error(err))
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}
	h.logger.Info("PlaygroundChatCompletions: channel selected", zap.Int64("channel_id", channel.ID), zap.String("channel_name", channel.Name))

	// 设置 SSE 头（流式响应）
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "streaming unsupported")
		return
	}

	// 构建请求体
	reqBody, err := json.Marshal(req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, err.Error())
		return
	}

	// 根据渠道类型构建请求
	httpReq, headers := h.buildHTTPRequest(c, channel, reqBody, req.Model)
	if httpReq == nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to build request")
		return
	}

	// 设置请求头
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		common.ErrorResponse(c, resp.StatusCode, util.InternalServerError, string(body))
		return
	}

	// 流式转发
	buf := make([]byte, 1024)
	totalTokens := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			// 写入响应
			if _, writeErr := c.Writer.Write(chunk); writeErr != nil {
				h.logger.Error("write to client failed", zap.Error(writeErr))
				break
			}
			flusher.Flush()

			// 简单估算 token 数量（用于计费）
			totalTokens += n / 4 // 粗略估算：4 字节 ≈ 1 token
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			h.logger.Error("read from channel failed", zap.Error(err))
			break
		}
	}

	// 记录使用统计（简化版）
	h.logger.Info("Playground chat completed",
		zap.Int64("userid", realUserID),
		zap.String("model", req.Model),
		zap.Int("estimated_tokens", totalTokens),
		zap.Int64("channel_id", channel.ID),
	)
}

func (h *RelayHandler) buildHTTPRequest(c *gin.Context, channel *model.Channel, reqBody []byte, modelName string) (*http.Request, map[string]string) {
	var httpReq *http.Request
	var headers map[string]string
	var err error

	switch channel.Type {
	case 1: // OpenAI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 2: // Anthropic Claude
		hanthropicReq := map[string]interface{}{
			"model":       modelName,
			"messages":    nil, // 需要从 reqBody 解析
			"temperature": 0.7,
			"max_tokens":  4096,
		}
		reqBody, _ = json.Marshal(hanthropicReq)
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/messages",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         channel.APIKey,
			"anthropic-version": "2023-06-01",
		}
	case 3: // Azure OpenAI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type": "application/json",
			"api-key":      channel.APIKey,
		}
	case 4: // Google Gemini
		geminiReq := map[string]interface{}{
			"model":             modelName,
			"messages":          nil,
			"temperature":       0.7,
			"max_output_tokens": 4096,
		}
		reqBody, _ = json.Marshal(geminiReq)
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/models/"+modelName+":generateContent",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	// 国内大模型
	case 14: // 豆包
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/api/v3/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 15: // 阿里通义
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/compatible-mode/v1/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 16: // DeepSeek
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 17: // MiniMax
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/text/chatcompletion_v2",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 18: // 智谱 AI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v4/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	default: // 默认 OpenAI 格式
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	}

	if err != nil {
		return nil, nil
	}
	return httpReq, headers
}

// ChatCompletionsStream 处理流式聊天补全请求
func (h *RelayHandler) ChatCompletionsStream(c *gin.Context) {
	// 获取用户信息
	userID, _ := c.Get("userid")
	if userID == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	// Parse userid as string then convert to int64
	userIDStr, ok := userID.(string)
	if !ok {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid userid format")
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid userid")
		return
	}

	// 解析请求体
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "streaming unsupported")
		return
	}

	// 转发流式请求到渠道
	resp, err := h.forwardStreamToChannel(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		return
	}
	defer resp.Body.Close()

	// 读取并转发流式响应
	reader := bufio.NewReader(resp.Body)
	totalCompletionTokens := 0

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		lineStr := string(line)

		// 跳过空行和注释
		if len(lineStr) < 6 || lineStr[:6] != "data: " {
			continue
		}

		// 解析数据
		data := lineStr[6:]
		if data == "[DONE]\n" {
			c.Writer.WriteString(lineStr)
			flusher.Flush()
			break
		}

		// 转发给客户端
		c.Writer.WriteString(lineStr)
		flusher.Flush()

		// 估算 completion tokens
		totalCompletionTokens += estimateCompletionTokensFromStream(data)
	}

	// 异步扣费
	go func() {
		promptTokens := estimatePromptTokens(req.Messages)
		h.billingService.ConsumeQuota(
			c.Request.Context(),
			realUserID,
			0,
			req.Model,
			promptTokens,
			totalCompletionTokens,
			channel.ID,
			channel.Name,
		)
	}()
}

// forwardToChannel 转发请求到渠道
func (h *RelayHandler) forwardToChannel(c *gin.Context, channel *model.Channel, req *ChatCompletionsRequest) (*ChatCompletionsResponse, error) {
	// 根据渠道类型构建请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var httpReq *http.Request
	var headers map[string]string

	switch channel.Type {
	case 1: // OpenAI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	case 2: // Anthropic Claude
		// 转换为 Anthropic 格式
		hanthropicReq := map[string]interface{}{
			"model":       req.Model,
			"messages":    req.Messages,
			"temperature": req.Temperature,
			"max_tokens":  req.MaxTokens,
		}
		reqBody, _ = json.Marshal(hanthropicReq)
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/messages",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         channel.APIKey,
			"anthropic-version": "2023-06-01",
		}
	case 3: // Azure OpenAI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type": "application/json",
			"api-key":      channel.APIKey,
		}
	case 4: // Google Gemini
		// 转换为 Gemini 格式
		geminiReq := map[string]interface{}{
			"model":             req.Model,
			"messages":          req.Messages,
			"temperature":       req.Temperature,
			"max_output_tokens": req.MaxTokens,
		}
		reqBody, _ = json.Marshal(geminiReq)
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/models/"+req.Model+":generateContent",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	// 国内大模型（兼容 OpenAI 格式）
	case 14: // 豆包 (ByteDance)
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/api/v3/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	case 15: // 阿里通义
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/compatible-mode/v1/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	case 16: // DeepSeek
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	case 17: // MiniMax
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/text/chatcompletion_v2",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	case 18: // 智谱 AI
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v4/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}

	default: // 默认为 OpenAI 格式
		httpReq, err = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
		}
	}

	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理不同渠道的响应格式
	if channel.Type == 4 { // Google Gemini
		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usageMetadata"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			return nil, err
		}

		// 转换为 OpenAI 格式
		response := &ChatCompletionsResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []Choice{},
			Usage: Usage{
				PromptTokens:     geminiResp.Usage.PromptTokens,
				CompletionTokens: geminiResp.Usage.CompletionTokens,
				TotalTokens:      geminiResp.Usage.PromptTokens + geminiResp.Usage.CompletionTokens,
			},
		}

		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			content := ""
			for _, part := range geminiResp.Candidates[0].Content.Parts {
				content += part.Text
			}
			response.Choices = append(response.Choices, Choice{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			})
		}

		return response, nil
	} else {
		var response ChatCompletionsResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, err
		}

		// 转换 Anthropic 响应格式
		if channel.Type == 2 {
			// 这里可以添加响应格式转换逻辑
		}

		return &response, nil
	}
}

// forwardStreamToChannel 转发流式请求到渠道
func (h *RelayHandler) forwardStreamToChannel(c *gin.Context, channel *model.Channel, req *ChatCompletionsRequest) (*http.Response, error) {
	req.Stream = true
	var reqBody []byte
	var httpReq *http.Request
	var headers map[string]string

	switch channel.Type {
	case 1: // OpenAI
		reqBody, _ = json.Marshal(req)
		httpReq, _ = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
			"Accept":        "text/event-stream",
		}
	case 2: // Anthropic Claude
		// 转换为 Anthropic 格式
		anthropicReq := map[string]interface{}{
			"model":       req.Model,
			"messages":    req.Messages,
			"temperature": req.Temperature,
			"max_tokens":  req.MaxTokens,
			"stream":      true,
		}
		reqBody, _ = json.Marshal(anthropicReq)
		httpReq, _ = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/messages",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         channel.APIKey,
			"anthropic-version": "2023-06-01",
			"Accept":            "text/event-stream",
		}
	case 3: // Azure OpenAI
		reqBody, _ = json.Marshal(req)
		httpReq, _ = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type": "application/json",
			"api-key":      channel.APIKey,
			"Accept":       "text/event-stream",
		}
	case 4: // Google Gemini
		// 转换为 Gemini 格式
		geminiReq := map[string]interface{}{
			"model":             req.Model,
			"messages":          req.Messages,
			"temperature":       req.Temperature,
			"max_output_tokens": req.MaxTokens,
			"stream":            true,
		}
		reqBody, _ = json.Marshal(geminiReq)
		httpReq, _ = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/v1/models/"+req.Model+":generateContent",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
			"Accept":        "text/event-stream",
		}
	default: // 默认为 OpenAI 格式
		reqBody, _ = json.Marshal(req)
		httpReq, _ = http.NewRequestWithContext(
			c.Request.Context(),
			"POST",
			channel.BaseURL+"/chat/completions",
			bytes.NewReader(reqBody),
		)
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + channel.APIKey,
			"Accept":        "text/event-stream",
		}
	}

	// 设置请求头
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(httpReq)
}

// estimatePromptTokens 估算 prompt tokens（简单估算）
func estimatePromptTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// 简单估算：每 4 个字符约 1 个 token
		total += len(msg.Content) / 4
	}
	return total + len(messages)*4 // 每条消息额外消耗
}

// estimateCompletionTokens 估算 completion tokens
func estimateCompletionTokens(choices []Choice) int {
	if len(choices) == 0 {
		return 0
	}
	total := 0
	for _, choice := range choices {
		total += len(choice.Message.Content) / 4
	}
	return total
}

// estimateCompletionTokensFromStream 从流式数据估算 tokens
func estimateCompletionTokensFromStream(data string) int {
	// 简单估算
	return len(data) / 4
}

// EmbeddingsRequest Embeddings 请求
type EmbeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
	User  string `json:"user,omitempty"`
}

// EmbeddingsResponse Embeddings 响应
type EmbeddingsResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingUsage  `json:"usage"`
}

// EmbeddingData Embedding 数据
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// EmbeddingUsage Embedding 使用情况
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Embeddings 处理 embeddings 请求
func (h *RelayHandler) Embeddings(c *gin.Context) {
	startTime := time.Now()

	// 获取用户信息
	userID, _ := c.Get("userid")
	if userID == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	// Parse userid as string then convert to int64
	userIDStr, ok := userID.(string)
	if !ok {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid userid format")
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid userid")
		return
	}

	// 解析请求体
	var req EmbeddingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}

	// 转发请求到渠道
	response, err := h.forwardEmbeddingsToChannel(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		return
	}

	// 计算 token 使用量
	promptTokens := len(req.Input) / 4

	// 扣费
	go func() {
		if err := h.billingService.ConsumeQuota(
			c.Request.Context(),
			realUserID,
			0,
			req.Model,
			promptTokens,
			0,
			channel.ID,
			channel.Name,
		); err != nil {
			fmt.Printf("failed to consume quota: %v\n", err)
		}
	}()

	// 更新渠道统计
	responseTime := time.Since(startTime).Milliseconds()
	h.channelService.UpdateChannelUsedTokens(channel.ID, int64(promptTokens), responseTime)

	// 返回响应
	common.SuccessResponse(c, util.Success, response)
}

// forwardEmbeddingsToChannel 转发 embeddings 请求到渠道
func (h *RelayHandler) forwardEmbeddingsToChannel(c *gin.Context, channel *model.Channel, req *EmbeddingsRequest) (*EmbeddingsResponse, error) {
	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		"POST",
		channel.BaseURL+"/embeddings",
		io.NopCloser(bytes.NewReader(reqBody)),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response EmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ImagesGenerationRequest 图片生成请求
type ImagesGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
}

// ImagesGenerationResponse 图片生成响应
type ImagesGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData 图片数据
type ImageData struct {
	URL           string `json:"url,omitempty"`
	Base64        string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImagesGenerations 处理图片生成请求
func (h *RelayHandler) ImagesGenerations(c *gin.Context) {
	startTime := time.Now()

	// 获取用户信息
	userID, _ := c.Get("userid")
	if userID == nil {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized)
		return
	}

	// Parse userid as string then convert to int64
	userIDStr, ok := userID.(string)
	if !ok {
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid userid format")
		return
	}
	realUserID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid userid")
		return
	}

	// 解析请求体
	var req ImagesGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}

	// 设置默认值
	if req.N == 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}

	// 转发请求到渠道
	response, err := h.forwardImagesToChannel(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		return
	}

	// 扣费
	go func() {
		if err := h.billingService.ConsumeQuota(
			c.Request.Context(),
			realUserID,
			0,
			req.Model,
			0,
			0,
			channel.ID,
			channel.Name,
		); err != nil {
			fmt.Printf("failed to consume quota: %v\n", err)
		}
	}()

	// 更新渠道统计
	responseTime := time.Since(startTime).Milliseconds()
	h.channelService.UpdateChannelUsedTokens(channel.ID, int64(req.N*100), responseTime)

	// 返回响应
	common.SuccessResponse(c, util.Success, response)
}

// forwardImagesToChannel 转发图片生成请求到渠道
func (h *RelayHandler) forwardImagesToChannel(c *gin.Context, channel *model.Channel, req *ImagesGenerationRequest) (*ImagesGenerationResponse, error) {
	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		"POST",
		channel.BaseURL+"/images/generations",
		io.NopCloser(bytes.NewReader(reqBody)),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	client := &http.Client{Timeout: 60 * time.Second} // 图片生成时间较长
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response ImagesGenerationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// TokenAuthMiddleware Token 认证中间件
func (h *RelayHandler) TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "missing authorization header")
			c.Abort()
			return
		}

		// 提取 Token
		tokenKey := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenKey == authHeader {
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "invalid authorization format")
			c.Abort()
			return
		}

		// 验证 Token
		token, err := h.tokenService.ValidateToken(c.Request.Context(), tokenKey)
		if err != nil {
			h.logger.Warn("token validation failed", zap.String("token", tokenKey), zap.Error(err))
			common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, err.Error())
			c.Abort()
			return
		}

		// 将 Token 和用户 ID 存入上下文（使用 userid 作为 key，存储为字符串）
		c.Set("token", token)
		c.Set("userid", fmt.Sprintf("%d", token.UserID))
		c.Set("token_key", token.Key)

		c.Next()
	}
}

// CheckModelPermission 检查模型权限（从上下文中获取 Token）
func (h *RelayHandler) CheckModelPermission(c *gin.Context, modelName string) error {
	tokenVal, exists := c.Get("token")
	if !exists {
		return fmt.Errorf("token not found in context")
	}

	token, ok := tokenVal.(*model.Token)
	if !ok {
		return fmt.Errorf("invalid token type")
	}

	return h.tokenService.CheckModelPermission(token, modelName)
}

// DeductQuotaAndRecord 扣减配额并记录使用
func (h *RelayHandler) DeductQuotaAndRecord(c *gin.Context, token *model.Token, modelName string, inputTokens, outputTokens int, channelID int64, durationMs int, success bool, errorMsg string) {
	ctx := c.Request.Context()

	// 计算配额
	quota, err := h.tokenService.CalculateQuota(modelName, inputTokens, outputTokens, token.Ratio)
	if err != nil {
		h.logger.Error("failed to calculate quota", zap.Error(err))
		quota = 0
	}

	// 扣减配额
	if quota > 0 {
		if err := h.tokenService.DeductTokenQuota(ctx, token, quota); err != nil {
			h.logger.Error("failed to deduct token quota", zap.Error(err))
			// 扣减失败，尝试从用户余额扣除
			userID := token.UserID
			if err := h.billingService.DeductUserBalance(ctx, userID, quota, "token_insufficient"); err != nil {
				h.logger.Error("failed to deduct user balance", zap.Error(err))
			}
		}
	}

	// 记录使用日志
	usageLog := &model.TokenUsageLog{
		TokenKey:      token.Key,
		UserID:        token.UserID,
		Model:         modelName,
		TokensUsed:    inputTokens + outputTokens,
		QuotaDeducted: quota,
		DurationMs:    durationMs,
		Success:       success,
		ErrorMessage:  errorMsg,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		ChannelID:     channelID,
	}

	if err := h.tokenService.RecordTokenUsage(ctx, usageLog); err != nil {
		h.logger.Error("failed to record token usage", zap.Error(err))
	}
}

// GetTokenFromContext 从上下文获取 Token
func (h *RelayHandler) GetTokenFromContext(c *gin.Context) (*model.Token, bool) {
	tokenVal, exists := c.Get("token")
	if !exists {
		return nil, false
	}

	token, ok := tokenVal.(*model.Token)
	return token, ok
}

// parseBearerToken 解析 Bearer Token
func parseBearerToken(authHeader string) (string, bool) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", false
	}
	return parts[1], true
}

// estimateTokens 估算 tokens 数量（用于流式响应）
func estimateTokens(text string) int {
	// 简单估算：每 4 个字符约等于 1 个 token
	return len([]rune(text)) / 4
}

// parseStreamResponse 解析流式响应，返回 tokens 统计
func parseStreamResponse(data []byte) (inputTokens, outputTokens int, err error) {
	lines := bytes.Split(data, []byte("\n"))
	outputText := ""

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		lineStr := strings.TrimSpace(string(line))
		if !strings.HasPrefix(lineStr, "data:") {
			continue
		}
		dataStr := strings.TrimPrefix(lineStr, "data: ")
		if dataStr == "[DONE]" {
			continue
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			outputText += chunk.Choices[0].Delta.Content
		}
	}

	outputTokens = estimateTokens(outputText)
	return 0, outputTokens, nil
}

// parseNonStreamResponse 解析非流式响应
func parseNonStreamResponse(data []byte) (inputTokens, outputTokens int, err error) {
	var response struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return 0, 0, err
	}

	return response.Usage.PromptTokens, response.Usage.CompletionTokens, nil
}

// convertMessageToMap 转换消息为 map
func convertMessageToMap(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

// countMessageTokens 简单计算消息的 tokens 数量
func countMessageTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateTokens(msg.Content)
	}
	return total
}

// formatDuration 格式化耗时
func formatDuration(start time.Time) int {
	return int(time.Since(start).Milliseconds())
}

// parseErrorResponse 解析错误响应
func parseErrorResponse(data []byte) string {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &errResp); err != nil {
		return string(data)
	}

	return errResp.Error.Message
}

// updateTokenAccessTime 更新 Token 最后访问时间
func (h *RelayHandler) updateTokenAccessTime(token *model.Token) {
	now := time.Now()
	h.db.Model(token).Update("accessed_time", &now)
}

// checkAndAutoRenewToken 检查并自动续期 Token
func (h *RelayHandler) checkAndAutoRenewToken(c *gin.Context, token *model.Token, requiredQuota int) error {
	if token.HasQuota(requiredQuota) {
		return nil // 配额充足
	}

	// TODO: 实现自动续费逻辑
	// 1. 检查用户是否开启了自动续费
	// 2. 从用户余额扣除到 Token
	// 3. 记录续费日志

	return fmt.Errorf("insufficient token quota")
}

// forwardToChannelWithBody 转发请求到渠道并返回响应体
func (h *RelayHandler) forwardToChannelWithBody(c *gin.Context, channel *model.Channel, req *ChatCompletionsRequest) (*ChatCompletionsResponse, []byte, error) {
	// 构建请求体
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := channel.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		"POST",
		url,
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	// 发送请求
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, responseBody, fmt.Errorf("channel returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response ChatCompletionsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, responseBody, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, responseBody, nil
}
