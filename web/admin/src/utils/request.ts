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

// 并发请求同时遇到登录失效时，只提示一次、只跳转一次。
let redirectingToLogin = false

// 401：清空登录态（含 Pinia store）并跳回登录页。
// 直接 clearAuth 而不调用 logout 接口，避免登出请求再次 401 造成递归。
function handleUnauthorized(text: string) {
  if (redirectingToLogin) return
  useUserStore().clearAuth()
  if (router.currentRoute.value.name === 'Login') return
  redirectingToLogin = true
  message.error(text)
  router.replace('/login').finally(() => {
    redirectingToLogin = false
  })
}

request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      if (res.code === 401) {
        handleUnauthorized(res.message || '登录状态已失效')
      } else {
        message.error(res.message || '请求失败')
      }
      return Promise.reject(new Error(res.message))
    }
    return res
  },
  (error) => {
    // 优先展示后端返回的业务提示（如"无访问权限"），而不是 axios 的通用文案
    const text = error.response?.data?.message || error.message || '网络错误'
    if (error.response?.status === 401) {
      handleUnauthorized(text)
    } else {
      message.error(text)
    }
    return Promise.reject(error)
  }
)

export default request
