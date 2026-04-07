# AI Model Scheduler - 项目结构详解

## 快速导航

- [架构设计](docs/ARCHITECTURE.md) - 详细架构说明
- [内部包说明](internal/README.md) - internal 目录使用指南
- [Web 部署](docs/WEB_DEPLOYMENT.md) - 前端部署指南
- [快速开始](QUICKSTART.md) - 5 分钟上手

## 完整目录结构

```
ai-api/
│
├── cmd/                          # 应用程序入口
│   └── server/
│       ├── main.go               # 主入口（启动流程）
│       ├── api-router.go         # API 路由注册（模块化）
│       └── config.go             # 配置结构定义
│
├── internal/                     # 核心业务逻辑（私有包）
│   ├── handler/                  # HTTP 层
│   │   ├── common/               # 公共处理器
│   │   │   ├── health.go         # 健康检查
│   │   │   ├── models.go         # 模型列表
│   │   │   ├── frontend.go       # 前端静态文件
│   │   │   └── index.go          # 模块导出
│   │   ├── user/                 # 用户相关
│   │   │   ├── auth.go           # 认证
│   │   │   ├── token.go          # Token 管理
│   │   │   ├── invitation.go     # 邀请系统
│   │   │   ├── redemption.go     # 兑换码
│   │   │   ├── statistics.go     # 统计
│   │   │   └── index.go          # 模块导出
│   │   ├── admin/                # 管理员相关
│   │   │   ├── admin.go          # 后台管理
│   │   │   ├── logger.go         # 日志管理
│   │   │   └── index.go          # 模块导出
│   │   ├── relay/                # API 转发
│   │   │   ├── relay.go          # OpenAI 兼容 API
│   │   │   └── index.go          # 模块导出
│   │   └── index.go              # 统一导出所有 Handler
│   │
│   ├── service/                  # 业务逻辑层
│   │   ├── channel.go            # 渠道服务（负载均衡）
│   │   ├── billing.go            # 计费服务（余额管理）
│   │   ├── scheduler.go          # 定时任务服务
│   │   ├── logger.go             # 日志服务
│   │   └── errors.go             # 业务错误定义
│   │
│   ├── repository/               # 数据访问层
│   │   ├── database.go           # 数据库连接和迁移
│   │   ├── redis.go              # Redis 连接
│   │   └── interfaces.go         # Repository 接口定义
│   │
│   ├── model/                    # 数据模型
│   │   ├── user.go               # 用户、API Key
│   │   ├── channel.go            # 渠道、Token、兑换码
│   │   ├── billing.go            # 账单、余额
│   │   ├── usage.go              # 使用记录
│   │   ├── provider.go           # 提供商、模型
│   │   └── log.go                # 日志模型
│   │
│   └── config/                   # 配置管理
│       └── config.go             # 配置加载和验证
│
├── pkg/                          # 公共包（可被外部使用）
│   ├── client/                   # SDK 客户端
│   │   ├── openai.go             # OpenAI 客户端
│   │   └── anthropic.go          # Anthropic 客户端
│   └── types/                    # 公共类型
│       ├── response.go           # 通用响应
│       └── pagination.go         # 分页类型
│
├── web/                          # 前端代码（Vue 3）
│   ├── src/
│   │   ├── api/                  # API 客户端
│   │   ├── views/                # 页面组件
│   │   ├── components/           # 公共组件
│   │   ├── layouts/              # 布局组件
│   │   ├── router/               # 路由配置
│   │   └── store/                # 状态管理
│   ├── public/                   # 静态资源
│   ├── package.json              # 依赖配置
│   └── vite.config.js            # Vite 配置
│
├── configs/                      # 配置文件
│   ├── config.default.yaml       # 默认配置
│   └── config.local.yaml         # 本地配置（不提交）
│
├── deploy/                       # 部署配置
│   └── docker/
│       ├── Dockerfile.web        # 前后端联合构建
│       ├── docker-compose.web.yml # Web 部署配置
│       └── prometheus.yml        # Prometheus 配置
│
├── docs/                         # 项目文档
│   ├── ARCHITECTURE.md           # 架构设计
│   ├── WEB_DEPLOYMENT.md         # Web 部署
│   ├── IMPLEMENTATION_SUMMARY.md # 实现总结
│   └── PROJECT_STRUCTURE.md      # 项目结构（本文档）
│
├── scripts/                      # 工具脚本
│   ├── build.sh                  # 构建脚本
│   └── deploy.sh                 # 部署脚本
│
├── tests/                        # 测试代码
│   ├── unit/                     # 单元测试
│   └── integration/              # 集成测试
│
├── .gitignore                    # Git 忽略配置
├── LICENSE                       # 开源协议
├── README.md                     # 项目说明
├── QUICKSTART.md                 # 快速开始
├── go.mod                        # Go 模块定义
├── go.sum                        # Go 依赖校验
└── Makefile                      # Make 命令（可选）
```

