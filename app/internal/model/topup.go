package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// TopUp 充值订单模型
type TopUp struct {
	ID              int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID         string          `gorm:"uniqueIndex;size:64;not null" json:"order_id"`
	UserID          int64           `gorm:"index" json:"user_id"`
	Amount          int             `json:"amount"`                                             // 充值额度 (积分)
	Money           decimal.Decimal `gorm:"type:decimal(10,2)" json:"money"`                    // 实际金额
	Currency        string          `gorm:"size:16;default:USD" json:"currency"`
	PaymentMethod   string          `gorm:"size:32" json:"payment_method"`                      // stripe/epay
	TradeNo         string          `gorm:"size:128" json:"trade_no"`                           // 外部交易号
	Status          int             `gorm:"default:0;index" json:"status"`                      // 0=待支付,1=成功,2=失败,3=过期,4=退款
	ReturnURL       string          `gorm:"size:512" json:"return_url"`
	WebhookPayload  string          `gorm:"type:text" json:"-"`                                 // 原始回调数据
	IdempotencyKey  string          `gorm:"uniqueIndex;size:128" json:"-"`                      // 幂等键
	ExpireAt        time.Time       `json:"expire_at"`
	PaidAt          *time.Time      `json:"paid_at"`
	CreatedAt       time.Time       `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (TopUp) TableName() string {
	return "top_ups"
}

// TopUp 状态常量
const (
	TopUpStatusPending  = 0
	TopUpStatusSuccess  = 1
	TopUpStatusFailed   = 2
	TopUpStatusExpired  = 3
	TopUpStatusRefunded = 4
)
