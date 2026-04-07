package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string    `gorm:"uniqueIndex;size:128" json:"email"`
	PasswordHash string    `gorm:"size:256" json:"-"`
	Name         string    `gorm:"size:64" json:"name"`
	Avatar       string    `gorm:"size:256" json:"avatar"`
	Status       int       `gorm:"default:1" json:"status"` // 1:正常 0:禁用
	Role         string    `gorm:"size:32;default:user" json:"role"`
	Tier         string    `gorm:"size:32;default:free" json:"tier"` // free/pro/enterprise
	Group        string    `gorm:"size:64;default:default" json:"group"`
	GitHubID     string    `gorm:"size:64;index" json:"github_id"`
	OIDCSubject  string    `gorm:"size:256;index" json:"oidc_subject"`
	MFASecret    string    `gorm:"size:256" json:"-"`
	MFAEnabled   bool      `gorm:"default:false" json:"mfa_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
