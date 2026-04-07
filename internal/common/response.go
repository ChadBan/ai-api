package common

import (
	"fmt"
	"net/http"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/gin-gonic/gin"
)

// JSONResult 统一成功响应结构
type JSONResult struct {
	Code          int         `json:"code"`
	Message       string      `json:"message"`
	MessageDetail interface{} `json:"message_detail,omitempty"`
	Data          interface{} `json:"data"`
}

// ErrorJSONResult 统一错误响应结构
type ErrorJSONResult struct {
	Code          int         `json:"code"`
	Message       string      `json:"message"`
	MessageDetail interface{} `json:"message_detail,omitempty"`
}

// SuccResult 创建成功响应
func SuccResult(code int, data interface{}) *JSONResult {
	message := util.CodeText(code)
	return &JSONResult{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// HttpErrorResult 创建 HTTP 错误响应
func HttpErrorResult(code int, message string) *ErrorJSONResult {
	return &ErrorJSONResult{
		Code:          code,
		Message:       message,
		MessageDetail: "",
	}
}

// ErrorResult 创建业务错误响应（支持详细错误信息）
func ErrorResult(code int, details ...interface{}) *ErrorJSONResult {
	message := util.CodeText(code, details...)
	if len(details) <= 0 {
		return &ErrorJSONResult{
			Code:          code,
			Message:       message,
			MessageDetail: "",
		}
	} else {
		return &ErrorJSONResult{
			Code:          code,
			Message:       message,
			MessageDetail: fmt.Sprintf("%s", details...),
		}
	}
}

// SuccessResponse 发送成功响应
func SuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(http.StatusOK, SuccResult(code, data))
}

// ErrorResponse 发送错误响应
func ErrorResponse(c *gin.Context, httpCode int, code int, details ...interface{}) {
	c.JSON(httpCode, ErrorResult(code, details...))
}

// HttpErrorResponse 发送 HTTP 错误响应
func HttpErrorResponse(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, HttpErrorResult(httpCode, message))
}
