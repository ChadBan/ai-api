<template>
  <div class="config-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>系统配置</span>
          <el-button type="primary" @click="saveConfig">
            <el-icon><Check /></el-icon>
            保存配置
          </el-button>
        </div>
      </template>
      
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="基本配置" name="basic">
          <el-form :model="config" label-width="120px">
            <el-form-item label="系统名称">
              <el-input v-model="config.system_name" placeholder="请输入系统名称" />
            </el-form-item>
            <el-form-item label="系统描述">
              <el-input v-model="config.system_description" type="textarea" placeholder="请输入系统描述" />
            </el-form-item>
            <el-form-item label="管理员邮箱">
              <el-input v-model="config.admin_email" placeholder="请输入管理员邮箱" />
            </el-form-item>
            <el-form-item label="系统公告">
              <el-input v-model="config.system_announcement" type="textarea" placeholder="请输入系统公告" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
        
        <el-tab-pane label="API配置" name="api">
          <el-form :model="config" label-width="120px">
            <el-form-item label="API请求超时(秒)">
              <el-input-number v-model="config.api_timeout" :min="1" :max="300" />
            </el-form-item>
            <el-form-item label="API最大并发数">
              <el-input-number v-model="config.api_max_concurrency" :min="1" :max="100" />
            </el-form-item>
            <el-form-item label="API速率限制(次/分钟)">
              <el-input-number v-model="config.api_rate_limit" :min="1" :max="1000" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
        
        <el-tab-pane label="计费配置" name="billing">
          <el-form :model="config" label-width="120px">
            <el-form-item label="默认token价格(分/1K)">
              <el-input-number v-model="config.default_token_price" :min="0.01" :step="0.01" />
            </el-form-item>
            <el-form-item label="充值倍率">
              <el-input-number v-model="config.recharge_multiplier" :min="0.1" :step="0.1" />
            </el-form-item>
            <el-form-item label="赠送比例(%)">
              <el-input-number v-model="config.bonus_percentage" :min="0" :max="100" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
        
        <el-tab-pane label="安全配置" name="security">
          <el-form :model="config" label-width="120px">
            <el-form-item label="JWT密钥">
              <el-input v-model="config.jwt_secret" type="password" show-password />
            </el-form-item>
            <el-form-item label="密码最小长度">
              <el-input-number v-model="config.password_min_length" :min="6" :max="20" />
            </el-form-item>
            <el-form-item label="登录失败限制">
              <el-input-number v-model="config.login_fail_limit" :min="0" :max="10" />
            </el-form-item>
            <el-form-item label="登录失败锁定时间(分钟)">
              <el-input-number v-model="config.login_fail_lock_time" :min="0" :max="1440" />
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import api from '@/api'
import { Check } from '@element-plus/icons-vue'
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const activeTab = ref('basic')
const config = ref({
  system_name: 'AI Model Scheduler',
  system_description: 'AI模型调度系统',
  admin_email: '',
  system_announcement: '',
  api_timeout: 30,
  api_max_concurrency: 10,
  api_rate_limit: 60,
  default_token_price: 0.1,
  recharge_multiplier: 1,
  bonus_percentage: 0,
  jwt_secret: '',
  password_min_length: 8,
  login_fail_limit: 5,
  login_fail_lock_time: 15
})

const loadConfig = async () => {
  try {
    const response = await api.admin.getSystemConfig()
    // 新响应格式: { code, message, data: {...} }
    if (response.data?.data) {
      Object.assign(config.value, response.data.data)
    } else if (response.data) {
      Object.assign(config.value, response.data)
    }
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
  }
}

const saveConfig = async () => {
  try {
    await api.admin.updateSystemConfig(config.value)
    ElMessage.success('配置保存成功')
  } catch (error) {
    console.error('保存配置失败:', error)
    ElMessage.error('保存配置失败')
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped lang="scss">
.config-page {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .el-tabs {
    margin-top: 20px;
  }
  
  .el-form {
    padding: 20px 0;
  }
  
  .el-form-item {
    margin-bottom: 20px;
  }
}
</style>