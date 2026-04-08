<template>
  <div class="playground-page">
    <el-card>
      <template #header>
        <div class="flex justify-between items-center">
          <span>AI 对话 </span>
          <div class="flex gap-2">
            <el-select v-model="selectedChannel" placeholder="选择渠道" class="w-48" @change="handleChannelChange">
              <el-option
                v-for="channel in channels"
                :key="channel.id"
                :label="channel.name"
                :value="channel.id"
              />
            </el-select>
            <el-select v-model="selectedModel" placeholder="选择模型" class="w-48">
              <el-option
                v-for="model in filteredModels"
                :key="model.value"
                :label="model.label"
                :value="model.value"
              />
            </el-select>
            <el-button @click="showAdvancedSettings = !showAdvancedSettings">
              {{ showAdvancedSettings ? '隐藏参数' : '显示参数' }}
            </el-button>
          </div>
        </div>
      </template>
      
      <!-- 高级参数设置 -->
      <el-collapse-transition>
        <div v-show="showAdvancedSettings" class="mb-4 p-4 bg-gray-50 rounded-lg">
          <el-row :gutter="20">
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Temperature">
                  <el-slider v-model="temperature" :min="0" :max="2" :step="0.01" />
                </el-form-item>
              </el-form>
            </el-col>
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Max Tokens">
                  <el-input-number v-model="maxTokens" :min="1" :max="4096" />
                </el-form-item>
              </el-form>
            </el-col>
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Top P">
                  <el-slider v-model="topP" :min="0" :max="1" :step="0.01" />
                </el-form-item>
              </el-form>
            </el-col>
          </el-row>
          <el-row :gutter="20" class="mt-2">
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Frequency Penalty">
                  <el-slider v-model="frequencyPenalty" :min="-2" :max="2" :step="0.01" />
                </el-form-item>
              </el-form>
            </el-col>
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Presence Penalty">
                  <el-slider v-model="presencePenalty" :min="-2" :max="2" :step="0.01" />
                </el-form-item>
              </el-form>
            </el-col>
            <el-col :span="8">
              <el-form label-position="top">
                <el-form-item label="Stop Sequences">
                  <el-input v-model="stopSequences" placeholder="逗号分隔的停止序列" />
                </el-form-item>
              </el-form>
            </el-col>
          </el-row>
        </div>
      </el-collapse-transition>
      
      <div class="chat-container">
        <div class="chat-messages" ref="messagesContainer">
          <div
            v-for="(message, index) in messages"
            :key="index"
            :class="['message', message.role]"
          >
            <div class="message-content">{{ message.content }}</div>
          </div>
          <div v-if="isStreaming" class="message assistant">
            <div class="message-content">{{ streamingContent }}</div>
          </div>
        </div>
        
        <div class="chat-input">
          <el-input
            v-model="inputMessage"
            type="textarea"
            :rows="3"
            placeholder="输入您的问题..."
            @keyup.enter.exact="sendMessage"
          />
          <div class="flex justify-end mt-2">
            <el-button
              type="primary"
              @click="sendMessage"
              :loading="isLoading"
              :disabled="isLoading || !inputMessage.trim()"
            >
              发送
            </el-button>
            <el-button @click="clearChat">清空</el-button>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script>
import api from '@/api';

