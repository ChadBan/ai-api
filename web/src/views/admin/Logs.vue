<template>
  <div class="logs-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>日志管理</span>
          <el-select v-model="activeTab" size="small">
            <el-option label="审计日志" value="audit" />
            <el-option label="请求日志" value="request" />
            <el-option label="错误日志" value="error" />
            <el-option label="登录日志" value="login" />
          </el-select>
        </div>
      </template>
      
      <div class="logs-content">
        <!-- 审计日志 -->
        <div v-if="activeTab === 'audit'">
          <el-row :gutter="20" class="mb-4">
            <el-col :span="6">
              <el-input v-model="auditSearch.action" placeholder="操作" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="auditSearch.resource" placeholder="资源" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="auditSearch.user_id" placeholder="用户ID" clearable />
            </el-col>
            <el-col :span="6">
              <el-button type="primary" @click="loadAuditLogs">查询</el-button>
            </el-col>
          </el-row>
          
          <el-table :data="auditLogs" style="width: 100%">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_id" label="用户ID" width="100" />
            <el-table-column prop="action" label="操作" />
            <el-table-column prop="resource" label="资源" />
            <el-table-column prop="ip_address" label="IP地址" width="150" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
          
          <el-pagination
            v-model:current-page="auditPage.current"
            v-model:page-size="auditPage.size"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="auditTotal"
            @size-change="handleAuditSizeChange"
            @current-change="handleAuditCurrentChange"
            class="mt-4"
          />
        </div>
        
        <!-- 请求日志 -->
        <div v-if="activeTab === 'request'">
          <el-row :gutter="20" class="mb-4">
            <el-col :span="6">
              <el-input v-model="requestSearch.path" placeholder="路径" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="requestSearch.model_name" placeholder="模型" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="requestSearch.status_code" placeholder="状态码" clearable />
            </el-col>
            <el-col :span="6">
              <el-button type="primary" @click="loadRequestLogs">查询</el-button>
            </el-col>
          </el-row>
          
          <el-table :data="requestLogs" style="width: 100%">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_id" label="用户ID" width="100" />
            <el-table-column prop="path" label="路径" />
            <el-table-column prop="model_name" label="模型" />
            <el-table-column prop="status_code" label="状态码" width="100" />
            <el-table-column prop="duration" label="响应时间(ms)" width="120" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
          
          <el-pagination
            v-model:current-page="requestPage.current"
            v-model:page-size="requestPage.size"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="requestTotal"
            @size-change="handleRequestSizeChange"
            @current-change="handleRequestCurrentChange"
            class="mt-4"
          />
        </div>
        
        <!-- 错误日志 -->
        <div v-if="activeTab === 'error'">
          <el-row :gutter="20" class="mb-4">
            <el-col :span="6">
              <el-input v-model="errorSearch.level" placeholder="级别" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="errorSearch.request_id" placeholder="请求ID" clearable />
            </el-col>
            <el-col :span="6">
              <el-button type="primary" @click="loadErrorLogs">查询</el-button>
            </el-col>
          </el-row>
          
          <el-table :data="errorLogs" style="width: 100%">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="level" label="级别" width="100" />
            <el-table-column prop="request_id" label="请求ID" width="150" />
            <el-table-column prop="message" label="消息" />
            <el-table-column prop="stack" label="堆栈" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
          
          <el-pagination
            v-model:current-page="errorPage.current"
            v-model:page-size="errorPage.size"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="errorTotal"
            @size-change="handleErrorSizeChange"
            @current-change="handleErrorCurrentChange"
            class="mt-4"
          />
        </div>
        
        <!-- 登录日志 -->
        <div v-if="activeTab === 'login'">
          <el-row :gutter="20" class="mb-4">
            <el-col :span="6">
              <el-input v-model="loginSearch.username" placeholder="用户名" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="loginSearch.status" placeholder="状态" clearable />
            </el-col>
            <el-col :span="6">
              <el-input v-model="loginSearch.ip_address" placeholder="IP地址" clearable />
            </el-col>
            <el-col :span="6">
              <el-button type="primary" @click="loadLoginLogs">查询</el-button>
            </el-col>
          </el-row>
          
          <el-table :data="loginLogs" style="width: 100%">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="username" label="用户名" />
            <el-table-column prop="status" label="状态" width="100" />
            <el-table-column prop="ip_address" label="IP地址" width="150" />
            <el-table-column prop="user_agent" label="用户代理" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
          
          <el-pagination
            v-model:current-page="loginPage.current"
            v-model:page-size="loginPage.size"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="loginTotal"
            @size-change="handleLoginSizeChange"
            @current-change="handleLoginCurrentChange"
            class="mt-4"
          />
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import api from '@/api';
import { onMounted, ref } from 'vue';

