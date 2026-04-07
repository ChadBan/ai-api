package common

import (
	"net/http"

	resp "github.com/ai-model-scheduler/ai-model-scheduler/internal/common"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/model"
	"github.com/ai-model-scheduler/ai-model-scheduler/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ModelHandler 模型管理 Handler
type ModelHandler struct {
	db *gorm.DB
}

// NewModelHandler 创建 ModelHandler
func NewModelHandler(db *gorm.DB) *ModelHandler {
	return &ModelHandler{db: db}
}

// ListModels 获取模型列表
func (h *ModelHandler) ListModels(c *gin.Context) {
	var models []model.Model

	query := h.db.Where("status = ?", 1)

	// 可选的过滤条件
	if providerID := c.Query("provider_id"); providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}

	if modelType := c.Query("type"); modelType != "" {
		query = query.Where("type = ?", modelType)
	}

	if err := query.Find(&models).Error; err != nil {
		resp.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to fetch models")
		return
	}

	resp.SuccessResponse(c, util.Success, models)
}

// GetModel 获取模型详情
func (h *ModelHandler) GetModel(c *gin.Context) {
	modelID := c.Param("id")

	var m model.Model
	if err := h.db.First(&m, modelID).Error; err != nil {
		resp.ErrorResponse(c, http.StatusNotFound, util.ModelNotFound)
		return
	}

	resp.SuccessResponse(c, util.Success, m)
}
