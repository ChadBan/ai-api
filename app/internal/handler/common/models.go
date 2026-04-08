package common

import (
	"encoding/json"
	"net/http"

	resp "ai-api/app/internal/common"
	"ai-api/app/internal/model"
	"ai-api/app/internal/util"

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

// ListAvailableModels 获取可用模型列表
// 从已配置的渠道中获取可用的模型列表，查询所有启用的渠道（status=1），
// 从渠道的 Models 字段中提取模型名称列表，然后查询模型表获取模型详细信息
func (h *ModelHandler) ListAvailableModels(c *gin.Context) {
	// 查询所有启用的渠道
	var channels []model.Channel
	if err := h.db.Where("status = ?", 1).Find(&channels).Error; err != nil {
		resp.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to fetch channels")
		return
	}

	// 从渠道中提取所有唯一的模型名称
	modelNameSet := make(map[string]bool)
	for _, channel := range channels {
		// 如果渠道没有配置模型列表，跳过
		if channel.Models == "" {
			continue
		}

		// 解析 Models JSON 字段
		var models []string
		if err := json.Unmarshal([]byte(channel.Models), &models); err != nil {
			// 如果解析失败，跳过该渠道
			continue
		}

		// 将模型名称添加到集合中
		for _, modelName := range models {
			// 忽略通配符
			if modelName == "*" || len(modelName) == 0 {
				continue
			}
			modelNameSet[modelName] = true
		}
	}

	// 如果没有找到任何模型，返回空列表
	if len(modelNameSet) == 0 {
		resp.SuccessResponse(c, util.Success, []model.Model{})
		return
	}

	// 将 map 转换为 slice
	modelNames := make([]string, 0, len(modelNameSet))
	for name := range modelNameSet {
		modelNames = append(modelNames, name)
	}

	// 查询模型表获取模型详细信息
	var models []model.Model
	if err := h.db.Where("name IN ? AND status = ?", modelNames, 1).Find(&models).Error; err != nil {
		resp.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to fetch model details")
		return
	}

	resp.SuccessResponse(c, util.Success, models)
}

// GetChannelModels 获取所有渠道的模型列表
// 直接返回渠道信息，包括每个渠道的模型列表
func (h *ModelHandler) GetChannelModels(c *gin.Context) {
	// 查询所有渠道
	var channels []model.Channel
	if err := h.db.Find(&channels).Error; err != nil {
		resp.ErrorResponse(c, http.StatusInternalServerError, util.InternalServerError, "failed to fetch channels")
		return
	}

	// 直接返回渠道信息，前端会解析每个渠道的模型列表
	resp.SuccessResponse(c, util.Success, channels)
}
