# PostgreSQL 数据库迁移指南

本文档介绍如何从 MySQL 迁移到 PostgreSQL。

## 变更概述

### 1. 数据库驱动

- **之前**: `gorm.io/driver/mysql`
- **之后**: `gorm.io/driver/postgres`

### 2. 连接字符串格式

**MySQL**:
```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

**PostgreSQL**:
```
host=host user=username password=password dbname=database port=port sslmode=disable TimeZone=UTC
```

### 3. 数据类型调整

为了兼容 PostgreSQL，对以下模型字段进行了调整：

| 模型 | 字段 | 原类型 | 新类型 |
|------|------|--------|--------|
| APIKey | Permissions | `[]string` | `datatypes.JSON` |
| Channel | Models | `[]string` | `datatypes.JSON` |
| Model | Capabilities | `[]string` | `datatypes.JSON` |

### 4. 配置文件

更新 `configs/config.yaml`:

```yaml
database:
  driver: postgres  # 原来是 mysql
  host: localhost
  port: 5432        # 原来是 3306
  username: postgres
  password: postgres_password
  database: ai_scheduler
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h
```

## 部署方式

### Docker Compose 部署

使用新的 Docker Compose 配置（已自动使用 PostgreSQL）：

```bash
cd deploy/docker
docker-compose up -d
```

服务会自动连接到 PostgreSQL 容器。

### 手动部署

1. **安装 PostgreSQL 15+**

   ```bash
   # Ubuntu/Debian
   sudo apt-get install postgresql-15
   
   # macOS (Homebrew)
   brew install postgresql@15
   ```

2. **创建数据库和用户**

   ```bash
   sudo -u postgres psql
   ```

   ```sql
   CREATE DATABASE ai_scheduler;
   CREATE USER ai_scheduler WITH PASSWORD 'your_password';
   GRANT ALL PRIVILEGES ON DATABASE ai_scheduler TO ai_scheduler;
   \q
   ```

3. **更新配置文件**

   编辑 `configs/config.yaml`，设置正确的数据库连接信息。

4. **启动应用**

   ```bash
   go build -o ai-scheduler ./cmd/server
   ./ai-scheduler
   ```

   应用会自动创建所有必要的表。

## 从 MySQL 迁移数据

如果需要从现有的 MySQL 数据库迁移数据到 PostgreSQL：

### 方法 1: 使用 pgloader

```bash
# 安装 pgloader
sudo apt-get install pgloader

# 迁移数据
pgloader mysql://user:pass@localhost/ai_scheduler \
         pgsql://user:pass@localhost/ai_scheduler
```

### 方法 2: 导出导入

1. **从 MySQL 导出数据**

   ```bash
   mysqldump -u root -p --no-create-info ai_scheduler > data.sql
   ```

2. **修改 SQL 语法**

   - 将反引号 `` ` `` 改为双引号 `"`
   - 调整日期时间格式
   - 修改自增字段语法

3. **导入到 PostgreSQL**

   ```bash
   psql -U postgres -d ai_scheduler -f data.sql
   ```

## PostgreSQL 特定优化

### 1. JSON 字段查询

PostgreSQL 对 JSON 的支持非常强大，可以使用以下查询：

```sql
-- 查询包含特定模型的渠道
SELECT * FROM channels 
WHERE models @> '["gpt-4"]'::jsonb;

-- 查询权限包含 admin 的 API Key
SELECT * FROM api_keys 
WHERE permissions @> '["admin"]'::jsonb;
```

### 2. 索引优化

为常用查询字段添加索引：

```sql
CREATE INDEX idx_channels_status_priority ON channels(status, priority);
CREATE INDEX idx_billings_user_created ON billings(user_id, created_at);
CREATE INDEX idx_users_email ON users(email);
```

### 3. 性能监控

```sql
-- 查看慢查询
SELECT * FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

-- 查看表大小
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## 常见问题

### Q: 为什么选择 PostgreSQL？

A: PostgreSQL 在以下方面优于 MySQL：
- 更好的 JSON 支持
- 更强大的查询功能
- 更好的并发性能
- 更丰富的数据类型
- 更好的扩展性

### Q: 迁移会影响现有数据吗？

A: 如果正确执行迁移步骤，不会影响数据。但建议：
1. 先在测试环境验证
2. 备份所有数据
3. 制定回滚计划

### Q: 性能会受影响吗？

A: 在我们的测试中，PostgreSQL 的性能与 MySQL 相当，在某些场景下（如复杂查询、JSON 操作）甚至更好。

## 回滚到 MySQL

如果需要回滚到 MySQL：

1. 更新配置文件：
   ```yaml
   database:
     driver: mysql
     port: 3306
   ```

2. 更新 Docker Compose：
   ```bash
   cd deploy/docker
   # 恢复 docker-compose.yml 中的 MySQL 配置
   docker-compose down
   docker-compose up -d
   ```

3. 重新编译（如果需要）：
   ```bash
   go build -o ai-scheduler ./cmd/server
   ```

## 参考资源

- [PostgreSQL 官方文档](https://www.postgresql.org/docs/)
- [GORM PostgreSQL 驱动](https://github.com/go-gorm/gorm.io/driver/postgres)
- [pgloader 数据迁移工具](https://pgloader.io/)
