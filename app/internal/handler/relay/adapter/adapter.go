package adapter

import (
	"ai-api/app/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChannelAdapter 渠道适配器接口
type ChannelAdapter interface {
	// BuildRequest 构建 HTTP 请求
	BuildRequest(c *gin.Context, channel *model.Channel, req interface{}) (*http.Request, error)
	
	// HandleResponse 处理非流式响应
	HandleResponse(resp *http.Response) (interface{}, error)
	
	// HandleStreamResponse 处理流式响应
	HandleStreamResponse(resp *http.Response, c *gin.Context) error
	
	// GetEndpoint 获取 API 端点
	GetEndpoint(modelName string) string
}

// BaseAdapter 基础适配器实现
type BaseAdapter struct {
	Channel *model.Channel
}

// NewBaseAdapter 创建基础适配器
func NewBaseAdapter(channel *model.Channel) *BaseAdapter {
	return &BaseAdapter{
		Channel: channel,
	}
}

// GetEndpoint 获取默认 API 端点
func (a *BaseAdapter) GetEndpoint(modelName string) string {
	return "/chat/completions"
}
