package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ai-api/app/internal/common"
	"ai-api/app/internal/handler/relay/adapter"
	"ai-api/app/internal/logger"
	"ai-api/app/internal/util"

	"ai-api/app/internal/model"
	"ai-api/app/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RelayHandler 转发处理器
type RelayHandler struct {
	db             *gorm.DB
	channelService *service.ChannelService
	billingService *service.BillingService
	tokenService   *service.TokenService
	logger         *logger.Logger
	adapterFactory *adapter.AdapterFactory
}

// NewRelayHandler 创建 RelayHandler
func NewRelayHandler(db *gorm.DB, channelService *service.ChannelService, billingService *service.BillingService, tokenService *service.TokenService, logger *logger.Logger) *RelayHandler {
	return &RelayHandler{
		db:             db,
		channelService: channelService,
		billingService: billingService,
		tokenService:   tokenService,
		logger:         logger,
		adapterFactory: adapter.NewAdapterFactory(),
	}
}

// ChatCompletionsRequest 聊天补全请求（OpenAI 格式）
type ChatCompletionsRequest struct {
	Model            string    `json:"model"`
	Messages         []Message `json:"messages"`
	Temperature      float64   `json:"temperature,omitempty"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	Stream           bool      `json:"stream,omitempty"`
	TopP             float64   `json:"top_p,omitempty"`
	User             string    `json:"user,omitempty"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64   `json:"presence_penalty,omitempty"`
	Stop             []string  `json:"stop,omitempty"`
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
		h.logger.Warn("model permission denied", logger.String("model", req.Model), logger.Int64("userid", token.UserID))
		common.ErrorResponse(c, http.StatusForbidden, util.Forbidden, err.Error())
		return
	}

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}

	// 创建适配器
	adap := h.adapterFactory.CreateAdapter(channel)

	// 构建请求
	httpReq, err := adap.BuildRequest(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to build request: %v", err))
		return
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		return
	}
	defer resp.Body.Close()

	durationMs := formatDuration(startTime)

	var inputTokens, outputTokens int
	success := true
	errorMsg := ""

	if req.Stream {
		// 处理流式响应
		if err := adap.HandleStreamResponse(resp, c); err != nil {
			h.logger.Error("Handle stream response failed", logger.Err(err))
			success = false
			errorMsg = err.Error()
		}
	} else {
		// 处理非流式响应
		response, err := adap.HandleResponse(resp)
		if err != nil {
			success = false
			errorMsg = err.Error()
			common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("channel error: %v", err))
		} else {
			// 计算 token 使用量
			inputTokens = estimatePromptTokens(req.Messages)
			// 尝试从响应中提取 outputTokens
			if respMap, ok := response.(map[string]interface{}); ok {
				if usage, ok := respMap["usage"].(map[string]interface{}); ok {
					if completionTokens, ok := usage["completion_tokens"].(float64); ok {
						outputTokens = int(completionTokens)
					}
				}

				// 只返回 message.content
				if choices, ok := respMap["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if message, ok := choice["message"].(map[string]interface{}); ok {
							if content, ok := message["content"].(string); ok {
								// 更新渠道统计
								responseTime := int64(durationMs)
								h.channelService.UpdateChannelUsedTokens(channel.ID, int64(inputTokens+outputTokens), responseTime)

								// 返回响应
								common.SuccessResponse(c, util.Success, content)
							}
						}
					}
				} else {
					// 如果无法提取 content，则返回完整响应
					// 更新渠道统计
					responseTime := int64(durationMs)
					h.channelService.UpdateChannelUsedTokens(channel.ID, int64(inputTokens+outputTokens), responseTime)

					// 返回响应
					common.SuccessResponse(c, util.Success, response)
				}
			} else {
				// 如果响应不是 map，则返回完整响应
				// 更新渠道统计
				responseTime := int64(durationMs)
				h.channelService.UpdateChannelUsedTokens(channel.ID, int64(inputTokens+outputTokens), responseTime)

				// 返回响应
				common.SuccessResponse(c, util.Success, response)
			}
		}
	}

	// 扣减配额并记录使用（异步）
	go func() {
		h.DeductQuotaAndRecord(c, token, req.Model, inputTokens, outputTokens, channel.ID, durationMs, success, errorMsg)
	}()
}

