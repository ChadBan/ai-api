# Internal Packages

本目录包含项目的核心业务逻辑代码，外部包不应依赖这些包。

## 目录结构

```
internal/
├── handler/          # HTTP 层 - 处理请求/响应
├── service/          # 业务逻辑层 - 核心业务规则
├── repository/       # 数据访问层 - 数据库操作
├── model/            # 数据模型 - 数据结构定义
└── config/           # 配置管理 - 配置加载和验证
```

## 分层说明

### Handler (HTTP 层)

**位置**: `internal/handler/`

**职责**:
- 解析 HTTP 请求
- 参数验证
- 调用 Service 层
- 格式化响应

**组织方式**:
```
handler/
├── common/        # 公共处理器（健康检查、静态文件）
├── user/          # 用户相关（认证、Token、兑换码）
├── admin/         # 管理员相关（后台管理、日志）
├── relay/         # API 转发（OpenAI 兼容）
└── index.go       # 统一导出
```

**示例**:
```go
// internal/handler/user/auth.go
package user

type AuthHandler struct {
    db *gorm.DB
    secret string
}

func (h *AuthHandler) Login(c *gin.Context) {
    // 1. 验证参数
    // 2. 调用 service
    // 3. 返回响应
}
```

### Service (业务逻辑层)

**位置**: `internal/service/`

**职责**:
- 实现核心业务逻辑
- 事务管理
- 业务规则验证
- 调用 Repository 层

**文件列表**:
- `channel.go` - 渠道管理、负载均衡
- `billing.go` - 计费、余额管理
- `scheduler.go` - 定时任务
- `logger.go` - 日志服务

**示例**:
```go
// internal/service/channel.go
package service

type ChannelService struct {
    db *gorm.DB
    cache *redis.Client
}

func (s *ChannelService) SelectChannel(model string) (*model.Channel, error) {
    // 业务逻辑：选择最优渠道
}
```

### Repository (数据访问层)

**位置**: `internal/repository/`

**职责**:
- 数据库连接管理
- CRUD 操作
- 缓存管理
- 事务支持

**示例**:
```go
// internal/repository/database.go
package repository

type Database struct {
    DB *gorm.DB
}

func (d *Database) AutoMigrate(models ...interface{}) error {
    return d.DB.AutoMigrate(models...)
}
```

### Model (数据模型)

**位置**: `internal/model/`

**职责**:
- 数据结构定义
- ORM 映射
- 验证规则
- 关联关系

**组织方式**:
```
model/
├── user.go          # 用户、API Key
├── channel.go       # 渠道、Token、兑换码
├── billing.go       # 账单、余额
├── usage.go         # 使用记录
└── log.go           # 日志模型
```

**示例**:
```go
// internal/model/user.go
package model

type User struct {
    ID        int64     `gorm:"primaryKey"`
    Email     string    `gorm:"uniqueIndex"`
    Name      string
    Role      string    `gorm:"default:user"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## 依赖关系

```
handler → service → repository → model
   ↓         ↓           ↓
   └─────────┴───────────┘
```

**原则**:
- ❌ 不允许反向依赖
- ❌ 不允许跨层调用
- ✅ 依赖抽象接口
- ✅ 使用依赖注入

## 错误处理

### 定义业务错误

```go
// internal/service/errors.go
package service

import "errors"

var (
    ErrUserNotFound     = errors.New("user not found")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrInsufficientBalance = errors.New("insufficient balance")
)
```

### 包装错误

```go
func (s *UserService) GetUser(id int64) (*model.User, error) {
    user, err := s.repo.FindByID(id)
    if err != nil {
        return nil, fmt.Errorf("get user failed: %w", err)
    }
    return user, nil
}
```

### 返回错误

```go
func (h *AuthHandler) Login(c *gin.Context) {
    err := h.service.Login(...)
    if err != nil {
        if errors.Is(err, service.ErrUserNotFound) {
            c.JSON(404, gin.H{"error": "user not found"})
            return
        }
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
}
```

## 测试规范

### 单元测试

```go
// internal/service/user_test.go
package service

func TestUserService_CreateUser(t *testing.T) {
    // 准备测试数据
    // 调用被测方法
    // 断言结果
}
```

### 集成测试

```go
// tests/integration/user_test.go
package integration

func TestUserAPI_Register(t *testing.T) {
    // 启动测试服务器
    // 发送 HTTP 请求
    // 验证响应
}
```

## 最佳实践

### ✅ 推荐

1. **小函数** - 每个函数不超过 50 行
2. **单一职责** - 每个文件只做一件事
3. **清晰命名** - 见名知义
4. **错误处理** - 不忽略任何错误
5. **注释文档** - 导出类型必须有注释

### ❌ 避免

1. **上帝函数** - 几百行的函数
2. **循环依赖** - 包之间相互依赖
3. **魔术数字** - 使用具名常量
4. **忽略错误** - `err != nil` 必须处理
5. **硬编码** - 使用配置文件

## 性能考虑

1. **数据库查询**: 使用索引、避免 N+1 查询
2. **缓存**: 热点数据使用 Redis 缓存
3. **连接池**: 复用数据库连接
4. **批量操作**: 减少网络往返
5. **异步处理**: 非关键路径异步化

## 安全注意

1. **SQL 注入**: 使用 GORM 参数化查询
2. **XSS**: 转义用户输入
3. **CSRF**: 使用 Token 验证
4. **认证**: JWT Token 过期时间
5. **敏感数据**: 加密存储

---

**维护者**: AI Model Scheduler Team  
**最后更新**: 2026-03-31
