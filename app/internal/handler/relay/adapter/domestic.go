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

// DomesticAdapter 国内模型适配器
type DomesticAdapter struct {
	*BaseAdapter
	Endpoint string
}

// NewDomesticAdapter 创建国内模型适配器
func NewDomesticAdapter(channel *model.Channel, endpoint string) *DomesticAdapter {
	return &DomesticAdapter{
		BaseAdapter: NewBaseAdapter(channel),
		Endpoint:    endpoint,
	}
}

// GetEndpoint 获取国内模型 API 端点
func (a *DomesticAdapter) GetEndpoint(modelName string) string {
	return a.Endpoint
}

// BuildRequest 构建国内模型请求
func (a *DomesticAdapter) BuildRequest(c *gin.Context, channel *model.Channel, req interface{}) (*http.Request, error) {
	// 验证渠道信息
	if channel.BaseURL == "" {
		return nil, fmt.Errorf("channel base URL is empty")
	}
	if channel.APIKey == "" {
		return nil, fmt.Errorf("channel API key is empty")
	}
	if a.Endpoint == "" {
		return nil, fmt.Errorf("adapter endpoint is empty")
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

// HandleResponse 处理国内模型响应
func (a *DomesticAdapter) HandleResponse(resp *http.Response) (interface{}, error) {
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

	// 检查是否有错误字段
	if respMap, ok := response.(map[string]interface{}); ok {
		if errorMsg, ok := respMap["error"].(map[string]interface{}); ok {
			if message, ok := errorMsg["message"].(string); ok {
				return nil, fmt.Errorf("domestic model error: %s", message)
			}
		}
	}

	return response, nil
}

// HandleStreamResponse 处理国内模型流式响应
func (a *DomesticAdapter) HandleStreamResponse(resp *http.Response, c *gin.Context) error {
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
