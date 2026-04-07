package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-model-scheduler/ai-model-scheduler/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware 限流中间件（基于 Redis）
func RateLimitMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	cfg := config.GetConfig().RateLimit

	if !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		ctx := context.Background()
		userID, exists := c.Get("user_id")
		
		var key string
		limit := cfg.UserQPS

		if exists {
			key = fmt.Sprintf("ratelimit:user:%d", userID)
		} else {
			// 未认证用户使用 IP 限流
			key = fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
			limit = 5 // 更严格的限制
		}

		// 使用滑动窗口限流
		now := time.Now().Unix()
		windowKey := fmt.Sprintf("%s:%d", key, now)

		// 检查当前秒的请求数
		current, err := redisClient.Incr(ctx, windowKey).Result()
		if err != nil {
			c.Next() // Redis 失败时不阻止请求
			return
		}

		if current == 1 {
			// 设置 1 秒过期
			redisClient.Expire(ctx, windowKey, 2*time.Second)
		}

		if current > int64(limit) {
			c.JSON(429, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
