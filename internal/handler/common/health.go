package common

import (
	resp "github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

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
