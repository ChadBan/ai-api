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
        <el-table-column prop="display_name" label="显示名称" />
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
        <el-form-item label="支持模型" required>
          <div class="model-management">
            <!-- 模型搜索框 -->
            <div class="model-search">
              <el-input 
                v-model="modelSearchQuery" 
                placeholder="搜索模型..." 
                @input="handleModelSearch"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
                <template #suffix>
                  <el-button 
                    v-if="modelSearchQuery" 
                    type="text" 
                    @click="clearModelSearch"
                  >
                    <el-icon><Close /></el-icon>
                  </el-button>
                </template>
              </el-input>
            </div>
            
            <!-- 模型标签显示 -->
            <div class="model-tags">
              <el-tag 
                v-for="model in filteredModels" 
                :key="model" 
                :effect="selectedModels.includes(model) ? 'dark' : 'plain'"
                :class="{'selected-tag': selectedModels.includes(model)}"
                @click="toggleModelSelection(model)"
              >
                {{ model }}
                <el-icon v-if="selectedModels.includes(model)">
                  <Check />
                </el-icon>
              </el-tag>
              <el-empty v-if="filteredModels.length === 0" description="未找到匹配的模型" />
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
              <el-button type="text" size="small" @click="showManualModelInput = !showManualModelInput">
                更多
              </el-button>
            </div>
            
            <div v-if="showManualModelInput" class="manual-model-input">
              <el-input v-model="manualModel" placeholder="输入自定义模型名称" @keyup.enter="addManualModel" />
              <el-button type="primary" size="small" @click="addManualModel">
                填入
              </el-button>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="分组">
          <el-select v-model="createForm.group" placeholder="选择分组">
            <el-option label="default" :value="'default'" />
            <el-option label="openai" :value="'openai'" />
            <el-option label="anthropic" :value="'anthropic'" />
            <el-option label="google" :value="'google'" />
            <el-option label="azure" :value="'azure'" />
            <el-option label="domestic" :value="'domestic'" />
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
        <el-form-item label="支持模型" required>
          <div class="model-management">
            <!-- 模型搜索框 -->
            <div class="model-search">
              <el-input 
                v-model="editModelSearchQuery" 
                placeholder="搜索模型..." 
                @input="handleEditModelSearch"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
                <template #suffix>
                  <el-button 
                    v-if="editModelSearchQuery" 
                    type="text" 
                    @click="clearEditModelSearch"
                  >
                    <el-icon><Close /></el-icon>
                  </el-button>
                </template>
              </el-input>
            </div>
            
            <!-- 模型标签显示 -->
            <div class="model-tags">
              <el-tag 
                v-for="model in filteredEditModels" 
                :key="model" 
                :effect="selectedEditModels.includes(model) ? 'dark' : 'plain'"
                :class="{'selected-tag': selectedEditModels.includes(model)}"
                @click="toggleEditModelSelection(model)"
              >
                {{ model }}
                <el-icon v-if="selectedEditModels.includes(model)">
                  <Check />
                </el-icon>
              </el-tag>
              <el-empty v-if="filteredEditModels.length === 0" description="未找到匹配的模型" />
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
              <el-button type="text" size="small" @click="showEditManualModelInput = !showEditManualModelInput">
                更多
              </el-button>
            </div>
            
            <div v-if="showEditManualModelInput" class="manual-model-input">
              <el-input v-model="editManualModel" placeholder="输入自定义模型名称" @keyup.enter="addEditManualModel" />
              <el-button type="primary" size="small" @click="addEditManualModel">
                填入
              </el-button>
            </div>
          </div>
        </el-form-item>
        
        <el-form-item label="分组">
          <el-select v-model="editForm.group" placeholder="选择分组">
            <el-option label="default" :value="'default'" />
            <el-option label="openai" :value="'openai'" />
            <el-option label="anthropic" :value="'anthropic'" />
            <el-option label="google" :value="'google'" />
            <el-option label="azure" :value="'azure'" />
            <el-option label="domestic" :value="'domestic'" />
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
const showEditManualModelInput = ref(false)
const editManualModel = ref('')
const customModels = ref({}) // 存储手动添加的模型
const modelSearchQuery = ref('')
const editModelSearchQuery = ref('')
const filteredModels = ref([])
const filteredEditModels = ref([])

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

