package model

import (
	"time"
)

// Group 分组模型
type Group struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	DisplayName string    `gorm:"size:128" json:"display_name"`
	Ratio       float64   `gorm:"default:1.0" json:"ratio"`      // 全局定价倍率
	Models      string    `gorm:"type:text" json:"models"`        // 允许的模型列表 JSON (空=全部)
	QPSLimit    int       `gorm:"default:0" json:"qps_limit"`     // QPS 限制 (0=系统默认)
	DailyLimit  int       `gorm:"default:0" json:"daily_limit"`   // 每日请求限制 (0=无限)
	Status      int       `gorm:"default:1" json:"status"`        // 1=启用, 0=禁用
	Description string    `gorm:"size:256" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Group) TableName() string {
	return "groups"
}
