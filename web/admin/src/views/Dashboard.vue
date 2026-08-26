<template>
  <div class="dashboard page-shell">
    <section class="welcome-band">
      <div>
        <span class="section-label">OVERVIEW</span>
        <h1>欢迎回来，{{ displayName }}</h1>
        <p>系统运行概览与常用管理入口</p>
      </div>
      <div class="system-status">
        <span class="status-dot" />
        服务运行正常
      </div>
    </section>

    <n-grid responsive="screen" cols="1 s:2 l:4" :x-gap="14" :y-gap="14">
      <n-grid-item v-for="item in stats" :key="item.label">
        <n-card class="stat-card" :bordered="true">
          <div class="stat-topline">
            <span>{{ item.label }}</span>
            <n-icon :component="item.icon" :color="item.color" size="20" />
          </div>
          <strong>{{ item.value }}</strong>
          <span class="stat-note">{{ item.note }}</span>
        </n-card>
      </n-grid-item>
    </n-grid>

    <section class="dashboard-grid">
      <n-card title="系统能力" class="page-panel capability-panel">
        <div class="capability-list">
          <div v-for="item in capabilities" :key="item.title" class="capability-item">
            <div class="capability-icon">
              <n-icon :component="item.icon" size="18" />
            </div>
            <div>
              <strong>{{ item.title }}</strong>
              <p>{{ item.description }}</p>
            </div>
          </div>
        </div>
      </n-card>

      <n-card title="技术栈" class="page-panel stack-panel">
        <n-descriptions label-placement="left" :column="1">
          <n-descriptions-item label="后端">Go / Gin / GORM</n-descriptions-item>
          <n-descriptions-item label="权限">Casbin RBAC + JWT</n-descriptions-item>
          <n-descriptions-item label="前端">Vue 3 + Naive UI</n-descriptions-item>
          <n-descriptions-item label="任务">Asynq + Redis</n-descriptions-item>
        </n-descriptions>
      </n-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NDescriptions, NDescriptionsItem, NGrid, NGridItem, NIcon } from 'naive-ui'
import {
  CodeSlashOutline,
  DocumentTextOutline,
  KeyOutline,
  MenuOutline,
  PeopleOutline,
  PersonOutline,
  SettingsOutline,
  ShieldCheckmarkOutline,
} from '@vicons/ionicons5'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()
const displayName = computed(
  () => userStore.userInfo?.nickname || userStore.userInfo?.username || '管理员',
)

const stats = [
  { label: '用户总数', value: '--', note: '系统账号', icon: PersonOutline, color: '#2563eb' },
  { label: '角色数量', value: '--', note: '权限角色', icon: PeopleOutline, color: '#0f766e' },
  { label: '菜单数量', value: '--', note: '导航与按钮', icon: MenuOutline, color: '#b45309' },
  { label: 'API 数量', value: '--', note: '受控接口', icon: CodeSlashOutline, color: '#7c3aed' },
]

const capabilities = [
  { title: '身份与权限', description: 'JWT 会话与按钮级 RBAC 权限控制', icon: ShieldCheckmarkOutline },
  { title: '系统配置', description: '数据库持久化、Redis 缓存、运行时生效', icon: SettingsOutline },
  { title: '操作审计', description: '记录管理操作、响应结果与调用耗时', icon: DocumentTextOutline },
  { title: '安全边界', description: '路由、菜单、接口三层权限映射', icon: KeyOutline },
]
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.welcome-band {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  min-height: 128px;
  padding: 26px 28px;
  border: 1px solid #dce5e4;
  border-left: 4px solid #0f766e;
  border-radius: 8px;
  background: #fff;
}

.section-label {
  color: #0f766e;
  font-size: 11px;
  font-weight: 700;
}

.welcome-band h1 {
  margin: 7px 0 3px;
  font-size: 24px;
  line-height: 1.35;
}

.welcome-band p {
  margin: 0;
  color: #667085;
}

.system-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #475467;
  font-size: 13px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #16a34a;
  box-shadow: 0 0 0 3px #dcfce7;
}

.stat-card {
  min-height: 138px;
}

.stat-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #667085;
}

.stat-card strong {
  display: block;
  margin: 12px 0 3px;
  font-size: 30px;
  line-height: 1;
}

.stat-note {
  color: #98a2b3;
  font-size: 12px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(280px, 1fr);
  gap: 16px;
}

.capability-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px 24px;
}

.capability-item {
  display: flex;
  gap: 12px;
  padding: 15px 0;
  border-bottom: 1px solid #eaecf0;
}

.capability-icon {
  display: grid;
  flex: 0 0 34px;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 6px;
  background: #ecfdf5;
  color: #0f766e;
}

.capability-item strong {
  font-size: 14px;
}

.capability-item p {
  margin: 3px 0 0;
  color: #667085;
  font-size: 12px;
  line-height: 1.55;
}

@media (max-width: 900px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .welcome-band {
    align-items: flex-start;
    flex-direction: column;
    gap: 18px;
    padding: 22px;
  }

  .capability-list {
    grid-template-columns: 1fr;
  }
}
</style>
