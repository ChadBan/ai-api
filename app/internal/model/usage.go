package model

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// UsageRecord 使用记录
type UsageRecord struct {
	ID           int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64           `gorm:"index" json:"user_id"`
	APIKeyID     int64           `gorm:"index" json:"api_key_id"`
	ModelID      int64           `gorm:"index" json:"model_id"`
	ModelName    string          `gorm:"size:128" json:"model_name"`
	ProviderName string          `gorm:"size:64" json:"provider_name"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	TotalTokens  int             `json:"total_tokens"`
	Cost         decimal.Decimal `gorm:"type:decimal(10,4)" json:"cost"`
	Currency     string          `gorm:"size:16;default:USD" json:"currency"`
	RequestType  string          `gorm:"size:32" json:"request_type"`
	StatusCode   int             `json:"status_code"`
	ErrorMessage string          `gorm:"size:512" json:"error_message"`
	Duration     int64           `json:"duration_ms"`
	Metadata     datatypes.JSON  `json:"metadata"`
	CreatedAt    time.Time       `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (UsageRecord) TableName() string {
	return "usage_records"
}

// DailyUsage 每日用量汇总
type DailyUsage struct {
	ID                int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int64           `gorm:"uniqueIndex:user_date" json:"user_id"`
	Date              time.Time       `gorm:"uniqueIndex:user_date;type:date" json:"date"`
	TotalRequests     int             `json:"total_requests"`
	TotalInputTokens  int             `json:"total_input_tokens"`
	TotalOutputTokens int             `json:"total_output_tokens"`
	TotalTokens       int             `json:"total_tokens"`
	TotalCost         decimal.Decimal `gorm:"type:decimal(10,4)" json:"total_cost"`
	ModelBreakdown    datatypes.JSON  `json:"model_breakdown"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (DailyUsage) TableName() string {
	return "daily_usage"
}
