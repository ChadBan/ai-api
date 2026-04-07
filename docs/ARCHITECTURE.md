# AI Model Scheduler - 架构设计文档

## 项目结构

```
ai-api/
├── cmd/
│   └── server/
│       ├── main.go           # 应用程序入口
│       ├── api-router.go     # API 路由注册
│       └── config.go         # 应用配置
├── internal/
│   ├── handler/              # HTTP 层（按功能模块组织）
│   │   ├── common/           # 公共处理器
│   │   │   ├── health.go     # 健康检查
│   │   │   ├── models.go     # 模型列表
│   │   │   └── frontend.go   # 前端静态文件
│   │   ├── user/             # 用户相关处理器
│   │   │   ├── auth.go       # 认证
│   │   │   ├── token.go      # Token 管理
│   │   │   ├── invitation.go # 邀请系统
│   │   │   ├── redemption.go # 兑换码
│   │   │   └── statistics.go # 统计
│   │   ├── admin/            # 管理员处理器
│   │   │   ├── admin.go      # 后台管理
│   │   │   └── logger.go     # 日志管理
│   │   ├── relay/            # API 转发
│   │   │   └── relay.go      # OpenAI 兼容 API
│   │   └── index.go          # 统一导出
│   ├── service/              # 业务逻辑层
│   │   ├── channel.go        # 渠道服务
│   │   ├── billing.go        # 计费服务
│   │   ├── scheduler.go      # 定时任务
│   │   └── logger.go         # 日志服务
│   ├── model/                # 数据模型
│   │   ├── user.go           # 用户模型
│   │   ├── channel.go        # 渠道模型
│   │   ├── billing.go        # 计费模型
│   │   └── ... 
│   ├── repository/           # 数据访问层
│   │   ├── database.go       # 数据库连接
│   │   └── redis.go          # Redis 连接
│   └── config/               # 配置管理
│       └── config.go         # 配置加载
├── pkg/                      # 公共包
│   ├── client/               # 客户端 SDK
│   └── types/                # 公共类型
├── web/                      # 前端代码
│   └── src/
├── deploy/                   # 部署配置
│   └── docker/
└── docs/                     # 文档
```

## 分层架构

### 1. HTTP 层 (handler)
- **职责**: 处理 HTTP 请求、参数验证、响应格式化
- **原则**: 
  - 每个功能模块独立目录
  - 不包含业务逻辑
  - 只调用 service 层
  - 统一的错误处理

### 2. 业务逻辑层 (service)
- **职责**: 核心业务逻辑、事务管理、数据验证
- **原则**:
  - 无状态设计
  - 可组合性
  - 依赖注入
  - 统一的错误定义

### 3. 数据访问层 (repository)
- **职责**: 数据库操作、缓存管理
- **原则**:
  - 单一职责
  - 接口抽象
  - 支持多数据源

### 4. 数据模型层 (model)
- **职责**: 数据结构定义、ORM 映射
- **原则**:
  - 按业务领域分组
  - 清晰的关联关系
  - 包含验证逻辑

## 设计原则

### 1. 单一职责 (SRP)
每个文件/包只负责一个明确的职责

### 2. 依赖倒置 (DIP)
高层模块不依赖低层模块，都依赖抽象

### 3. 接口隔离 (ISP)
使用小接口而不是大接口

### 4. 开闭原则 (OCP)
对扩展开放，对修改关闭

## 代码规范

### Handler 命名
```go
// ✅ 好的命名
user/AuthHandler.go
admin/UserManager.go
relay/OpenAIProxy.go

// ❌ 坏的命名
handler.go
handlers_v2.go
```

### Service 接口
```go
type UserService interface {
    CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
    GetUserByID(ctx context.Context, id int64) (*User, error)
    UpdateUser(ctx context.Context, user *User) error
}
```

### 错误处理
```go
// 定义业务错误
var (
    ErrUserNotFound = errors.New("user not found")
    ErrUnauthorized = errors.New("unauthorized")
)

// 使用 errors.Wrap
if err != nil {
    return fmt.Errorf("create user failed: %w", err)
}
```

## 测试策略

```
internal/
├── handler/
│   └── user/
│       ├── auth.go
│       └── auth_test.go
├── service/
│   └── user.go
│   └── user_test.go
└── repository/
    └── user.go
    └── user_test.go
```

## 性能优化

1. **连接池**: 数据库、Redis 连接复用
2. **缓存策略**: 热点数据缓存
3. **批量操作**: 减少数据库往返
4. **异步处理**: 非关键路径异步化

---

Last updated: 2026-03-31