export default {
  name: 'EnhancedPlayground',
  data() {
    return {
      messages: [],
      inputMessage: '',
      selectedChannel: '',
      selectedModel: '',
      isLoading: false,
      isStreaming: false,
      streamingContent: '',
      channels: [],
      models: [],
      showAdvancedSettings: false,
      // 高级参数
      temperature: 0.7,
      maxTokens: 2048,
      topP: 0.9,
      frequencyPenalty: 0,
      presencePenalty: 0,
      stopSequences: ''
    }
  },
  mounted() {
    // 加载渠道列表
    this.loadChannels()
    // 初始化时添加欢迎消息
    this.messages.push({ 
      role: 'assistant', 
      content: '你好！我是您的 AI 助手，有什么可以帮助您的吗？' 
    })
  },
  computed: {
    filteredModels() {
      if (!this.selectedChannel) {
        return this.models
      }
      // 找到选中的渠道
      const channel = this.channels.find(c => c.id === this.selectedChannel)
      if (!channel || !channel.models) {
        return []
      }
      // 解析渠道的模型列表
      try {
        const channelModels = JSON.parse(channel.models)
        if (Array.isArray(channelModels)) {
          // 过滤出该渠道支持的模型
          return this.models.filter(model => channelModels.includes(model.value))
        }
      } catch (e) {
        console.error('解析模型列表失败:', e)
      }
      return []
    }
  },
  methods: {
    async loadChannels() {
      try {
        // 从渠道获取模型列表，同时也获取渠道信息
        const res = await api.getChannelModels()
        if (res.data && res.data.data) {
          // 保存渠道信息
          this.channels = res.data.data
          
          // 提取所有唯一的模型
          const uniqueModels = new Set()
          this.channels.forEach(channel => {
            if (channel.models) {
              try {
                const models = JSON.parse(channel.models)
                if (Array.isArray(models)) {
                  models.forEach(model => uniqueModels.add(model))
                }
              } catch (e) {
                console.error('解析模型列表失败:', e)
              }
            }
          })
          // 转换为下拉选项格式
          this.models = Array.from(uniqueModels).map(model => ({
            label: model,
            value: model
          }))
          // 默认选择第一个渠道和第一个模型
          if (this.channels.length > 0) {
            this.selectedChannel = this.channels[0].id
          }
          if (this.models.length > 0) {
            this.selectedModel = this.models[0].value
          }
        } else {
          console.error('API 返回数据格式不对:', res)
          // 如果加载失败，使用默认模型列表
          this.useDefaultModels()
        }
      } catch (error) {
        console.error('Failed to load channels and models:', error)
        // 如果从渠道获取失败，尝试使用旧的模型 API
        try {
          const res = await api.get('/v1/models/available')
          if (res.data && res.data.data) {
            const availableModels = Array.isArray(res.data.data) ? res.data.data : []
            this.models = availableModels.map(m => ({
              label: m.display_name || m.name,
              value: m.name
            }))
            if (this.models.length > 0) {
              this.selectedModel = this.models[0].value
            } else {
              this.useDefaultModels()
            }
          } else {
            this.useDefaultModels()
          }
        } catch (e) {
          console.error('Failed to load models from available API:', e)
          this.useDefaultModels()
        }
      }
    },
    handleChannelChange() {
      // 当渠道变化时，清空模型选择
      this.selectedModel = ''
      // 如果有过滤后的模型，选择第一个
      if (this.filteredModels.length > 0) {
        this.selectedModel = this.filteredModels[0].value
      }
    },
    useDefaultModels() {
      this.models = [
        { label: 'GPT-3.5 Turbo', value: 'gpt-3.5-turbo' },
        { label: 'GPT-4', value: 'gpt-4' },
        { label: 'Claude 3', value: 'claude-3' },
        { label: '豆包', value: 'doubao-pro' },
        { label: '通义千问', value: 'qwen-turbo' },
        { label: 'DeepSeek', value: 'deepseek-chat' },
        { label: 'MiniMax', value: 'abab6' },
        { label: '智谱 AI', value: 'glm-4' }
      ]
      if (this.models.length > 0) {
        this.selectedModel = this.models[0].value
      }
    },
    async sendMessage() {
      if (!this.inputMessage.trim() || this.isLoading) return
      
      const userMessage = this.inputMessage.trim()
      this.messages.push({ role: 'user', content: userMessage })
      this.inputMessage = ''
      this.isLoading = true
      this.isStreaming = true
      this.streamingContent = ''
      
      try {
        // 构建请求参数
        const requestData = {
          model: this.selectedModel,
          messages: this.messages.filter(msg => msg.role !== 'assistant' || msg.content !== this.streamingContent)
            .map(msg => ({
              role: msg.role,
              content: msg.content
            })),
          stream: true
        }
        
        // 添加高级参数
        requestData.temperature = this.temperature
        requestData.max_tokens = this.maxTokens
        requestData.top_p = this.topP
        requestData.frequency_penalty = this.frequencyPenalty
        requestData.presence_penalty = this.presencePenalty
        
        if (this.stopSequences) {
          const sequences = this.stopSequences.split(',').map(s => s.trim()).filter(s => s)
          if (sequences.length > 0) {
            requestData.stop = sequences
          }
        }
        
        const response = await api.chatCompletions(requestData)
        this.handleStreamResponse(response)
      } catch (error) {
        this.isLoading = false
        this.isStreaming = false
        
        // 显示详细的错误信息
        let errorMsg = '请求失败'
        // fetch 的错误处理
        if (error.status) {
          errorMsg = `HTTP ${error.status}`
        } else if (error.message) {
          errorMsg = error.message
        }
        
        this.messages.push({ 
          role: 'assistant', 
          content: `错误：${errorMsg}` 
        })
        this.scrollToBottom()
      }
    },
    handleStreamResponse(response) {
      const reader = response.body.getReader()
      const decoder = new TextDecoder('utf-8')
      
      const processStream = async () => {
        try {
          const { done, value } = await reader.read()
          
          if (done) {
            this.messages.push({ 
              role: 'assistant', 
              content: this.streamingContent 
            })
            this.isLoading = false
            this.isStreaming = false
            this.streamingContent = ''
            this.scrollToBottom()
            return
          }
          
          const chunk = decoder.decode(value, { stream: true })
          const lines = chunk.split('\n')
          
          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6)
              if (data === '[DONE]') {
                continue
              }
              
              try {
                const json = JSON.parse(data)
                if (json.choices && json.choices[0] && json.choices[0].delta) {
                  const content = json.choices[0].delta.content
                  if (content) {
                    this.streamingContent += content
                    this.scrollToBottom()
                  }
                }
              } catch (e) {
                console.error('解析流式响应失败:', e)
              }
            }
          }
          
          processStream()
        } catch (error) {
          console.error('读取流式响应失败:', error)
          this.isLoading = false
          this.isStreaming = false
        }
      }
      
      processStream()
    },
    scrollToBottom() {
      this.$nextTick(() => {
        const container = this.$refs.messagesContainer
        if (container) {
          container.scrollTop = container.scrollHeight
        }
      })
    },
    clearChat() {
      this.messages = []
      this.messages.push({ 
        role: 'assistant', 
        content: '你好！我是您的 AI 助手，有什么可以帮助您的吗？' 
      })
    }
  }
}
</script>

