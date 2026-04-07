# 内部架构设计

本文档描述 ai-api 项目的内部架构设计。

## 目录结构

```
internal/
├── config/          # 配置管理
│   └── config.go    # 应用配置结构体
├── handler/         # HTTP 处理器层（Controller 层）
│   ├── index.go     # 统一导出所有 handler
│   ├── admin/       # 管理员相关 handler
│   │   ├── admin.go     # 用户/渠道/系统管理
│   │   └── logger.go    # 日志管理
│   ├── common/      # 公共 handler
│   │   ├── frontend.go  # 前端静态文件服务
│   │   ├── health.go    # 健康检查
│   │   └── models.go    # 模型管理
│   ├── middleware/  # Gin 中间件
│   │   ├── auth.go      # JWT 认证
│   │   ├── cors.go      # CORS 跨域
│   │   ├── logging.go   # 请求日志
│   │   └── ratelimit.go # 限流
│   ├── relay/       # API 转发
│   │   └── relay.go     # OpenAI 兼容 API 转发
│   └── user/        # 用户相关 handler
│       ├── auth.go        # 注册/登录
│       ├── invitation.go  # 邀请系统
│       ├── redemption.go  # 兑换码
│       ├── statistics.go  # 统计数据
│       └── token.go       # Token 管理
├── model/           # 数据模型层（Entity 层）
│   ├── apikey.go    # API Key 模型
│   ├── channel.go   # 渠道模型
│   ├── log.go       # 日志模型
│   ├── provider.go  # 提供商模型
│   ├── usage.go     # 使用量模型
│   └── user.go      # 用户模型
├── repository/      # 数据访问层
│   ├── database.go  # 数据库连接管理
│   └── redis.go     # Redis 连接管理
└── service/         # 业务逻辑层
    ├── billing.go    # 计费服务
    ├── channel.go    # 渠道服务
    ├── logger.go     # 日志服务
    └── scheduler.go  # 定时任务服务
```

## 分层架构

项目采用经典的三层架构：

### 1. Handler 层（Controller 层）

**职责**：
- 接收 HTTP 请求
- 参数验证
- 调用 Service 层处理业务
- 返回 HTTP 响应

**设计原则**：
- 不包含业务逻辑
- 只负责请求处理和响应格式化
- 按功能模块划分：admin、common、user、relay
- 通过 `index.go` 统一导出，避免循环依赖

**示例**：
```go
// user/auth.go
package user

type AuthHandler struct {
    db *gorm.DB
    jwtSecret string
}

func (h *AuthHandler) Login(c *gin.Context) {
    // 1. 参数验证
    // 2. 调用 service 层
    // 3. 返回响应
}
```

### 2. Service 层（Business Logic 层）

**职责**：
- 实现核心业务逻辑
- 调用 Repository 层进行数据持久化
- 事务管理
- 业务规则验证

**设计原则**：
- 可复用，不依赖特定 HTTP 上下文
- 可以组合多个 Repository 调用
- 包含业务规则和数据验证

**示例**：
```go
// service/billing.go
package service

type BillingService struct {
    db *gorm.DB
}

func (s *BillingService) CalculateTokens(modelName string, promptTokens, completionTokens int) (int, decimal.Decimal, error) {
    // 业务逻辑实现
}
```

### 3. Model 层（Entity 层）

**职责**：
- 定义数据结构
- 数据库表映射
- 基础验证规则

**设计原则**：
- 只包含数据和简单验证
- 不包含业务逻辑
- 按领域模型划分文件

### 4. Repository 层（Data Access 层）

**职责**：
- 数据库连接管理
- Redis 连接管理
- 基础 CRUD 操作（可选）

**设计原则**：
- 不包含业务逻辑
- 提供通用的数据访问方法
- 对 Service 层透明

## 依赖关系

```
handler → service → model
              ↓
         repository
```

**规则**：
- Handler 可以调用 Service
- Service 可以调用 Repository 和 Model
- 不允许反向依赖
- Model 层不被允许依赖任何其他层

## 模块化设计

### Handler 模块化

Handler 按功能划分为子模块：

- **admin/**: 后台管理功能（需要管理员权限）
- **user/**: 用户自助功能（需要用户认证）
- **common/**: 公共功能（无需认证或低权限）
- **relay/**: API 转发功能（需要 API Key 认证）
- **middleware/**: 通用中间件

每个子模块都是独立的 Go package，通过父 package 的 `index.go` 统一导出。

### 为什么使用 index.go？

1. **避免循环依赖**：cmd/server 只需要 import 一个 package
2. **统一接口**：所有 handler 构造函数都在 index.go 中声明
3. **清晰的分层**：子模块内部实现对外透明

## 最佳实践

### 1. 错误处理

```go
// 错误应该被包装并返回到 handler 层
if err := s.db.Create(&user).Error; err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// handler 层统一格式化错误
c.JSON(http.StatusInternalServerError, gin.H{
    "error": err.Error(),
})
```

### 2. 依赖注入

```go
// 在 cmd/server/api-router.go 中构造
authHandler := handler.NewAuthHandler(db, jwtSecret)
channelService := service.NewChannelService(db, logger)

// 注入到 handler
relayHandler := handler.NewRelayHandler(db, channelService, billingService)
```

### 3. 事务管理

```go
// 在 service 层管理事务
tx := s.db.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

if err := tx.Create(&user).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Create(&profile).Error; err != nil {
    tx.Rollback()
    return err
}

return tx.Commit().Error
```

## 扩展指南

### 添加新的 Handler

1. 在对应的子目录下创建文件（如 `user/new_feature.go`）
2. 定义 Handler 结构和构造函数
3. 确保 package 名称正确（如 `package user`）
4. 在 `cmd/server/api-router.go` 中注册路由

### 添加新的 Service

1. 在 `internal/service/` 下创建文件
2. 定义 Service 结构和构造函数
3. 实现业务逻辑
4. 在需要的 Handler 中注入使用

### 添加新的 Model

1. 在 `internal/model/` 下创建文件
2. 定义结构体和 GORM 钩子
3. 在 `cmd/server/main.go` 的 AutoMigrate 中注册

## 代码规范

1. **命名规范**：
   - Handler: `XxxHandler`
   - Service: `XxxService`
   - 构造函数：`NewXxx()`
   - 方法：驼峰命名，首字母大写表示公开

2. **文件组织**：
   - 每个文件不超过 500 行
   - 超过则考虑拆分
   - 相关的类型和方法放在同一文件

3. **注释**：
   - 所有公开的类型和方法必须有注释
   - 复杂逻辑需要说明意图
   - 使用中文注释
