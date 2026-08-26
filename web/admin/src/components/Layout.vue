<template>
  <div class="app-shell">
    <n-layout has-sider class="app-layout">
      <n-layout-sider
        v-if="!isMobile"
        bordered
        collapse-mode="width"
        :collapsed="collapsed"
        :collapsed-width="68"
        :width="232"
        :native-scrollbar="false"
        class="sidebar"
      >
        <div class="brand" :class="{ compact: collapsed }">
          <div class="brand-mark">B</div>
          <div v-if="!collapsed" class="brand-copy">
            <strong>Backplane</strong>
            <span>Admin Console</span>
          </div>
        </div>
        <n-menu
          inverted
          :collapsed="collapsed"
          :collapsed-width="68"
          :collapsed-icon-size="21"
          :indent="20"
          :options="menuOptions"
          :value="route.path"
          @update:value="handleMenuSelect"
        />
      </n-layout-sider>

      <n-layout class="workspace">
        <n-layout-header bordered class="topbar">
          <div class="topbar-left">
            <n-button quaternary circle aria-label="切换导航" @click="toggleNavigation">
              <template #icon>
                <n-icon :component="isMobile ? MenuOutline : collapsed ? ChevronForwardOutline : ChevronBackOutline" />
              </template>
            </n-button>
            <div class="page-context">
              <span class="context-label">工作台</span>
              <strong>{{ route.meta.title || '控制台' }}</strong>
            </div>
          </div>

          <n-dropdown :options="userOptions" trigger="click" @select="handleCommand">
            <button class="user-trigger" type="button">
              <n-avatar round size="small" color="#0f766e">{{ userInitial }}</n-avatar>
              <span class="user-name">{{ displayName }}</span>
              <n-icon :component="ChevronDownOutline" size="14" />
            </button>
          </n-dropdown>
        </n-layout-header>

        <TabBar ref="tabBarRef" @refresh="handleRefresh" />

        <n-layout-content class="content-area" :native-scrollbar="false">
          <div class="content-inner">
            <router-view v-slot="{ Component }">
              <keep-alive :include="tabStore.cachedNames()">
                <component :is="Component" v-if="showPage" :key="route.path" />
              </keep-alive>
            </router-view>
          </div>
        </n-layout-content>
      </n-layout>
    </n-layout>

    <n-drawer v-model:show="mobileMenuVisible" placement="left" :width="264">
      <n-drawer-content class="mobile-drawer" body-content-style="padding: 0; background: #111827;" closable>
        <template #header>
          <div class="drawer-brand">
            <div class="brand-mark">B</div>
            <strong>Backplane</strong>
          </div>
        </template>
        <n-menu
          inverted
          :indent="20"
          :options="menuOptions"
          :value="route.path"
          @update:value="handleMenuSelect"
        />
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { MenuOption } from 'naive-ui'
import {
  NAvatar,
  NButton,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
} from 'naive-ui'
import {
  ChevronBackOutline,
  ChevronDownOutline,
  ChevronForwardOutline,
  LogOutOutline,
  MenuOutline,
  PersonCircleOutline,
  SpeedometerOutline,
} from '@vicons/ionicons5'
import { useUserStore, type MenuNode } from '@/store/user'
import { useTabStore } from '@/store/tab'
import { renderIcon, resolveIcon } from '@/utils/icons'
import TabBar from './TabBar.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const tabStore = useTabStore()
const collapsed = ref(false)
const mobileMenuVisible = ref(false)
const isMobile = ref(false)
const showPage = ref(true)
const tabBarRef = ref()
let mediaQuery: MediaQueryList | null = null

const displayName = computed(
  () => userStore.userInfo?.nickname || userStore.userInfo?.username || '管理员',
)
const userInitial = computed(() => displayName.value.slice(0, 1).toUpperCase())

function toMenuOption(item: MenuNode): MenuOption {
  const hasChildren = Boolean(item.children?.length)
  return {
    key: item.path || `menu-${item.id}`,
    label: hasChildren
      ? item.name
      : () => h(RouterLink, { to: item.path }, { default: () => item.name }),
    icon: renderIcon(resolveIcon(item.icon)),
    children: hasChildren ? item.children!.map(toMenuOption) : undefined,
  }
}