const activeTab = ref('audit')

// 审计日志
const auditLogs = ref([])
const auditTotal = ref(0)
const auditPage = ref({ current: 1, size: 20 })
const auditSearch = ref({ action: '', resource: '', user_id: '' })

// 请求日志
const requestLogs = ref([])
const requestTotal = ref(0)
const requestPage = ref({ current: 1, size: 20 })
const requestSearch = ref({ path: '', model_name: '', status_code: '' })

// 错误日志
const errorLogs = ref([])
const errorTotal = ref(0)
const errorPage = ref({ current: 1, size: 20 })
const errorSearch = ref({ level: '', request_id: '' })

// 登录日志
const loginLogs = ref([])
const loginTotal = ref(0)
const loginPage = ref({ current: 1, size: 20 })
const loginSearch = ref({ username: '', status: '', ip_address: '' })

// 加载审计日志
const loadAuditLogs = async () => {
  try {
    const response = await api.admin.getAuditLogs({
      page: auditPage.value.current,
      page_size: auditPage.value.size,
      action: auditSearch.value.action,
      resource: auditSearch.value.resource,
      user_id: auditSearch.value.user_id
    })
    // 新响应格式: { code, message, data: { items: [...], total: N } }
    auditLogs.value = response.data.data?.items || []
    auditTotal.value = response.data.data?.total || 0
  } catch (error) {
    console.error('加载审计日志失败:', error)
  }
}

// 加载请求日志
const loadRequestLogs = async () => {
  try {
    const response = await api.admin.getRequestLogs({
      page: requestPage.value.current,
      page_size: requestPage.value.size,
      path: requestSearch.value.path,
      model_name: requestSearch.value.model_name,
      status_code: requestSearch.value.status_code
    })
    requestLogs.value = response.data.data?.items || []
    requestTotal.value = response.data.data?.total || 0
  } catch (error) {
    console.error('加载请求日志失败:', error)
  }
}

// 加载错误日志
const loadErrorLogs = async () => {
  try {
    const response = await api.admin.getErrorLogs({
      page: errorPage.value.current,
      page_size: errorPage.value.size,
      level: errorSearch.value.level,
      request_id: errorSearch.value.request_id
    })
    errorLogs.value = response.data.data?.items || []
    errorTotal.value = response.data.data?.total || 0
  } catch (error) {
    console.error('加载错误日志失败:', error)
  }
}

// 加载登录日志
const loadLoginLogs = async () => {
  try {
    const response = await api.admin.getLoginLogs({
      page: loginPage.value.current,
      page_size: loginPage.value.size,
      username: loginSearch.value.username,
      status: loginSearch.value.status,
      ip_address: loginSearch.value.ip_address
    })
    loginLogs.value = response.data.data?.items || []
    loginTotal.value = response.data.data?.total || 0
  } catch (error) {
    console.error('加载登录日志失败:', error)
  }
}

// 分页处理
const handleAuditSizeChange = (size) => {
  auditPage.value.size = size
  loadAuditLogs()
}

const handleAuditCurrentChange = (page) => {
  auditPage.value.current = page
  loadAuditLogs()
}

const handleRequestSizeChange = (size) => {
  requestPage.value.size = size
  loadRequestLogs()
}

const handleRequestCurrentChange = (page) => {
  requestPage.value.current = page
  loadRequestLogs()
}

const handleErrorSizeChange = (size) => {
  errorPage.value.size = size
  loadErrorLogs()
}

const handleErrorCurrentChange = (page) => {
  errorPage.value.current = page
  loadErrorLogs()
}

const handleLoginSizeChange = (size) => {
  loginPage.value.size = size
  loadLoginLogs()
}

const handleLoginCurrentChange = (page) => {
  loginPage.value.current = page
  loadLoginLogs()
}

// 初始加载
onMounted(() => {
  loadAuditLogs()
})
</script>

<style scoped lang="scss">
.logs-page {
  padding: 20px;
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .logs-content {
    margin-top: 20px;
  }
  
  .mb-4 {
    margin-bottom: 16px;
  }
}
</style>
