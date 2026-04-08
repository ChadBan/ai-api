import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/common/Login.vue')
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/common/Register.vue')
  },
  {
    path: '/',
    component: () => import('@/layouts/UserLayout.vue'),
    children: [
      {
        path: '',
        name: 'UserDashboard',
        component: () => import('@/views/user/Dashboard.vue')
      },
      {
        path: 'tokens',
        name: 'UserTokens',
        component: () => import('@/views/user/Tokens.vue')
      },
      {
        path: 'token-usage',
        name: 'TokenUsage',
        component: () => import('@/views/user/TokenUsage.vue')
      },
      {
        path: 'balance',
        name: 'UserBalance',
        component: () => import('@/views/user/Balance.vue')
      },
      {
        path: 'playground',
        name: 'Playground',
        component: () => import('@/views/user/Playground.vue')
      }
    ]
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    children: [
      {
        path: '',
        name: 'AdminDashboard',
        component: () => import('@/views/admin/Dashboard.vue')
      },
      {
        path: 'channels',
        name: 'AdminChannels',
        component: () => import('@/views/admin/Channels.vue')
      },
      {
        path: 'users',
        name: 'AdminUsers',
        component: () => import('@/views/admin/Users.vue')
      },
      {
        path: 'tokens',
        name: 'AdminTokens',
        component: () => import('@/views/admin/Tokens.vue')
      },
      {
        path: 'redemptions',
        name: 'AdminRedemptions',
        component: () => import('@/views/admin/Redemptions.vue')
      },
      {
        path: 'logs',
        name: 'AdminLogs',
        component: () => import('@/views/admin/Logs.vue')
      },
      {
        path: 'config',
        name: 'AdminConfig',
        component: () => import('@/views/admin/Config.vue')
      },
      {
        path: 'groups',
        name: 'AdminGroups',
        component: () => import('@/views/admin/Groups.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  
  // 没有 token 且访问需要认证的页面
  if (!token && to.path !== '/login' && to.path !== '/register') {
    console.log('Router guard: No token, redirecting to login')
    next('/login')
    return
  }
  
  // 有 token 但访问登录/注册页
  if (token && (to.path === '/login' || to.path === '/register')) {
    next('/')
    return
  }
  
  // 允许访问
  next()
})

export default router
