package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-api/app/internal/logger"
	"ai-api/app/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditService 审计服务
type AuditService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewAuditService 创建审计服务
func NewAuditService(db *gorm.DB, logger *logger.Logger) *AuditService {
	return &AuditService{
		db:     db,
		logger: logger,
	}
}

// LogOptions 日志选项
type LogOptions struct {
	UserID       int64
	Username     string
	Action       model.OperationType
	Resource     model.ResourceType
	ResourceID   string
	IPAddress    string
	UserAgent    string
	RequestURI   string
	Method       string
	StatusCode   int
	Duration     time.Duration
	RequestBody  interface{}
	ResponseBody interface{}
	Error        string
	Metadata     map[string]interface{}
}

// Log 记录审计日志
func (s *AuditService) Log(ctx context.Context, opts LogOptions) error {
	var reqBodyJSON, respBodyJSON, metaJSON []byte
	var err error

	if opts.RequestBody != nil {
		reqBodyJSON, err = json.Marshal(opts.RequestBody)
		if err != nil {
			reqBodyJSON = []byte("{}")
		}
	} else {
		reqBodyJSON = []byte("null")
	}

	if opts.ResponseBody != nil {
		respBodyJSON, err = json.Marshal(opts.ResponseBody)
		if err != nil {
			respBodyJSON = []byte("{}")
		}
	} else {
		respBodyJSON = []byte("null")
	}

	if opts.Metadata != nil {
		metaJSON, err = json.Marshal(opts.Metadata)
		if err != nil {
			metaJSON = []byte("{}")
		}
	} else {
		metaJSON = []byte("null")
	}

	log := model.AuditLog{
		UserID:       opts.UserID,
		Username:     opts.Username,
		Action:       string(opts.Action),
		Resource:     string(opts.Resource),
		ResourceID:   opts.ResourceID,
		IPAddress:    opts.IPAddress,
		UserAgent:    opts.UserAgent,
		RequestURI:   opts.RequestURI,
		Method:       opts.Method,
		StatusCode:   opts.StatusCode,
		Duration:     opts.Duration.Milliseconds(),
		RequestBody:  reqBodyJSON,
		ResponseBody: respBodyJSON,
		Error:        opts.Error,
		Metadata:     metaJSON,
		CreatedAt:    time.Now(),
	}

	// 异步写入数据库
	go func() {
		if err := s.db.Create(&log).Error; err != nil {
			s.logger.Error("failed to save audit log", logger.Err(err))
		}
	}()

	// 同时写入 Zap 日志
	s.logger.Info("audit_log",
		logger.Int64("user_id", opts.UserID),
		logger.String("username", opts.Username),
		logger.String("action", string(opts.Action)),
		logger.String("resource", string(opts.Resource)),
		logger.String("resource_id", opts.ResourceID),
		logger.String("ip", opts.IPAddress),
		logger.Int("status", opts.StatusCode),
		logger.Duration("duration", opts.Duration),
	)

	return nil
}

// LogUserAction 记录用户操作（便捷方法）
func (s *AuditService) LogUserAction(c *gin.Context, action model.OperationType, resource model.ResourceType, resourceID string, extra map[string]interface{}) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("email")

	opts := LogOptions{
		UserID:     getInt64(userID),
		Username:   getString(username),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		RequestURI: c.Request.URL.String(),
		Method:     c.Request.Method,
		StatusCode: c.Writer.Status(),
		Metadata:   extra,
	}

	// 从上下文获取请求/响应体（如果有的话）
	if reqBody, exists := c.Get("request_body"); exists {
		opts.RequestBody = reqBody
	}
	if respBody, exists := c.Get("response_body"); exists {
		opts.ResponseBody = respBody
	}

	s.Log(c.Request.Context(), opts)
}

// RequestLogService 请求日志服务
type RequestLogService struct {
	db *gorm.DB
}

// NewRequestLogService 创建请求日志服务
func NewRequestLogService(db *gorm.DB) *RequestLogService {
	return &RequestLogService{db: db}
}

// LogRequest 记录请求日志
func (s *RequestLogService) LogRequest(c *gin.Context, duration time.Duration, tokens int, cost float64, modelName, providerName string) error {
	userID, _ := c.Get("user_id")
	apiKeyID, _ := c.Get("api_key_id")

	traceID, _ := c.Get("trace_id")
	if traceID == nil {
		traceID = ""
	}

	log := model.RequestLog{
		TraceID:      getString(traceID),
		UserID:       getInt64(userID),
		APIKeyID:     getInt64(apiKeyID),
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		Query:        c.Request.URL.RawQuery,
		StatusCode:   c.Writer.Status(),
		RequestSize:  int64(c.Request.ContentLength),
		ResponseSize: int64(c.Writer.Size()),
		Duration:     duration.Milliseconds(),
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		ModelName:    modelName,
		ProviderName: providerName,
		Tokens:       tokens,
		Cost:         cost,
		CreatedAt:    time.Now(),
	}

	return s.db.Create(&log).Error
}

// ErrorLogService 错误日志服务
type ErrorLogService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewErrorLogService 创建错误日志服务
func NewErrorLogService(db *gorm.DB, logger *logger.Logger) *ErrorLogService {
	return &ErrorLogService{
		db:     db,
		logger: logger,
	}
}

// LogError 记录错误日志
func (s *ErrorLogService) LogError(level, message, file string, line int, function string, stackTrace string, context map[string]interface{}, userID int64, requestID string) error {
	ctxJSON, _ := json.Marshal(context)

	log := model.ErrorLog{
		Level:      level,
		Message:    message,
		File:       file,
		Line:       line,
		Function:   function,
		StackTrace: stackTrace,
		Context:    ctxJSON,
		UserID:     userID,
		RequestID:  requestID,
		CreatedAt:  time.Now(),
	}

	// 同时写入 Zap 和数据库
	switch level {
	case "critical":
		s.logger.Fatal(message, logger.String("request_id", requestID))
	case "error":
		s.logger.Error(message, logger.String("request_id", requestID))
	default:
		s.logger.Warn(message, logger.String("request_id", requestID))
	}

	return s.db.Create(&log).Error
}

// LoginLogService 登录日志服务
type LoginLogService struct {
	db *gorm.DB
}

// NewLoginLogService 创建登录日志服务
func NewLoginLogService(db *gorm.DB) *LoginLogService {
	return &LoginLogService{db: db}
}

// LogLogin 记录登录日志
func (s *LoginLogService) LogLogin(userID int64, username, ipAddress, userAgent string, success bool, reason string) error {
	status := 1
	if !success {
		status = 0
	}

	log := model.LoginLog{
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Status:    status,
		Reason:    reason,
		CreatedAt: time.Now(),
	}

	return s.db.Create(&log).Error
}

// 辅助函数
func getInt64(v interface{}) int64 {
	switch v := v.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
