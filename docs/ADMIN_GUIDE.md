# 管理后台使用指南

## 概述

本项目提供完整的后台管理系统，功能参考了 [new-api](https://github.com/QuantumNous/new-api) 项目。

## 管理员账号

### 创建第一个管理员

默认情况下，所有注册用户都是普通用户（`role: user`）。需要通过数据库手动设置第一个管理员：

```sql
-- 连接到 PostgreSQL 数据库
psql -U postgres -d ai_scheduler

-- 将指定用户设置为管理员
UPDATE users SET role = 'admin' WHERE email = 'your-admin-email@example.com';

-- 验证
SELECT id, email, name, role FROM users;
```

或者在注册后直接修改数据库：

```sql
-- 查看最新注册的用户
SELECT * FROM users ORDER BY created_at DESC LIMIT 1;

-- 设置为管理员
UPDATE users SET role = 'admin' WHERE id = 1;
```

## 管理后台 API

所有管理接口都需要管理员权限，路径前缀为 `/v1/admin`。

### 1. 用户管理

#### 获取用户列表
```bash
GET /v1/admin/users?page=1&page_size=20&status=1&role=user
Authorization: Bearer <admin_token>
```

**响应示例**:
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

#### 获取用户详情
```bash
GET /v1/admin/users/:id
Authorization: Bearer <admin_token>
```

#### 更新用户信息
```bash
PUT /v1/admin/users/:id
Authorization: Bearer <admin_token>

{
  "name": "新用户名",
  "status": 1,      // 1:正常 0:禁用
  "role": "user",   // user/admin
  "tier": "free"    // free/pro/enterprise
}
```

#### 封禁/解封用户
```bash
POST /v1/admin/users/:id/ban
Authorization: Bearer <admin_token>

{
  "ban": true  // true:封禁 false:解封
}
```

#### 删除用户
```bash
DELETE /v1/admin/users/:id
Authorization: Bearer <admin_token>
```

#### 给用户增加余额
```bash
POST /v1/admin/users/:id/balance
Authorization: Bearer <admin_token>

{
  "quota": 10000,
  "reason": "奖励"
}
```

### 2. 渠道管理

#### 获取渠道列表
```bash
GET /v1/admin/channels?page=1&page_size=20&status=1
Authorization: Bearer <admin_token>
```

#### 创建渠道
```bash
POST /v1/admin/channels
Authorization: Bearer <admin_token>

{
  "type": 1,              // 1:OpenAI, 2:Anthropic, 3:Azure,etc.
  "name": "My OpenAI",
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-xxx",
  "test_model": "gpt-3.5-turbo",
  "models": ["gpt-3.5-turbo", "gpt-4"],  // 支持的模型
  "priority": 1,          // 优先级（数字越小优先级越高）
  "weight": 100           // 权重（用于负载均衡）
}
```

#### 更新渠道
```bash
PUT /v1/admin/channels/:id
Authorization: Bearer <admin_token>

{
  "name": "Updated Name",
  "priority": 2,
  "weight": 80
}
```

#### 删除渠道
```bash
DELETE /v1/admin/channels/:id
Authorization: Bearer <admin_token>
```

#### 测试渠道
```bash
POST /v1/admin/channels/:id/test
Authorization: Bearer <admin_token>
```

### 3. 系统配置

#### 获取系统配置
```bash
GET /v1/admin/config
Authorization: Bearer <admin_token>
```

**配置项说明**:
- `register_enabled`: 是否允许注册
- `default_quota`: 新用户默认配额
- `price`: 美元兑人民币汇率
- `display_in_currency`: 是否显示货币金额
- `mfa_required`: 是否要求 MFA

#### 更新系统配置
```bash
PUT /v1/admin/config
Authorization: Bearer <admin_token>

{
  "register_enabled": true,
  "default_quota": 1000,
  "price": 7.2
}
```

### 4. 日志管理

#### 获取使用日志
```bash
GET /v1/admin/logs?key=gpt&page=1&page_size=20
Authorization: Bearer <admin_token>
```

## 前端管理界面

### 访问管理后台

1. 使用管理员账号登录
2. 系统会自动识别管理员角色并显示管理菜单
3. 访问路径：`http://localhost:3000/admin`

### 管理功能模块

- **仪表盘**: 查看系统统计数据和图表
- **渠道管理**: 添加、编辑、删除 AI 渠道
- **用户管理**: 查看用户列表、调整余额、封禁用户
- **Token 管理**: 管理 API Token
- **兑换码**: 创建和管理兑换码
- **日志查询**: 查看使用和请求日志
- **系统设置**: 配置系统参数

## 快速测试

### 1. 创建管理员

```bash
# 注册一个普通账号
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"123456"}'

# 在数据库中设置为管理员
psql -U postgres -d ai_scheduler -c "UPDATE users SET role='admin' WHERE email='admin@example.com'"
```

### 2. 登录获取 Token

```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"123456"}'
```

### 3. 测试管理接口

```bash
# 获取用户列表
curl http://localhost:8080/v1/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# 创建渠道
curl -X POST http://localhost:8080/v1/admin/channels \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": 1,
    "name": "Test Channel",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-test123",
    "priority": 1,
    "weight": 100
  }'
```

## 安全建议

1. **生产环境**: 不要使用默认的管理员邮箱
2. **密码策略**: 强制要求复杂密码
3. **访问控制**: 限制管理后台 IP 访问
4. **审计日志**: 记录所有管理员操作
5. **定期备份**: 定期备份数据库

## 故障排查

### Q: 登录后看不到管理菜单？
A: 检查用户的 `role` 字段是否为 `admin`

### Q: 调用管理接口返回 403？
A: 确认 Token 属于管理员账号

### Q: 如何重置管理员密码？
A: 直接更新数据库：
```sql
UPDATE users SET password_hash = 'bcrypt_hash' WHERE email = 'admin@example.com';
```

## 参考资源

- [new-api 项目](https://github.com/QuantumNous/new-api)
- [API 文档](./API.md)
- [部署指南](./DEPLOYMENT.md)
