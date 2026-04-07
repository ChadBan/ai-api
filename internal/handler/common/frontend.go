package common

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// FrontendHandler 前端静态文件处理器
type FrontendHandler struct {
	rootDir string
}

// NewFrontendHandler 创建前端处理器
func NewFrontendHandler(rootDir string) *FrontendHandler {
	return &FrontendHandler{
		rootDir: rootDir,
	}
}

// ServeStatic 提供静态文件服务
func (h *FrontendHandler) ServeStatic(c *gin.Context) {
	path := c.Request.URL.Path
	
	// 构建文件路径
	filePath := filepath.Join(h.rootDir, path)
	
	// 检查文件是否存在
	if _, err := filepath.Abs(filePath); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	
	c.File(filePath)
}

// ServeIndex 处理 SPA 路由，返回 index.html
func (h *FrontendHandler) ServeIndex(c *gin.Context) {
	indexFile := filepath.Join(h.rootDir, "index.html")
	c.File(indexFile)
}

// RegisterRoutes 注册前端路由
func (h *FrontendHandler) RegisterRoutes(r *gin.Engine) {
	// 静态文件
	r.Static("/static", filepath.Join(h.rootDir, "static"))
	r.Static("/assets", filepath.Join(h.rootDir, "assets"))
	
	// API 路由后的所有路径都返回 index.html（SPA 支持）
	noAPI := r.Group("")
	{
		noAPI.GET("/*filepath", h.ServeIndex)
	}
}
