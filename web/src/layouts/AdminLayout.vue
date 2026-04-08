<template>
  <div class="admin-layout">
    <el-container>
      <el-aside width="240px" class="admin-sidebar">
        <div class="logo">
          <el-icon><Setting /></el-icon>
          <span>管理后台</span>
        </div>
        
        <el-menu
          :default-active="activeMenu"
          router
          background-color="#304156"
          text-color="#bfcbd9"
          active-text-color="#409EFF"
          class="admin-menu"
        >
          <el-menu-item index="/admin">
            <el-icon><DataAnalysis /></el-icon>
            <span>数据看板</span>
          </el-menu-item>
          
          <el-menu-item index="/admin/channels">
            <el-icon><Connection /></el-icon>
            <span>渠道管理</span>
          </el-menu-item>
          
          <el-menu-item index="/admin/users">
            <el-icon><User /></el-icon>
            <span>用户管理</span>
          </el-menu-item>
          
          <el-menu-item index="/admin/tokens">
            <el-icon><Key /></el-icon>
            <span>Token 管理</span>
          </el-menu-item>
          
          <el-menu-item index="/admin/redemptions">
            <el-icon><Ticket /></el-icon>
            <span>兑换码管理</span>
          </el-menu-item>
          
          <el-menu-item index="/admin/logs">
              <el-icon><Reading /></el-icon>
              <span>日志管理</span>
            </el-menu-item>
            <el-menu-item index="/admin/config">
              <el-icon><Setting /></el-icon>
              <span>系统配置</span>
            </el-menu-item>
            
            <el-menu-item index="/admin/groups">
              <el-icon><Collection /></el-icon>
              <span>分组管理</span>
            </el-menu-item>
        </el-menu>
      </el-aside>
      
      <el-container>
        <el-header class="admin-header">
          <div class="header-left">
            <el-breadcrumb separator="/" class="breadcrumb">
              <el-breadcrumb-item :to="{ path: '/admin' }">
                <el-icon><House /></el-icon>
                <span>首页</span>
              </el-breadcrumb-item>
              <el-breadcrumb-item>{{ currentRouteName }}</el-breadcrumb-item>
            </el-breadcrumb>
          </div>
          
          <div class="header-right">
            <el-dropdown trigger="click">
              <div class="user-info">
                <el-avatar :size="36" :src="userStore.userInfo?.avatar" :icon="UserFilled">
                  {{ userStore.userInfo?.name?.charAt(0) || 'A' }}
                </el-avatar>
                <div class="user-details">
                  <div class="username">{{ userStore.userInfo?.name || '管理员' }}</div>
                  <div class="user-email">{{ userStore.userInfo?.email || '' }}</div>
                </div>
                <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleProfile">
                    <el-icon><User /></el-icon>
                    <span>个人资料</span>
                  </el-dropdown-item>
                  <el-dropdown-item @click="handleLogout">
                    <el-icon><SwitchButton /></el-icon>
                    <span>退出登录</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </el-header>
        
        <el-main class="admin-main">
          <router-view />
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<script setup>
import { useUserStore } from '@/store/user'
import { ArrowDown, Collection, Connection, DataAnalysis, House, Key, Setting, SwitchButton, Ticket, User, UserFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const userStore = useUserStore()
const router = useRouter()
const route = useRoute()

const activeMenu = computed(() => route.path)

const currentRouteName = computed(() => {
  const nameMap = {
    '/admin': '数据看板',
    '/admin/channels': '渠道管理',
    '/admin/users': '用户管理',
    '/admin/tokens': 'Token 管理',
    '/admin/redemptions': '兑换码管理',
    '/admin/logs': '日志管理',
    '/admin/config': '系统配置',
    '/admin/groups': '分组管理'
  }
  return nameMap[route.path] || '未知'
})

const handleLogout = () => {
  userStore.clearToken()
  router.push('/login')
}

const handleProfile = () => {
  // 未来实现个人资料页面
  console.log('个人资料')
}

onMounted(async () => {
  // 只有在没有用户信息时才获取
  if (!userStore.userInfo) {
    await userStore.fetchUserInfo()
  }
  
  console.log('User Info:', userStore.userInfo)
  console.log('Is Admin:', userStore.isAdmin)
  
  if (!userStore.isAdmin) {
    ElMessage.error('没有管理员权限')
    router.push('/')
  }
})
</script>

<style scoped lang="scss">
.admin-layout {
  height: 100vh;
  
  .el-container {
    height: 100%;
  }
}

.admin-sidebar {
  background-color: #304156;
  color: #fff;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  
  .logo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    height: 64px;
    font-size: 18px;
    font-weight: 700;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    transition: all 0.3s ease;
    
    .el-icon {
      font-size: 28px;
    }
  }
}

.admin-menu {
  border-right: none;
  height: calc(100vh - 64px);
  overflow-y: auto;
  
  .el-menu-item {
    height: 56px;
    line-height: 56px;
    margin: 0 12px;
    border-radius: 8px;
    transition: all 0.3s ease;
    font-size: 14px;
    
    &:hover {
      background-color: rgba(255, 255, 255, 0.1) !important;
    }
    
    &.is-active {
      background-color: rgba(64, 158, 255, 0.2) !important;
      font-weight: 600;
    }
    
    .el-icon {
      margin-right: 12px;
      font-size: 16px;
    }
  }
}

.admin-header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 64px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  
  .breadcrumb {
    .el-breadcrumb__item {
      display: flex;
      align-items: center;
      gap: 4px;
      
      .el-icon {
        font-size: 14px;
      }
    }
  }
  
  .header-right {
    .user-info {
      display: flex;
      align-items: center;
      gap: 12px;
      cursor: pointer;
      padding: 8px 12px;
      border-radius: 8px;
      transition: all 0.3s ease;
      
      &:hover {
        background-color: #f5f7fa;
      }
      
      .user-details {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        
        .username {
          font-size: 14px;
          font-weight: 600;
          color: #303133;
        }
        
        .user-email {
          font-size: 12px;
          color: #909399;
          margin-top: 2px;
        }
      }
      
      .dropdown-icon {
        font-size: 12px;
        color: #909399;
        transition: transform 0.3s ease;
      }
      
      &:hover .dropdown-icon {
        transform: rotate(180deg);
      }
    }
  }
}

.admin-main {
  background: #f5f7fa;
  padding: 24px;
  overflow-y: auto;
  min-height: calc(100vh - 64px);
}

@media (max-width: 768px) {
  .admin-sidebar {
    width: 200px !important;
    
    .logo {
      font-size: 16px;
      
      .el-icon {
        font-size: 24px;
      }
    }
  }
  
  .admin-menu {
    .el-menu-item {
      font-size: 12px;
      
      .el-icon {
        font-size: 14px;
      }
    }
  }
  
  .admin-header {
    padding: 0 16px;
    
    .header-right {
      .user-info {
        .user-details {
          display: none;
        }
      }
    }
  }
  
  .admin-main {
    padding: 16px;
  }
}
</style>
