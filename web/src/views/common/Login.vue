<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <el-icon :size="48" color="#409EFF"><Monitor /></el-icon>
        <h1>AI Model Scheduler</h1>
        <p>AI 模型调度管理系统</p>
      </div>
      
      <el-form
        ref="formRef"
        :model="loginForm"
        :rules="rules"
        class="login-form"
        @keyup.enter="handleLogin"
      >
        <el-form-item prop="email">
          <el-input
            v-model="loginForm.email"
            placeholder="请输入邮箱"
            prefix-icon="Message"
            size="large"
          />
        </el-form-item>
        
        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="Lock"
            size="large"
            show-password
          />
        </el-form-item>
        
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="login-btn"
            @click="handleLogin"
          >
            登录
          </el-button>
        </el-form-item>
        
        <div class="login-footer">
          <span>还没有账号？</span>
          <router-link to="/register">立即注册</router-link>
        </div>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import api from '@/api'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref(null)
const loading = ref(false)

const loginForm = reactive({
  email: '',
  password: ''
})

const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少为 6 位', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    loading.value = true
    
    try {
      const res = await api.login(loginForm.email, loginForm.password)
      
      console.log('Login response:', res)
      console.log('Login response.data:', res.data)
      
      // 新响应格式: { code: 1000, message: "Success", data: { token: "..." } }
      if (res.data && res.data.data && res.data.data.token) {
        const newToken = res.data.data.token
        console.log('Setting token:', newToken.substring(0, 20) + '...')
        userStore.setToken(newToken)
        
        // 重新获取用户信息
        console.log('Fetching user info...')
        await userStore.fetchUserInfo()
        console.log('User info fetched:', userStore.userInfo)
        
        ElMessage.success('登录成功')
        
        // 根据角色跳转
        if (userStore.isAdmin) {
          router.push('/admin')
        } else {
          router.push('/')
        }
      } else {
        console.error('Invalid login response format:', res.data)
        ElMessage.error('登录响应格式错误')
      }
    } catch (error) {
      console.error('Login failed:', error)
      ElMessage.error(error.message || '登录失败，请检查邮箱和密码')
    } finally {
      loading.value = false
    }
  })
}
</script>

<style scoped lang="scss">
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-box {
  width: 420px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  
  .login-header {
    text-align: center;
    margin-bottom: 30px;
    
    h1 {
      margin: 16px 0 8px;
      font-size: 24px;
      color: #303133;
    }
    
    p {
      color: #909399;
      font-size: 14px;
    }
  }
}

.login-form {
  .login-btn {
    width: 100%;
  }
}

.login-footer {
  text-align: center;
  margin-top: 16px;
  color: #909399;
  
  a {
    color: #409EFF;
    text-decoration: none;
    margin-left: 8px;
    
    &:hover {
      text-decoration: underline;
    }
  }
}
</style>
