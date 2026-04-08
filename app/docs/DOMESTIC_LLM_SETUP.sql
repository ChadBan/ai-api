-- 国内大模型渠道和模型初始化 SQL
-- 执行此脚本添加豆包、阿里通义、DeepSeek、MiniMax、智谱 AI 支持

-- 1. 添加国内大模型提供商（如果 providers 表存在）
-- 注意：如果系统使用 channels 直接管理，可跳过此步

-- 2. 添加默认渠道配置示例
-- 这些是示例配置，实际使用时需要替换为真实的 API Key 和 BaseURL

-- 豆包 (ByteDance)
INSERT INTO channels (type, name, base_url, api_key, status, models, priority, weight)
VALUES (
  14, -- 豆包类型
  '豆包官方',
  'https://ark.cn-beijing.volces.com/api/v3',
  'YOUR_DOUBAO_API_KEY', -- 需要替换为真实 API Key
  1,
  '["doubao-lite-4k","doubao-pro-4k","doubao-lite-32k","doubao-pro-32k"]',
  1,
  100
) ON CONFLICT DO NOTHING;

-- 阿里通义千问
INSERT INTO channels (type, name, base_url, api_key, status, models, priority, weight)
VALUES (
  15, -- 阿里类型
  '阿里通义官方',
  'https://dashscope.aliyuncs.com/compatible-mode',
  'YOUR_ALI_API_KEY', -- 需要替换为真实 API Key
  1,
  '["qwen-turbo","qwen-plus","qwen-max","qwen-max-longcontext"]',
  1,
  100
) ON CONFLICT DO NOTHING;

-- DeepSeek
INSERT INTO channels (type, name, base_url, api_key, status, models, priority, weight)
VALUES (
  16, -- DeepSeek 类型
  'DeepSeek 官方',
  'https://api.deepseek.com',
  'YOUR_DEEPSEEK_API_KEY', -- 需要替换为真实 API Key
  1,
  '["deepseek-chat","deepseek-coder"]',
  1,
  100
) ON CONFLICT DO NOTHING;

-- MiniMax
INSERT INTO channels (type, name, base_url, api_key, status, models, priority, weight)
VALUES (
  17, -- MiniMax 类型
  'MiniMax 官方',
  'https://api.minimax.chat/v1',
  'YOUR_MINIMAX_API_KEY', -- 需要替换为真实 API Key
  1,
  '["abab6.5-chat","abab6.5g-chat","abab6.5t-chat"]',
  1,
  100
) ON CONFLICT DO NOTHING;

-- 智谱 AI
INSERT INTO channels (type, name, base_url, api_key, status, models, priority, weight)
VALUES (
  18, -- 智谱 AI 类型
  '智谱 AI 官方',
  'https://open.bigmodel.cn/api/paas/v4',
  'YOUR_ZHIPU_API_KEY', -- 需要替换为真实 API Key
  1,
  '["glm-4","glm-4-flash","glm-3-turbo"]',
  1,
  100
) ON CONFLICT DO NOTHING;

-- 3. 添加模型到 models 表（如果使用了 models 表）
-- 这样 Playground 页面才能动态加载这些模型

-- 豆包模型
INSERT INTO models (provider_id, name, display_name, type, context_window, max_tokens, input_price, output_price, status)
VALUES 
(1, 'doubao-lite-4k', '豆包 Lite 4K', 'chat', 4096, 2048, 0.0003, 0.0006, 1),
(1, 'doubao-pro-4k', '豆包 Pro 4K', 'chat', 4096, 2048, 0.0008, 0.0020, 1),
(1, 'doubao-lite-32k', '豆包 Lite 32K', 'chat', 32768, 4096, 0.0006, 0.0012, 1),
(1, 'doubao-pro-32k', '豆包 Pro 32K', 'chat', 32768, 4096, 0.0015, 0.0030, 1)
ON CONFLICT DO NOTHING;

-- 阿里通义模型
INSERT INTO models (provider_id, name, display_name, type, context_window, max_tokens, input_price, output_price, status)
VALUES 
(1, 'qwen-turbo', '通义千问 Turbo', 'chat', 8192, 4096, 0.0003, 0.0006, 1),
(1, 'qwen-plus', '通义千问 Plus', 'chat', 32768, 8192, 0.0008, 0.0020, 1),
(1, 'qwen-max', '通义千问 Max', 'chat', 32768, 8192, 0.0015, 0.0030, 1),
(1, 'qwen-max-longcontext', '通义千问 Max 长文本', 'chat', 131072, 8192, 0.0020, 0.0040, 1)
ON CONFLICT DO NOTHING;

-- DeepSeek 模型
INSERT INTO models (provider_id, name, display_name, type, context_window, max_tokens, input_price, output_price, status)
VALUES 
(1, 'deepseek-chat', 'DeepSeek Chat', 'chat', 32768, 4096, 0.0002, 0.0004, 1),
(1, 'deepseek-coder', 'DeepSeek Coder', 'chat', 16384, 2048, 0.0002, 0.0004, 1)
ON CONFLICT DO NOTHING;

-- MiniMax 模型
INSERT INTO models (provider_id, name, display_name, type, context_window, max_tokens, input_price, output_price, status)
VALUES 
(1, 'abab6.5-chat', 'MiniMax Abab6.5 Chat', 'chat', 8192, 4096, 0.0005, 0.0010, 1),
(1, 'abab6.5g-chat', 'MiniMax Abab6.5G Chat', 'chat', 8192, 4096, 0.0005, 0.0010, 1),
(1, 'abab6.5t-chat', 'MiniMax Abab6.5T Chat', 'chat', 8192, 4096, 0.0005, 0.0010, 1)
ON CONFLICT DO NOTHING;

-- 智谱 AI 模型
INSERT INTO models (provider_id, name, display_name, type, context_window, max_tokens, input_price, output_price, status)
VALUES 
(1, 'glm-4', '智谱 GLM-4', 'chat', 128000, 4096, 0.0010, 0.0020, 1),
(1, 'glm-4-flash', '智谱 GLM-4 Flash', 'chat', 8192, 4096, 0.0001, 0.0002, 1),
(1, 'glm-3-turbo', '智谱 GLM-3 Turbo', 'chat', 128000, 4096, 0.0005, 0.0010, 1)
ON CONFLICT DO NOTHING;

-- 4. 查询验证
SELECT 'Channels' as table_name, COUNT(*) as count FROM channels WHERE type IN (14,15,16,17,18)
UNION ALL
SELECT 'Models' as table_name, COUNT(*) as count FROM models WHERE name IN (
  'doubao-lite-4k', 'doubao-pro-4k', 'doubao-lite-32k', 'doubao-pro-32k',
  'qwen-turbo', 'qwen-plus', 'qwen-max', 'qwen-max-longcontext',
  'deepseek-chat', 'deepseek-coder',
  'abab6.5-chat', 'abab6.5g-chat', 'abab6.5t-chat',
  'glm-4', 'glm-4-flash', 'glm-3-turbo'
);
