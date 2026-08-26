<template>
  <div class="page-shell">
    <n-card class="page-panel" :bordered="true">
      <template #header>
        <div class="page-toolbar">
          <div>
            <h1 class="page-title">系统配置</h1>
            <span class="muted-text">运行时配置保存后立即同步到共享缓存</span>
          </div>
          <n-space>
            <n-button v-if="canAdd" @click="openCreate">
              <template #icon><n-icon :component="AddOutline" /></template>
              新增配置
            </n-button>
            <n-button v-if="canEdit" :loading="refreshing" @click="handleRefreshAll">
              <template #icon><n-icon :component="RefreshOutline" /></template>
              刷新缓存
            </n-button>
            <n-button v-if="canEdit" type="primary" :loading="saving" @click="handleSave">
              <template #icon><n-icon :component="SaveOutline" /></template>
              保存配置
            </n-button>
          </n-space>
        </div>
      </template>

      <n-spin :show="loading">
        <n-tabs v-if="groups.length" v-model:value="activeGroup" type="line" animated>
          <n-tab-pane v-for="group in groups" :key="group" :name="group" :tab="group">
            <n-data-table
              :columns="columns"
              :data="grouped[group]"
              :row-key="(row: any) => row.id"
              :scroll-x="980"
            />
          </n-tab-pane>
        </n-tabs>
        <n-empty v-else description="暂无配置" />
      </n-spin>
    </n-card>

    <n-modal
      v-model:show="dialogVisible"
      preset="card"
      :title="dialogMode === 'edit' ? '编辑配置' : '新增配置'"
      :style="modalStyle"
      :mask-closable="false"
    >
      <n-form ref="formRef" :model="form" :rules="formRules" label-placement="left" label-width="76">
        <n-form-item label="分组">
          <n-input v-model:value="form.group" placeholder="如：站点" />
        </n-form-item>
        <n-form-item label="键" path="key">
          <n-input v-model:value="form.key" :disabled="dialogMode === 'edit'" placeholder="如：site.name" />
        </n-form-item>
        <n-form-item label="名称" path="name">
          <n-input v-model:value="form.name" placeholder="后台显示名称" />
        </n-form-item>
        <n-form-item label="类型">
          <n-select v-model:value="form.type" :options="typeOptions" />
        </n-form-item>

        <n-form-item v-if="form.type === 'select'" label="选项">
          <div class="option-editor">
            <div v-for="(option, index) in optionRows" :key="index" class="option-row">
              <n-input v-model:value="option.label" placeholder="显示文字" />
              <n-input v-model:value="option.value" placeholder="存储值" />
              <n-button quaternary circle type="error" aria-label="删除选项" @click="optionRows.splice(index, 1)">
                <template #icon><n-icon :component="TrashOutline" /></template>
              </n-button>
            </div>
            <n-button dashed block @click="optionRows.push({ label: '', value: '' })">
              <template #icon><n-icon :component="AddOutline" /></template>
              添加选项
            </n-button>
          </div>
        </n-form-item>

        <n-form-item label="值">
          <n-switch
            v-if="form.type === 'bool'"
            v-model:value="form.value"
            checked-value="true"
            unchecked-value="false"
          />
          <n-select
            v-else-if="form.type === 'select'"
            v-model:value="form.value"
            :options="validOptionRows"
            placeholder="请选择默认值"
          />
          <n-input
            v-else-if="form.type === 'text' || form.type === 'json'"
            v-model:value="form.value"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 8 }"
          />
          <n-input v-else v-model:value="form.value" :placeholder="numericPlaceholder(form.type)" />
        </n-form-item>

        <n-grid cols="1 s:2" responsive="screen" :x-gap="16">
          <n-form-item-gi label="公开读取">
            <n-switch v-model:value="form.is_public" />
          </n-form-item-gi>
          <n-form-item-gi label="排序">
            <n-input-number v-model:value="form.sort" :min="0" />
          </n-form-item-gi>
        </n-grid>
        <n-form-item label="备注">
          <n-input v-model:value="form.remark" placeholder="配置用途或取值说明" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-actions">
          <n-button @click="dialogVisible = false">取消</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NButton,
  NCard,
  NDataTable,
  NEmpty,
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { AddOutline, CreateOutline, RefreshOutline, SaveOutline, TrashOutline } from '@vicons/ionicons5'
import {
  batchUpdateConfig,
  createConfig,
  deleteConfig,
  getConfigList,
  refreshConfig,
  updateConfig,
} from '@/api/config'
import TableActionButton from '@/components/TableActionButton.vue'
import { useUserStore } from '@/store/user'
import { dialog, message } from '@/utils/ui'

const userStore = useUserStore()
const canEdit = computed(() => userStore.hasPermission('system:config:edit'))
const canAdd = computed(() => userStore.hasPermission('system:config:add'))
const canDelete = computed(() => userStore.hasPermission('system:config:delete'))
const loading = ref(false)
const saving = ref(false)
const refreshing = ref(false)
const submitLoading = ref(false)
const configs = ref<any[]>([])
const activeGroup = ref('')
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref(0)
const formRef = ref<FormInst | null>(null)
const optionRows = ref<Array<{ label: string; value: string }>>([])
const modalStyle = { width: 'min(600px, calc(100vw - 32px))' }
const types = ['string', 'int', 'float', 'bool', 'text', 'json', 'select']
const typeOptions = types.map((type) => ({ label: type, value: type }))

const defaultForm = {
  group: '',
  key: '',
  name: '',
  type: 'string',
  value: '',
  is_public: false,
  sort: 0,
  remark: '',
}
const form = reactive({ ...defaultForm })
const formRules: FormRules = {
  key: { required: true, message: '请输入配置键', trigger: ['input', 'blur'] },
  name: { required: true, message: '请输入名称', trigger: ['input', 'blur'] },
}

