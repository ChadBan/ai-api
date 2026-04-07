# 管理后台配置指南

## 问题诊断

当访问 `/admin` 提示"没有管理员权限"时，按以下步骤排查：

### 1. 确认数据库中的管理员角色

连接到 PostgreSQL 数据库：

```bash
# 使用 Docker
docker exec -it ai-scheduler-postgres psql -U postgres -d ai_scheduler

# 或本地连接
psql -U postgres -d ai_scheduler
```

查看所有用户的角色：

```sql
SELECT id, email, name, role FROM users ORDER BY created_at;
```

设置管理员：

```sql
-- 将指定用户设置为管理员
UPDATE users SET role = 'admin' WHERE id = 4;

-- 或者通过邮箱
UPDATE users SET role = 'admin' WHERE email = 'your-email@example.com';
```

验证：

```sql
SELECT id, email, role FROM users WHERE role = 'admin';
```

### 2. 清除浏览器缓存并重新登录

1. 打开浏览器开发者工具（F12）
2. 清除 localStorage：
   ```javascript
   localStorage.clear()
   ```
3. 刷新页面
4. 使用管理员账号重新登录

### 3. 检查前端是否正确获取用户信息

打开浏览器开发者工具，查看 Network 标签：

1. 登录后应该调用 `GET /v1/user/self`
2. 检查响应中是否包含 `role: "admin"`

**正确响应示例**：
```json
{
  "user": {
    "id": 4,
    "email": "admin@example.com",
    "name": "Admin User",
    "role": "admin",
    "tier": "free",
    "status": 1
  }
}
```

### 4. 检查 Pinia Store

在浏览器控制台执行：

```javascript
// 查看当前用户信息
console.log(window.__PINIA__.state.value.user)

// 检查 isAdmin 值
console.log('Is Admin:', window.__PINIA__.state.value.user.userInfo?.role === 'admin')
```

## 快速创建管理员（脚本方式）

### 方法 1: 使用 SQL 脚本

创建文件 `create_admin.sql`：

```sql
-- 将第一个注册用户设为管理员
UPDATE users SET role = 'admin' WHERE id = (SELECT MIN(id) FROM users);

-- 验证
SELECT id, email, role FROM users WHERE role = 'admin';
```

执行：

```bash
docker exec -i ai-scheduler-postgres psql -U postgres -d ai_scheduler < create_admin.sql
```

### 方法 2: 使用 Go 脚本

创建文件 `cmd/admin/create_admin.go`：

```go
package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID    int64  `gorm:"primaryKey"`
	Email string `gorm:"size:128"`
	Role  string `gorm:"size:32"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=ai_scheduler port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 将第一个用户设为管理员
	var user User
	result := db.First(&user)
	if result.Error != nil {
		log.Fatal("No users found")
	}

	db.Model(&user).Update("role", "admin")
	fmt.Printf("User %d (%s) is now admin!\n", user.ID, user.Email)
}
```

执行：

```bash
cd cmd/admin
go run create_admin.go
```

## 前端调试

### 修改 AdminLayout.vue 添加调试信息

在 `onMounted` 中添加：

```javascript
onMounted(async () => {
  await userStore.fetchUserInfo()
  
  console.log('User Info:', userStore.userInfo)
  console.log('Is Admin:', userStore.isAdmin)
  
  if (!userStore.isAdmin) {
    ElMessage.error('没有管理员权限')
    router.push('/')
  }
})
```

### 常见错误

**错误 1**: Token 过期
- 解决：清除 localStorage，重新登录

**错误 2**: API 返回 404
- 解决：确保后端服务正在运行

**错误 3**: userInfo 为 null
- 解决：检查 `/v1/user/self` 接口是否正常

## 完整测试流程

1. **启动后端**
   ```bash
   cd cmd/server
   go run .
   ```

2. **启动前端**
   ```bash
   cd web
   npm run dev
   ```

3. **创建管理员**
   ```bash
   docker exec -i ai-scheduler-postgres psql -U postgres -d ai_scheduler \
     -c "UPDATE users SET role='admin' WHERE email='test@example.com'"
   ```

4. **登录测试**
   - 访问 http://localhost:3000/login
   - 使用管理员邮箱登录
   - 应该自动跳转到 `/admin`

5. **验证功能**
   - 访问 http://localhost:3000/admin/channels
   - 尝试创建渠道
   - 访问 http://localhost:3000/admin/users
   - 查看用户列表

## 安全提醒

⚠️ **生产环境注意事项**：

1. 不要硬编码管理员密码
2. 实现密码重置功能
3. 添加双因素认证（2FA）
4. 记录所有管理员操作日志
5. 限制管理后台 IP 访问
6. 定期审计管理员权限
