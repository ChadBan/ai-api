package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ai-api/app/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GroupService 分组服务
type GroupService struct {
	db         *gorm.DB
	groupCache sync.Map // map[string]*model.Group
}

// NewGroupService 创建分组服务
func NewGroupService(db *gorm.DB) *GroupService {
	s := &GroupService{db: db}
	s.SeedDefault()
	s.RefreshCache()
	return s
}

// SeedDefault 确保默认分组存在
func (s *GroupService) SeedDefault() {
	now := time.Now()
	defaultGroup := model.Group{
		Name:        "default",
		DisplayName: "默认分组",
		Ratio:       1.0,
		Models:      "",
		Status:      1,
		Description: "系统默认分组",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&defaultGroup)
}

// RefreshCache 刷新分组缓存
func (s *GroupService) RefreshCache() {
	var groups []model.Group
	s.db.Where("status = ?", 1).Find(&groups)
	for _, g := range groups {
		gg := g
		s.groupCache.Store(g.Name, &gg)
	}
}

// GetGroup 获取分组
func (s *GroupService) GetGroup(name string) (*model.Group, error) {
	if v, ok := s.groupCache.Load(name); ok {
		return v.(*model.Group), nil
	}

	var group model.Group
	if err := s.db.Where("name = ? AND status = ?", name, 1).First(&group).Error; err != nil {
		return nil, fmt.Errorf("group not found: %s", name)
	}

	s.groupCache.Store(name, &group)
	return &group, nil
}

// ValidateModelAccess 验证分组是否允许访问指定模型
func (s *GroupService) ValidateModelAccess(groupName, modelName string) error {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return err
	}

	// 空模型列表表示允许所有模型
	if group.Models == "" {
		return nil
	}

	var models []string
	if err := json.Unmarshal([]byte(group.Models), &models); err != nil {
		return nil // 解析失败则允许
	}

	if len(models) == 0 {
		return nil
	}

	for _, m := range models {
		if m == modelName || m == "*" {
			return nil
		}
		// 通配符匹配
		if len(m) > 0 && m[len(m)-1] == '*' {
			prefix := m[:len(m)-1]
			if len(modelName) >= len(prefix) && modelName[:len(prefix)] == prefix {
				return nil
			}
		}
	}

	return fmt.Errorf("model %s is not allowed in group %s", modelName, groupName)
}

// GetGroupRatio 获取分组定价倍率
func (s *GroupService) GetGroupRatio(groupName string) float64 {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return 1.0
	}
	return group.Ratio
}

// --- Admin CRUD ---

// ListGroups 列出分组
func (s *GroupService) ListGroups(page, pageSize int) ([]model.Group, int64, error) {
	var groups []model.Group
	var total int64

	s.db.Model(&model.Group{}).Count(&total)
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&groups).Error; err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

// CreateGroup 创建分组
func (s *GroupService) CreateGroup(group *model.Group) error {
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	if err := s.db.Create(group).Error; err != nil {
		return err
	}
	s.groupCache.Store(group.Name, group)
	return nil
}

// UpdateGroup 更新分组
func (s *GroupService) UpdateGroup(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	if err := s.db.Model(&model.Group{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	s.RefreshCache()
	return nil
}

// DeleteGroup 删除分组
func (s *GroupService) DeleteGroup(id int64) error {
	var group model.Group
	if err := s.db.First(&group, id).Error; err != nil {
		return err
	}
	if group.Name == "default" {
		return fmt.Errorf("cannot delete default group")
	}
	if err := s.db.Delete(&group).Error; err != nil {
		return err
	}
	s.groupCache.Delete(group.Name)
	return nil
}
