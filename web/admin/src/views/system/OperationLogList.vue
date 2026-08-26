<template>
  <div class="page-shell">
    <n-card class="page-panel" :bordered="true">
      <template #header>
        <div class="page-toolbar">
          <div>
            <h1 class="page-title">操作日志</h1>
            <span class="muted-text">查询管理操作、响应结果与调用耗时</span>
          </div>
          <n-button v-if="canClear" type="error" secondary @click="confirmClear">
            <template #icon><n-icon :component="TrashBinOutline" /></template>
            清空日志
          </n-button>
        </div>
      </template>

      <div class="filter-bar">
        <n-input v-model:value="query.username" clearable placeholder="操作人" @keyup.enter="handleSearch" />
        <n-input v-model:value="query.module" clearable placeholder="模块" @keyup.enter="handleSearch" />
        <n-select v-model:value="query.success" clearable :options="successOptions" placeholder="执行状态" />
        <n-input v-model:value="query.keyword" clearable placeholder="操作或路径" @keyup.enter="handleSearch" />
        <n-date-picker
          v-model:value="dateRange"
          type="datetimerange"
          clearable
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          class="date-filter"
        />
        <n-space>
          <n-button type="primary" @click="handleSearch">
            <template #icon><n-icon :component="SearchOutline" /></template>
            查询
          </n-button>
          <n-button @click="handleReset">
            <template #icon><n-icon :component="RefreshOutline" /></template>
            重置
          </n-button>
        </n-space>
      </div>

      <n-data-table
        remote
        striped
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :scroll-x="1320"
      />

      <div class="table-footer">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          :prefix="({ itemCount }) => `共 ${itemCount} 条`"
          @update:page="fetchData"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </n-card>

    <n-modal
      v-model:show="detailVisible"
      preset="card"
      title="日志详情"
      :style="detailModalStyle"
    >
      <n-descriptions v-if="detail" bordered label-placement="left" :column="2">
        <n-descriptions-item label="操作人">{{ detail.username }}（ID {{ detail.user_id }}）</n-descriptions-item>
        <n-descriptions-item label="角色">{{ detail.role_codes || '-' }}</n-descriptions-item>
        <n-descriptions-item label="模块">{{ detail.module }}</n-descriptions-item>
        <n-descriptions-item label="操作">{{ detail.action }}</n-descriptions-item>
        <n-descriptions-item label="方法">
          <n-tag size="small" :type="methodType(detail.method)" :bordered="false">{{ detail.method }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="结果">
          <n-tag size="small" :type="detail.success ? 'success' : 'error'" :bordered="false">
            {{ detail.success ? '成功' : '失败' }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="HTTP 状态">{{ detail.status }}</n-descriptions-item>
        <n-descriptions-item label="业务码">{{ detail.biz_code }}</n-descriptions-item>
        <n-descriptions-item label="IP">{{ detail.client_ip }}</n-descriptions-item>
        <n-descriptions-item label="耗时">{{ detail.latency_ms }} ms</n-descriptions-item>
        <n-descriptions-item label="路径" :span="2">
          <span class="code-text">{{ detail.method }} {{ detail.path }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="时间" :span="2">{{ formatTime(detail.created_at) }}</n-descriptions-item>
        <n-descriptions-item label="User-Agent" :span="2">{{ detail.user_agent || '-' }}</n-descriptions-item>
        <n-descriptions-item v-if="!detail.success" label="错误信息" :span="2">
          <span class="error-text">{{ detail.error_msg }}</span>
        </n-descriptions-item>
        <n-descriptions-item label="请求参数" :span="2">
          <span v-if="!hasParams(detail.req_params)" class="param-empty">无</span>
          <div v-else class="param-viewer">
            <div class="param-summary">{{ formatParamSummary(detail.req_params) }}</div>
            <n-collapse arrow-placement="right">
              <n-collapse-item title="查看完整请求" name="request">
                <pre class="param-pre">{{ formatParams(detail.req_params) }}</pre>
              </n-collapse-item>
            </n-collapse>
          </div>
        </n-descriptions-item>
        <n-descriptions-item label="响应参数" :span="2">
          <span v-if="!hasParams(detail.resp_params)" class="param-empty">无</span>
          <div v-else class="param-viewer">
            <div class="param-summary">{{ formatParamSummary(detail.resp_params) }}</div>
            <n-collapse arrow-placement="right">
              <n-collapse-item title="查看完整响应" name="response">
                <pre class="param-pre">{{ formatParams(detail.resp_params) }}</pre>
              </n-collapse-item>
            </n-collapse>
          </div>
        </n-descriptions-item>
      </n-descriptions>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NCard,
  NCollapse,
  NCollapseItem,
  NDataTable,
  NDatePicker,
  NDescriptions,
  NDescriptionsItem,
  NIcon,
  NInput,
  NModal,
  NPagination,
  NSelect,
  NSpace,
  NTag,
  type DataTableColumns,
} from 'naive-ui'
import { EyeOutline, RefreshOutline, SearchOutline, TrashBinOutline, TrashOutline } from '@vicons/ionicons5'
import {
  clearOperationLogs,
  deleteOperationLog,
  getOperationLogById,
  getOperationLogList,
} from '@/api/operationLog'
import TableActionButton from '@/components/TableActionButton.vue'
import { useUserStore } from '@/store/user'
import { dialog, message } from '@/utils/ui'

type TagType = 'success' | 'info' | 'warning' | 'error' | 'default'

const userStore = useUserStore()
const canDelete = computed(() => userStore.hasPermission('system:operlog:delete'))
const canClear = computed(() => userStore.hasPermission('system:operlog:clear'))
const loading = ref(false)
const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const query = reactive({ username: '', module: '', success: null as string | null, keyword: '' })
const dateRange = ref<[number, number] | null>(null)
const detailVisible = ref(false)
const detail = ref<any>(null)
const detailModalStyle = { width: 'min(780px, calc(100vw - 24px))' }
const successOptions = [
  { label: '成功', value: 'true' },
  { label: '失败', value: 'false' },
]

const columns = computed<DataTableColumns<any>>(() => [
  { title: 'ID', key: 'id', width: 68 },
  { title: '时间', key: 'created_at', width: 168, render: (row) => formatTime(row.created_at) },
  { title: '操作人', key: 'username', width: 110, render: (row) => row.username || '-' },
  { title: '模块', key: 'module', width: 110 },
  { title: '操作', key: 'action', width: 125, ellipsis: { tooltip: true } },
  {
    title: '方法',
    key: 'method',
    width: 86,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: methodType(row.method) }, {
      default: () => row.method,
    }),
  },
  {
    title: '路径',
    key: 'path',
    minWidth: 210,
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'code-text' }, row.path),
  },
  { title: 'IP', key: 'client_ip', width: 132 },
  {
    title: '结果',
    key: 'success',
    width: 80,
    render: (row) => h(NTag, { size: 'small', bordered: false, type: row.success ? 'success' : 'error' }, {
      default: () => (row.success ? '成功' : '失败'),
    }),
  },
  { title: '耗时', key: 'latency_ms', width: 88, render: (row) => `${row.latency_ms} ms` },
  {
    title: '操作',
    key: 'actions',
    width: canDelete.value ? 96 : 62,
    fixed: 'right',
    render: (row) => h('div', { class: 'table-actions' }, [
        h(TableActionButton, {
          label: '详情',
          icon: EyeOutline,
          type: 'primary',
          onClick: () => openDetail(row),
        }),
        canDelete.value
          ? h(TableActionButton, {
              label: '删除',
              icon: TrashOutline,
              type: 'error',
              onClick: () => confirmDelete(row.id),
            })
          : null,
      ]),
  },
])

