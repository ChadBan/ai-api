-- Token 管理功能数据库迁移脚本
-- 执行时间：2026-04-01

-- 1. 更新 tokens 表，添加新字段
DO $$ 
BEGIN 
    -- 添加 model_limit 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'tokens' AND column_name = 'model_limit') THEN
        ALTER TABLE tokens ADD COLUMN model_limit TEXT DEFAULT '[]';
        COMMENT ON COLUMN tokens.model_limit IS 'JSON 数组，允许的模型列表';
    END IF;

    -- 添加 ratio 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'tokens' AND column_name = 'ratio') THEN
        ALTER TABLE tokens ADD COLUMN ratio DECIMAL(10,6) DEFAULT 1.0;
        COMMENT ON COLUMN tokens.ratio IS '汇率倍率';
    END IF;

    -- 添加 accessed_time 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'tokens' AND column_name = 'accessed_time') THEN
        ALTER TABLE tokens ADD COLUMN accessed_time TIMESTAMP;
        CREATE INDEX IF NOT EXISTS idx_tokens_accessed ON tokens(accessed_time);
    END IF;

    -- 添加 group 字段
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'tokens' AND column_name = 'group') THEN
        ALTER TABLE tokens ADD COLUMN group VARCHAR(64) DEFAULT 'default';
        CREATE INDEX IF NOT EXISTS idx_tokens_group ON tokens(group);
    END IF;
END $$;

-- 2. 创建 token_usage_logs 表
CREATE TABLE IF NOT EXISTS token_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    token_key VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    model VARCHAR(64) NOT NULL,
    tokens_used INT DEFAULT 0,
    quota_deducted INT DEFAULT 0,
    request_time TIMESTAMP NOT NULL,
    duration_ms INT DEFAULT 0,
    success BOOLEAN DEFAULT true,
    error_message VARCHAR(512),
    input_tokens INT DEFAULT 0,
    output_tokens INT DEFAULT 0,
    channel_id BIGINT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_token_usage_token ON token_usage_logs(token_key);
CREATE INDEX IF NOT EXISTS idx_token_usage_user ON token_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_time ON token_usage_logs(request_time);
CREATE INDEX IF NOT EXISTS idx_token_usage_model ON token_usage_logs(model);
CREATE INDEX IF NOT EXISTS idx_token_usage_channel ON token_usage_logs(channel_id);

-- 添加注释
COMMENT ON TABLE token_usage_logs IS 'Token 使用记录表';
COMMENT ON COLUMN token_usage_logs.token_key IS '使用的 Token Key';
COMMENT ON COLUMN token_usage_logs.user_id IS '用户 ID';
COMMENT ON COLUMN token_usage_logs.model IS '使用的模型名称';
COMMENT ON COLUMN token_usage_logs.tokens_used IS '使用的 tokens 总数';
COMMENT ON COLUMN token_usage_logs.quota_deducted IS '扣除的配额';
COMMENT ON COLUMN token_usage_logs.request_time IS '请求时间';
COMMENT ON COLUMN token_usage_logs.duration_ms IS '请求耗时（毫秒）';
COMMENT ON COLUMN token_usage_logs.success IS '是否成功';
COMMENT ON COLUMN token_usage_logs.error_message IS '错误信息';
COMMENT ON COLUMN token_usage_logs.input_tokens IS '输入 tokens';
COMMENT ON COLUMN token_usage_logs.output_tokens IS '输出 tokens';
COMMENT ON COLUMN token_usage_logs.channel_id IS '使用的渠道 ID';

-- 3. 创建统计视图（可选）
CREATE OR REPLACE VIEW v_token_daily_usage AS
SELECT 
    DATE(request_time) as usage_date,
    token_key,
    user_id,
    model,
    COUNT(*) as request_count,
    SUM(tokens_used) as total_tokens,
    SUM(quota_deducted) as total_quota
FROM token_usage_logs
GROUP BY DATE(request_time), token_key, user_id, model;

COMMENT ON VIEW v_token_daily_usage IS 'Token 每日使用统计视图';
