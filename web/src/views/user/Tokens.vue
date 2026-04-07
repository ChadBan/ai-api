<template>
  <div class="tokens-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Token 管理</span>
          <el-button type="primary" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon>
            创建 Token
          </el-button>
        </div>
      </template>
      
      <el-table :data="tokens" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="key" label="Token" show-overflow-tooltip />
        <el-table-column prop="remain_quota" label="剩余配额" width="100" />
        <el-table-column prop="ratio" label="汇率倍率" width="100">
          <template #default="{ row }">
            {{ row.ratio?.toFixed(2) || '1.00' }}
          </template>
        </el-table-column>
        <el-table-column prop="model_limit" label="模型限制" width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ formatModelLimit(row.model_limit) }}
          </template>
        </el-table-column>
        <el-table-column prop="expired_time" label="过期时间" width="160">
          <template #default="{ row }">
            {{ row.expired_time ? formatDate(row.expired_time) : '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="toggleStatus(row)">
              {{ row.status === 1 ? '禁用' : '启用' }}
            </el-button>
            <el-button size="small" type="primary" @click="showUsageLogs(row)">
              使用记录
            </el-button>
            <el-button size="small" type="danger" @click="deleteToken(row.id)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 创建对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建 Token" width="600px">
      <el-form :model="createForm" label-width="120px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="配额" required>
          <el-input-number v-model="createForm.remain_quota" :min="0" :precision="0" />
        </el-form-item>
        <el-form-item label="无限配额">
          <el-switch v-model="createForm.unlimited_quota" />
        </el-form-item>
        <el-form-item label="汇率倍率">
          <el-input-number v-model="createForm.ratio" :min="0.1" :max="10" :precision="2" :step="0.1" />
          <div class="form-tip">默认 1.0，可根据需求调整</div>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="createForm.expired_time"
            type="datetime"
            placeholder="选择过期时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 100%"
            clearable
          />
          <div class="form-tip">不设置表示永不过期</div>
        </el-form-item>
        <el-form-item label="允许的模型">
          <el-select
            v-model="createForm.model_limit"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入或选择允许的模型"
            style="width: 100%"
          >
            <el-option label="GPT-3.5" value="gpt-3.5-turbo" />
            <el-option label="GPT-4" value="gpt-4" />
            <el-option label="GPT-4 Turbo" value="gpt-4-turbo" />
            <el-option label="Claude 3" value="claude-3" />
            <el-option label="通配符 GPT" value="gpt-*" />
            <el-option label="通配符 Claude" value="claude-*" />
          </el-select>
          <div class="form-tip">支持通配符（如 gpt-*），不填表示允许所有模型</div>
        </el-form-item>
        <el-form-item label="用户组">
          <el-input v-model="createForm.group" placeholder="default" />
          <div class="form-tip">用于分组管理</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="creating">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import api from '@/api';
import { ElMessage } from 'element-plus';
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';

const router = useRouter()

const loading = ref(false)
const tokens = ref([])
const showCreateDialog = ref(false)

const createForm = reactive({
  name: '',
  remain_quota: 1000,
  unlimited_quota: false,
  ratio: 1.0,
  expired_time: null,
  model_limit: [],
  group: 'default'
})

const creating = ref(false)

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await api.getTokens({ page: 1, page_size: 100 })
    // 新响应格式: { code, message, data: { items: [...], total: N } }
    tokens.value = res.data.data?.items || []
  } catch (error) {
    console.error('Failed to load tokens:', error)
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!createForm.name) {
    ElMessage.warning('请输入名称')
    return
  }
  
  creating.value = true
  try {
    await api.createToken(createForm)
    ElMessage.success('创建成功')
    showCreateDialog.value = false
    loadTokens()
  } catch (error) {
    console.error('Failed to create token:', error)
  } finally {
    creating.value = false
  }
}

const toggleStatus = async (token) => {
  try {
    await api.toggleTokenStatus(token.id)
    ElMessage.success('操作成功')
    loadTokens()
  } catch (error) {
    console.error('Failed to toggle status:', error)
  }
}

const deleteToken = async (id) => {
  if (!confirm('确定要删除此 Token 吗？')) return
  
  try {
    await api.deleteToken(id)
    ElMessage.success('删除成功')
    loadTokens()
  } catch (error) {
    console.error('Failed to delete token:', error)
  }
}

const showUsageLogs = (token) => {
  // TODO: 跳转到使用记录页面或打开对话框
  ElMessage.info('使用记录功能开发中...')
}

const formatModelLimit = (modelLimit) => {
  if (!modelLimit || modelLimit === '[]' || modelLimit === 'null') {
    return '所有模型'
  }
  try {
    const models = JSON.parse(modelLimit)
    if (Array.isArray(models)) {
      return models.join(', ')
    }
  } catch (e) {
    return modelLimit
  }
  return modelLimit
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
})
</script>

<style scoped lang="scss">
.tokens-page {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  :deep(.form-tip) {
    font-size: 12px;
    color: #999;
    margin-top: 4px;
    line-height: 1.5;
  }
}
</style>
