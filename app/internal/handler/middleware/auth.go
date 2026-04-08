package middleware

import (
	"net/http"
	"strings"

	"ai-api/app/internal/common"
	"ai-api/app/internal/security"
	"ai-api/app/internal/util"

	"github.com/gin-gonic/gin"
)

const ApiKeyHeader = "M-API-KEY"

// AuthMiddleware JWT 认证中间件（参考 beijing-car-api 模式）
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 白名单路径 - 不需要认证
		if strings.Contains(c.FullPath(), "/auth/register") ||
			strings.Contains(c.FullPath(), "/auth/login") ||
			strings.Contains(c.FullPath(), "/models") ||
			strings.Contains(c.FullPath(), "/health") ||
			strings.Contains(c.FullPath(), "/ready") {
			c.Next()
			return
		}

		// 从 M-API-KEY Header 获取 Token
		token := c.GetHeader(ApiKeyHeader)
		if token == "" {
			c.JSON(http.StatusUnauthorized, common.HttpErrorResult(
				http.StatusUnauthorized,
				http.StatusText(http.StatusUnauthorized),
			))
			c.Abort()
			return
		}

		// 解析 Token
		sub, err := security.DecodeJwtToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, common.ErrorResult(util.TokenInvalid, err.Error()))
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("userid", sub)
		c.Set(ApiKeyHeader, token)

		c.Next()
	}
}

// TokenAuthMiddleware API Key 认证中间件（用于 /v1/* 接口）
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 白名单路径
		if strings.Contains(c.FullPath(), "/auth/register") ||
			strings.Contains(c.FullPath(), "/auth/login") ||
			strings.Contains(c.FullPath(), "/models") ||
			strings.Contains(c.FullPath(), "/health") ||
			strings.Contains(c.FullPath(), "/ready") {
			c.Next()
			return
		}

		// 从 M-API-KEY Header 获取 Token
		apiKey := c.GetHeader(ApiKeyHeader)

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, common.ErrorResult(util.ApiKeyInvalid))
			c.Abort()
			return
		}

		// TODO: 验证 API Key 的有效性（查询数据库）
		// 这里暂时跳过详细验证，后续可以添加

		c.Set("api_key", apiKey)
		c.Next()
	}
}