const grouped = computed<Record<string, any[]>>(() => {
  const result: Record<string, any[]> = {}
  for (const config of configs.value) {
    const group = config.group || '其它'
    if (!result[group]) result[group] = []
    result[group].push(config)
  }
  return result
})
const groups = computed(() => Object.keys(grouped.value))
const validOptionRows = computed(() =>
  optionRows.value
    .filter((option) => option.value !== '')
    .map((option) => ({ label: option.label || option.value, value: option.value })),
)

function renderConfigValue(row: any) {
  if (row.type === 'bool') {
    return h(NSwitch, {
      value: row.value,
      checkedValue: 'true',
      uncheckedValue: 'false',
      'onUpdate:value': (value: string) => { row.value = value },
    })
  }
  if (row.type === 'select') {
    return h(NSelect, {
      value: row.value,
      options: parseOptions(row.options),
      style: { width: '300px' },
      'onUpdate:value': (value: string) => { row.value = value },
    })
  }
  if (row.type === 'text' || row.type === 'json') {
    return h(NInput, {
      value: row.value,
      type: 'textarea',
      autosize: { minRows: 2, maxRows: 6 },
      'onUpdate:value': (value: string) => { row.value = value },
    })
  }
  return h(NInput, {
    value: row.value,
    style: { width: row.type === 'int' || row.type === 'float' ? '180px' : '360px' },
    placeholder: numericPlaceholder(row.type),
    'onUpdate:value': (value: string) => { row.value = value },
  })
}

const columns = computed<DataTableColumns<any>>(() => [
  {
    title: '名称',
    key: 'name',
    width: 210,
    render: (row) => h('div', [
      h('div', { class: 'config-name' }, row.name || row.key),
      row.remark ? h('div', { class: 'config-remark' }, row.remark) : null,
    ]),
  },
  {
    title: '键',
    key: 'key',
    width: 230,
    render: (row) => h(NTag, { size: 'small', bordered: false }, {
      default: () => h('span', { class: 'code-text' }, row.key),
    }),
  },
  { title: '值', key: 'value', minWidth: 390, render: renderConfigValue },
  {
    title: '操作',
    key: 'actions',
    width: 128,
    fixed: 'right',
    render: (row) => h('div', { class: 'table-actions' }, [
        canEdit.value
          ? h(TableActionButton, {
              label: '编辑',
              icon: CreateOutline,
              onClick: () => openEdit(row),
            })
          : null,
        canEdit.value
          ? h(TableActionButton, {
              label: '刷新',
              icon: RefreshOutline,
              type: 'primary',
              onClick: () => handleRefreshKey(row.key),
            })
          : null,
        canDelete.value && !row.builtin
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
    const res: any = await getConfigList()
    configs.value = res.data || []
    if ((!activeGroup.value || !groups.value.includes(activeGroup.value)) && groups.value.length) {
      activeGroup.value = groups.value[0]
    }
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    const items = configs.value.map((config) => ({ key: config.key, value: String(config.value ?? '') }))
    await batchUpdateConfig(items)
    message.success('配置已保存')
    fetchData()
  } finally {
    saving.value = false
  }
}

async function handleRefreshAll() {
  refreshing.value = true
  try {
    await refreshConfig()
    message.success('全部缓存已刷新')
  } finally {
    refreshing.value = false
  }
}

async function handleRefreshKey(key: string) {
  await refreshConfig(key)
  message.success(`已刷新：${key}`)
}

function confirmDelete(id: number) {
  dialog.warning({
    title: '删除配置',
    content: '确认删除该自定义配置？',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteConfig(id)
      message.success('删除成功')
      fetchData()
    },
  })
}

function parseOptions(options: string): Array<{ label: string; value: string }> {
  if (!options) return []
  try {
    const parsed = JSON.parse(options)
    return parsed.map((option: any) =>
      typeof option === 'object'
        ? { label: option.label ?? option.value, value: String(option.value) }
        : { label: String(option), value: String(option) },
    )
  } catch {
    return []
  }
}

function numericPlaceholder(type: string) {
  return type === 'int' || type === 'float' ? '请输入数字' : ''
}

function openCreate() {
  dialogMode.value = 'create'
  editingId.value = 0
  Object.assign(form, defaultForm)
  optionRows.value = []
  dialogVisible.value = true
}

function openEdit(row: any) {
  dialogMode.value = 'edit'
  editingId.value = row.id
  Object.assign(form, {
    group: row.group,
    key: row.key,
    name: row.name,
    type: row.type || 'string',
    value: row.value,
    is_public: row.is_public,
    sort: row.sort,
    remark: row.remark,
  })
  optionRows.value = parseOptions(row.options)
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  let options = ''
  if (form.type === 'select') {
    if (validOptionRows.value.length === 0) {
      message.warning('请至少添加一个选项')
      return
    }
    options = JSON.stringify(validOptionRows.value)
  }

  submitLoading.value = true
  try {
    const payload = { ...form, options }
    if (dialogMode.value === 'edit') await updateConfig(editingId.value, payload)
    else await createConfig(payload)
    message.success('配置已保存')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.muted-text {
  display: block;
  margin-top: 3px;
  font-size: 12px;
}

.config-name {
  font-weight: 600;
}

.config-remark {
  margin-top: 3px;
  color: #98a2b3;
  font-size: 12px;
}

.option-editor {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 8px;
}

.option-row {
  display: grid;
  grid-template-columns: 1fr 1fr 34px;
  gap: 8px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

@media (max-width: 640px) {
  .option-row {
    grid-template-columns: 1fr 34px;
  }

  .option-row :deep(.n-input):nth-child(2) {
    grid-column: 1;
  }
}
</style>
