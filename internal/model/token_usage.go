package model

import (
	"time"
)

// TokenUsageLog Token 使用记录
type TokenUsageLog struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenKey      string    `gorm:"size:128;index;not null" json:"token_key"`
	UserID        int64     `gorm:"index;not null" json:"user_id"`
	Model         string    `gorm:"size:64;index" json:"model"`
	TokensUsed    int       `gorm:"default:0" json:"tokens_used"`
	QuotaDeducted int       `gorm:"default:0" json:"quota_deducted"`
	RequestTime   time.Time `gorm:"autoCreateTime;index" json:"request_time"`
	DurationMs    int       `gorm:"default:0" json:"duration_ms"`
	Success       bool      `gorm:"default:true" json:"success"`
	ErrorMessage  string    `gorm:"size:512" json:"error_message"`
	InputTokens   int       `gorm:"default:0" json:"input_tokens"`
	OutputTokens  int       `gorm:"default:0" json:"output_tokens"`
	ChannelID     int64     `gorm:"index" json:"channel_id"`
}

// TableName 指定表名
func (TokenUsageLog) TableName() string {
	return "token_usage_logs"
}