// 渠道类型配置
const channelConfigs = {
  1: { // OpenAI
    baseUrl: 'https://api.openai.com/v1',
    testModel: 'gpt-3.5-turbo',
    models: ['gpt-3.5-turbo', 'gpt-4', 'gpt-4o', 'gpt-4-turbo'],
    apiKeyTip: 'OpenAI API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  2: { // Anthropic
    baseUrl: 'https://api.anthropic.com/v1',
    testModel: 'claude-3-sonnet-20240229',
    models: ['claude-3-sonnet-20240229', 'claude-3-opus-20240229', 'claude-3-haiku-20240307'],
    apiKeyTip: 'Anthropic API Key 格式为 sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  3: { // Azure
    baseUrl: 'https://your-resource.openai.azure.com/openai/deployments/{deployment-name}',
    testModel: 'gpt-35-turbo',
    models: ['gpt-35-turbo', 'gpt-4', 'gpt-4-turbo'],
    apiKeyTip: 'Azure API Key 格式为 xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  4: { // Google Gemini
    baseUrl: 'https://generativeai.googleapis.com/v1',
    testModel: 'gemini-pro',
    models: ['gemini-pro', 'gemini-1.5-pro', 'gemini-1.5-flash'],
    apiKeyTip: 'Google API Key 格式为 AIzaSyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  14: { // 豆包
    baseUrl: 'https://ark.cn-beijing.volces.com/api/v3',
    testModel: 'ep-20240604172028-4q775',
    models: ['ep-20240604172028-4q775', 'ep-20240610144610-g97pz'],
    apiKeyTip: '豆包 API Key 格式为 ak-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  15: { // 阿里通义
    baseUrl: 'https://dashscope.aliyuncs.com/api/v1',
    testModel: 'qwen-turbo',
    models: ['qwen-turbo', 'qwen-plus', 'qwen-max'],
    apiKeyTip: '阿里通义 API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  16: { // DeepSeek
    baseUrl: 'https://api.deepseek.com/v1',
    testModel: 'deepseek-chat',
    models: ['deepseek-chat', 'deepseek-llm'],
    apiKeyTip: 'DeepSeek API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  17: { // MiniMax
    baseUrl: 'https://api.minimax.chat/v1',
    testModel: 'abab5.5-chat',
    models: ['abab5.5-chat', 'abab6-chat'],
    apiKeyTip: 'MiniMax API Key 格式为 sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
  },
  18: { // 智谱 AI
    baseUrl: 'https://open.bigmodel.cn/api/mcp',
    testModel: 'chatglm3-6b',
    models: ['chatglm3-6b', 'chatglm3-6b-32k', 'glm-4'],
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
  if (!modelSearchQuery.value) {
    filteredModels.value = getAvailableModels(createForm.type)
  } else {
    filteredModels.value = getAvailableModels(createForm.type).filter(model => 
      model.toLowerCase().includes(modelSearchQuery.value.toLowerCase())
    )
  }
}

const updateFilteredEditModels = () => {
  if (!editModelSearchQuery.value) {
    filteredEditModels.value = getAvailableModels(editForm.type)
  } else {
    filteredEditModels.value = getAvailableModels(editForm.type).filter(model => 
      model.toLowerCase().includes(editModelSearchQuery.value.toLowerCase())
    )
  }
}

// 处理模型搜索
const handleModelSearch = () => {
  updateFilteredModels()
}

const handleEditModelSearch = () => {
  updateFilteredEditModels()
}

// 清空搜索
const clearModelSearch = () => {
  modelSearchQuery.value = ''
  updateFilteredModels()
}

const clearEditModelSearch = () => {
  editModelSearchQuery.value = ''
  updateFilteredEditModels()
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
    
    const response = await api.admin.getLatestModels({ type })
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
    
    const response = await api.admin.getLatestModels({ type })
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
      return ['ep-20240604172028-4q775', 'ep-20240610144610-g97pz', 'doubao-1.5-pro-20240528']
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
}

const loadChannels = async () => {
  loading.value = true
  try {
    const res = await api.admin.getChannels({ page: 1, page_size: 100 })
    // 新响应格式: { code, message, data: { items: [...], total: N } }
    channels.value = res.data.data?.items || []
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
    // 将选中的模型转换为JSON字符串
    createForm.models = JSON.stringify(selectedModels.value)
    
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
  editForm.api_key = channel.api_key
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
  
  showEditDialog.value = true
  showEditManualModelInput.value = false
  editManualModel.value = ''
}

const handleEdit = async () => {
  try {
    // 将选中的模型转换为JSON字符串
    editForm.models = JSON.stringify(selectedEditModels.value)
    
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

onMounted(() => {
  loadChannels()
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
    
    .model-search {
      margin-bottom: 5px;
      
      .el-input {
        width: 100%;
      }
    }
    
    .model-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      padding: 10px;
      background-color: #f5f7fa;
      border-radius: 4px;
      min-height: 80px;
      
      .el-tag {
        cursor: pointer;
        transition: all 0.3s ease;
        
        &:hover {
          transform: translateY(-2px);
        }
      }
      
      .selected-tag {
        font-weight: bold;
      }
    }
    
    .model-actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      padding: 5px 0;
    }
    
    .manual-model-input {
      display: flex;
      gap: 10px;
      margin-top: 5px;
      padding: 10px;
      background-color: #f0f2f5;
      border-radius: 4px;
      
      .el-input {
        flex: 1;
      }
    }
  }
}
</style>
