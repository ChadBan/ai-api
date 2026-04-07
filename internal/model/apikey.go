package model

import (
	"time"

	"gorm.io/gorm"
)

// APIKey API Key 模型
type APIKey struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64          `gorm:"index" json:"user_id"`
	Key         string         `gorm:"uniqueIndex;size:128" json:"key"`
	Name        string         `gorm:"size:128" json:"name"`
	Status      int            `gorm:"default:1" json:"status"` // 1:启用 0:禁用
	ExpiredAt   *time.Time     `json:"expired_at"`
	LastUsedAt  *time.Time     `json:"last_used_at"`
	Permissions string         `gorm:"type:text" json:"permissions"` // JSON 字符串
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
