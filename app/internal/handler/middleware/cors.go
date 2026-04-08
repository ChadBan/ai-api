package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS 跨域中间件
func CORSMiddleware() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"X-Requested-With",
		"X-Request-ID",
	}
	config.ExposeHeaders = []string{
		"Content-Length",
		"X-Request-ID",
	}
	config.AllowCredentials = true
	config.MaxAge = 12 * 3600 // 12 hours

	return cors.New(config)
}