async function fetchData() {
  loading.value = true
  try {
    const params: any = {
      page: page.value,
      page_size: pageSize.value,
      username: query.username || undefined,
      module: query.module || undefined,
      success: query.success || undefined,
      keyword: query.keyword || undefined,
      start_time: dateRange.value ? formatQueryTime(dateRange.value[0]) : undefined,
      end_time: dateRange.value ? formatQueryTime(dateRange.value[1]) : undefined,
    }
    const res: any = await getOperationLogList(params)
    tableData.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function handlePageSizeChange() {
  page.value = 1
  fetchData()
}

function handleSearch() {
  page.value = 1
  fetchData()
}

function handleReset() {
  query.username = ''
  query.module = ''
  query.success = null
  query.keyword = ''
  dateRange.value = null
  page.value = 1
  fetchData()
}

async function openDetail(row: any) {
  try {
    const res: any = await getOperationLogById(row.id)
    detail.value = res.data
    detailVisible.value = true
  } catch {
    // The shared interceptor displays request errors.
  }
}

function confirmDelete(id: number) {
  dialog.warning({
    title: '删除日志',
    content: '确认删除该条操作日志？',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteOperationLog(id)
      message.success('删除成功')
      fetchData()
    },
  })
}

function confirmClear() {
  dialog.error({
    title: '清空操作日志',
    content: '将永久删除全部操作日志，此操作无法恢复。',
    positiveText: '确认清空',
    negativeText: '取消',
    onPositiveClick: async () => {
      await clearOperationLogs()
      message.success('操作日志已清空')
      page.value = 1
      fetchData()
    },
  })
}

function methodType(method: string): TagType {
  const types: Record<string, TagType> = {
    GET: 'info',
    POST: 'success',
    PUT: 'warning',
    PATCH: 'warning',
    DELETE: 'error',
  }
  return types[method] || 'default'
}

function formatTime(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

function formatQueryTime(timestamp: number) {
  const date = new Date(timestamp)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function parseParams(raw: string) {
  if (!raw) return null
  try {
    const value = JSON.parse(raw)
    if (value.body && typeof value.body === 'string') {
      try {
        value.body = JSON.parse(value.body)
      } catch {
        // Keep non-JSON request bodies as text.
      }
    }
    return value
  } catch {
    return raw
  }
}

function isEmptyValue(value: any): boolean {
  if (value == null || value === '') return true
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.values(value).every(isEmptyValue)
  return false
}

function valueSize(value: any): string {
  if (value == null || value === '') return '空'
  if (Array.isArray(value)) return `${value.length} 项`
  if (typeof value === 'object') return `${Object.keys(value).length} 项`
  return '1 项'
}

function hasParams(raw: string) {
  return !isEmptyValue(parseParams(raw))
}

function formatParamSummary(raw: string) {
  const value = parseParams(raw)
  if (isEmptyValue(value)) return '无'
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return Object.entries(value)
      .filter(([, item]) => !isEmptyValue(item))
      .map(([key, item]) => `${key}: ${valueSize(item)}`)
      .join('，')
  }
  return valueSize(value)
}

function formatParams(raw: string) {
  const value = parseParams(raw)
  if (isEmptyValue(value)) return '-'
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

onMounted(fetchData)
</script>

<style scoped>
.muted-text {
  display: block;
  margin-top: 3px;
  font-size: 12px;
}

.filter-bar {
  display: grid;
  grid-template-columns: 130px 130px 120px 170px minmax(300px, 1fr) auto;
  gap: 10px;
  margin-bottom: 16px;
}

.date-filter {
  width: 100%;
}

.param-empty {
  color: #98a2b3;
}

.param-viewer {
  width: 100%;
  min-width: 0;
}

.param-summary {
  margin-bottom: 6px;
  color: #667085;
}

.param-pre {
  max-height: 280px;
  overflow: auto;
  margin: 0;
  padding: 12px;
  border: 1px solid #e4e7ec;
  border-radius: 6px;
  background: #f8fafc;
  color: #1f2937;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.error-text {
  color: #b91c1c;
}

@media (max-width: 1280px) {
  .filter-bar {
    grid-template-columns: repeat(4, minmax(120px, 1fr));
  }

  .date-filter {
    grid-column: span 3;
  }
}

@media (max-width: 760px) {
  .filter-bar {
    grid-template-columns: 1fr 1fr;
  }

  .date-filter {
    grid-column: 1 / -1;
  }
}

@media (max-width: 480px) {
  .filter-bar {
    grid-template-columns: 1fr;
  }

  .date-filter {
    grid-column: auto;
  }
}
</style>
