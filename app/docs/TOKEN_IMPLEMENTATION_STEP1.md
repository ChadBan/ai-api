# Token 管理功能实现 - Step 1

## 完成时间
2026-04-01

## 实现内容

### 1. 数据模型更新

#### Token 模型（internal/model/channel.go）

新增了以下字段和方法：

**新增字段**：
- `ModelLimit string` - JSON 数组，允许的模型列表
- `Ratio float64` - 汇率倍率（默认 1.0）
- `AccessedTime *time.Time` - 最后访问时间

**新增方法**：
- `IsEnabled()` - 检查 Token 是否启用
- `IsExpired()` - 检查 Token 是否过期
- `HasQuota(quota int)` - 检查是否有足够配额
- `IsModelAllowed(modelName string)` - 检查模型是否在允许列表中
- `DeductQuota(quota int)` - 扣减配额

#### TokenUsageLog 模型（internal/model/token_usage.go）

新建 Token 使用记录模型，包含字段：
- `TokenKey` - 使用的 Token Key
- `UserID` - 用户 ID
- `Model` - 使用的模型名称
- `TokensUsed` - 使用的 tokens 总数
- `QuotaDeducted` - 扣除的配额
- `RequestTime` - 请求时间
- `DurationMs` - 请求耗时（毫秒）
- `Success` - 是否成功
- `ErrorMessage` - 错误信息
- `InputTokens` - 输入 tokens
- `OutputTokens` - 输出 tokens
- `ChannelID` - 使用的渠道 ID

### 2. API 接口增强

#### 创建 Token（增强版）

**端点**: `POST /v1/tokens`

**请求体**：
```json
{
  "name": "My API Key",
  "remain_quota": 10000,
  "unlimited_quota": false,
  "expired_time": 1735689600,
  "model_limit": ["gpt-3.5-turbo", "gpt-4"],
  "ratio": 1.0,
  "group": "default"
}
```

**新增参数说明**：
- `model_limit`: 允许的模型列表，如 `["gpt-3.5-turbo"]`
- `ratio`: 汇率倍率，VIP 用户可设置为 0.8 享受 8 折优惠

#### Token 充值

**端点**: `POST /v1/tokens/:id/topup`

**请求体**：
```json
{
  "quota": 5000,
  "reason": "手动充值"
}
```

**响应**：
```json
{
  "message": "Token topped up successfully",
  "quota_added": 5000,
  "remain_quota": 15000
}
```

#### Token 使用记录查询

**端点**: `GET /v1/tokens/:id/usage?page=1&page_size=20`

**响应**：
```json
{
  "items": [
    {
      "id": 1,
      "token_key": "sk-xxx",
      "model": "gpt-3.5-turbo",
      "tokens_used": 1500,
      "quota_deducted": 17280,
      "request_time": "2026-04-01T10:00:00Z",
      "duration_ms": 234,
      "success": true,
      "input_tokens": 1000,
      "output_tokens": 500
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### 3. 数据库迁移

执行迁移脚本：

```bash
docker exec -i ai-scheduler-postgres psql -U postgres -d ai_scheduler < scripts/migrate_tokens.sql
```

或手动执行 SQL：

```sql
-- 添加新字段到 tokens 表
ALTER TABLE tokens ADD COLUMN model_limit TEXT DEFAULT '[]';
ALTER TABLE tokens ADD COLUMN ratio DECIMAL(10,6) DEFAULT 1.0;
ALTER TABLE tokens ADD COLUMN accessed_time TIMESTAMP;
ALTER TABLE tokens ADD COLUMN group VARCHAR(64) DEFAULT 'default';

-- 创建 token_usage_logs 表
CREATE TABLE token_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    token_key VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    model VARCHAR(64) NOT NULL,
    tokens_used INT DEFAULT 0,
    quota_deducted INT DEFAULT 0,
    request_time TIMESTAMP NOT NULL,
    duration_ms INT DEFAULT 0,
    success BOOLEAN DEFAULT true,
    error_message VARCHAR(512),
    input_tokens INT DEFAULT 0,
    output_tokens INT DEFAULT 0,
    channel_id BIGINT
);

-- 创建索引
CREATE INDEX idx_token_usage_token ON token_usage_logs(token_key);
CREATE INDEX idx_token_usage_user ON token_usage_logs(user_id);
CREATE INDEX idx_token_usage_time ON token_usage_logs(request_time);
```

### 4. 使用示例

#### 创建只能访问 GPT-3.5 的 Token

```bash
curl -X POST http://localhost:8080/v1/tokens \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-3.5 Only",
    "remain_quota": 10000,
    "model_limit": ["gpt-3.5-turbo", "gpt-3.5-turbo-16k"],
    "ratio": 1.0
  }'
```

#### 创建 VIP 优惠汇率 Token

```bash
curl -X POST http://localhost:8080/v1/tokens \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "VIP Token",
    "remain_quota": 50000,
    "ratio": 0.8,
    "group": "vip"
  }'
```

#### Token 充值

```bash
curl -X POST http://localhost:8080/v1/tokens/123/topup \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "quota": 10000,
    "reason": "月度预算"
  }'
```

#### 查看使用记录

```bash
curl http://localhost:8080/v1/tokens/123/usage?page=1&page_size=50 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 下一步计划（Step 2）

### P0 - 核心功能

1. **完善 IsModelAllowed 方法**
   - 实现正确的 JSON 解析
   - 支持通配符匹配（如 `gpt-*`）

2. **Token 验证中间件**
   - 在 relay 层集成 Token 验证
   - 验证模型权限
   - 扣减配额并记录日志

3. **配额计算逻辑**
   - 根据模型价格计算配额
   - 应用 ratio 汇率倍率
   - 处理无限配额情况

### P1 - 重要功能

1. **使用记录自动写入**
   - 在 API 调用完成后异步写入
   - 记录详细信息（tokens、耗时、成功状态）

2. **统计图表**
   - 每日使用趋势
   - 模型使用分布
   - Token 消耗排行

## 测试验证

### 编译测试
```bash
cd /usr/local/go-path/ai-api
go build -o /tmp/ai-api-token ./cmd/server
# ✅ 编译成功
```

### 数据库迁移
```sql
-- 验证新字段
\d tokens

-- 验证新表
\d token_usage_logs
```

## 文件清单

### 新增文件
- `internal/model/token_usage.go` - Token 使用记录模型
- `scripts/migrate_tokens.sql` - 数据库迁移脚本
- `docs/TOKEN_IMPLEMENTATION_STEP1.md` - 本文档

### 修改文件
- `internal/model/channel.go` - 更新 Token 模型，添加方法
- `internal/handler/user/token.go` - 增强创建逻辑，添加充值和使用记录接口
- `cmd/server/api-router.go` - 注册新路由
- `cmd/server/main.go` - 注册 TokenUsageLog 模型
- `web/src/views/admin/Dashboard.vue` - 修复 ECharts 导入

## 注意事项

1. **向后兼容**：所有新增字段都有默认值，不影响现有 Token
2. **性能考虑**：使用记录表添加了多个索引，便于查询
3. **安全性**：Token 充值需要验证所有权
4. **TODO**：`IsModelAllowed` 方法需要实现完整的 JSON 解析和通配符匹配

## 参考资源

- [new-api Token 实现](https://github.com/QuantumNous/new-api)
- [PostgreSQL JSON 操作](https://www.postgresql.org/docs/current/functions-json.html)
- [配额计算最佳实践](./TOKEN_FEATURE_PLAN.md)
