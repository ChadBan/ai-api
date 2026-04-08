package model

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 渠道类型常量
const (
	ChannelTypeOpenAI     = 1  // OpenAI
	ChannelTypeAnthropic  = 2  // Anthropic
	ChannelTypeAzure      = 3  // Azure
	ChannelTypeCloseAI    = 4  // CloseAI
	ChannelTypeOpenAISB   = 5  // OpenAI-SB
	ChannelTypeOHMyGPT    = 6  // OHMyGPT
	ChannelTypeCustom     = 7  // Custom
	ChannelTypeAesop      = 8  // Aesop
	ChannelTypeProxy      = 9  // Proxy
	ChannelTypeAPI2D      = 10 // API2D
	ChannelTypeAIProxy    = 11 // AIProxy
	ChannelTypeFastGPT    = 12 // FastGPT
	ChannelTypeCloudflare = 13 // Cloudflare
	ChannelTypeDoubao     = 14 // 豆包 (ByteDance)
	ChannelTypeAli        = 15 // 阿里通义
	ChannelTypeDeepSeek   = 16 // DeepSeek
	ChannelTypeMiniMax    = 17 // MiniMax
	ChannelTypeZhipu      = 18 // 智谱 AI
)

// Channel 渠道模型（对应 new-api 的 channels 表）
type Channel struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Type         int            `gorm:"default:1" json:"type"` // 渠道类型：1=OpenAI, 2=Anthropic, 3=Azure, 4=CloseAI, 5=OpenAI-SB, 6=OHMyGPT, 7=Custom, 8=Aesop, 9=Proxy, 10=API2D, 11=AIProxy, 12=FastGPT, 13=Cloudflare, 14=豆包 (Doubao), 15=阿里通义 (Ali), 16=DeepSeek, 17=MiniMax, 18=智谱 AI (Zhipu)
	Name         string         `gorm:"size:128;index" json:"name"`
	DisplayName  string         `gorm:"size:128" json:"display_name"`
	BaseURL      string         `gorm:"size:256" json:"base_url"`
	APIKey       string         `gorm:"size:512" json:"api_key"` // 加密存储
	TestModel    string         `gorm:"size:128" json:"test_model"`
	Models       string         `gorm:"type:text" json:"models"`        // 支持的模型列表（JSON 字符串）
	Status       int            `gorm:"default:1" json:"status"`        // 1=启用，0=禁用
	Priority     int            `gorm:"default:0" json:"priority"`      // 优先级（数字越小优先级越高）
	Weight       int            `gorm:"default:0" json:"weight"`        // 权重（用于负载均衡）
	UsedTokens   int64          `gorm:"default:0" json:"used_tokens"`   // 已用 token 数
	ResponseTime int64          `gorm:"default:0" json:"response_time"` // 平均响应时间（毫秒）
	LastTestTime *time.Time     `json:"last_test_time"`
	Config       string         `gorm:"type:text" json:"config"` // 额外配置（JSON 字符串）
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// TableName 指定表名
func (Channel) TableName() string {
	return "channels"
}

// Token 令牌模型（对应 new-api 的 tokens 表）
type Token struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64          `gorm:"index;not null" json:"user_id"`
	Key            string         `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Name           string         `gorm:"size:128" json:"name"`
	Status         int            `gorm:"default:1;index" json:"status"` // 1:启用 0:禁用
	CreatedTime    time.Time      `gorm:"autoCreateTime" json:"created_time"`
	AccessedTime   *time.Time     `json:"accessed_time"`                  // 最后访问时间
	ExpiredTime    *time.Time     `gorm:"index" json:"expired_time"`      // 过期时间（nil 表示永不过期）
	RemainQuota    int            `gorm:"default:-1" json:"remain_quota"` // 剩余配额（-1 表示无限）
	UnlimitedQuota bool           `gorm:"default:false" json:"unlimited_quota"`
	UsedQuota      int            `gorm:"default:0" json:"used_quota"`
	ModelLimit     string         `gorm:"type:text;default:'[]'" json:"model_limit"`   // JSON 数组，允许的模型列表
	Ratio          float64        `gorm:"type:decimal(10,6);default:1.0" json:"ratio"` // 汇率倍率
	Group          string         `gorm:"size:64;default:'default';index" json:"group"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Token) TableName() string {
	return "tokens"
}

// Redemption 兑换码模型（对应 new-api的 redemptions 表）
type Redemption struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId       int64          `gorm:"index" json:"user_id"`
	Key          string         `gorm:"uniqueIndex;size:512" json:"key"`
	Status       int            `gorm:"default:1" json:"status"`        // 1=未使用，0=已使用，-1=已禁用
	Quota        int            `gorm:"default:0" json:"quota"`         // 额度（积分）
	NominalQuota int            `gorm:"default:0" json:"nominal_quota"` // 名义额度（展示用）
	Count        int            `gorm:"default:0" json:"count"`         // 已兑换次数
	Group        string         `gorm:"size:32" json:"group"`           // 可用用户组
	Side         int            `gorm:"default:0" json:"side"`          // 0=给所有人，1=给邀请人，2=给被邀请人
	IsPublic     bool           `gorm:"default:false" json:"is_public"` // 是否公开
	Verified     bool           `gorm:"default:false" json:"verified"`  // 是否验证
	CustomCredit int            `gorm:"default:0" json:"custom_credit"` // 自定义积分
	CreatedAt    time.Time      `json:"created_at"`
	RedeemedTime *time.Time     `json:"redeemed_time"` // 兑换时间
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Redemption) TableName() string {
	return "redemptions"
}

