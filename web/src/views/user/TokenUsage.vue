<template>
  <div class="token-usage-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Token 使用记录</span>
          <el-button @click="goBack">
            <el-icon><ArrowLeft /></el-icon>
            返回
          </el-button>
        </div>
      </template>

      <!-- 筛选条件 -->
      <el-form :model="filters" inline>
        <el-form-item label="Token">
          <el-select v-model="filters.token_id" placeholder="选择 Token" clearable style="width: 200px">
            <el-option
              v-for="token in tokens"
              :key="token.id"
              :label="`${token.name} (${token.key.substring(0, 12)}...)`"
              :value="token.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="模型">
          <el-input v-model="filters.model" placeholder="输入模型名称" clearable style="width: 150px" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.success" placeholder="全部" clearable style="width: 100px">
            <el-option label="成功" :value="true" />
            <el-option label="失败" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="filters.date_range"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>

      <!-- 统计卡片 -->
      <el-row :gutter="16" style="margin-top: 20px">
        <el-col :span="6">
          <el-statistic title="总请求数" :value="stats.total_requests" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="成功请求" :value="stats.success_requests">
            <template #suffix>
              <span v-if="stats.total_requests > 0">
                ({{ ((stats.success_requests / stats.total_requests) * 100).toFixed(1) }}%)
              </span>
            </template>
          </el-statistic>
        </el-col>
        <el-col :span="6">
          <el-statistic title="总 Token 消耗" :value="stats.total_tokens" />
        </el-col>
        <el-col :span="6">
          <el-statistic title="总配额扣除" :value="stats.total_quota" />
        </el-col>
      </el-row>

      <!-- 数据表格 -->
      <el-table :data="logs" v-loading="loading" style="margin-top: 20px">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="token_key" label="Token" width="200" show-overflow-tooltip>
          <template #default="{ row }">
            {{ maskToken(row.token_key) }}
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" width="150" />
        <el-table-column prop="input_tokens" label="输入 Tokens" width="100" />
        <el-table-column prop="output_tokens" label="输出 Tokens" width="100" />
        <el-table-column label="总 Tokens" width="100">
          <template #default="{ row }">
            {{ row.input_tokens + row.output_tokens }}
          </template>
        </el-table-column>
        <el-table-column prop="quota_deducted" label="扣除配额" width="100" />
        <el-table-column prop="duration_ms" label="耗时" width="80">
          <template #default="{ row }">
            {{ row.duration_ms }}ms
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'">
              {{ row.success ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_message" label="错误信息" show-overflow-tooltip />
        <el-table-column prop="request_time" label="请求时间" width="160">
          <template #default="{ row }">
            {{ formatDate(row.request_time) }}
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.page_size"
        :total="pagination.total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>
  </div>
</template>

<script setup>
import api from '@/api'
import { ElMessage } from 'element-plus'
import { onMounted, reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const loading = ref(false)
const logs = ref([])
const tokens = ref([])

const filters = reactive({
  token_id: null,
  model: '',
  success: null,
  date_range: []
})

const stats = reactive({
  total_requests: 0,
  success_requests: 0,
  total_tokens: 0,
  total_quota: 0
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const loadTokens = async () => {
  try {
    const res = await api.getTokens({ page: 1, page_size: 100 })
    // 新响应格式: { code, message, data: { items: [...], total: N } }
    tokens.value = res.data.data?.items || []
    
    // 如果 URL 中有 token_id 参数，自动选中
    if (route.query.token_id) {
      filters.token_id = parseInt(route.query.token_id)
    }
  } catch (error) {
    console.error('Failed to load tokens:', error)
  }
}

const loadLogs = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.page_size
    }
    
    if (filters.token_id) {
      params.token_id = filters.token_id
    }
    if (filters.model) {
      params.model = filters.model
    }
    if (filters.success !== null && filters.success !== undefined) {
      params.success = filters.success
    }
    if (filters.date_range && filters.date_range.length === 2) {
      params.start_time = filters.date_range[0]
      params.end_time = filters.date_range[1]
    }
    
    const res = await api.getTokenUsageLogs(params)
    // 新响应格式: { code, message, data: { items: [...], total: N } }
    logs.value = res.data.data?.items || []
    pagination.total = res.data.data?.total || 0
    
    // 更新统计
    updateStats()
  } catch (error) {
    console.error('Failed to load usage logs:', error)
    ElMessage.error('加载使用记录失败')
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  // 从当前页数据计算统计（简化版，实际应该从后端获取总体统计）
  stats.total_requests = pagination.total
  stats.success_requests = logs.value.filter(log => log.success).length
  stats.total_tokens = logs.value.reduce((sum, log) => sum + log.input_tokens + log.output_tokens, 0)
  stats.total_quota = logs.value.reduce((sum, log) => sum + log.quota_deducted, 0)
}

const handleSearch = () => {
  pagination.page = 1
  loadLogs()
}

const handleReset = () => {
  filters.token_id = null
  filters.model = ''
  filters.success = null
  filters.date_range = []
  pagination.page = 1
  loadLogs()
}

const handleSizeChange = () => {
  loadLogs()
}

const handlePageChange = (page) => {
  pagination.page = page
  loadLogs()
}

const goBack = () => {
  router.back()
}

const maskToken = (token) => {
  if (!token) return ''
  if (token.length <= 16) return '***'
  return `${token.substring(0, 8)}...${token.substring(token.length - 4)}`
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  loadTokens()
  loadLogs()
})
</script>

<style scoped lang="scss">
.token-usage-page {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  :deep(.el-statistic__head) {
    font-size: 14px;
    color: #909399;
  }
  
  :deep(.el-statistic__content) {
    font-size: 24px;
    font-weight: bold;
  }
}
</style>
