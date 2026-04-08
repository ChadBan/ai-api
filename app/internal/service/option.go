package service

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"ai-api/app/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OptionService 系统设置服务
type OptionService struct {
	db    *gorm.DB
	cache sync.Map
}

// NewOptionService 创建系统设置服务
func NewOptionService(db *gorm.DB) *OptionService {
	s := &OptionService{db: db}
	s.seedDefaults()
	return s
}

// seedDefaults 初始化默认设置值
func (s *OptionService) seedDefaults() {
	defaults := []model.SystemOption{
		{Key: model.OptRegisterEnabled, Value: "true", Type: "bool", Description: "是否开放注册"},
		{Key: model.OptDefaultQuota, Value: "1000", Type: "int", Description: "新用户默认额度"},
		{Key: model.OptPreConsumedQuota, Value: "0", Type: "int", Description: "预扣费额度 (0=禁用)"},
		{Key: model.OptTopUpLink, Value: "", Type: "string", Description: "充值链接"},
		{Key: model.OptMfaRequired, Value: "false", Type: "bool", Description: "是否强制 MFA"},
		{Key: model.OptPrice, Value: "1.0", Type: "float", Description: "积分价格比率"},
		{Key: model.OptDisplayInCurrency, Value: "false", Type: "bool", Description: "是否按货币显示"},
		{Key: model.OptDisplayName, Value: "true", Type: "bool", Description: "是否显示用户名"},
		{Key: model.OptDrawNotify, Value: "false", Type: "bool", Description: "绘图通知"},
		{Key: model.OptCriticalNotify, Value: "false", Type: "bool", Description: "关键通知"},
		{Key: model.OptGroupRatio, Value: "{}", Type: "json", Description: "分组倍率配置"},
		{Key: model.OptModelRatio, Value: "{}", Type: "json", Description: "模型倍率配置"},
	}

	for _, opt := range defaults {
		opt.UpdatedAt = time.Now()
		s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).Create(&opt)
	}
}

// GetOption 获取设置值
func (s *OptionService) GetOption(key string) (string, error) {
	// 先查缓存
	if v, ok := s.cache.Load(key); ok {
		return v.(string), nil
	}

	var opt model.SystemOption
	if err := s.db.Where("`key` = ?", key).First(&opt).Error; err != nil {
		return "", err
	}

	s.cache.Store(key, opt.Value)
	return opt.Value, nil
}

// GetOptionWithDefault 获取设置值，失败时返回默认值
func (s *OptionService) GetOptionWithDefault(key, defaultValue string) string {
	val, err := s.GetOption(key)
	if err != nil || val == "" {
		return defaultValue
	}
	return val
}

// GetBool 获取布尔设置
func (s *OptionService) GetBool(key string, defaultValue bool) bool {
	val := s.GetOptionWithDefault(key, strconv.FormatBool(defaultValue))
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

// GetInt 获取整数设置
func (s *OptionService) GetInt(key string, defaultValue int) int {
	val := s.GetOptionWithDefault(key, strconv.Itoa(defaultValue))
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

// GetFloat 获取浮点设置
func (s *OptionService) GetFloat(key string, defaultValue float64) float64 {
	val := s.GetOptionWithDefault(key, strconv.FormatFloat(defaultValue, 'f', -1, 64))
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetJSON 获取 JSON 设置到目标结构
func (s *OptionService) GetJSON(key string, target interface{}) error {
	val, err := s.GetOption(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), target)
}

// SetOption 设置单个选项
func (s *OptionService) SetOption(key, value, valueType string) error {
	opt := model.SystemOption{
		Key:       key,
		Value:     value,
		Type:      valueType,
		UpdatedAt: time.Now(),
	}

	result := s.db.Where("`key` = ?", key).First(&model.SystemOption{})
	if result.Error == gorm.ErrRecordNotFound {
		if err := s.db.Create(&opt).Error; err != nil {
			return err
		}
	} else {
		if err := s.db.Model(&model.SystemOption{}).Where("`key` = ?", key).Updates(map[string]interface{}{
			"value":      value,
			"type":       valueType,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
	}

	// 失效缓存
	s.cache.Delete(key)
	return nil
}

// SetBulk 批量设置选项
func (s *OptionService) SetBulk(options map[string]string) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for key, value := range options {
		result := tx.Where("`key` = ?", key).First(&model.SystemOption{})
		if result.Error == gorm.ErrRecordNotFound {
			opt := model.SystemOption{
				Key:       key,
				Value:     value,
				Type:      "string",
				UpdatedAt: time.Now(),
			}
			if err := tx.Create(&opt).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Model(&model.SystemOption{}).Where("`key` = ?", key).Updates(map[string]interface{}{
				"value":      value,
				"updated_at": time.Now(),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
		s.cache.Delete(key)
	}

	return tx.Commit().Error
}

// GetAll 获取所有设置
func (s *OptionService) GetAll() (map[string]string, error) {
	var options []model.SystemOption
	if err := s.db.Find(&options).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, opt := range options {
		result[opt.Key] = opt.Value
		s.cache.Store(opt.Key, opt.Value)
	}
	return result, nil
}

// InvalidateCache 清除所有缓存
func (s *OptionService) InvalidateCache() {
	s.cache.Range(func(key, value interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}
