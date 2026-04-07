<template>
  <div class="playground-page">
    <el-card>
      <template #header>
        <div class="flex justify-between items-center">
          <span>AI 对话 - 模型数：{{ models.length }}</span>
          <el-select v-model="selectedModel" placeholder="选择模型" class="w-48">
            <el-option
              v-for="model in models"
              :key="model.value"
              :label="model.label"
              :value="model.value"
            />
          </el-select>
        </div>
      </template>
      
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
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script>
import api from '@/api'

export default {
  data() {
    return {
      messages: [],
      inputMessage: '',
      selectedModel: '',
      isLoading: false,
      isStreaming: false,
      streamingContent: '',
      models: []
    }
  },
  mounted() {
    // 加载模型列表
    this.loadModels()
    // 初始化时添加欢迎消息
    this.messages.push({ 
      role: 'assistant', 
      content: '你好！我是您的 AI 助手，有什么可以帮助您的吗？' 
    })
  },
  methods: {
    async loadModels() {
      console.log('开始加载模型列表...')
      try {
        console.log('调用 api.getModels()...')
        const res = await api.getModels()
        console.log('API 响应:', res)
        if (res.data && res.data.data) {
          console.log('模型数据:', res.data.data)
          this.models = res.data.data.map(m => ({
            label: m.display_name || m.name,
            value: m.name
          }))
          console.log('转换后的模型列表:', this.models)
          // 默认选择第一个模型
          if (this.models.length > 0) {
            this.selectedModel = this.models[0].value
            console.log('已设置默认模型:', this.selectedModel)
          }
        } else {
          console.error('API 返回数据格式不对:', res)
        }
      } catch (error) {
        console.error('Failed to load models:', error)
        // 如果加载失败，使用默认模型列表
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
        const response = await api.chatCompletions({
          model: this.selectedModel,
          messages: this.messages.map(msg => ({
            role: msg.role,
            content: msg.content
          })),
          stream: true
        })
        
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
    }
  }
}
</script>

<style scoped>
.playground-page {
  padding: 20px;
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
</style>
