# AI Model Scheduler - Web 前端

基于 Vue 3 + Element Plus 的现代化管理后台前端。

## 技术栈

- **框架**: Vue 3.4
- **UI 组件库**: Element Plus 2.6
- **状态管理**: Pinia 2.1
- **路由**: Vue Router 4.3
- **HTTP 客户端**: Axios 1.6
- **构建工具**: Vite 5.2
- **图表**: ECharts 5.5
- **图标**: Element Plus Icons

## 功能模块

### 用户端
- ✅ 登录/注册
- ✅ 控制台（余额、统计）
- ✅ Token 管理
- ✅ 余额充值
- ✅ AI 对话 Playground

### 管理端
- ✅ 数据看板
- ✅ 渠道管理
- ✅ 用户管理
- ✅ Token 管理
- ✅ 兑换码管理
- ✅ 日志管理

## 快速开始

### 安装依赖

```bash
cd web
npm install
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:3000

### 生产构建

```bash
npm run build
```

构建产物输出到 `dist` 目录

## 项目结构

```
web/
├── src/
│   ├── api/              # API 接口
│   │   └── index.js
│   ├── assets/           # 静态资源
│   ├── components/       # 公共组件
│   ├── layouts/          # 布局组件
│   │   ├── UserLayout.vue
│   │   └── AdminLayout.vue
│   ├── router/           # 路由配置
│   │   └── index.js
│   ├── store/            # 状态管理
│   │   └── user.js
│   ├── views/            # 页面组件
│   │   ├── common/       # 公共页面
│   │   │   ├── Login.vue
│   │   │   └── Register.vue
│   │   ├── user/         # 用户页面
│   │   │   ├── Dashboard.vue
│   │   │   ├── Tokens.vue
│   │   │   ├── Balance.vue
│   │   │   └── Playground.vue
│   │   └── admin/        # 管理页面
│   │       ├── Dashboard.vue
│   │       ├── Channels.vue
│   │       ├── Users.vue
│   │       ├── Tokens.vue
│   │       ├── Redemptions.vue
│   │       └── Logs.vue
│   ├── App.vue
│   └── main.js
├── index.html
├── package.json
├── vite.config.js
└── README.md
```

## API 配置

前端通过 Vite 代理连接到后端 API：

- 开发环境：http://localhost:8080
- 生产环境：同域名 /api 路径

## 环境变量

创建 `.env` 文件配置环境变量：

```env
VITE_API_BASE_URL=/api
```

## Docker 部署

使用 Docker Compose 一键部署：

```bash
docker-compose up -d
```

## 开发规范

- 使用 ESLint 进行代码检查
- 组件采用 Composition API (setup 语法)
- 使用 SCSS 进行样式开发
- 遵循 Vue 3 最佳实践

## License

Apache 2.0