// PlaygroundChatCompletions 处理 Playground 聊天补全请求（使用 JWT 用户认证）
func (h *RelayHandler) PlaygroundChatCompletions(c *gin.Context) {
	// 从 JWT 认证中间件获取用户 ID
	userID, _ := c.Get("userid")
	userId := userID.(string)
	if userId == "" {
		h.logger.Error("PlaygroundChatCompletions: userid not found in context")
		common.ErrorResponse(c, http.StatusUnauthorized, util.Unauthorized, "userid not set by middleware")
		return
	}

	realUserID, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to parse userid", logger.String("userid", userId), logger.Err(err))
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, "invalid userid")
		return
	}
	h.logger.Info("PlaygroundChatCompletions: user authenticated", logger.Int64("userid", realUserID))

	// 解析请求体
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to bind JSON", logger.Err(err))
		common.ErrorResponse(c, http.StatusBadRequest, util.InvalidParam, err.Error())
		return
	}
	h.logger.Info("PlaygroundChatCompletions: request parsed", logger.String("model", req.Model), logger.Int("messages", len(req.Messages)), logger.Bool("stream", req.Stream))

	// 选择渠道
	channel, err := h.channelService.SelectChannel(req.Model)
	if err != nil {
		h.logger.Error("PlaygroundChatCompletions: failed to select channel", logger.String("model", req.Model), logger.Err(err))
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to select channel: %v", err))
		return
	}
	h.logger.Info("PlaygroundChatCompletions: channel selected", logger.Int64("channel_id", channel.ID), logger.String("channel_name", channel.Name))

	// 创建适配器
	adap := h.adapterFactory.CreateAdapter(channel)

	// 构建请求
	httpReq, err := adap.BuildRequest(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to build request: %v", err))
		return
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

	// 处理流式响应
	if err := adap.HandleStreamResponse(resp, c); err != nil {
		h.logger.Error("Handle stream response failed", logger.Err(err))
	}

	// 记录使用统计
	h.logger.Info("Playground chat completed",
		logger.Int64("userid", realUserID),
		logger.String("model", req.Model),
		logger.Int64("channel_id", channel.ID),
	)
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

	// 创建适配器
	adap := h.adapterFactory.CreateAdapter(channel)

	// 构建请求
	httpReq, err := adap.BuildRequest(c, channel, &req)
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, fmt.Sprintf("failed to build request: %v", err))
		return
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
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

	// 处理流式响应
	if err := adap.HandleStreamResponse(resp, c); err != nil {
		h.logger.Error("Handle stream response failed", logger.Err(err))
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
			0, // 流式响应中难以准确计算 token 数
			channel.ID,
			channel.Name,
		)
	}()
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
			h.logger.Warn("token validation failed", logger.String("token", tokenKey), logger.Err(err))
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
		h.logger.Error("failed to calculate quota", logger.Err(err))
		quota = 0
	}

	// 扣减配额
	if quota > 0 {
		if err := h.tokenService.DeductTokenQuota(ctx, token, quota); err != nil {
			h.logger.Error("failed to deduct token quota", logger.Err(err))
			// 扣减失败，尝试从用户余额扣除
			userID := token.UserID
			if err := h.billingService.DeductUserBalance(ctx, userID, quota, "token_insufficient"); err != nil {
				h.logger.Error("failed to deduct user balance", logger.Err(err))
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
		h.logger.Error("failed to record token usage", logger.Err(err))
	}

	// 记录 UsageRecord
	statusCode := 200
	if !success {
		statusCode = 500
	}

	channel, err := h.channelService.GetChannelByID(channelID)
	if err != nil {
		h.logger.Error("failed to get channel", logger.Err(err))
		return
	}

	usageRecord := &model.UsageRecord{
		UserID:       token.UserID,
		APIKeyID:     token.ID,
		ModelName:    modelName,
		ProviderName: channel.Name,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		RequestType:  "chat_completions",
		StatusCode:   statusCode,
		ErrorMessage: errorMsg,
		Duration:     int64(durationMs),
		CreatedAt:    time.Now(),
	}

	if err := h.db.Create(usageRecord).Error; err != nil {
		h.logger.Error("failed to create usage record", logger.Err(err))
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
