# Token 管理 Step 3 实现文档

## 概述

本文档记录了 Token 管理功能的 Step 3 实现，包括前端 UI 增强、自动续期功能和使用记录展示页面。

**实现时间**: 2026-04-01  
**版本**: v3.0  
**状态**: ✅ 已完成

---

## 一、前端 UI 增强

### 1.1 Token 创建表单升级

**文件**: `web/src/views/user/Tokens.vue`

#### 新增字段

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `model_limit` | Array | [] | 允许的模型列表，支持通配符 |
| `ratio` | Number | 1.0 | 汇率倍率（0.1-10） |
| `expired_time` | DateTime | null | 过期时间（null=永不过期） |
| `unlimited_quota` | Boolean | false | 是否无限配额 |
| `group` | String | 'default' | 用户组 |

#### UI 组件

```vue
<!-- 汇率倍率 -->
<el-input-number v-model="createForm.ratio" 
                 :min="0.1" 
                 :max="10" 
                 :precision="2" 
                 :step="0.1" />

<!-- 过期时间 -->
<el-date-picker v-model="createForm.expired_time"
                type="datetime"
                placeholder="选择过期时间"
                value-format="YYYY-MM-DD HH:mm:ss" />

<!-- 允许的模型 -->
<el-select v-model="createForm.model_limit"
           multiple
           filterable
           allow-create
           default-first-option>
  <el-option label="GPT-3.5" value="gpt-3.5-turbo" />
  <el-option label="GPT-4" value="gpt-4" />
  <el-option label="通配符 GPT" value="gpt-*" />
</el-select>
```

#### 辅助函数

```javascript
// 格式化模型限制显示
const formatModelLimit = (modelLimit) => {
  if (!modelLimit || modelLimit === '[]' || modelLimit === 'null') {
    return '所有模型'
  }
  try {
    const models = JSON.parse(modelLimit)
    if (Array.isArray(models)) {
      return models.join(', ')
    }
  } catch (e) {
    return modelLimit
  }
  return modelLimit
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {...})
}
```

### 1.2 表格增强

新增列显示：
- **汇率倍率**: 显示 token 的 ratio 值（保留 2 位小数）
- **模型限制**: 显示允许的模型列表（解析 JSON）
- **过期时间**: 显示过期时间或"永不过期"
- **操作列**: 新增"使用记录"按钮

---

## 二、Token 自动续期功能

### 2.1 核心服务

**文件**: `internal/service/token.go`

#### AutoRenewalConfig 配置结构

```go
type AutoRenewalConfig struct {
    Enabled      bool  // 是否启用自动续期
    RenewalCycle int   // 续期周期（天数）
    MinQuota     int   // 触发续期的最低配额阈值
    RenewalQuota int   // 续期配额量
}
```

#### CheckAndAutoRenewToken 方法

**逻辑流程**:
1. 检查是否启用自动续期
2. 检查 Token 是否已过期或即将过期（提前 7 天）
3. 检查配额是否低于阈值
4. 查询用户余额是否充足
5. 开启事务执行：
   - 扣减用户余额
   - 增加 Token 配额
   - 延长过期时间（如已过期）
6. 记录日志

**代码示例**:
```go
func (s *TokenService) CheckAndAutoRenewToken(ctx context.Context, token *model.Token, config *AutoRenewalConfig) error {
    // 检查续期条件
    shouldRenew := false
    
    // 检查过期时间
    if token.ExpiredTime != nil {
        if token.ExpiredTime.Before(now) || 
           token.ExpiredTime.Before(now.Add(7*24*time.Hour)) {
            shouldRenew = true
        }
    }
    
    // 检查配额
    if token.RemainQuota < config.MinQuota {
        shouldRenew = true
    }
    
    // 执行续期...
}
```

#### 辅助方法