// Invitation 邀请关系模型（对应 new-api 的 invitations 表）
type Invitation struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	InviterID     int64     `gorm:"index" json:"inviter_id"`          // 邀请人 ID
	InviteeID     int64     `gorm:"uniqueIndex" json:"invitee_id"`    // 被邀请人 ID
	InviterCode   string    `gorm:"size:32" json:"inviter_code"`      // 邀请码
	Credit        int       `gorm:"default:0" json:"credit"`          // 获得的积分奖励
	InviteeCredit int       `gorm:"default:0" json:"invitee_credit"`  // 被邀请人获得的积分
	CashbackRate  float64   `gorm:"default:0.2" json:"cashback_rate"` // 返现比例（20%）
	CreatedAt     time.Time `json:"created_at"`
}

// TableName 指定表名
func (Invitation) TableName() string {
	return "invitations"
}

// Billing 账单模型（对应 new-api 的 billings 表）
type Billing struct {
	ID               int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID           int64           `gorm:"index" json:"user_id"`
	TokenID          int64           `gorm:"index" json:"token_id"`
	Amount           decimal.Decimal `gorm:"type:decimal(10,4)" json:"amount"`
	Quota            int             `json:"quota"` // 使用的积分
	ModelName        string          `gorm:"size:128" json:"model_name"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	TotalTokens      int             `json:"total_tokens"`
	Multiplier       float64         `json:"multiplier"`            // 倍率
	Type             int             `gorm:"default:0" json:"type"` // 0=消费,1=充值,2=退款,3=兑换,4=赠送,5=扣减,6=提现,7=预扣费
	ChannelId        int64           `gorm:"index" json:"channel_id"`
	ChannelName      string          `gorm:"size:128" json:"channel_name"`
	InputQuota       int             `json:"input_quota"`                     // 输入 token 消耗积分
	OutputQuota      int             `json:"output_quota"`                    // 输出 token 消耗积分
	GroupRatio       float64         `json:"group_ratio"`                     // 使用的分组倍率
	PreConsumeID     int64           `gorm:"index" json:"pre_consume_id"`     // 关联预扣费记录
	BillingStatus    int             `gorm:"default:1" json:"billing_status"` // 0=pending,1=completed,2=reconciled
	CreatedAt        time.Time       `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (Billing) TableName() string {
	return "billings"
}

// UserBalance 用户余额模型
type UserBalance struct {
	ID           int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64           `gorm:"uniqueIndex" json:"user_id"`
	Balance      decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"balance"`       // 现金余额
	Quota        int             `gorm:"default:0" json:"quota"`                            // 积分余额
	UsedQuota    int             `gorm:"default:0" json:"used_quota"`                       // 已用积分
	TotalQuota   int             `gorm:"default:0" json:"total_quota"`                      // 总积分（历史累计）
	FrozenQuota  int             `gorm:"default:0" json:"frozen_quota"`                     // 冻结积分（预扣费）
	TotalBalance decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"total_balance"` // 总余额（历史累计）
	CashBack     decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"cash_back"`     // 返利金额
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (UserBalance) TableName() string {
	return "user_balances"
}

// IsEnabled 检查 Token 是否启用
func (t *Token) IsEnabled() bool {
	return t.Status == 1
}

// IsExpired 检查 Token 是否过期
func (t *Token) IsExpired() bool {
	if t.ExpiredTime == nil {
		return false // 永不过期
	}
	return time.Now().After(*t.ExpiredTime)
}

// HasQuota 检查是否有足够配额
func (t *Token) HasQuota(quota int) bool {
	if t.UnlimitedQuota || t.RemainQuota < 0 {
		return true // 无限配额
	}
	return t.RemainQuota >= quota
}

// IsModelAllowed 检查模型是否在允许列表中
func (t *Token) IsModelAllowed(modelName string) bool {
	// 如果没有设置限制，允许所有模型
	if t.ModelLimit == "" || t.ModelLimit == "[]" || t.ModelLimit == "null" {
		return true
	}
	// TODO: 实现 JSON 解析
	return true
}

// DeductQuota 扣减配额
func (t *Token) DeductQuota(quota int) {
	if t.UnlimitedQuota || t.RemainQuota < 0 {
		return // 无限配额不扣减
	}
	if t.RemainQuota >= quota {
		t.RemainQuota -= quota
		t.UsedQuota += quota
	}
}
