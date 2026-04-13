package adapter

import (
	"ai-api/app/internal/model"
)

// AdapterFactory 适配器工厂
type AdapterFactory struct{}

// NewAdapterFactory 创建适配器工厂
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{}
}

// CreateAdapter 根据渠道类型创建适配器
func (f *AdapterFactory) CreateAdapter(channel *model.Channel) ChannelAdapter {
	switch channel.Type {
	case model.ChannelTypeOpenAI, model.ChannelTypeAzure, model.ChannelTypeOpenAISB, model.ChannelTypeOHMyGPT, model.ChannelTypeCustom, model.ChannelTypeAesop, model.ChannelTypeProxy, model.ChannelTypeAPI2D, model.ChannelTypeAIProxy, model.ChannelTypeFastGPT, model.ChannelTypeCloudflare:
		return NewOpenAIAdapter(channel)
	case model.ChannelTypeAnthropic:
		return NewAnthropicAdapter(channel)
	case model.ChannelTypeCloseAI:
		return NewGeminiAdapter(channel)
	case model.ChannelTypeDoubao:
		return NewDomesticAdapter(channel, "/chat/completions")
	case model.ChannelTypeAli:
		return NewDomesticAdapter(channel, "/compatible-mode/v1/chat/completions")
	case model.ChannelTypeDeepSeek:
		return NewDomesticAdapter(channel, "/chat/completions")
	case model.ChannelTypeMiniMax:
		return NewDomesticAdapter(channel, "/v1/text/chatcompletion_v2")
	case model.ChannelTypeZhipu:
		return NewDomesticAdapter(channel, "/v4/chat/completions")
	default:
		return NewOpenAIAdapter(channel)
	}
}
