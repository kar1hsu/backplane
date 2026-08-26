<template>
  <div class="tab-bar">
    <div class="tab-list">
      <button
        v-for="tab in tabStore.tabs"
        :key="tab.path"
        type="button"
        class="tab-item"
        :class="{ active: tabStore.activeTab === tab.path }"
        @click="switchTab(tab.path)"
        @contextmenu.prevent="openContextMenu($event, tab)"
      >
        <span class="tab-dot" />
        <span class="tab-title">{{ tab.title }}</span>
        <n-icon
          v-if="!tab.affix"
          :component="CloseOutline"
          class="tab-close"
          @click.stop="closeTab(tab.path)"
        />
      </button>
    </div>

    <n-dropdown
      placement="bottom-start"
      trigger="manual"
      :x="contextX"
      :y="contextY"
      :options="contextOptions"
      :show="contextVisible"
      :on-clickoutside="hideContextMenu"
      @select="handleContextSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NDropdown, NIcon } from 'naive-ui'
import { CloseCircleOutline, CloseOutline, RefreshOutline } from '@vicons/ionicons5'
import { useTabStore, type TabItem } from '@/store/tab'
import { renderIcon } from '@/utils/icons'

const router = useRouter()
const tabStore = useTabStore()
const emit = defineEmits<{ refresh: [] }>()
const contextVisible = ref(false)
const contextX = ref(0)
const contextY = ref(0)
const contextTab = ref<TabItem | null>(null)

const contextOptions = computed(() => [
  { label: '刷新当前', key: 'refresh', icon: renderIcon(RefreshOutline) },
  {
    label: '关闭当前',
    key: 'close',
    icon: renderIcon(CloseOutline),
    disabled: Boolean(contextTab.value?.affix),
  },
  { label: '关闭其他', key: 'others', icon: renderIcon(CloseCircleOutline) },
  { label: '关闭所有', key: 'all', icon: renderIcon(CloseCircleOutline) },
])

function switchTab(path: string) {
  tabStore.activeTab = path
  router.push(path)
}

function closeTab(path: string) {
  const redirect = tabStore.removeTab(path)
  if (redirect) router.push(redirect)
}

function openContextMenu(event: MouseEvent, tab: TabItem) {
  contextTab.value = tab
  contextX.value = event.clientX
  contextY.value = event.clientY
  contextVisible.value = true
}

function hideContextMenu() {
  contextVisible.value = false
}

function handleContextSelect(key: string) {
  hideContextMenu()
  if (key === 'refresh') emit('refresh')
  if (key === 'close' && contextTab.value) closeTab(contextTab.value.path)
  if (key === 'others' && contextTab.value) {
    tabStore.closeOthers(contextTab.value.path)
    router.push(contextTab.value.path)
  }
  if (key === 'all') router.push(tabStore.closeAll())
}
</script>

<style scoped>
.tab-bar {
  display: flex;
  align-items: flex-end;
  height: 41px;
  padding: 0 14px;
  border-bottom: 1px solid #e4e7ec;
  background: #fff;
  user-select: none;
}

.tab-list {
  display: flex;
  flex: 1;
  gap: 2px;
  overflow-x: auto;
}

.tab-list::-webkit-scrollbar {
  height: 0;
}

.tab-item {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 92px;
  height: 40px;
  padding: 0 12px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: #667085;
  cursor: pointer;
  white-space: nowrap;
}

.tab-item:hover {
  background: #f8fafb;
  color: #344054;
}

.tab-item.active {
  border-bottom-color: #0f766e;
  color: #0f766e;
  font-weight: 600;
}

.tab-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #d0d5dd;
}

.tab-item.active .tab-dot {
  background: #14b8a6;
}

.tab-title {
  overflow: hidden;
  max-width: 120px;
  text-overflow: ellipsis;
}

.tab-close {
  flex: 0 0 auto;
  padding: 2px;
  border-radius: 50%;
}

.tab-close:hover {
  background: #fee2e2;
  color: #b91c1c;
}

@media (max-width: 720px) {
  .tab-bar {
    padding: 0 8px;
  }

  .tab-item {
    min-width: 82px;
    padding: 0 9px;
  }
}
</style>
