-- AI Model Scheduler PostgreSQL 初始化脚本

-- 创建扩展（如果需要）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 注意：GORM 会自动创建表，这里可以添加一些初始化数据或自定义配置

-- 插入默认管理员用户（可选）
-- 密码：admin123 (需要在应用中哈希)
-- INSERT INTO users (email, password_hash, name, role, status, created_at, updated_at)
-- VALUES ('admin@example.com', '$2a$10$...', 'Admin', 'admin', 1, NOW(), NOW());

-- 创建索引（如果需要优化查询性能）
-- CREATE INDEX idx_users_email ON users(email);
-- CREATE INDEX idx_channels_status ON channels(status);
-- CREATE INDEX idx_billing_created_at ON billings(created_at);

