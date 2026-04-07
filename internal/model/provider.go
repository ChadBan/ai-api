package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Provider 模型提供商
type Provider struct {
	ID          int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string  `gorm:"uniqueIndex;size:64" json:"name"`
	DisplayName string  `gorm:"size:128" json:"display_name"`
	BaseURL     string  `gorm:"size:256" json:"base_url"`
	APIKey      string  `gorm:"size:256" json:"-"`
	Status      int     `gorm:"default:1" json:"status"`
	Priority    int     `gorm:"default:0" json:"priority"`
	Weight      int     `gorm:"default:100" json:"weight"`
	Config      string  `gorm:"type:text" json:"config"` // JSON 字符串
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Provider) TableName() string {
	return "providers"
}

// Model 模型定义
type Model struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID    int64           `gorm:"index" json:"provider_id"`
	Name          string          `gorm:"size:128;uniqueIndex:idx_provider_model" json:"name"`
	DisplayName   string          `gorm:"size:128" json:"display_name"`
	Type          string          `gorm:"size:32" json:"type"`
	ContextWindow int             `json:"context_window"`
	MaxTokens     int             `json:"max_tokens"`
	InputPrice    decimal.Decimal `gorm:"type:decimal(10,6)" json:"input_price"`
	OutputPrice   decimal.Decimal `gorm:"type:decimal(10,6)" json:"output_price"`
	Status        int             `gorm:"default:1" json:"status"`
	Capabilities  string          `gorm:"type:text" json:"capabilities"` // JSON 字符串
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (Model) TableName() string {
	return "models"
}
