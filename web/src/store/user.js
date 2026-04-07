import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useUserStore = defineStore('user', () => {
  const userInfo = ref(null)
  const token = ref(localStorage.getItem('token') || '')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => userInfo.value?.role === 'admin')

  function setToken(newToken) {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  function clearToken() {
    token.value = ''
    localStorage.removeItem('token')
    userInfo.value = null
    
    // 同时清除 sessionStorage 中的临时数据
    sessionStorage.removeItem('user_info')
  }

  async function fetchUserInfo() {
    if (!token.value) {
      console.warn('fetchUserInfo: No token available')
      return
    }
    
    try {
      console.log('fetchUserInfo: Making API request with token:', token.value.substring(0, 20) + '...')
      const res = await api.getUserInfo()
      console.log('fetchUserInfo: API response:', res)
      console.log('fetchUserInfo: response.data:', res.data)
      
      // 新响应格式: { code: 1000, message: "Success", data: { id, email, name, ... } }
      userInfo.value = res.data?.data || res.data
      console.log('Fetched user info:', userInfo.value)
    } catch (error) {
      console.error('Failed to fetch user info:', error)
      // 不要在这里清除 token，让响应拦截器处理 401 错误
      // clearToken() 会导致重复跳转
    }
  }

  return {
    userInfo,
    token,
    isLoggedIn,
    isAdmin,
    setToken,
    clearToken,
    fetchUserInfo
  }
})
