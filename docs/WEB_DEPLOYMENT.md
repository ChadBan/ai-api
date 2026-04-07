# Web 前端部署指南

## 概述

本项目现在包含完整的前后端分离架构：
- **后端**: Golang + Gin + GORM
- **前端**: Vue 3 + Element Plus
- **部署**: Docker Compose 一键部署

## 快速开始

### 方式一：Docker Compose 部署（推荐）

```bash
# 进入项目目录
cd /usr/local/go-path/ai-api

# 使用 Docker Compose 启动所有服务
docker-compose -f deploy/docker/docker-compose.web.yml up -d

# 查看日志
docker-compose -f deploy/docker/docker-compose.web.yml logs -f app

# 访问应用
# http://localhost:8080
```

### 方式二：本地开发部署

#### 1. 启动后端服务

```bash
# 确保 MySQL 和 Redis 已安装
# 修改配置文件 configs/config.default.yaml

# 启动后端
go run cmd/server/main.go
```

#### 2. 安装并启动前端

```bash
cd web

# 安装依赖
npm install

# 开发模式（自动热重载）
npm run dev

# 访问 http://localhost:3000
```

#### 3. 生产构建

```bash
cd web

# 构建生产版本
npm run build

# 构建产物在 dist/ 目录
# 后端会自动服务这些静态文件
```

或者使用一键安装脚本：

```bash
./web/install.sh
```

## 项目结构

```
ai-api/
├── cmd/                    # 后端入口
├── internal/               # 后端核心代码
│   ├── handler/           # HTTP 处理器
│   ├── service/           # 业务逻辑
│   ├── model/             # 数据模型
│   └── repository/        # 数据访问层
├── web/                    # 前端代码
│   ├── src/
│   │   ├── api/           # API 接口
│   │   ├── views/         # 页面组件
│   │   ├── components/    # 公共组件
│   │   ├── layouts/       # 布局组件
│   │   ├── router/        # 路由配置
│   │   └── store/         # 状态管理
│   ├── package.json
│   └── vite.config.js
├── deploy/docker/          # Docker 部署配置
└── docs/                   # 文档
```

## 功能模块

### 用户端功能
- ✅ 用户登录/注册
- ✅ 控制台（余额、统计信息）
- ✅ Token 管理（创建、删除、启用/禁用）
- ✅ 余额充值（占位，待实现支付）
- ✅ AI 对话 Playground（占位，待实现）

### 管理端功能
- ✅ 数据看板（用户数、渠道数等统计）
- ✅ 渠道管理（增删改查、测试）
- ✅ 用户管理（占位）
- ✅ Token 管理（占位）
- ✅ 兑换码管理（占位）
- ✅ 日志管理（占位）

## 默认账号

首次启动后，需要注册第一个用户。

通过数据库手动设置管理员：

```sql
UPDATE users SET role = 'admin' WHERE email = 'your@email.com';
```

## API 接口

所有 API 接口都在 `/api/v1` 路径下：

### 认证接口
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/register` - 用户注册

### 用户接口
- `GET /api/v1/balance` - 查询余额
- `GET /api/v1/tokens` - Token 列表
- `POST /api/v1/tokens` - 创建 Token
- `DELETE /api/v1/tokens/:id` - 删除 Token

### 管理接口
- `GET /api/v1/admin/channels` - 渠道列表
- `POST /api/v1/admin/channels` - 创建渠道
- `GET /api/v1/admin/users` - 用户列表

详细 API 文档请查看 [docs/API.md](API.md)

## 环境变量

可以通过环境变量配置：

```bash
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=ai_scheduler

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT 配置
JWT_SECRET=your-secret-key-change-in-production
```

## 开发指南

### 前端开发

1. 安装依赖：`npm install`
2. 启动开发服务器：`npm run dev`
3. 访问：http://localhost:3000
4. API 代理到后端：http://localhost:8080

### 添加新页面

在 `web/src/views/` 目录下创建新的 `.vue` 文件，然后在 `web/src/router/index.js` 中添加路由。

### 添加新 API

在 `web/src/api/index.js` 中添加新的 API 方法。

## 常见问题

### Q: 前端无法连接后端？
A: 确保后端运行在 8080 端口，并且 CORS 已配置。

### Q: 登录后跳转到 admin 页面报错？
A: 确保用户有 admin 角色权限。

### Q: Docker 启动失败？
A: 检查端口是否被占用，确保 MySQL 和 Redis 端口可用。

## 下一步开发计划

1. ✅ 基础 UI 框架
2. ✅ 登录/注册
3. ✅ Token 管理
4. ✅ 渠道管理
5. ⏳ 完整的用户管理
6. ⏳ AI 对话 Playground
7. ⏳ 支付集成
8. ⏳ 2FA/OAuth 登录
9. ⏳ 数据可视化图表

## License

Apache 2.0