```go
// GetLowQuotaTokens 获取低配额 Token 列表
func (s *TokenService) GetLowQuotaTokens(ctx context.Context, threshold int, limit int) ([]model.Token, error)

// GetExpiringTokens 获取即将过期的 Token 列表
func (s *TokenService) GetExpiringTokens(ctx context.Context, daysUntilExpiry int, limit int) ([]model.Token, error)
```

### 2.2 定时任务服务

**文件**: `internal/service/token_renewal.go`

#### TokenRenewalService 结构

```go
type TokenRenewalService struct {
    tokenService *TokenService
    logger       *zap.Logger
    config       *AutoRenewalConfig
    stopChan     chan struct{}
    wg           sync.WaitGroup
}
```

#### 主要方法

| 方法 | 说明 |
|------|------|
| `Start()` | 启动定时任务（每 24 小时检查一次） |
| `Stop()` | 停止服务 |
| `checkAndRenewAllTokens()` | 检查并续期所有符合条件的 Token |
| `ManualRenew(ctx, tokenID)` | 手动续期指定 Token |

**定时任务逻辑**:
```go
func (s *TokenRenewalService) Start() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    go func() {
        for {
            select {
            case <-ticker.C:
                s.checkAndRenewAllTokens()
            case <-s.stopChan:
                return
            }
        }
    }()
}
```

### 2.3 使用示例

```go
// 在应用启动时初始化
config := &AutoRenewalConfig{
    Enabled:      true,
    RenewalCycle: 30,      // 30 天
    MinQuota:     100000,  // 配额低于 10 万触发
    RenewalQuota: 1000000, // 续期 100 万配额
}

renewalService := service.NewTokenRenewalService(tokenService, logger, config)
renewalService.Start()

// 手动续期（可选）
err := renewalService.ManualRenew(ctx, tokenID)
```

---

## 三、Token 使用记录展示页面

### 3.1 页面路由

**文件**: `web/src/router/index.js`

```javascript
{
  path: 'token-usage',
  name: 'TokenUsage',
  component: () => import('@/views/user/TokenUsage.vue')
}
```

### 3.2 页面功能

**文件**: `web/src/views/user/TokenUsage.vue`

#### 筛选条件

- **Token 选择**: 下拉选择具体 Token
- **模型名称**: 文本输入框模糊搜索
- **状态筛选**: 成功/失败
- **时间范围**: 日期时间范围选择器

#### 统计卡片

| 指标 | 说明 |
|------|------|
| 总请求数 | 筛选条件下的总请求次数 |
| 成功请求 | 成功请求数及成功率 |
| 总 Token 消耗 | input_tokens + output_tokens 总和 |
| 总配额扣除 | quota_deducted 总和 |

#### 数据表格

| 列名 | 字段 | 说明 |
|------|------|------|
| ID | id | 记录 ID |
| Token | token_key | 脱敏显示（前 8 后 4） |
| 模型 | model | 使用的模型名称 |
| 输入 Tokens | input_tokens | 输入 token 数 |
| 输出 Tokens | output_tokens | 输出 token 数 |
| 总 Tokens | - | input + output |
| 扣除配额 | quota_deducted | 实际扣除的配额 |
| 耗时 | duration_ms | 请求耗时（毫秒） |
| 状态 | success | 成功/失败标签 |
| 错误信息 | error_message | 失败时的错误信息 |
| 请求时间 | request_time | 格式化显示 |

#### API 调用

```javascript
// 获取使用记录列表
const loadLogs = async () => {
  const params = {
    page: pagination.page,
    page_size: pagination.page_size,
    token_id: filters.token_id,
    model: filters.model,
    success: filters.success,
    start_time: filters.date_range[0],
    end_time: filters.date_range[1]
  }
  
  const res = await api.getTokenUsageLogs(params)
  logs.value = res.data.items || []
  pagination.total = res.data.total || 0
}
```

### 3.3 后端接口

**文件**: `internal/handler/user/token.go`

#### ListTokenUsageLogs Handler

