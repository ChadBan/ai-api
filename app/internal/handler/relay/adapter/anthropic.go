package adapter

import (
	"ai-api/app/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AnthropicAdapter Anthropic Claude 适配器
type AnthropicAdapter struct {
	*BaseAdapter
}

// NewAnthropicAdapter 创建 Anthropic 适配器
func NewAnthropicAdapter(channel *model.Channel) *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseAdapter: NewBaseAdapter(channel),
	}
}

// GetEndpoint 获取 Anthropic API 端点
func (a *AnthropicAdapter) GetEndpoint(modelName string) string {
	return "/v1/messages"
}

// BuildRequest 构建 Anthropic 请求
func (a *AnthropicAdapter) BuildRequest(c *gin.Context, channel *model.Channel, req interface{}) (*http.Request, error) {
	// 验证渠道信息
	if channel.BaseURL == "" {
		return nil, fmt.Errorf("channel base URL is empty")
	}
	if channel.APIKey == "" {
		return nil, fmt.Errorf("channel API key is empty")
	}

	// 解析统一请求格式
	reqMap, err := convertRequestToMap(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request to map: %w", err)
	}

	// 验证必要字段
	if _, ok := reqMap["model"]; !ok {
		return nil, fmt.Errorf("model field is required")
	}
	if _, ok := reqMap["messages"]; !ok {
		return nil, fmt.Errorf("messages field is required")
	}

	// 转换为 Anthropic 格式
	hanthropicReq := map[string]interface{}{
		"model": reqMap["model"],
		"messages": reqMap["messages"],
	}

	// 添加可选参数
	if temp, ok := reqMap["temperature"].(float64); ok {
		hanthropicReq["temperature"] = temp
	}
	if maxTokens, ok := reqMap["max_tokens"].(int); ok {
		hanthropicReq["max_tokens"] = maxTokens
	}
	if topP, ok := reqMap["top_p"].(float64); ok {
		hanthropicReq["top_p"] = topP
	}
	if freqPenalty, ok := reqMap["frequency_penalty"].(float64); ok {
		hanthropicReq["frequency_penalty"] = freqPenalty
	}
	if presPenalty, ok := reqMap["presence_penalty"].(float64); ok {
		hanthropicReq["presence_penalty"] = presPenalty
	}
	if stop, ok := reqMap["stop"].([]string); ok && len(stop) > 0 {
		hanthropicReq["stop_sequences"] = stop
	}
	if stream, ok := reqMap["stream"].(bool); ok {
		hanthropicReq["stream"] = stream
	}

	// 将请求转换为 JSON
	reqBody, err := json.Marshal(hanthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		"POST",
		channel.BaseURL+a.GetEndpoint(""),
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", channel.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	return httpReq, nil
}

// HandleResponse 处理 Anthropic 响应
func (a *AnthropicAdapter) HandleResponse(resp *http.Response) (interface{}, error) {
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("channel returned non-OK status: %d, body: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查响应体是否为空
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	// 解析 Anthropic 响应
	var anthropicResp map[string]interface{}
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	// 检查是否有错误字段
	if errorMsg, ok := anthropicResp["error"].(map[string]interface{}); ok {
		if message, ok := errorMsg["message"].(string); ok {
			return nil, fmt.Errorf("anthropic error: %s", message)
		}
	}

	// 转换为统一格式
	response := map[string]interface{}{
		"id":      anthropicResp["id"],
		"object":  "chat.completion",
		"created": anthropicResp["created"],
		"model":   anthropicResp["model"],
	}

	// 转换 choices
	if content, ok := anthropicResp["content"].([]interface{}); ok && len(content) > 0 {
		if msg, ok := content[0].(map[string]interface{}); ok {
			choices := []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    msg["role"],
						"content": msg["text"],
					},
					"finish_reason": anthropicResp["stop_reason"],
				},
			}
			response["choices"] = choices
		}
	}

	// 转换 usage
	if usage, ok := anthropicResp["usage"].(map[string]interface{}); ok {
		response["usage"] = map[string]interface{}{
			"prompt_tokens":     usage["input_tokens"],
			"completion_tokens": usage["output_tokens"],
			"total_tokens":      usage["input_tokens"].(float64) + usage["output_tokens"].(float64),
		}
	}

	return response, nil
}

// HandleStreamResponse 处理 Anthropic 流式响应
func (a *AnthropicAdapter) HandleStreamResponse(resp *http.Response, c *gin.Context) error {
	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 获取 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil
	}

	// 流式转发
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
		}
		if err != nil {
			break
		}
	}

	return nil
}

// convertRequestToMap 将请求转换为 map
func convertRequestToMap(req interface{}) (map[string]interface{}, error) {
	// 将请求转换为 JSON
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// 解析为 map
	var reqMap map[string]interface{}
	if err := json.Unmarshal(data, &reqMap); err != nil {
		return nil, err
	}

	return reqMap, nil
}