## 关键文件说明

### 应用启动流程

```
cmd/server/main.go
  ↓
1. 加载配置 (config.Load)
2. 初始化日志 (initLogger)
3. 连接数据库 (repository.NewDatabase)
4. 数据库迁移 (db.AutoMigrate)
5. 创建服务实例 (NewChannelService, NewBillingService)
6. 初始化定时任务 (scheduler.InitTasks)
7. 注册 API 路由 (apiRouter.RegisterRoutes)
8. 启动 HTTP 服务器 (http.ListenAndServe)
```

### API 路由组织

```
cmd/server/api-router.go
  ↓
├── registerHealthRoutes()      # /health, /ready
├── registerAuthRoutes()        # /v1/auth/*
├── registerModelRoutes()       # /v1/models/*
├── registerRelayRoutes()       # /v1/chat/completions
├── registerBalanceRoutes()     # /v1/balance
├── registerRedemptionRoutes()  # /v1/redemptions/*
├── registerInvitationRoutes()  # /v1/invitations/*
├── registerStatisticsRoutes()  # /v1/statistics/*
├── registerTokenRoutes()       # /v1/tokens/*
├── registerAdminRoutes()       # /v1/admin/*
└── registerLogRoutes()         # /v1/logs/*
```

### 数据流转

```
HTTP Request
  ↓
Handler (参数验证)
  ↓
Service (业务逻辑)
  ↓
Repository (数据访问)
  ↓
Database/Redis
  ↓
Response
```

## 代码统计

| 目录 | 文件数 | 代码行数 | 说明 |
|------|--------|----------|------|
| cmd/server | 3 | ~200 | 应用入口 |
| internal/handler | 15 | ~3200 | HTTP 层 |
| internal/service | 4 | ~1400 | 业务逻辑 |
| internal/model | 6 | ~800 | 数据模型 |
| internal/repository | 2 | ~200 | 数据访问 |
| web/src | 20+ | ~3000 | 前端代码 |
| **总计** | **50+** | **~8800** | |

## 开发指南

### 添加新的 API 端点

1. 在 `internal/handler/{module}/` 创建处理器
2. 在 `internal/service/` 实现业务逻辑
3. 在 `cmd/server/api-router.go` 注册路由
4. 编写单元测试

### 添加新的数据模型

1. 在 `internal/model/` 定义结构体
2. 在 `cmd/server/main.go` 添加数据库迁移
3. 在 `internal/repository/` 实现 CRUD 方法
4. 在 `internal/service/` 实现业务逻辑

### 修改配置

1. 在 `configs/config.default.yaml` 添加配置项
2. 在 `internal/config/config.go` 添加结构体字段
3. 在代码中使用 `cfg.XXX` 访问

## 最佳实践

### ✅ 推荐做法

```go
// 1. 使用依赖注入
func NewUserService(db *gorm.DB, cache *redis.Client) *UserService {
    return &UserService{db: db, cache: cache}
}

// 2. 错误包装
if err != nil {
    return fmt.Errorf("create user failed: %w", err)
}

// 3. 上下文传递
func (s *Service) DoSomething(ctx context.Context, id int64) error {
    // 使用 ctx 控制超时和取消
}

// 4. 接口抽象
type UserRepository interface {
    FindByID(id int64) (*model.User, error)
    Create(user *model.User) error
}
```

### ❌ 避免的做法

```go
// 1. 全局变量
var globalDB *gorm.DB  // ❌

// 2. 忽略错误
doSomething()  // ❌ 没有检查错误

// 3. 硬编码
url := "http://localhost:8080"  // ❌

// 4. 大函数
func Process() {
    // 200 行代码...  // ❌
}
```

## 性能优化建议

1. **数据库查询**: 使用索引、预加载关联
2. **缓存**: 热点数据使用 Redis
3. **连接池**: 配置合理的 MaxOpenConns
4. **批量操作**: 使用 Batch Insert
5. **异步处理**: 非关键路径使用 goroutine

## 安全注意事项

1. **SQL 注入**: 使用 GORM 参数化查询
2. **XSS**: 转义 HTML 输出
3. **CSRF**: 验证请求来源
4. **认证**: JWT Token 设置合理过期时间
5. **敏感数据**: 密码 bcrypt 加密存储

---

**维护者**: AI Model Scheduler Team  
**最后更新**: 2026-03-31  
**版本**: v1.0.0
