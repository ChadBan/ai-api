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

// GeminiAdapter Google Gemini 适配器
type GeminiAdapter struct {
	*BaseAdapter
}

// NewGeminiAdapter 创建 Google Gemini 适配器
func NewGeminiAdapter(channel *model.Channel) *GeminiAdapter {
	return &GeminiAdapter{
		BaseAdapter: NewBaseAdapter(channel),
	}
}

// GetEndpoint 获取 Gemini API 端点
func (a *GeminiAdapter) GetEndpoint(modelName string) string {
	return "/v1/models/" + modelName + ":generateContent"
}

// BuildRequest 构建 Gemini 请求
func (a *GeminiAdapter) BuildRequest(c *gin.Context, channel *model.Channel, req interface{}) (*http.Request, error) {
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

	modelName, ok := reqMap["model"].(string)
	if !ok {
		return nil, fmt.Errorf("model field must be a string")
	}

	// 转换为 Gemini 格式
	geminiReq := map[string]interface{}{
		"model": modelName,
	}

	// 转换 messages
	if messages, ok := reqMap["messages"].([]interface{}); ok {
		geminiMessages := []map[string]interface{}{}
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				geminiMessage := map[string]interface{}{
					"role": msgMap["role"],
					"parts": []map[string]interface{}{
						{
							"text": msgMap["content"],
						},
					},
				}
				geminiMessages = append(geminiMessages, geminiMessage)
			}
		}
		geminiReq["messages"] = geminiMessages
	}

	// 添加可选参数
	if temp, ok := reqMap["temperature"].(float64); ok {
		geminiReq["temperature"] = temp
	}
	if maxTokens, ok := reqMap["max_tokens"].(int); ok {
		geminiReq["max_output_tokens"] = maxTokens
	}
	if topP, ok := reqMap["top_p"].(float64); ok {
		geminiReq["top_p"] = topP
	}
	if stream, ok := reqMap["stream"].(bool); ok {
		geminiReq["stream"] = stream
	}

	// 将请求转换为 JSON
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		"POST",
		channel.BaseURL+a.GetEndpoint(modelName),
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	return httpReq, nil
}

// HandleResponse 处理 Gemini 响应
func (a *GeminiAdapter) HandleResponse(resp *http.Response) (interface{}, error) {
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

	// 解析 Gemini 响应
	var geminiResp map[string]interface{}
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	// 检查是否有错误字段
	if errorMsg, ok := geminiResp["error"].(map[string]interface{}); ok {
		if message, ok := errorMsg["message"].(string); ok {
			return nil, fmt.Errorf("gemini error: %s", message)
		}
	}

	// 转换为统一格式
	response := map[string]interface{}{
		"id":      geminiResp["name"],
		"object":  "chat.completion",
		"created": geminiResp["createTime"],
		"model":   geminiResp["model"],
	}

	// 转换 choices
	if candidates, ok := geminiResp["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					contentText := ""
					for _, part := range parts {
						if partMap, ok := part.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								contentText += text
							}
						}
					}
					choices := []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]interface{}{
								"role":    "assistant",
								"content": contentText,
							},
							"finish_reason": candidate["finishReason"],
						},
					}
					response["choices"] = choices
				}
			}
		}
	}

	// 转换 usage
	if usage, ok := geminiResp["usageMetadata"].(map[string]interface{}); ok {
		response["usage"] = map[string]interface{}{
			"prompt_tokens":     usage["promptTokens"],
			"completion_tokens": usage["candidatesTokens"],
			"total_tokens":      usage["totalTokens"],
		}
	}

	return response, nil
}

// HandleStreamResponse 处理 Gemini 流式响应
func (a *GeminiAdapter) HandleStreamResponse(resp *http.Response, c *gin.Context) error {
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
