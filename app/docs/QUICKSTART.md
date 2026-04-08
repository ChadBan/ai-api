# 快速启动指南

## 运行应用

### 方式 1: 从项目根目录运行

```bash
cd /usr/local/go-path/ai-api
go run cmd/server/*.go
```

### 方式 2: 从 cmd/server 目录运行

```bash
cd cmd/server
go run .
```

### 方式 3: 编译后运行

```bash
go build -o ai-scheduler ./cmd/server
./ai-scheduler
```

## 首次运行 - 配置数据库

### 使用 Docker Compose（推荐）

最简单的启动方式是使用 Docker Compose，它会自动创建 PostgreSQL 和 Redis 容器：

```bash
cd deploy/docker
docker-compose up -d
```

这会启动：
- PostgreSQL 15 (端口 5432)
- Redis 7 (端口 6379)
- 应用 (端口 8080)
- Prometheus (端口 9090)
- Grafana (端口 3000)

查看日志：
```bash
docker-compose logs -f app
```

### 手动配置 PostgreSQL

如果要在本地运行 PostgreSQL：

#### 1. 安装 PostgreSQL

**macOS (Homebrew)**:
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian**:
```bash
sudo apt-get update
sudo apt-get install postgresql-15 postgresql-contrib-15
sudo systemctl start postgresql
```

#### 2. 创建数据库和用户

```bash
sudo -u postgres psql
```

在 PostgreSQL 命令行中执行：
```sql
CREATE DATABASE ai_scheduler;
CREATE USER ai_scheduler WITH PASSWORD 'ai_scheduler_password';
GRANT ALL PRIVILEGES ON DATABASE ai_scheduler TO ai_scheduler;
\q
```

#### 3. 更新配置文件

编辑 `configs/config.default.yaml`：

```yaml
database:
  driver: postgres
  host: localhost
  port: 5432
  username: ai_scheduler
  password: ai_scheduler_password
  database: ai_scheduler
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
```

#### 4. 运行应用

```bash
cd /usr/local/go-path/ai-api
go run cmd/server/*.go
```

## 验证运行

应用启动后，你会看到类似日志：

```json
{"level":"info","ts":1774947190,"msg":"starting AI Model Scheduler","version":"0.1.0","mode":"debug"}
{"level":"info","ts":1774947190,"msg":"database migration completed"}
{"level":"info","ts":1774947190,"msg":"server starting","port":8080,"mode":"debug"}
```

### 测试健康检查

```bash
curl http://localhost:8080/health
```

应该返回：
```json
{"status":"ok"}
```

### 测试 API

```bash
# 获取模型列表
curl http://localhost:8080/v1/models

# 注册新用户
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

## 常见问题

### Q: 提示 "database does not exist"

A: 这是因为 PostgreSQL 还没有创建数据库。请按照上面的步骤创建数据库和用户。

### Q: 提示 "connection refused"

A: 检查 PostgreSQL 是否正在运行：

```bash
# macOS
brew services list

# Linux
systemctl status postgresql
```

### Q: 如何重置数据库？

A: 删除所有表并重新迁移：

```bash
# 连接到数据库
psql -U ai_scheduler -d ai_scheduler

# 删除所有表
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;

# 重启应用，GORM 会自动重新创建表
```

### Q: 如何查看日志？

A: 默认日志输出到控制台。要保存到文件，修改配置：

```yaml
log:
  level: info
  format: json
  output: file
  file_path: logs/app.log
```

然后创建日志目录：
```bash
mkdir -p logs
```

## 下一步

- [API 文档](./API.md) - 查看所有可用的 API 端点
- [部署指南](./DEPLOYMENT.md) - 生产环境部署说明
- [PostgreSQL 迁移指南](./POSTGRESQL_MIGRATION.md) - 数据库详细信息
