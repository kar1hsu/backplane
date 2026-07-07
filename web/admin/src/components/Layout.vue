<template>
  <el-container class="layout-container">
    <el-aside :width="isCollapse ? '64px' : '220px'" class="layout-aside">
      <div class="logo">
        <el-icon :size="24"><Monitor /></el-icon>
        <span v-show="!isCollapse" class="logo-text">Frame Admin</span>
      </div>
      <el-menu
        :default-active="route.path"
        :collapse="isCollapse"
        router
        background-color="#001529"
        text-color="#ffffffa6"
        active-text-color="#ffffff"
        class="aside-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <template #title>控制台</template>
        </el-menu-item>
        <MenuItem v-for="menu in userStore.menuTree" :key="menu.id" :item="menu" />
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="layout-header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse" :size="20">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><Avatar /></el-icon>
              {{ userStore.userInfo?.nickname || userStore.userInfo?.username || '管理员' }}
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <TabBar ref="tabBarRef" @refresh="handleRefresh" />
      <el-main class="layout-main">
        <router-view v-slot="{ Component }">
          <keep-alive :include="tabStore.cachedNames()">
            <component :is="Component" v-if="showPage" :key="route.path" />
          </keep-alive>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { useTabStore } from '@/store/tab'
import TabBar from './TabBar.vue'
import MenuItem from './MenuItem.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const tabStore = useTabStore()
const isCollapse = ref(false)
const showPage = ref(true)
const tabBarRef = ref()

async function handleRefresh() {
  showPage.value = false
  await nextTick()
  showPage.value = true
}

onMounted(async () => {
  if (userStore.token) {
    try {
      if (!userStore.userInfo) {
        await userStore.fetchProfile()
      }
      if (userStore.menuTree.length === 0) {
        await userStore.fetchMenus()
      }
      if (userStore.permissions.length === 0) {
        await userStore.fetchPermissions()
      }
    } catch {
      // 401 已由 request 拦截器统一清登录态并跳转；
      // 其他错误（网络抖动 / 5xx）不强制登出，避免把用户误踢下线
    }
  }
})

async function handleCommand(command: string) {
  if (command === 'logout') {
    await userStore.logout()
    tabStore.closeAll()
    router.push('/login')
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}
.layout-aside {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #001529;
  transition: width 0.3s;
  overflow: hidden;
}
.logo {
  height: 60px;
  flex: 0 0 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  border-bottom: 1px solid #ffffff1a;
}
.logo-text {
  white-space: nowrap;
}
.aside-menu {
  flex: 1 1 auto;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  border-right: none;
}
.aside-menu::-webkit-scrollbar {
  width: 6px;
}
.aside-menu::-webkit-scrollbar-thumb {
  background: #ffffff24;
  border-radius: 999px;
}
.aside-menu::-webkit-scrollbar-track {
  background: transparent;
}
.layout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #f0f0f0;
  background: #fff;
  padding: 0 20px;
}
.header-left {
  display: flex;
  align-items: center;
}
.collapse-btn {
  cursor: pointer;
}
.header-right {
  display: flex;
  align-items: center;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 14px;
}
.layout-main {
  background: #f5f5f5;
  overflow-y: auto;
}
</style>
