package service

import (
	"sync"
	"time"

	"ai-api/app/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PricingService 定价服务
type PricingService struct {
	db          *gorm.DB
	priceCache  sync.Map // map[string]*model.ModelPrice
	groupCache  sync.Map // map[string]float64  key="group:model"
}

// NewPricingService 创建定价服务
func NewPricingService(db *gorm.DB) *PricingService {
	s := &PricingService{db: db}
	s.seedDefaults()
	s.RefreshCache()
	return s
}

// seedDefaults 初始化默认模型定价
func (s *PricingService) seedDefaults() {
	defaults := []model.ModelPrice{
		{ModelName: "gpt-4", InputRatio: 15.0, OutputRatio: 30.0, Enabled: true},
		{ModelName: "gpt-4-32k", InputRatio: 30.0, OutputRatio: 60.0, Enabled: true},
		{ModelName: "gpt-4-turbo", InputRatio: 5.0, OutputRatio: 15.0, Enabled: true},
		{ModelName: "gpt-4o", InputRatio: 2.5, OutputRatio: 10.0, Enabled: true},
		{ModelName: "gpt-4o-mini", InputRatio: 0.15, OutputRatio: 0.6, Enabled: true},
		{ModelName: "gpt-3.5-turbo", InputRatio: 0.75, OutputRatio: 1.0, Enabled: true},
		{ModelName: "gpt-3.5-turbo-16k", InputRatio: 1.5, OutputRatio: 2.0, Enabled: true},
		{ModelName: "claude-3-opus-20240229", InputRatio: 7.5, OutputRatio: 37.5, Enabled: true},
		{ModelName: "claude-3-sonnet-20240229", InputRatio: 1.5, OutputRatio: 7.5, Enabled: true},
		{ModelName: "claude-3-haiku-20240307", InputRatio: 0.125, OutputRatio: 0.625, Enabled: true},
		{ModelName: "claude-3-5-sonnet-20241022", InputRatio: 1.5, OutputRatio: 7.5, Enabled: true},
		{ModelName: "claude-3-5-haiku-20241022", InputRatio: 0.5, OutputRatio: 2.0, Enabled: true},
		{ModelName: "gemini-pro", InputRatio: 0.25, OutputRatio: 0.5, Enabled: true},
		{ModelName: "gemini-1.5-pro", InputRatio: 1.75, OutputRatio: 3.5, Enabled: true},
		{ModelName: "gemini-1.5-flash", InputRatio: 0.075, OutputRatio: 0.15, Enabled: true},
		{ModelName: "deepseek-chat", InputRatio: 0.14, OutputRatio: 0.28, Enabled: true},
		{ModelName: "deepseek-reasoner", InputRatio: 0.55, OutputRatio: 2.19, Enabled: true},
		{ModelName: "text-embedding-ada-002", InputRatio: 0.05, OutputRatio: 0.0, Enabled: true},
		{ModelName: "text-embedding-3-small", InputRatio: 0.01, OutputRatio: 0.0, Enabled: true},
		{ModelName: "text-embedding-3-large", InputRatio: 0.065, OutputRatio: 0.0, Enabled: true},
		{ModelName: "dall-e-3", InputRatio: 0.0, OutputRatio: 0.0, FixedCost: decimal.NewFromFloat(20.0), Enabled: true},
		{ModelName: "dall-e-2", InputRatio: 0.0, OutputRatio: 0.0, FixedCost: decimal.NewFromFloat(10.0), Enabled: true},
		{ModelName: "whisper-1", InputRatio: 0.0, OutputRatio: 0.0, FixedCost: decimal.NewFromFloat(5.0), Enabled: true},
		{ModelName: "tts-1", InputRatio: 7.5, OutputRatio: 0.0, Enabled: true},
		{ModelName: "tts-1-hd", InputRatio: 15.0, OutputRatio: 0.0, Enabled: true},
	}

	now := time.Now()
	for i := range defaults {
		defaults[i].CreatedAt = now
		defaults[i].UpdatedAt = now
		s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "model_name"}},
			DoNothing: true,
		}).Create(&defaults[i])
	}
}

// RefreshCache 刷新定价缓存
func (s *PricingService) RefreshCache() {
	var prices []model.ModelPrice
	s.db.Where("enabled = ?", true).Find(&prices)
	for _, p := range prices {
		pp := p
		s.priceCache.Store(p.ModelName, &pp)
	}

	var multipliers []model.GroupPriceMultiplier
	s.db.Find(&multipliers)
	for _, m := range multipliers {
		key := m.GroupName + ":" + m.ModelName
		s.groupCache.Store(key, m.Multiplier)
	}
}

