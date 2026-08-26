import axios from 'axios'
import router from '@/router'
import { useUserStore } from '@/store/user'
import { message } from '@/utils/ui'

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '',
  timeout: 10000,
})

request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 401：清空登录态（含 Pinia store）并跳回登录页。
// 直接 clearAuth 而不调用 logout 接口，避免登出请求再次 401 造成递归。
function handleUnauthorized() {
  useUserStore().clearAuth()
  if (router.currentRoute.value.name !== 'Login') {
    router.replace('/login')
  }
}

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      message.error(res.message || '请求失败')
      if (res.code === 401) {
        handleUnauthorized()
      }
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      handleUnauthorized()
    }
    message.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

export default request
