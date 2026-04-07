# 快速开始指南

## 5 分钟快速部署

### 方式一：Docker Compose（最简单）

```bash
# 1. 克隆项目
git clone <your-repo-url>
cd ai-api

# 2. 启动所有服务
docker-compose -f deploy/docker/docker-compose.web.yml up -d

# 3. 查看状态
docker-compose -f deploy/docker/docker-compose.web.yml ps

# 4. 访问应用
# http://localhost:8080
```

**完成！** 🎉

### 方式二：本地开发环境

#### 前置要求
- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 7.0+

#### 步骤

**1. 准备数据库**

```sql
CREATE DATABASE ai_scheduler CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

**2. 配置后端**

```bash
# 复制配置文件
cp configs/config.default.yaml configs/config.yaml

# 编辑配置，修改数据库连接信息
vim configs/config.yaml
```

**3. 启动后端**

```bash
go mod download
go run cmd/server/main.go
```

**4. 安装并启动前端**

```bash
cd web

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

**5. 访问应用**

- 前端：http://localhost:3000
- 后端 API：http://localhost:8080

## 首次使用

### 1. 注册账号

访问 http://localhost:3000/register，填写信息注册。

### 2. 设置管理员权限

```sql
-- 连接数据库
mysql -u root -p ai_scheduler

-- 将第一个用户设置为管理员
UPDATE users SET role = 'admin' WHERE id = 1;
```

### 3. 登录管理后台

使用管理员账号登录，访问 http://localhost:3000/admin

### 4. 添加 AI 渠道

在管理后台添加你的第一个 AI 渠道：
- 类型：OpenAI
- 名称：My OpenAI Channel
- BaseURL: https://api.openai.com/v1
- API Key: sk-your-openai-key

### 5. 创建 Token

在用户中心创建 Token，用于 API 调用。

### 6. 开始使用

- **API 调用**: 使用创建的 Token 访问 `/api/v1/chat/completions`
- **Web 对话**: 访问 Playground 页面（开发中）

## 常见问题

### Q: Docker 启动失败？

检查端口占用：
```bash
lsof -i :8080
lsof -i :3306
lsof -i :6379
```

### Q: 前端无法连接后端？

确保后端运行在 8080 端口，或修改 `web/vite.config.js` 中的代理配置。

### Q: 数据库迁移失败？

删除数据库重新创建：
```sql
DROP DATABASE IF EXISTS ai_scheduler;
CREATE DATABASE ai_scheduler;
```

### Q: 忘记密码？

```sql
UPDATE users SET password = '$2a$10$...' WHERE email = 'your@email.com';
-- 需要使用 bcrypt 生成新密码哈希
```

或者重新注册一个账号。

## 下一步

- 📖 查看 [API 文档](docs/API.md)
- 🔧 查看 [部署指南](docs/WEB_DEPLOYMENT.md)
- 📊 查看 [实现总结](docs/IMPLEMENTATION_SUMMARY.md)

## 技术支持

遇到问题？
1. 查看 [GitHub Issues](https://github.com/ai-model-scheduler/ai-model-scheduler/issues)
2. 查看日志：`docker-compose logs app`
3. 检查配置文件

---

祝你使用愉快！🚀