```go
func (h *TokenHandler) ListTokenUsageLogs(c *gin.Context) {
    userID := c.Get("user_id")
    
    // 构建查询
    query := h.db.Model(&model.TokenUsageLog{}).Where("user_id = ?", userID)
    
    // 支持筛选
    if tokenID := c.Query("token_id"); tokenID != "" { ... }
    if model := c.Query("model"); model != "" { ... }
    if success := c.Query("success"); success != "" { ... }
    if startTime := c.Query("start_time"); startTime != "" { ... }
    if endTime := c.Query("end_time"); endTime != "" { ... }
    
    // 分页查询
    // ...
}
```

**路由**: `GET /v1/tokens/usage-logs`

**查询参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码（默认 1） |
| page_size | int | 否 | 每页数量（默认 20） |
| token_id | int | 否 | Token ID |
| model | string | 否 | 模型名称 |
| success | bool | 否 | 是否成功 |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |

---

## 四、数据库表结构

### tokens 表

```sql
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS model_limit TEXT DEFAULT '[]';
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS ratio DECIMAL(10,6) DEFAULT 1.0;
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS group VARCHAR(64) DEFAULT 'default';
```

### token_usage_logs 表

```sql
CREATE TABLE IF NOT EXISTS token_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    token_key VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    model VARCHAR(64),
    tokens_used INTEGER DEFAULT 0,
    quota_deducted INTEGER DEFAULT 0,
    request_time TIMESTAMP NOT NULL,
    duration_ms INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT TRUE,
    error_message VARCHAR(512),
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    channel_id BIGINT
);

CREATE INDEX idx_token_usage_logs_token_key ON token_usage_logs(token_key);
CREATE INDEX idx_token_usage_logs_user_id ON token_usage_logs(user_id);
CREATE INDEX idx_token_usage_logs_request_time ON token_usage_logs(request_time);
CREATE INDEX idx_token_usage_logs_model ON token_usage_logs(model);
```

---

## 五、API 接口清单

### 5.1 已有接口（不变）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /v1/tokens | 创建 Token |
| GET | /v1/tokens | 获取 Token 列表 |
| GET | /v1/tokens/:id | 获取 Token 详情 |
| PUT | /v1/tokens/:id | 更新 Token |
| DELETE | /v1/tokens/:id | 删除 Token |
| POST | /v1/tokens/:id/status | 切换状态 |
| GET | /v1/tokens/stats | 获取统计 |
| POST | /v1/tokens/:id/topup | Token 充值 |
| GET | /v1/tokens/:id/usage | 获取单 Token 使用记录 |

### 5.2 新增接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /v1/tokens/usage-logs | 获取使用记录列表（支持多 Token 筛选） |

**请求示例**:
```bash
GET /v1/tokens/usage-logs?page=1&page_size=20&token_id=123&model=gpt-4&success=true
```

**响应示例**:
```json
{
  "items": [
    {
      "id": 1,
      "token_key": "sk-abc...",
      "user_id": 1,
      "model": "gpt-4",
      "input_tokens": 100,
      "output_tokens": 50,
      "quota_deducted": 1500,
      "duration_ms": 234,
      "success": true,
      "request_time": "2026-04-01T10:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

---

## 六、配置说明

### 6.1 自动续期配置

在应用配置文件中添加：

```yaml
token:
  auto_renewal:
    enabled: true
    renewal_cycle_days: 30
    min_quota_threshold: 100000  # 10 万配额
    renewal_quota: 1000000       # 100 万配额
