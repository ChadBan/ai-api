package util

// 错误码定义
const (
	Success             = 1000 // 成功
	InvalidParam        = 4000 // 参数错误
	Unauthorized        = 4001 // 未授权
	Forbidden           = 4052 // 禁止访问
	NotFound            = 4004 // 资源不存在
	DuplicateEntry      = 4009 // 重复记录
	BadRequest          = 4010 // 请求错误
	InternalServerError = 4500 // 服务器内部错误

	// 用户相关错误
	UserNotFound       = 4100 // 用户不存在
	UserDisabled       = 4101 // 用户已禁用
	InvalidCredentials = 4102 // 用户名或密码错误
	TokenExpired       = 4103 // Token 已过期
	TokenInvalid       = 4104 // Token 无效

	// 配额相关错误
	QuotaExhausted    = 4200 // 配额不足
	QuotaUpdateFailed = 4201 // 配额更新失败

	// Token 相关错误
	TokenNotFound     = 4300 // Token 不存在
	TokenDisabled     = 4301 // Token 已禁用
	TokenExpiredError = 4302 // Token 已过期

	// API Key 相关错误
	ApiKeyInvalid   = 4400 // API Key 无效
	ApiKeyDisabled  = 4401 // API Key 已禁用
	ApiKeyQuotaUsed = 4402 // API Key 配额已用完

	// 渠道相关错误
	ChannelNotFound    = 4501 // 渠道不存在
	ChannelDisabled    = 4502 // 渠道已禁用
	ChannelSetupNeeded = 4503 // 渠道需要配置

	// 模型相关错误
	ModelNotFound   = 4600 // 模型不存在
	ModelDisabled   = 4601 // 模型已禁用
	ModelNotSupport = 4602 // 模型不支持

	// 充值相关错误
	TopupNotFound      = 4700 // 充值记录不存在
	TopupAlreadyUsed   = 4701 // 兑换码已使用
	TopupExpired       = 4702 // 兑换码已过期
	TopupQuotaInsuffic = 4703 // 充值额度不足

	// 邀请相关错误
	InvitationCodeInvalid = 4800 // 邀请码无效
	SelfInvitation        = 4801 // 不能邀请自己

	// 日志相关错误
	LogNotFound = 4900 // 日志不存在
)

// Error_desc 错误码描述映射
var Error_desc = map[int]string{
	Success:             "Success",
	InvalidParam:        "Invalid request parameters",
	Unauthorized:        "Unauthorized access",
	Forbidden:           "Access forbidden",
	NotFound:            "Resource not found",
	DuplicateEntry:      "Duplicate entry",
	BadRequest:          "Bad request",
	InternalServerError: "Internal server error",

	// 用户相关
	UserNotFound:       "User not found",
	UserDisabled:       "User account is disabled",
	InvalidCredentials: "Invalid username or password",
	TokenExpired:       "Token has expired",
	TokenInvalid:       "Invalid token",

	// 配额相关
	QuotaExhausted:    "Insufficient quota",
	QuotaUpdateFailed: "Failed to update quota",

	// Token 相关
	TokenNotFound:     "Token not found",
	TokenDisabled:     "Token is disabled",
	TokenExpiredError: "Token has expired",

	// API Key 相关
	ApiKeyInvalid:   "Invalid API key",
	ApiKeyDisabled:  "API key is disabled",
	ApiKeyQuotaUsed: "API key quota exhausted",

	// 渠道相关
	ChannelNotFound:    "Channel not found",
	ChannelDisabled:    "Channel is disabled",
	ChannelSetupNeeded: "Channel needs setup",

	// 模型相关
	ModelNotFound:   "Model not found",
	ModelDisabled:   "Model is disabled",
	ModelNotSupport: "Model not supported",

	// 充值相关
	TopupNotFound:      "Topup record not found",
	TopupAlreadyUsed:   "Redemption code already used",
	TopupExpired:       "Redemption code expired",
	TopupQuotaInsuffic: "Insufficient topup quota",

	// 邀请相关
	InvitationCodeInvalid: "Invalid invitation code",
	SelfInvitation:        "Cannot invite yourself",

	// 日志相关
	LogNotFound: "Log not found",
}

// CodeText 获取错误码对应的文本描述
func CodeText(code int, args ...interface{}) string {
	if desc, ok := Error_desc[code]; ok {
		return desc
	}
	return "Unknown error"
}