<style scoped>
.playground-page {
  padding: 20px;
}

.flex {
  display: flex;
}

.justify-between {
  justify-content: space-between;
}

.items-center {
  align-items: center;
}

.w-48 {
  width: 12rem;
}

.gap-2 {
  gap: 0.5rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mt-2 {
  margin-top: 0.5rem;
}

.mt-4 {
  margin-top: 1rem;
}

.bg-gray-50 {
  background-color: #f9fafb;
}

.rounded-lg {
  border-radius: 0.5rem;
}

.p-4 {
  padding: 1rem;
}

.chat-container {
  height: 70vh;
  display: flex;
  flex-direction: column;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  margin-bottom: 16px;
}

.message {
  margin-bottom: 16px;
  max-width: 80%;
}

.message.user {
  align-self: flex-end;
  margin-left: auto;
  text-align: right;
}

.message.assistant {
  align-self: flex-start;
}

.message-content {
  padding: 10px 14px;
  border-radius: 8px;
  line-height: 1.5;
}

.message.user .message-content {
  background-color: #ecf5ff;
  color: #409eff;
  border-top-right-radius: 2px;
}

.message.assistant .message-content {
  background-color: #f5f7fa;
  color: #303133;
  border-top-left-radius: 2px;
}

.chat-input {
  width: 100%;
}

.el-textarea {
  width: 100%;
}

.el-slider {
  width: 100%;
}

.el-input-number {
  width: 100%;
}

.el-collapse-transition {
  transition: all 0.3s ease;
}
</style>