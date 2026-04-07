package model

import (
	"time"

	"gorm.io/datatypes"
)

// AuditLog 审计日志模型
type AuditLog struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64          `gorm:"index" json:"user_id"`
	Username    string         `gorm:"size:128" json:"username"`
	Action      string         `gorm:"size:64;index" json:"action"` // 操作类型：create/update/delete/login/etc.
	Resource    string         `gorm:"size:128;index" json:"resource"` // 资源类型：user/channel/token/etc.
	ResourceID  string         `gorm:"size:64" json:"resource_id"`
	IPAddress   string         `gorm:"size:64" json:"ip_address"`
	UserAgent   string         `gorm:"size:512" json:"user_agent"`
	RequestURI  string         `gorm:"size:512" json:"request_uri"`
	Method      string         `gorm:"size:16" json:"method"`
	StatusCode  int            `gorm:"index" json:"status_code"`
	Duration    int64          `json:"duration_ms"` // 请求耗时（毫秒）
	RequestBody datatypes.JSON `json:"request_body"`
	ResponseBody datatypes.JSON `json:"response_body"`
	Error       string         `gorm:"size:1024" json:"error"`
	Metadata    datatypes.JSON `json:"metadata"`
	CreatedAt   time.Time      `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// RequestLog 请求日志模型
type RequestLog struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TraceID      string    `gorm:"size:64;index" json:"trace_id"`
	SpanID       string    `gorm:"size:64" json:"span_id"`
	UserID       int64     `gorm:"index" json:"user_id"`
	APIKeyID     int64     `gorm:"index" json:"api_key_id"`
	Method       string    `gorm:"size:16" json:"method"`
	Path         string    `gorm:"size:512;index" json:"path"`
	Query        string    `gorm:"size:1024" json:"query"`
	StatusCode   int       `gorm:"index" json:"status_code"`
	RequestSize  int64     `json:"request_size"`
	ResponseSize int64     `json:"response_size"`
	Duration     int64     `json:"duration_ms"`
	ClientIP     string    `gorm:"size:64" json:"client_ip"`
	UserAgent    string    `gorm:"size:512" json:"user_agent"`
	Referer      string    `gorm:"size:512" json:"referer"`
	Error        string    `gorm:"size:1024" json:"error"`
	ModelName    string    `gorm:"size:128;index" json:"model_name"`
	ProviderName string    `gorm:"size:64;index" json:"provider_name"`
	Tokens       int       `json:"tokens"`
	Cost         float64   `json:"cost"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (RequestLog) TableName() string {
	return "request_logs"
}

// ErrorLog 错误日志模型
type ErrorLog struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Level     string         `gorm:"size:32;index" json:"level"` // error/warn/critical
	Message   string         `gorm:"size:1024" json:"message"`
	File      string         `gorm:"size:256" json:"file"`
	Line      int            `json:"line"`
	Function  string         `gorm:"size:256" json:"function"`
	StackTrace string        `gorm:"type:text" json:"stack_trace"`
	Context   datatypes.JSON `json:"context"`
	UserID    int64          `gorm:"index" json:"user_id"`
	RequestID string         `gorm:"size:64;index" json:"request_id"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (ErrorLog) TableName() string {
	return "error_logs"
}

// LoginLog 登录日志模型
type LoginLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"index" json:"user_id"`
	Username  string    `gorm:"size:128;index" json:"username"`
	IPAddress string    `gorm:"size:64" json:"ip_address"`
	UserAgent string    `gorm:"size:512" json:"user_agent"`
	Status    int       `gorm:"index" json:"status"` // 0=失败，1=成功
	Reason    string    `gorm:"size:256" json:"reason"` // 失败原因
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (LoginLog) TableName() string {
	return "login_logs"
}

// OperationType 操作类型定义
type OperationType string

const (
	OpCreate  OperationType = "create"
	OpUpdate  OperationType = "update"
	OpDelete  OperationType = "delete"
	OpQuery   OperationType = "query"
	OpExport  OperationType = "export"
	OpImport  OperationType = "import"
	OpLogin   OperationType = "login"
	OpLogout  OperationType = "logout"
	OpUpload  OperationType = "upload"
	OpDownload OperationType = "download"
)

// ResourceType 资源类型定义
type ResourceType string

const (
	ResUser       ResourceType = "user"
	ResChannel    ResourceType = "channel"
	ResToken      ResourceType = "token"
	ResModel      ResourceType = "model"
	ResProvider   ResourceType = "provider"
	ResRedemption ResourceType = "redemption"
	ResBilling    ResourceType = "billing"
	ResConfig     ResourceType = "config"
	ResAPIKey     ResourceType = "api_key"
)
