<template>
  <div class="channels-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>渠道管理</span>
          <div class="header-actions">
            <el-button type="primary" @click="showCreateDialog = true">
              <el-icon><Plus /></el-icon>
              添加渠道
            </el-button>
            <el-button @click="batchTestChannels" :loading="batchTesting">
              <el-icon><Refresh /></el-icon>
              批量测试
            </el-button>
            <el-button @click="refreshChannels">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="channels" v-loading="loading" border style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="api_key" label="API Key" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.api_key ? '******' + row.api_key.slice(-4) : '' }}
          </template>
        </el-table-column>
        <el-table-column prop="group" label="分组" width="120">
          <template #default="{ row }">
            <el-tag size="small">
              {{ row.group || 'default' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="120">
          <template #default="{ row }">
            <el-tag :type="getChannelTypeTag(row.type)">
              {{ channelType(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="base_url" label="BaseURL" show-overflow-tooltip />
        <el-table-column prop="priority" label="优先级" width="80" />
        <el-table-column prop="weight" label="权重" width="80" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="last_test_time" label="最后测试时间" width="160">
          <template #default="{ row }">
            {{ row.last_test_time || '未测试' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="300">
          <template #default="{ row }">
            <el-button size="small" @click="testChannel(row.id)" :loading="testingChannels.includes(row.id)">
              <el-icon><Check /></el-icon>
              测试
            </el-button>
            <el-button size="small" type="warning" @click="editChannel(row)">
              <el-icon><Edit /></el-icon>
              编辑
            </el-button>
            <el-button size="small" type="danger" @click="deleteChannel(row.id)">
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 创建渠道对话框 -->
    <el-dialog v-model="showCreateDialog" title="添加渠道" width="600px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="渠道类型" required>
          <el-select v-model="createForm.type" placeholder="请选择渠道类型" @change="handleChannelTypeChange">
            <el-option label="OpenAI" :value="1" />
            <el-option label="Anthropic" :value="2" />
            <el-option label="Azure" :value="3" />
            <el-option label="Google Gemini" :value="4" />
            <el-option label="豆包 (ByteDance)" :value="14" />
            <el-option label="阿里通义" :value="15" />
            <el-option label="DeepSeek" :value="16" />
            <el-option label="MiniMax" :value="17" />
            <el-option label="智谱 AI" :value="18" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="显示名称" required>
          <el-input v-model="createForm.display_name" placeholder="请输入显示名称" />
        </el-form-item>
        <el-form-item label="BaseURL" required>
          <el-input v-model="createForm.base_url" placeholder="https://api.openai.com/v1" />
          <el-tooltip content="根据选择的渠道类型，系统已自动填充默认BaseURL" placement="top" :visible="showBaseUrlTip">
            <el-button type="text" size="small" @click="showBaseUrlTip = !showBaseUrlTip">
              显示提示
            </el-button>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="API Key" required>
          <el-input v-model="createForm.api_key" type="password" show-password />
          <el-tooltip content="{{ getApiKeyTip(createForm.type) }}" placement="top" :visible="showApiKeyTip">
            <el-button type="text" size="small" @click="showApiKeyTip = !showApiKeyTip">
              API Key提示
            </el-button>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="测试模型">
          <el-select v-model="createForm.test_model" placeholder="请选择测试模型">
            <el-option 
              v-for="model in getAvailableModels(createForm.type)" 
              :key="model" 
              :label="model" 
              :value="model" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="createForm.priority" :min="1" :step="1" />
          <el-tooltip content="优先级越高，越先被使用" placement="top">
            <el-button type="text" size="small">
              ？
            </el-button>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="权重">
          <el-input-number v-model="createForm.weight" :min="1" :step="10" />
          <el-tooltip content="权重越高，被使用的概率越大" placement="top">
            <el-button type="text" size="small">
              ？
            </el-button>
          </el-tooltip>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="createForm.status">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
            <el-option label="维护中" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型" required>
          <div class="model-management">
            <!-- 模型选择器 -->
            <el-select
              v-model="selectedModels"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="请选择该渠道所支持的模型"
              style="width: 100%"
              @change="updateFilteredModels"
            >
              <el-option
                v-for="model in filteredModels"
                :key="model"
                :label="model"
                :value="model"
              />
            </el-select>
            
            <!-- 已选择模型标签显示 -->
            <div class="selected-model-tags" v-if="selectedModels.length > 0">
              <el-tag
                v-for="model in selectedModels"
                :key="model"
                closable
                @close="removeModel(model)"
                class="selected-tag"
              >
                {{ model }}
              </el-tag>
            </div>
            
            <div class="model-actions">
              <el-button type="text" size="small" @click="selectAllModels">
                全选
              </el-button>
              <el-button type="text" size="small" @click="clearSelectedModels">
                清空
              </el-button>
              <el-button type="text" size="small" @click="fillRelatedModels(createForm.type)">
                填入相关模型
              </el-button>
              <el-button type="text" size="small" @click="syncModels(createForm.type)" :loading="syncingModels">
                <el-icon><Refresh /></el-icon>
                同步最新模型
              </el-button>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="分组">
          <el-select v-model="createForm.group" placeholder="选择分组">
            <el-option 
              v-for="group in groups" 
              :key="group.name" 
              :label="group.display_name || group.name" 
              :value="group.name" 
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetCreateForm">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
    
    <!-- 编辑渠道对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑渠道" width="600px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="渠道类型" required>
          <el-select v-model="editForm.type" @change="handleEditChannelTypeChange">
            <el-option label="OpenAI" :value="1" />
            <el-option label="Anthropic" :value="2" />
            <el-option label="Azure" :value="3" />
            <el-option label="Google Gemini" :value="4" />
            <el-option label="豆包 (ByteDance)" :value="14" />
            <el-option label="阿里通义" :value="15" />
            <el-option label="DeepSeek" :value="16" />
            <el-option label="MiniMax" :value="17" />
            <el-option label="智谱 AI" :value="18" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="editForm.name" placeholder="请输入名称" />
        </el-form-item>
        <el-form-item label="显示名称" required>
          <el-input v-model="editForm.display_name" placeholder="请输入显示名称" />
        </el-form-item>
        <el-form-item label="BaseURL" required>
          <el-input v-model="editForm.base_url" placeholder="https://api.openai.com/v1" />
        </el-form-item>
        <el-form-item label="API Key" required>
          <el-input v-model="editForm.api_key" type="password" show-password />
        </el-form-item>
        <el-form-item label="测试模型">
          <el-select v-model="editForm.test_model" placeholder="请选择测试模型">
            <el-option 
              v-for="model in getAvailableModels(editForm.type)" 
              :key="model" 
              :label="model" 
              :value="model" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="editForm.priority" :min="1" :step="1" />
        </el-form-item>
        <el-form-item label="权重">
          <el-input-number v-model="editForm.weight" :min="1" :step="10" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editForm.status">
            <el-option label="启用" :value="1" />
            <el-option label="禁用" :value="0" />
            <el-option label="维护中" :value="2" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型" required>
          <div class="model-management">
            <!-- 模型选择器 -->
            <el-select
              v-model="selectedEditModels"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="请选择该渠道所支持的模型"
              style="width: 100%"
              @change="updateFilteredEditModels"
            >
              <el-option
                v-for="model in filteredEditModels"
                :key="model"
                :label="model"
                :value="model"
              />
            </el-select>
            
            <!-- 已选择模型标签显示 -->
            <div class="selected-model-tags" v-if="selectedEditModels.length > 0">
              <el-tag
                v-for="model in selectedEditModels"
                :key="model"
                closable
                @close="removeEditModel(model)"
                class="selected-tag"
              >
                {{ model }}
              </el-tag>
            </div>
            
            <div class="model-actions">
              <el-button type="text" size="small" @click="selectAllEditModels">
                全选
              </el-button>
              <el-button type="text" size="small" @click="clearSelectedEditModels">
                清空
              </el-button>
              <el-button type="text" size="small" @click="fillEditRelatedModels(editForm.type)">
                填入相关模型
              </el-button>
              <el-button type="text" size="small" @click="syncEditModels(editForm.type)" :loading="syncingEditModels">
                <el-icon><Refresh /></el-icon>
                同步最新模型
              </el-button>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="分组">
          <el-select v-model="editForm.group" placeholder="选择分组">
            <el-option 
              v-for="group in groups" 
              :key="group.name" 
              :label="group.display_name || group.name" 
              :value="group.name" 
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import api from '@/api';
import { Check, Delete, Edit, Plus, Refresh } from '@element-plus/icons-vue';
import { ElButton, ElMessage, ElTooltip } from 'element-plus';
import { onMounted, reactive, ref } from 'vue';

const loading = ref(false)
const batchTesting = ref(false)
const testingChannels = ref([])
const channels = ref([])
const showCreateDialog = ref(false)
const showEditDialog = ref(false)
const showBaseUrlTip = ref(false)
const showApiKeyTip = ref(false)
const selectedModels = ref([])
const selectedEditModels = ref([])
const syncingModels = ref(false)
const showManualModelInput = ref(false)
const manualModel = ref('')
const syncingEditModels = ref(false)
const customModels = ref({}) // 存储手动添加的模型
const filteredModels = ref([])
const filteredEditModels = ref([])
const groups = ref([]) // 存储分组列表
const hasSyncedModels = ref({}) // 记录是否已经同步过模型

const createForm = reactive({
  type: 1,
  name: '',
  display_name: '',
  base_url: 'https://api.openai.com/v1',
  api_key: '',
  test_model: 'gpt-3.5-turbo',
  priority: 1,
  weight: 100,
  status: 1,
  models: '',
  group: 'default'
})

const editForm = reactive({
  id: '',
  type: 1,
  name: '',
  display_name: '',
  base_url: '',
  api_key: '',
  test_model: '',
  priority: 1,
  weight: 100,
  status: 1,
  models: '',
  group: 'default'
})

// 模型名称映射
const modelNameMap = {
  // 豆包模型映射
  'Doubao-pro-128k': 'doubao-pro-128k',
  'Doubao-pro-32k': 'doubao-pro-32k',
  'Doubao-pro-4k': 'doubao-pro-4k',
  'Doubao-lite-128k': 'doubao-lite-128k',
  'Doubao-lite-32k': 'doubao-lite-32k',
  'Doubao-lite-4k': 'doubao-lite-4k',
  'Doubao-embedding': 'doubao-embedding-1.0'
}

// 渠道类型配置
const channelConfigs = {
  1: { // OpenAI
    baseUrl: 'https://api.openai.com/v1',
    testModel: 'gpt-3.5-turbo',
    models: [], // 从API获取
    apiKeyTip: 'OpenAI API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  2: { // Anthropic
    baseUrl: 'https://api.anthropic.com/v1',
    testModel: 'claude-3-sonnet-20240229',
    models: [], // 从API获取
    apiKeyTip: 'Anthropic API Key 格式为 sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  3: { // Azure
    baseUrl: 'https://your-resource.openai.azure.com/openai/deployments/{deployment-name}',
    testModel: 'gpt-35-turbo',
    models: [], // 从API获取
    apiKeyTip: 'Azure API Key 格式为 xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  4: { // Google Gemini
    baseUrl: 'https://generativeai.googleapis.com/v1',
    testModel: 'gemini-pro',
    models: [], // 从API获取
    apiKeyTip: 'Google API Key 格式为 AIzaSyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  14: { // 豆包
    baseUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    testModel: 'Doubao-pro-128k',
    models: [], // 从API获取
    apiKeyTip: '豆包 API Key 格式为 ak-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  15: { // 阿里通义
    baseUrl: 'https://dashscope.aliyuncs.com/api/v1',
    testModel: 'qwen-turbo',
    models: [], // 从API获取
    apiKeyTip: '阿里通义 API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  16: { // DeepSeek
    baseUrl: 'https://api.deepseek.com/v1',
    testModel: 'deepseek-chat',
    models: [], // 从API获取
    apiKeyTip: 'DeepSeek API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  17: { // MiniMax
    baseUrl: 'https://api.minimax.chat/v1',
    testModel: 'abab5.5-chat',
    models: [], // 从API获取
    apiKeyTip: 'MiniMax API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  18: { // 智谱 AI
    baseUrl: 'https://open.bigmodel.cn/api/mcp',
    testModel: 'chatglm3-6b',
    models: [], // 从API获取
    apiKeyTip: '智谱 AI API Key 格式为 xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  }
}

const channelType = (type) => {
  const types = { 
    1: 'OpenAI', 
    2: 'Anthropic', 
    3: 'Azure', 
    4: 'Google Gemini',
    14: '豆包',
    15: '阿里通义',
    16: 'DeepSeek',
    17: 'MiniMax',
    18: '智谱 AI'
  }
  return types[type] || 'Unknown'
}

const getChannelTypeTag = (type) => {
  switch (type) {
    case 1: return 'primary'
    case 2: return 'warning'
    case 3: return 'info'
    case 4: return 'success'
    case 14: return 'danger'
    case 15: return 'purple'
    case 16: return 'blue'
    case 17: return 'cyan'
    case 18: return 'teal'
    default: return 'default'
  }
}

const getStatusText = (status) => {
  const statuses = { 0: '禁用', 1: '启用', 2: '维护中' }
  return statuses[status] || '未知'
}

const getStatusTagType = (status) => {
  switch (status) {
    case 1: return 'success'
    case 0: return 'danger'
    case 2: return 'warning'
    default: return 'default'
  }
}

const getAvailableModels = (type) => {
  const defaultModels = channelConfigs[type]?.models || []
  const customTypeModels = customModels.value[type] || []
  return [...defaultModels, ...customTypeModels]
}

// 计算过滤后的模型列表
const updateFilteredModels = () => {
  filteredModels.value = getAvailableModels(createForm.type)
}

const updateFilteredEditModels = () => {
  filteredEditModels.value = getAvailableModels(editForm.type)
}

// 移除模型
const removeModel = (model) => {
  const index = selectedModels.value.indexOf(model)
  if (index > -1) {
    selectedModels.value.splice(index, 1)
  }
}

// 移除编辑模型
const removeEditModel = (model) => {
  const index = selectedEditModels.value.indexOf(model)
  if (index > -1) {
    selectedEditModels.value.splice(index, 1)
  }
}

const getApiKeyTip = (type) => {
  return channelConfigs[type]?.apiKeyTip || '请输入API Key'
}

const handleChannelTypeChange = () => {
  const config = channelConfigs[createForm.type]
  if (config) {
    createForm.base_url = config.baseUrl
    createForm.test_model = config.testModel
    selectedModels.value = []
  }
  // 更新过滤后的模型列表
  updateFilteredModels()
}

const handleEditChannelTypeChange = () => {
  const config = channelConfigs[editForm.type]
  if (config) {
    editForm.base_url = config.baseUrl
    editForm.test_model = config.testModel
    selectedEditModels.value = []
  }
  // 更新过滤后的模型列表
  updateFilteredEditModels()
}

const selectAllModels = () => {
  selectedModels.value = [...getAvailableModels(createForm.type)]
}

const clearSelectedModels = () => {
  selectedModels.value = []
}

const selectAllEditModels = () => {
  selectedEditModels.value = [...getAvailableModels(editForm.type)]
}

const clearSelectedEditModels = () => {
  selectedEditModels.value = []
}

const syncModels = async (type) => {
  syncingModels.value = true
  try {
    // 从后端API获取最新模型列表
    ElMessage.info('正在同步最新模型...')
    
    const response = await api.admin.getLatestModels({
      type,
      api_key: createForm.api_key,
      base_url: createForm.base_url
    })
    const latestModels = response.data.data || []
    
    // 更新渠道配置中的模型列表
    if (channelConfigs[type]) {
      channelConfigs[type].models = latestModels
    }
    
    // 更新过滤后的模型列表
    updateFilteredModels()
    
    ElMessage.success('模型同步成功')
  } catch (error) {
    console.error('Failed to sync models:', error)
    ElMessage.error('模型同步失败')
  } finally {
    syncingModels.value = false
  }
}

const syncEditModels = async (type) => {
  syncingEditModels.value = true
  try {
    // 从后端API获取最新模型列表
    ElMessage.info('正在同步最新模型...')
    
    const response = await api.admin.getLatestModels({
      type,
      api_key: editForm.api_key,
      base_url: editForm.base_url
    })
    const latestModels = response.data.data || []
    
    // 更新渠道配置中的模型列表
    if (channelConfigs[type]) {
      channelConfigs[type].models = latestModels
    }
    
    // 更新过滤后的模型列表
    updateFilteredEditModels()
    
    ElMessage.success('模型同步成功')
  } catch (error) {
    console.error('Failed to sync models:', error)
    ElMessage.error('模型同步失败')
  } finally {
    syncingEditModels.value = false
  }
}

// 初始化模型列表
const initModels = async () => {
  try {
    // 为所有渠道类型同步模型列表
    for (const type in channelConfigs) {
      const response = await api.admin.getLatestModels({ type })
      const latestModels = response.data.data || []
      if (channelConfigs[type]) {
        channelConfigs[type].models = latestModels
      }
    }
    // 更新过滤后的模型列表
    updateFilteredModels()
    updateFilteredEditModels()
  } catch (error) {
    console.error('Failed to initialize models:', error)
  }
}

const addManualModel = () => {
  if (!manualModel.value.trim()) {
    ElMessage.warning('请输入模型名称')
    return
  }
  
  const type = createForm.type
  if (!customModels.value[type]) {
    customModels.value[type] = []
  }
  
  // 检查模型是否已存在
  if (getAvailableModels(type).includes(manualModel.value.trim())) {
    ElMessage.warning('模型已存在')
    return
  }
  
  // 添加到自定义模型列表
  customModels.value[type].push(manualModel.value.trim())
  ElMessage.success('模型添加成功')
  manualModel.value = ''
}

const addEditManualModel = () => {
  if (!editManualModel.value.trim()) {
    ElMessage.warning('请输入模型名称')
    return
  }
  
  const type = editForm.type
  if (!customModels.value[type]) {
    customModels.value[type] = []
  }
  
  // 检查模型是否已存在
  if (getAvailableModels(type).includes(editManualModel.value.trim())) {
    ElMessage.warning('模型已存在')
    return
  }
  
  // 添加到自定义模型列表
  customModels.value[type].push(editManualModel.value.trim())
  ElMessage.success('模型添加成功')
  editManualModel.value = ''
}

// 根据渠道类型获取最新模型列表
const getLatestModelsByType = (type) => {
  // 模拟从API获取最新模型
  // 实际项目中，这里应该调用后端API获取最新模型
  switch (type) {
    case 1: // OpenAI
      return ['gpt-3.5-turbo', 'gpt-4', 'gpt-4o', 'gpt-4-turbo', 'gpt-4o-mini', 'gpt-5-alpha']
    case 2: // Anthropic
      return ['claude-3-sonnet-20240229', 'claude-3-opus-20240229', 'claude-3-haiku-20240307', 'claude-3.5-sonnet-20240620']
    case 3: // Azure
      return ['gpt-35-turbo', 'gpt-4', 'gpt-4-turbo', 'gpt-4o']
    case 4: // Google Gemini
      return ['gemini-pro', 'gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-2.0-pro']
    case 14: // 豆包
      return [
        'doubao-1.5-pro-20240528',
        'doubao-1.5-flash-20240528',
        'doubao-1.0-pro-20240528',
        'doubao-1.0-flash-20240528',
        'ep-20240604172028-4q775',
        'ep-20240610144610-g97pz',
        'ep-20240701171326-7q7d2',
        'ep-20240715143158-4v8z9',
        'ep-20240801150102-5g6h7',
        'doubao-embedding-1.0'
      ]
    case 15: // 阿里通义
      return ['qwen-turbo', 'qwen-plus', 'qwen-max', 'qwen-2.5-turbo', 'qwen-2.5-plus']
    case 16: // DeepSeek
      return ['deepseek-chat', 'deepseek-llm', 'deepseek-v3.1']
    case 17: // MiniMax
      return ['abab5.5-chat', 'abab6-chat', 'abab6.5-chat']
    case 18: // 智谱 AI
      return ['chatglm3-6b', 'chatglm3-6b-32k', 'glm-4', 'glm-4-flash']
    default:
      return []
  }
}

const toggleModelSelection = (model) => {
  const index = selectedModels.value.indexOf(model)
  if (index > -1) {
    selectedModels.value.splice(index, 1)
  } else {
    selectedModels.value.push(model)
  }
}

const toggleEditModelSelection = (model) => {
  const index = selectedEditModels.value.indexOf(model)
  if (index > -1) {
    selectedEditModels.value.splice(index, 1)
  } else {
    selectedEditModels.value.push(model)
  }
}

const fillRelatedModels = (type) => {
  // 填入该渠道类型的所有模型
  selectedModels.value = [...getAvailableModels(type)]
  ElMessage.success('已填入相关模型')
}

const fillEditRelatedModels = (type) => {
  // 填入该渠道类型的所有模型
  selectedEditModels.value = [...getAvailableModels(type)]
  ElMessage.success('已填入相关模型')
}

// 转换模型名称为API使用的名称
const convertModelNames = (models) => {
  return models.map(model => {
    return modelNameMap[model] || model
  })
}

const resetCreateForm = () => {
  createForm.type = 1
  createForm.name = ''
  createForm.display_name = ''
  createForm.base_url = 'https://api.openai.com/v1'
  createForm.api_key = ''
  createForm.test_model = 'gpt-3.5-turbo'
  createForm.priority = 1
  createForm.weight = 100
  createForm.status = 1
  createForm.models = ''
  createForm.group = 'default'
  selectedModels.value = []
  showCreateDialog.value = false
  showManualModelInput.value = false
  manualModel.value = ''
  // 清除同步标记
  hasSyncedModels.value = {}
  // 更新过滤后的模型列表
  updateFilteredModels()
}

const loadChannels = async () => {
  loading.value = true
  try {
    const res = await api.admin.getChannels({ page: 1, page_size: 100 })
    // 新响应格式: { code, message, data: { data: [...], total: N } }
    channels.value = res.data.data?.data || []
  } catch (error) {
    console.error('Failed to load channels:', error)
    ElMessage.error('加载渠道失败')
  } finally {
    loading.value = false
  }
}

const refreshChannels = () => {
  loadChannels()
}

const handleCreate = async () => {
  try {
    // 转换模型名称并转换为JSON字符串
    const convertedModels = convertModelNames(selectedModels.value)
    createForm.models = JSON.stringify(convertedModels)
    
    const data = {
      ...createForm
    }
    await api.admin.createChannel(data)
    ElMessage.success('创建成功')
    resetCreateForm()
    loadChannels()
  } catch (error) {
    console.error('Failed to create channel:', error)
    ElMessage.error('创建失败')
  }
}

const editChannel = (channel) => {
  editForm.id = channel.id
  editForm.type = channel.type
  editForm.name = channel.name
  editForm.display_name = channel.display_name
  editForm.base_url = channel.base_url
  editForm.api_key = channel.api_key || ''
  editForm.test_model = channel.test_model
  editForm.priority = channel.priority
  editForm.weight = channel.weight
  editForm.status = channel.status
  editForm.models = channel.models
  editForm.group = channel.group || 'default'
  
  // 解析模型列表
  try {
    selectedEditModels.value = JSON.parse(channel.models || '[]')
  } catch (e) {
    selectedEditModels.value = []
  }
  
  // 清除同步标记
  hasSyncedModels.value = {}
  
  // 更新过滤后的模型列表
  updateFilteredEditModels()
  
  showEditDialog.value = true
}

const handleEdit = async () => {
  try {
    // 转换模型名称并转换为JSON字符串
    const convertedModels = convertModelNames(selectedEditModels.value)
    editForm.models = JSON.stringify(convertedModels)
    
    const data = {
      ...editForm
    }
    await api.admin.updateChannel(editForm.id, data)
    ElMessage.success('更新成功')
    showEditDialog.value = false
    loadChannels()
  } catch (error) {
    console.error('Failed to update channel:', error)
    ElMessage.error('更新失败')
  }
}

const testChannel = async (id) => {
  testingChannels.value.push(id)
  try {
    const response = await api.admin.testChannel(id)
    ElMessage.success(`测试通过，响应时间：${response.data.response_time || 0}ms`)
  } catch (error) {
    ElMessage.error('测试失败：' + (error.response?.data?.error || '未知错误'))
  } finally {
    testingChannels.value = testingChannels.value.filter(item => item !== id)
  }
}

const batchTestChannels = async () => {
  batchTesting.value = true
  try {
    const response = await api.admin.batchTestChannels()
    ElMessage.success('批量测试完成')
    loadChannels()
  } catch (error) {
    ElMessage.error('批量测试失败')
  } finally {
    batchTesting.value = false
  }
}

const deleteChannel = async (id) => {
  if (!confirm('确定要删除此渠道吗？')) return
  try {
    await api.admin.deleteChannel(id)
    ElMessage.success('删除成功')
    loadChannels()
  } catch (error) {
    console.error('Failed to delete channel:', error)
    ElMessage.error('删除失败')
  }
}

const loadGroups = async () => {
  try {
    const res = await api.admin.getGroups({ page: 1, page_size: 100 })
    groups.value = res.data.data?.data || []
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const handleModelFocus = async (form, mode) => {
  const key = `${mode}-${form.type}`
  
  // 如果已经同步过，就不再同步
  if (hasSyncedModels.value[key]) {
    return
  }
  
  if (!form.api_key) {
    ElMessage.warning('请先填写 API Key')
    return
  }
  if (!form.base_url) {
    ElMessage.warning('请先填写 Base URL')
    return
  }
  if (!form.type) {
    ElMessage.warning('请先选择渠道类型')
    return
  }
  
  try {
    const response = await api.admin.getLatestModels({
      type: form.type,
      api_key: form.api_key,
      base_url: form.base_url
    })
    const latestModels = response.data.data || []
    
    if (channelConfigs[form.type]) {
      channelConfigs[form.type].models = latestModels
    }
    
    if (mode === 'create') {
      updateFilteredModels()
    } else {
      updateFilteredEditModels()
    }
    
    // 标记为已同步
    hasSyncedModels.value[key] = true
    
    ElMessage.success('模型列表获取成功')
  } catch (error) {
    console.error('Failed to get models:', error)
    // 不显示错误消息，避免影响用户体验
  }
}

onMounted(() => {
  loadChannels()
  loadGroups()
  // 初始化过滤后的模型列表
  updateFilteredModels()
  updateFilteredEditModels()
})
</script>

<style scoped lang="scss">
.channels-page {
  padding: 20px;
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    
    .header-actions {
      display: flex;
      gap: 10px;
    }
  }
  
  .el-table {
    margin-top: 20px;
    
    .el-table__row {
      transition: all 0.3s ease;
      
      &:hover {
        background-color: #f5f7fa;
      }
    }
  }
  
  .model-management {
    display: flex;
    flex-direction: column;
    gap: 10px;
    
    .selected-model-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      padding: 10px;
      background-color: #f5f7fa;
      border-radius: 4px;
      min-height: 40px;
      
      .selected-tag {
        font-weight: bold;
        cursor: pointer;
        transition: all 0.3s ease;
        
        &:hover {
          transform: translateY(-2px);
        }
      }
    }
    
    .model-actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      padding: 5px 0;
    }
  }
}
</style>
