# ai-api is a Model Scheduler

[![License](https://img.shields.io/badge/License-AGPL%203.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![Vue Version](https://img.shields.io/badge/Vue-3.4+-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Status](https://img.shields.io/badge/status-beta-yellow)]()

> **注意**: 本项目正在积极开发中（Beta 阶段）。基础架构和 UI 已完成，更多功能正在陆续实现。

**ai-api** 是一个面向开发者和中小企业的开源 AI 模型调度系统。它提供统一的 API 接口，支持 OpenAI、Claude、Gemini 等主流 AI 模型的智能路由、计费管理和限流控制。**现在包含完整的 Web 管理界面！**

## ✨ 特性

### 核心功能

- 🔄 **统一 API 接口** - OpenAI 兼容的 API 格式，无缝切换不同模型提供商
- 🎯 **智能路由** - 基于成本、延迟、可用性的自动路由选择
- 💰 **计费管理** - 精确的 Token 计量和费用统计
- 🔒 **限流控制** - 多层级限流（全局/用户/API Key）
- 📊 **监控告警** - Prometheus + Grafana 完整监控方案
- 🔐 **认证授权** - JWT Token + API Key 双重认证

### 支持的模型

| 提供商 | 模型 | 状态 |
|--------|------|------|
| OpenAI | GPT-4, GPT-3.5-Turbo | ✅ |
| Anthropic | Claude 3, Claude 2 | 🔄 |
| Google | Gemini Pro | 🔄 |
| Azure | Azure OpenAI | 🔄 |

*✅ 已实现 🔄 计划中*

## 🚀 快速开始

### 方式一：Docker Compose（推荐）

```bash
# 克隆项目
git clone git@github.com:ChadBan/ai-api.git
cd ai-api/deploy/docker

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f app
```

服务启动后：
- API 服务：http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin_password)

### 方式二：本地运行

```bash
# 安装依赖
go mod download

# 配置数据库（MySQL 8.0+）
mysql -u root -p -e "CREATE DATABASE ai_scheduler;"

# 修改配置文件 configs/config.default.yaml

# 运行服务
go run cmd/server/main.go
```

## 📖 API 文档

### 认证

```bash
# 用户注册
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'

# 用户登录
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### 模型调用

```bash
# 获取模型列表
curl http://localhost:8080/v1/models

# 聊天补全（OpenAI 兼容格式）
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7
  }'
```

完整 API 文档请查看：[docs/API.md](docs/API.md)

## 🏗️ 架构设计

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│   API Gateway Layer     │
│  (Auth + Rate Limit)    │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│   Core Service Layer    │
│  (Router + Billing)     │
└──────┬──────────────────┘
       │
       ▼
┌─────────────────────────┐
│  Model Provider Layer   │
│  (OpenAI/Claude/etc.)   │
└─────────────────────────┘
```

详细架构说明：[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## ⚙️ 配置说明

主要配置项在 `configs/config.default.yaml`：

```yaml
server:
  port: 8080
  mode: debug

database:
  host: localhost
  port: 3306
  username: root
  password: root
  database: ai_scheduler

jwt:
  secret: "your-secret-key"  # 生产环境务必修改！
  expire: 720h

rate_limit:
  enabled: true
  user_qps: 10
  user_daily_requests: 1000
```

## 📦 部署

### Docker 部署

```bash
docker build -t ai-model-scheduler .
docker run -d -p 8080:8080 ai-model-scheduler
```

### Kubernetes 部署

详见 [deploy/kubernetes](deploy/kubernetes) 目录。

## 📊 监控

### Prometheus 指标

- `http_requests_total` - HTTP 请求总数
- `http_request_duration_seconds` - 请求延迟分布
- `model_requests_total` - 模型调用次数
- `model_tokens_total` - Token 使用量
- `rate_limit_hits_total` - 限流触发次数

### Grafana 仪表板

导入预设的仪表板配置，实时监控：
- QPS 和响应时间
- 各模型调用量
- Token 消耗统计
- 错误率和告警

## 🔧 开发

### 添加新的模型提供商

1. 在 `internal/model/provider.go` 定义提供商配置
2. 在 `internal/service/model/` 实现适配器
3. 更新配置文件

### 运行测试

```bash
go test ./...
```

### 代码规范

```bash
go fmt ./...
go vet ./...
golangci-lint run
```

## 🤝 贡献

我们欢迎各种形式的贡献！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

详见 [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## 📄 许可证

本项目采用 GNU Affero General Public License v3.0 许可证 - 详见 [LICENSE](LICENSE) 文件

## 📞 联系方式

- GitHub Issues: [提交问题](https://github.com/ai-model-scheduler/ai-model-scheduler/issues)
- 技术讨论：[GitHub Discussions](https://github.com/ai-model-scheduler/ai-model-scheduler/discussions)

## 🙏 致谢

感谢以下开源项目：

- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [GORM](https://github.com/go-gorm/gorm) - ORM 库
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Prometheus](https://prometheus.io/) - 监控系统

---

**Made with ❤️ by the AI Model Scheduler Team**
