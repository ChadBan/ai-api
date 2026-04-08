package middleware

import (
	"ai-api/app/internal/logger"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware 日志中间件
func LoggingMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码
		statusCode := c.Writer.Status()

		// 获取客户端 IP
		clientIP := c.ClientIP()

		// 获取 Request ID
		requestID, _ := c.Get("request_id")
		if requestID == nil {
			requestID = "unknown"
		}

		// 记录日志
		log.Info("request",
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", query),
			logger.Int("status", statusCode),
			logger.Duration("latency", latency),
			logger.String("client_ip", clientIP),
			logger.Any("request_id", requestID),
		)
	}
}
