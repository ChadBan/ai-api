package common

import (
	resp "ai-api/app/internal/common"
	"ai-api/app/internal/util"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查
func HealthHandler(c *gin.Context) {
	resp.SuccessResponse(c, util.Success, gin.H{
		"status": "healthy",
	})
}

// ReadyHandler 就绪检查
func ReadyHandler(c *gin.Context) {
	resp.SuccessResponse(c, util.Success, gin.H{
		"status": "ready",
	})
}