```

### 6.2 环境变量

```bash
# 自动续期开关
TOKEN_AUTO_RENEWAL_ENABLED=true
# 续期周期（天）
TOKEN_RENEWAL_CYCLE_DAYS=30
# 最低配额阈值
TOKEN_MIN_QUOTA_THRESHOLD=100000
# 续期配额量
TOKEN_RENEWAL_QUOTA=1000000
```

---

## 七、测试验证

### 7.1 后端编译

```bash
cd /usr/local/go-path/ai-api
go build -o /tmp/ai-api-step3 ./cmd/server
# ✅ 编译成功
```

### 7.2 前端检查

- ✅ Tokens.vue 语法检查通过
- ✅ TokenUsage.vue 语法检查通过
- ✅ Router 配置正确
- ✅ API 方法已添加

### 7.3 功能测试清单

- [ ] 创建带 model_limit 的 Token
- [ ] 创建带 ratio 的 Token
- [ ] 创建带过期时间的 Token
- [ ] 查看 Token 使用记录
- [ ] 筛选使用记录（按 Token、模型、状态、时间）
- [ ] 自动续期定时任务启动
- [ ] 手动触发 Token 续期
- [ ] 低配额 Token 自动检测
- [ ] 即将过期 Token 自动检测

---

## 八、注意事项

### 8.1 安全性

1. **Token 脱敏**: 前端显示时只展示前后各 4 个字符
2. **权限验证**: 所有接口都经过 AuthMiddleware 验证
3. **用户隔离**: 查询时强制加上 `user_id` 条件

### 8.2 性能优化

1. **分页查询**: 所有列表接口都支持分页
2. **索引优化**: token_usage_logs 表建立多个索引
3. **异步日志**: 使用记录写入不阻塞主流程

### 8.3 并发控制

1. **事务处理**: 自动续期使用数据库事务
2. **配额重查**: 扣减前再次检查防止超扣
3. **锁机制**: 使用 GORM 的行级锁

### 8.4 监控告警

建议在自动续期服务中添加监控：

```go
if err != nil {
    metrics.TokenAutoRenewalFailed.Inc()
    logger.Error("Auto-renewal failed", zap.Error(err))
    
    // 发送告警通知
    if config.AlertEnabled {
        alert.Send("Token 自动续期失败", err.Error())
    }
}
```

---

## 九、后续优化建议

### 9.1 短期优化

1. **批量写入**: 使用记录改为批量插入（每 100 条或每秒）
2. **缓存优化**: Token 验证结果缓存 5 分钟
3. **导出功能**: 支持导出使用记录为 CSV/Excel

### 9.2 中期优化

1. **多级续期**: 支持配置多个续期档位（如低/中/高配额）
2. ** webhook 通知**: 续期前后发送 webhook 通知
3. **续期策略**: 支持自定义续期策略（如仅工作日续期）

### 9.3 长期优化

1. **预测续期**: 基于历史使用量预测最佳续期时间
2. **智能推荐**: 根据使用模式推荐合适的配额量
3. **成本分析**: 提供 Token 使用成本分析报告

---

## 十、相关文件清单

### 后端文件

- `internal/service/token.go` - Token 核心服务（新增自动续期方法）
- `internal/service/token_renewal.go` - Token 自动续期服务（新建）
- `internal/handler/user/token.go` - Token Handler（新增使用记录接口）
- `internal/model/channel.go` - Token 模型定义
- `internal/model/token_usage.go` - Token 使用记录模型
- `cmd/server/api-router.go` - 路由配置（新增 usage-logs 路由）

### 前端文件

- `web/src/views/user/Tokens.vue` - Token 管理页面（增强表单和表格）
- `web/src/views/user/TokenUsage.vue` - Token 使用记录页面（新建）
- `web/src/router/index.js` - 路由配置（新增 token-usage 路由）
- `web/src/api/index.js` - API 封装（新增 getTokenUsageLogs 方法）

---

## 总结

Step 3 实现了完整的 Token 管理前端 UI 和自动续期功能，主要包括：

✅ **前端 UI 增强**: 支持 model_limit、ratio、expired_time 等字段配置  
✅ **自动续期功能**: 定时任务检测 + 自动扣费续期  
✅ **使用记录页面**: 完整的筛选、统计、展示功能  

所有代码已编译通过，可以进行下一步测试和部署。