const menuOptions = computed<MenuOption[]>(() => [
  {
    key: '/dashboard',
    label: () => h(RouterLink, { to: '/dashboard' }, { default: () => '控制台' }),
    icon: renderIcon(SpeedometerOutline),
  },
  ...userStore.menuTree.map(toMenuOption),
])

const userOptions = [
  {
    label: '个人信息',
    key: 'profile',
    icon: renderIcon(PersonCircleOutline),
    disabled: true,
  },
  { type: 'divider' as const, key: 'divider' },
  {
    label: '退出登录',
    key: 'logout',
    icon: renderIcon(LogOutOutline),
  },
]

async function handleRefresh() {
  showPage.value = false
  await nextTick()
  showPage.value = true
}

function handleMenuSelect(key: string) {
  if (key.startsWith('/')) router.push(key)
  mobileMenuVisible.value = false
}

function toggleNavigation() {
  if (isMobile.value) mobileMenuVisible.value = true
  else collapsed.value = !collapsed.value
}

function syncViewport(event?: MediaQueryListEvent) {
  isMobile.value = event?.matches ?? mediaQuery?.matches ?? false
  if (!isMobile.value) mobileMenuVisible.value = false
}

onMounted(async () => {
  mediaQuery = window.matchMedia('(max-width: 860px)')
  syncViewport()
  mediaQuery.addEventListener('change', syncViewport)

  if (userStore.token) {
    try {
      if (!userStore.userInfo) await userStore.fetchProfile()
      if (userStore.menuTree.length === 0) await userStore.fetchMenus()
      if (userStore.permissions.length === 0) await userStore.fetchPermissions()
    } catch {
      // Request errors and expired sessions are handled by the shared interceptor.
    }
  }
})

onUnmounted(() => mediaQuery?.removeEventListener('change', syncViewport))

async function handleCommand(command: string) {
  if (command !== 'logout') return
  await userStore.logout()
  tabStore.closeAll()
  router.push('/login')
}
</script>

<style scoped>
.app-shell,
.app-layout,
.workspace {
  height: 100%;
}

.sidebar {
  background: #111827;
}

.sidebar :deep(.n-layout-sider-scroll-container) {
  display: flex;
  flex-direction: column;
}

.sidebar :deep(.n-menu) {
  flex: 1;
  padding: 10px 8px 20px;
}

.sidebar :deep(.n-menu--collapsed) {
  padding-right: 0;
  padding-left: 0;
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  height: 68px;
  padding: 0 18px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  color: #fff;
  overflow: hidden;
}

.brand.compact {
  justify-content: center;
  padding: 0;
}

.brand-mark {
  display: grid;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 7px;
  background: #14b8a6;
  color: #062c2a;
  font-size: 17px;
  font-weight: 800;
}

.brand-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.2;
  white-space: nowrap;
}

.brand-copy strong {
  font-size: 16px;
}

.brand-copy span {
  margin-top: 3px;
  color: #94a3b8;
  font-size: 11px;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 68px;
  padding: 0 20px;
  background: #fff;
}

.topbar-left,
.page-context,
.user-trigger,
.drawer-brand {
  display: flex;
  align-items: center;
}

.topbar-left {
  gap: 12px;
  min-width: 0;
}

.page-context {
  align-items: flex-start;
  flex-direction: column;
  min-width: 0;
}

.page-context strong {
  overflow: hidden;
  max-width: 260px;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.context-label {
  color: #98a2b3;
  font-size: 11px;
  line-height: 1.2;
}

.user-trigger {
  gap: 8px;
  max-width: 220px;
  padding: 5px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #344054;
  cursor: pointer;
}

.user-trigger:hover {
  background: #f2f4f7;
}

.user-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content-area {
  height: calc(100vh - 109px);
  background: #f4f6f8;
}

.content-inner {
  min-width: 0;
  padding: 20px;
}

.drawer-brand {
  gap: 10px;
  color: #fff;
}

.mobile-drawer :deep(.n-drawer-header) {
  border-bottom-color: rgba(255, 255, 255, 0.08);
  background: #111827;
}

.mobile-drawer :deep(.n-drawer-header__close) {
  color: #cbd5e1;
}

@media (max-width: 860px) {
  .topbar {
    height: 60px;
    padding: 0 12px;
  }

  .content-area {
    height: calc(100vh - 101px);
  }

  .content-inner {
    padding: 12px;
  }

  .context-label,
  .user-name {
    display: none;
  }
}
</style>