// GetModelPrice 获取模型定价
func (s *PricingService) GetModelPrice(modelName string) *model.ModelPrice {
	if v, ok := s.priceCache.Load(modelName); ok {
		return v.(*model.ModelPrice)
	}

	// 从 DB 查询
	var price model.ModelPrice
	if err := s.db.Where("model_name = ? AND enabled = ?", modelName, true).First(&price).Error; err != nil {
		// 返回默认定价
		return &model.ModelPrice{
			ModelName:   modelName,
			InputRatio:  1.0,
			OutputRatio: 1.0,
			Enabled:     true,
		}
	}

	s.priceCache.Store(modelName, &price)
	return &price
}

// GetGroupMultiplier 获取分组倍率
func (s *PricingService) GetGroupMultiplier(groupName, modelName string) float64 {
	// 先查特定模型
	key := groupName + ":" + modelName
	if v, ok := s.groupCache.Load(key); ok {
		return v.(float64)
	}
	// 再查通配符
	wildcardKey := groupName + ":*"
	if v, ok := s.groupCache.Load(wildcardKey); ok {
		return v.(float64)
	}
	return 1.0
}

// CalculateQuota 计算配额消耗
// 返回: inputQuota, outputQuota, totalQuota
func (s *PricingService) CalculateQuota(modelName, groupName string, promptTokens, completionTokens int) (int, int, int) {
	price := s.GetModelPrice(modelName)
	groupMultiplier := s.GetGroupMultiplier(groupName, modelName)

	// 固定费用模型 (如 DALL-E)
	if !price.FixedCost.IsZero() {
		total := int(price.FixedCost.InexactFloat64() * groupMultiplier)
		return 0, 0, total
	}

	// Token 计费: (tokens / 1000) * ratio * groupMultiplier
	inputQuota := float64(promptTokens) / 1000.0 * price.InputRatio * groupMultiplier
	outputQuota := float64(completionTokens) / 1000.0 * price.OutputRatio * groupMultiplier
	totalQuota := inputQuota + outputQuota

	// 最小消耗 1 积分
	total := int(totalQuota)
	if total < 1 && (promptTokens > 0 || completionTokens > 0) {
		total = 1
	}

	return int(inputQuota), int(outputQuota), total
}

// --- Admin CRUD ---

// ListModelPrices 列出所有模型定价
func (s *PricingService) ListModelPrices(page, pageSize int, search string) ([]model.ModelPrice, int64, error) {
	var prices []model.ModelPrice
	var total int64

	query := s.db.Model(&model.ModelPrice{})
	if search != "" {
		query = query.Where("model_name LIKE ?", "%"+search+"%")
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	if err := query.Order("model_name ASC").Offset(offset).Limit(pageSize).Find(&prices).Error; err != nil {
		return nil, 0, err
	}
	return prices, total, nil
}

// CreateModelPrice 创建模型定价
func (s *PricingService) CreateModelPrice(price *model.ModelPrice) error {
	price.CreatedAt = time.Now()
	price.UpdatedAt = time.Now()
	if err := s.db.Create(price).Error; err != nil {
		return err
	}
	s.priceCache.Store(price.ModelName, price)
	return nil
}

// UpdateModelPrice 更新模型定价
func (s *PricingService) UpdateModelPrice(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	if err := s.db.Model(&model.ModelPrice{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	// 失效缓存并重载
	s.RefreshCache()
	return nil
}

// DeleteModelPrice 删除模型定价
func (s *PricingService) DeleteModelPrice(id int64) error {
	var price model.ModelPrice
	if err := s.db.First(&price, id).Error; err != nil {
		return err
	}
	if err := s.db.Delete(&price).Error; err != nil {
		return err
	}
	s.priceCache.Delete(price.ModelName)
	return nil
}

// ListGroupMultipliers 列出所有分组倍率
func (s *PricingService) ListGroupMultipliers(page, pageSize int) ([]model.GroupPriceMultiplier, int64, error) {
	var multipliers []model.GroupPriceMultiplier
	var total int64

	s.db.Model(&model.GroupPriceMultiplier{}).Count(&total)
	offset := (page - 1) * pageSize
	if err := s.db.Order("group_name ASC").Offset(offset).Limit(pageSize).Find(&multipliers).Error; err != nil {
		return nil, 0, err
	}
	return multipliers, total, nil
}

// CreateGroupMultiplier 创建分组倍率
func (s *PricingService) CreateGroupMultiplier(m *model.GroupPriceMultiplier) error {
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	if err := s.db.Create(m).Error; err != nil {
		return err
	}
	key := m.GroupName + ":" + m.ModelName
	s.groupCache.Store(key, m.Multiplier)
	return nil
}

// UpdateGroupMultiplier 更新分组倍率
func (s *PricingService) UpdateGroupMultiplier(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	if err := s.db.Model(&model.GroupPriceMultiplier{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	s.RefreshCache()
	return nil
}

// DeleteGroupMultiplier 删除分组倍率
func (s *PricingService) DeleteGroupMultiplier(id int64) error {
	if err := s.db.Delete(&model.GroupPriceMultiplier{}, id).Error; err != nil {
		return err
	}
	s.RefreshCache()
	return nil
}
