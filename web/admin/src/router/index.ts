import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useTabStore } from '@/store/tab'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录', noAuth: true },
  },
  {
    path: '/',
    component: () => import('@/components/Layout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '控制台' },
      },
      {
        path: 'system/user',
        name: 'SystemUser',
        component: () => import('@/views/system/UserList.vue'),
        meta: { title: '用户管理' },
      },
      {
        path: 'system/role',
        name: 'SystemRole',
        component: () => import('@/views/system/RoleList.vue'),
        meta: { title: '角色管理' },
      },
      {
        path: 'system/menu',
        name: 'SystemMenu',
        component: () => import('@/views/system/MenuList.vue'),
        meta: { title: '菜单管理' },
      },
      {
        path: 'system/config',
        name: 'SystemConfig',
        component: () => import('@/views/system/ConfigList.vue'),
        meta: { title: '系统配置' },
      },
      {
        path: 'system/operlog',
        name: 'SystemOperLog',
        component: () => import('@/views/system/OperationLogList.vue'),
        meta: { title: '操作日志' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || ''} - Backplane Admin`
  const token = localStorage.getItem('token')
  if (!to.meta.noAuth && !token) {
    next('/login')
    return
  }

  if (to.name && to.meta.title && !to.meta.noAuth) {
    const tabStore = useTabStore()
    tabStore.addTab({
      path: to.path,
      name: to.name as string,
      title: to.meta.title as string,
      affix: to.path === '/dashboard',
    })
  }

  next()
})

export default router
