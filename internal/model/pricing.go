package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ModelPrice 模型定价
type ModelPrice struct {
	ID          int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelName   string          `gorm:"uniqueIndex;size:128;not null" json:"model_name"`
	InputRatio  float64         `gorm:"default:1.0" json:"input_ratio"`   // 输入 token 倍率
	OutputRatio float64         `gorm:"default:1.0" json:"output_ratio"`  // 输出 token 倍率
	FixedCost   decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"fixed_cost"` // 固定费用
	Enabled     bool            `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (ModelPrice) TableName() string {
	return "model_prices"
}

// GroupPriceMultiplier 分组价格倍率
type GroupPriceMultiplier struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupName  string    `gorm:"uniqueIndex:idx_group_model;size:64;not null" json:"group_name"`
	ModelName  string    `gorm:"uniqueIndex:idx_group_model;size:128;not null" json:"model_name"` // * 表示所有模型
	Multiplier float64   `gorm:"default:1.0" json:"multiplier"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (GroupPriceMultiplier) TableName() string {
	return "group_price_multipliers"
}
