<template>
  <div class="user-layout">
    <el-container>
      <el-header class="layout-header">
        <div class="header-content">
          <div class="logo" @click="$router.push('/')">
            <el-icon><Monitor /></el-icon>
            <span>AI Model Scheduler</span>
          </div>
          
          <el-menu 
            mode="horizontal" 
            :ellipsis="false" 
            router 
            :default-active="$route.path"
            class="main-menu"
          >
            <el-menu-item index="/">
              <el-icon><House /></el-icon>
              <span>控制台</span>
            </el-menu-item>
            <el-menu-item index="/tokens">
              <el-icon><Key /></el-icon>
              <span>Token 管理</span>
            </el-menu-item>
            <el-menu-item index="/balance">
              <el-icon><Wallet /></el-icon>
              <span>余额充值</span>
            </el-menu-item>
            <el-menu-item index="/playground">
              <el-icon><ChatDotRound /></el-icon>
              <span>AI 对话</span>
            </el-menu-item>
          </el-menu>
          
          <div class="user-actions">
            <el-dropdown trigger="click">
              <div class="user-info">
                <el-avatar :size="36" :src="userStore.userInfo?.avatar" :icon="UserFilled">
                  {{ userStore.userInfo?.name?.charAt(0) || 'U' }}
                </el-avatar>
                <div class="user-details">
                  <div class="username">{{ userStore.userInfo?.name || '用户' }}</div>
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
        </div>
      </el-header>
      
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script setup>
import { useUserStore } from '@/store/user';
import { ArrowDown, ChatDotRound, House, Key, SwitchButton, User, UserFilled, Wallet } from '@element-plus/icons-vue';
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';

const userStore = useUserStore()
const router = useRouter()

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
})
</script>

<style scoped lang="scss">
.user-layout {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.layout-header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  padding: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  position: sticky;
  top: 0;
  z-index: 100;
  
  .header-content {
    display: flex;
    align-items: center;
    height: 64px;
    padding: 0 24px;
    max-width: 1440px;
    margin: 0 auto;
    width: 100%;
    
    .logo {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 20px;
      font-weight: 700;
      color: #409EFF;
      cursor: pointer;
      margin-right: 48px;
      transition: all 0.3s ease;
      
      &:hover {
        color: #66b1ff;
      }
      
      .el-icon {
        font-size: 28px;
      }
    }
  }
}

.main-menu {
  flex: 1;
  border-bottom: none;
  
  .el-menu-item {
    height: 64px;
    line-height: 64px;
    margin: 0 8px;
    border-radius: 8px;
    transition: all 0.3s ease;
    font-size: 14px;
    
    &:hover {
      background-color: #ecf5ff !important;
      color: #409EFF !important;
    }
    
    &.is-active {
      background-color: #ecf5ff !important;
      color: #409EFF !important;
      font-weight: 600;
    }
    
    .el-icon {
      margin-right: 8px;
      font-size: 16px;
    }
  }
}

.layout-main {
  padding: 24px;
  max-width: 1440px;
  margin: 0 auto;
  width: 100%;
  min-height: calc(100vh - 64px);
}

.user-actions {
  margin-left: auto;
  
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

@media (max-width: 768px) {
  .layout-header {
    .header-content {
      padding: 0 16px;
      
      .logo {
        font-size: 18px;
        margin-right: 24px;
        
        .el-icon {
          font-size: 24px;
        }
      }
    }
  }
  
  .main-menu {
    .el-menu-item {
      font-size: 12px;
      
      .el-icon {
        font-size: 14px;
      }
    }
  }
  
  .user-actions {
    .user-info {
      .user-details {
        display: none;
      }
    }
  }
  
  .layout-main {
    padding: 16px;
  }
}
</style>
