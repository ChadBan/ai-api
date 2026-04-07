package model

import (
	"time"
)

// SystemOption 系统设置模型 (Key-Value EAV 模式)
type SystemOption struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Type        string    `gorm:"size:32;default:string" json:"type"` // bool/int/float/string/json
	Description string    `gorm:"size:256" json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SystemOption) TableName() string {
	return "system_options"
}

// 预定义设置键常量
const (
	OptRegisterEnabled  = "register_enabled"
	OptDefaultQuota     = "default_quota"
	OptPreConsumedQuota = "pre_consumed_quota"
	OptTopUpLink        = "top_up_link"
	OptMfaRequired      = "mfa_required"
	OptPrice            = "price"
	OptDisplayInCurrency = "display_in_currency"
	OptDisplayName      = "display_name"
	OptDrawNotify       = "draw_notify"
	OptCriticalNotify   = "critical_notify"
	OptGroupRatio       = "group_ratio"
	OptModelRatio       = "model_ratio"
)
