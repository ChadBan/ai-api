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

// OpenAIAdapter OpenAI 适配器
type OpenAIAdapter struct {
	*BaseAdapter
}

// NewOpenAIAdapter 创建 OpenAI 适配器
func NewOpenAIAdapter(channel *model.Channel) *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseAdapter: NewBaseAdapter(channel),
	}
}

// BuildRequest 构建 OpenAI 请求
func (a *OpenAIAdapter) BuildRequest(c *gin.Context, channel *model.Channel, req interface{}) (*http.Request, error) {
	// 验证渠道信息
	if channel.BaseURL == "" {
		return nil, fmt.Errorf("channel base URL is empty")
	}
	if channel.APIKey == "" {
		return nil, fmt.Errorf("channel API key is empty")
	}

	// 将请求转换为 JSON
	reqBody, err := json.Marshal(req)
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
	httpReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	return httpReq, nil
}

// HandleResponse 处理 OpenAI 响应
func (a *OpenAIAdapter) HandleResponse(resp *http.Response) (interface{}, error) {
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

	// 解析响应
	var response interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	return response, nil
}

// HandleStreamResponse 处理 OpenAI 流式响应
func (a *OpenAIAdapter) HandleStreamResponse(resp *http.Response, c *gin.Context) error {
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
