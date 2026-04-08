import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      // 使用 M-API-KEY header 传递 token（与后端保持一致）
      config.headers['M-API-KEY'] = token
    }
    return config
  },
  error => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    const res = response.data

    // 检查是否为新的统一响应格式（有 code 字段）
    if (res && typeof res.code === 'number') {
      console.log('[API Response]', response.config.url, 'code:', res.code)

      // 成功响应 (code: 1000)
      if (res.code === 1000) {
        return response
      }

      // 业务错误
      const errorMsg = res.message_detail || res.message || '请求失败'
      console.warn('[API Error]', response.config.url, 'code:', res.code, 'message:', errorMsg)
      ElMessage.error(errorMsg)

      // Token 相关错误，跳转到登录页
      if (res.code === 4001 || res.code === 4103 || res.code === 4104) {
        console.warn('[API] Token error, redirecting to login')
        localStorage.removeItem('token')
        window.location.href = '/login'
      }

      return Promise.reject(new Error(errorMsg))
    }

    // 旧格式或无 code 字段的响应，直接返回
    return response
  },
  error => {
    console.error('[API HTTP Error]', error.response?.status, error.config?.url)

    if (error.response) {
      const res = error.response.data

      switch (error.response.status) {
        case 401:
          console.warn('[API 401] Unauthorized, clearing token and redirecting to login')
          // 检查是否为新格式
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '未授权'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error(res?.error || '未授权')
          }
          // localStorage.removeItem('token')
          // window.location.href = '/login'
          break
        case 400:
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '请求参数错误'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error(res?.error || '请求参数错误')
          }
          break
        case 403:
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '没有权限访问此资源'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error('没有权限访问此资源')
          }
          break
        case 404:
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '请求的资源不存在'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error('请求的资源不存在')
          }
          break
        case 500:
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '服务器错误'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error('服务器错误')
          }
          break
        default:
          if (res && res.code) {
            const errorMsg = res.message_detail || res.message || '请求失败'
            ElMessage.error(errorMsg)
          } else {
            ElMessage.error(res?.error || '请求失败')
          }
      }
    } else if (error.request) {
      ElMessage.error('网络错误，请检查网络连接')
    } else {
      ElMessage.error('请求配置错误')
    }
    return Promise.reject(error)
  }
)

// API 方法
export default {
  // 认证
  login(email, password) {
    return api.post('/v1/auth/login', { email, password })
  },

  register(username, email, password) {
    return api.post('/v1/auth/register', { name: username, email, password })
  },

  getUserInfo() {
    return api.get('/v1/user/self')
  },

  // 余额
  getBalance() {
    return api.get('/v1/balance')
  },

  // 消费记录
  getBillings(params) {
    return api.get('/v1/billings', { params })
  },

  // Token 管理
  getTokens(params) {
    return api.get('/v1/tokens', { params })
  },

  createToken(data) {
    return api.post('/v1/tokens', data)
  },

  updateToken(id, data) {
    return api.put(`/v1/tokens/${id}`, data)
  },

  deleteToken(id) {
    return api.delete(`/v1/tokens/${id}`)
  },

  toggleTokenStatus(id) {
    return api.post(`/v1/tokens/${id}/status`)
  },

  getTokenStats() {
    return api.get('/v1/tokens/stats')
  },

  getTokenUsageLogs(params) {
    return api.get('/v1/tokens/usage-logs', { params })
  },

  // 兑换码
  getRedemptions(params) {
    return api.get('/v1/redemptions', { params })
  },

  createRedemption(data) {
    return api.post('/v1/redemptions', data)
  },

  redeem(code) {
    return api.post('/v1/redemptions/redeem', { code })
  },

  deleteRedemption(id) {
    return api.delete(`/v1/redemptions/${id}`)
  },

  // 统计数据
  getDashboard() {
    return api.get('/v1/statistics/dashboard')
  },

  getUserStats(params) {
    return api.get('/v1/statistics/user', { params })
  },

  // 模型列表
  getModels() {
    return api.get('/v1/models')
  },

  // 获取所有渠道的模型列表
  getChannelModels() {
    return api.get('/v1/models/channel')
  },

  // 聊天
  chatCompletions(data) {
    // 使用原生 fetch 进行流式请求（axios 的 responseType: 'stream' 在浏览器中不可靠）
    const token = localStorage.getItem('token')
    const headers = {
      'Content-Type': 'application/json'
    }
    if (token) {
      // 使用 M-API-KEY header 传递 token（与后端保持一致）
      headers['M-API-KEY'] = token
    }

    return fetch('/v1/playground/chat/completions', {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(data)
    })
  },

  // 管理员接口
  admin: {
    // 渠道管理
    getChannels(params) {
      return api.get('/v1/admin/channels', { params })
    },

    createChannel(data) {
      return api.post('/v1/admin/channels', data)
    },

    updateChannel(id, data) {
      return api.put(`/v1/admin/channels/${id}`, data)
    },

    deleteChannel(id) {
      return api.delete(`/v1/admin/channels/${id}`)
    },

    testChannel(id) {
      return api.post(`/v1/admin/channels/${id}/test`)
    },

    batchTestChannels() {
      return api.post(`/v1/admin/channels/test-all`)
    },

    getLatestModels(params) {
      return api.get(`/v1/admin/models/latest`, { params })
    },

    searchModels(params) {
      return api.get(`/v1/admin/models/search`, { params })
    },

    // 用户管理
    getUsers(params) {
      return api.get('/v1/admin/users', { params })
    },

    getUser(id) {
      return api.get(`/v1/admin/users/${id}`)
    },

    updateUser(id, data) {
      return api.put(`/v1/admin/users/${id}`, data)
    },

    deleteUser(id) {
      return api.delete(`/v1/admin/users/${id}`)
    },

    banUser(id, status) {
      return api.post(`/v1/admin/users/${id}/ban`, { status })
    },

    addBalance(id, data) {
      return api.post(`/v1/admin/users/${id}/balance`, data)
    },

    // 系统配置
    getSystemConfig() {
      return api.get('/v1/admin/config')
    },

    updateSystemConfig(data) {
      return api.put('/v1/admin/config', data)
    },

    // 模型定价管理
    getModelPrices(params) {
      return api.get('/v1/admin/pricing/models', { params })
    },

    createModelPrice(data) {
      return api.post('/v1/admin/pricing/models', data)
    },

    updateModelPrice(id, data) {
      return api.put(`/v1/admin/pricing/models/${id}`, data)
    },

    deleteModelPrice(id) {
      return api.delete(`/v1/admin/pricing/models/${id}`)
    },

    // 分组管理
    getGroups(params) {
      return api.get('/v1/admin/groups', { params })
    },

    createGroup(data) {
      return api.post('/v1/admin/groups', data)
    },

    updateGroup(id, data) {
      return api.put(`/v1/admin/groups/${id}`, data)
    },

    deleteGroup(id) {
      return api.delete(`/v1/admin/groups/${id}`)
    },

    // 日志
    getLogs(params) {
      return api.get('/v1/admin/logs', { params })
    },

    getAuditLogs(params) {
      return api.get('/v1/logs/audit', { params })
    },
    getRequestLogs(params) {
      return api.get('/v1/logs/request', { params })
    },
    getErrorLogs(params) {
      return api.get('/v1/logs/error', { params })
    },
    getLoginLogs(params) {
      return api.get('/v1/logs/login', { params })
    }
  }
}
