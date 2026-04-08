# Token 管理功能完善计划

## 当前问题

1. **创建 Token 过于简单** - 缺少实际使用场景的考虑
2. **缺少模型限额控制** - 无法限制 Token 只能访问特定模型
3. **缺少使用记录追踪** - 无法查看 Token 的详细使用情况
4. **缺少配额扣减逻辑** - Token 使用时没有正确的配额计算

## 参考 new-api 的实现

### 核心字段

```go
type Token struct {
    ID             int64          `gorm:"primaryKey"`
    UserID         int64          `gorm:"index"`
    Key            string         `gorm:"unique;size:128"`
    Status         int            `gorm:"default:1"`  // 1:启用 0:禁用
    Name           string         `gorm:"size:128"`
    CreatedTime    time.Time
    AccessedTime   time.Time      // 最后访问时间
    ExpiredTime    *time.Time     // 过期时间（nil 表示永不过期）
    RemainQuota    int            `gorm:"default:-1"`  // 剩余配额（-1 表示无限）
    UnlimitedQuota bool           `gorm:"default:false"`
    UsedQuota      int            `gorm:"default:0"`
    ModelLimit     string         `gorm:"type:text"`   // JSON，限制的模型列表
    Ratio          float64        `gorm:"type:decimal(10,6)"`  // 汇率倍率
}
```

## 需要新增的功能

### 1. 模型限额（Model Limit）

**使用场景**：
- 用户创建一个 Token，只能访问 `gpt-3.5-turbo` 和 `gpt-4`，不能访问其他模型
- 防止 Token 被滥用访问高价模型

**实现**：
```json
{
  "model_limit": ["gpt-3.5-turbo", "gpt-4"]
}
```

### 2. 汇率倍率（Ratio）

**使用场景**：
- 给不同用户群体不同的汇率
- VIP 用户享受更优惠的汇率（如 0.8 倍）
- 普通用户标准汇率（1.0 倍）

**实现**：
```json
{
  "ratio": 0.8  // 8 折优惠
}
```

### 3. Token 使用记录

**使用场景**：
- 查看每个 Token 的使用历史
- 审计和计费依据
- 异常检测

**数据表设计**：
```sql
CREATE TABLE token_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    token_key VARCHAR(128),
    user_id BIGINT,
    model VARCHAR(64),
    tokens_used INT,
    quota_deducted INT,
    request_time TIMESTAMP,
    duration_ms INT,
    success BOOLEAN
);
```

### 4. Token 验证中间件

**验证流程**：
1. Token 是否存在且启用
2. Token 是否过期
3. Token 配额是否充足
4. 请求的模型是否在允许列表中
5. 计算配额并扣减

### 5. 自动续费/充值

**使用场景**：
- Token 配额不足时自动从用户余额扣除
- 设置自动充值阈值

## API 接口完善

### 创建 Token（增强版）

```bash
POST /v1/tokens
{
  "name": "My API Key",
  "remain_quota": 10000,
  "unlimited_quota": false,
  "expired_time": 1735689600,  // Unix 时间戳，-1 表示永不过期
  "model_limit": ["gpt-3.5-turbo", "gpt-4"],  // 允许的模型列表
  "ratio": 1.0,  // 汇率倍率
  "group": "default"
}
```

### 更新 Token（增强版）

```bash
PUT /v1/tokens/:id
{
  "name": "Updated Name",
  "remain_quota": 20000,
  "model_limit": ["gpt-3.5-turbo", "gpt-4", "claude-3"],
  "ratio": 0.9
}
```

### 获取 Token 使用记录

```bash
GET /v1/tokens/:id/usage?page=1&page_size=20
```

响应：
```json
{
  "items": [
    {
      "model": "gpt-3.5-turbo",
      "tokens_used": 1500,
      "quota_deducted": 1500,
      "request_time": "2026-04-01T10:00:00Z",
      "duration_ms": 234,
      "success": true
    }
  ],
  "total": 100
}
```

### Token 充值

```bash
POST /v1/tokens/:id/topup
{
  "quota": 5000,
  "reason": "手动充值"
}
```

## 配额计算逻辑

### 计费公式

```
实际扣费 = (输入 tokens + 输出 tokens) * 模型价格 * ratio
```

### 示例

```javascript
// GPT-3.5-Turbo 价格：$0.002 / 1K tokens
// 用户 ratio: 0.8
// 输入：1000 tokens, 输出：500 tokens

tokens_total = 1500
price_per_1k = 0.002
ratio = 0.8

cost_usd = (1500 / 1000) * 0.002 * 0.8 = 0.0024
quota = cost_usd * 汇率（如 7.2）* 1000000 = 17280
```

## 实施步骤

### P0 - 核心功能（必须）
1. ✅ 添加 model_limit 字段
2. ✅ 添加 ratio 字段  
3. ✅ 完善 Token 验证中间件
4. ✅ 实现配额扣减逻辑

### P1 - 重要功能
1. ⏳ Token 使用记录表
2. ⏳ 使用记录查询 API
3. ⏳ Token 充值功能
4. ⏳ 自动续费逻辑

### P2 - 优化功能
1. ⏳ 使用统计图表
2. ⏳ 异常使用告警
3. ⏳ Token 分组管理
4. ⏳ 批量操作

## 数据库迁移

```sql
-- 添加新字段
ALTER TABLE tokens 
ADD COLUMN model_limit TEXT DEFAULT '[]',
ADD COLUMN ratio DECIMAL(10,6) DEFAULT 1.0;

-- 创建使用记录表
CREATE TABLE token_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    token_key VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    model VARCHAR(64) NOT NULL,
    tokens_used INT NOT NULL,
    quota_deducted INT NOT NULL,
    request_time TIMESTAMP NOT NULL,
    duration_ms INT DEFAULT 0,
    success BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_token_usage_token ON token_usage_logs(token_key);
CREATE INDEX idx_token_usage_user ON token_usage_logs(user_id);
CREATE INDEX idx_token_usage_time ON token_usage_logs(request_time);
```
