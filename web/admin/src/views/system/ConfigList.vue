<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>系统配置</span>
          <div>
            <el-button v-if="canAdd" @click="openCreate">新增配置</el-button>
            <el-button v-if="canEdit" :loading="refreshing" @click="handleRefreshAll">刷新全部缓存</el-button>
            <el-button v-if="canEdit" type="primary" :loading="saving" @click="handleSave">保存</el-button>
          </div>
        </div>
      </template>

      <el-tabs v-if="groups.length" v-model="activeGroup" v-loading="loading">
        <el-tab-pane v-for="g in groups" :key="g" :label="g" :name="g">
          <el-table :data="grouped[g]" row-key="id">
            <el-table-column label="名称" width="200">
              <template #default="{ row }">
                <div>{{ row.name || row.key }}</div>
                <div v-if="row.remark" style="color: #909399; font-size: 12px">{{ row.remark }}</div>
              </template>
            </el-table-column>
            <el-table-column label="键" width="210">
              <template #default="{ row }"><el-tag size="small" type="info">{{ row.key }}</el-tag></template>
            </el-table-column>
            <el-table-column label="值" min-width="360">
              <template #default="{ row }">
                <!-- 按类型给合理宽高：数字短、文本中等、select 中等，text/json 宽且高 -->
                <el-switch v-if="row.type === 'bool'" v-model="row.value" active-value="true" inactive-value="false" :disabled="!isEditable(row)" />
                <el-select v-else-if="row.type === 'select'" v-model="row.value" style="width: 320px" placeholder="请选择" :disabled="!isEditable(row)">
                  <el-option v-for="opt in parseOptions(row.options)" :key="opt.value" :label="opt.label" :value="opt.value" />
                </el-select>
                <el-input
                  v-else-if="row.type === 'json'"
                  v-model="row.value"
                  :disabled="!isEditable(row)"
                  type="textarea"
                  :autosize="{ minRows: 3, maxRows: 12 }"
                  style="width: 100%; font-family: monospace"
                />
                <el-input
                  v-else-if="row.type === 'text'"
                  v-model="row.value"
                  :disabled="!isEditable(row)"
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 6 }"
                  style="width: 100%"
                />
                <el-input
                  v-else-if="row.type === 'int' || row.type === 'float'"
                  v-model="row.value"
                  :disabled="!isEditable(row)"
                  style="width: 180px"
                  placeholder="数字"
                />
                <el-input v-else v-model="row.value" style="width: 360px" :disabled="!isEditable(row)" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }">
                <el-button v-if="canEdit" link type="primary" @click="openEdit(row)">编辑</el-button>
                <el-button v-if="canEdit" link type="primary" @click="handleRefreshKey(row.key)">刷新</el-button>
                <el-popconfirm
                  v-if="canDelete && !row.builtin"
                  title="确认删除该配置？"
                  @confirm="handleDelete(row.id)"
                >
                  <template #reference><el-button link type="danger">删除</el-button></template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
      <el-empty v-else description="暂无配置" />
    </el-card>

    <!-- 新增 / 编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogMode === 'edit' ? '编辑配置' : '新增配置'" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="90px">
        <el-form-item label="分组" prop="group"><el-input v-model="form.group" placeholder="如：站点" /></el-form-item>
        <el-form-item label="键" prop="key">
          <el-input v-model="form.key" placeholder="如：site.name" :disabled="dialogMode === 'edit'" />
        </el-form-item>
        <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width: 100%">
            <el-option v-for="t in types" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>

        <!-- select 选项编辑器：键值对，避免手写 JSON -->
        <el-form-item v-if="form.type === 'select'" label="选项">
          <div style="width: 100%">
            <div v-for="(opt, i) in optionRows" :key="i" style="display: flex; gap: 8px; margin-bottom: 6px">
              <el-input v-model="opt.label" placeholder="显示文字" />
              <el-input v-model="opt.value" placeholder="存储值" />
              <el-button link type="danger" @click="optionRows.splice(i, 1)">删除</el-button>
            </div>
            <el-button size="small" @click="optionRows.push({ label: '', value: '' })">+ 添加选项</el-button>
          </div>
        </el-form-item>

        <el-form-item label="值">
          <el-switch v-if="form.type === 'bool'" v-model="form.value" active-value="true" inactive-value="false" />
          <el-select v-else-if="form.type === 'select'" v-model="form.value" style="width: 100%" placeholder="默认值（从选项中选）">
            <el-option v-for="(o, i) in validOptionRows" :key="i" :label="o.label || o.value" :value="o.value" />
          </el-select>
          <el-input v-else-if="form.type === 'text' || form.type === 'json'" v-model="form.value" type="textarea" :rows="3" />
          <el-input v-else v-model="form.value" :placeholder="form.type === 'int' || form.type === 'float' ? '数字' : ''" />
        </el-form-item>

        <el-form-item label="公开读"><el-switch v-model="form.is_public" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getConfigList, batchUpdateConfig, createConfig, updateConfig, deleteConfig, refreshConfig } from '@/api/config'
import { useUserStore } from '@/store/user'

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

const types = ['string', 'int', 'float', 'bool', 'text', 'json', 'select']

const grouped = computed<Record<string, any[]>>(() => {
  const m: Record<string, any[]> = {}
  for (const c of configs.value) {
    const g = c.group || '其它'
    if (!m[g]) m[g] = []
    m[g].push(c)
  }
  return m
})
const groups = computed(() => Object.keys(grouped.value))

// Values as loaded, keyed by config key: "保存" submits only what changed, so
// it cannot overwrite settings another admin edited in the meantime.
const savedValues = new Map<string, string>()

function isEditable(row: any): boolean {
  return canEdit.value && row.editable !== false
}

// el-switch rewrites any value other than its active/inactive value to
// "false", so bool values stored as "1" are normalized to "true" first.
function normalizeValue(row: any): string {
  const value = String(row.value ?? '')
  if (row.type === 'bool') return value === 'true' || value === '1' ? 'true' : 'false'
  return value
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getConfigList()
    configs.value = (res.data || []).map((c: any) => ({ ...c, value: normalizeValue(c) }))
    savedValues.clear()
    for (const c of configs.value) savedValues.set(c.key, c.value)
    if ((!activeGroup.value || !groups.value.includes(activeGroup.value)) && groups.value.length) {
      activeGroup.value = groups.value[0]
    }
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const items = configs.value
    .map((c) => ({ key: c.key, value: String(c.value ?? '') }))
    .filter((item) => item.value !== savedValues.get(item.key))
  if (!items.length) {
    ElMessage.info('没有需要保存的修改')
    return
  }
  saving.value = true
  try {
    await batchUpdateConfig(items)
    ElMessage.success('保存成功')
    fetchData()
  } finally {
    saving.value = false
  }
}

async function handleRefreshAll() {
  refreshing.value = true
  try {
    await refreshConfig()
    ElMessage.success('已刷新全部缓存')
  } finally {
    refreshing.value = false
  }
}

async function handleRefreshKey(key: string) {
  await refreshConfig(key)
  ElMessage.success(`已刷新：${key}`)
}

async function handleDelete(id: number) {
  await deleteConfig(id)
  ElMessage.success('删除成功')
  fetchData()
}

function parseOptions(options: string): { label: string; value: string }[] {
  if (!options) return []
  try {
    const arr = JSON.parse(options)
    return arr.map((o: any) =>
      typeof o === 'object' ? { label: o.label ?? o.value, value: String(o.value) } : { label: String(o), value: String(o) },
    )
  } catch {
    return []
  }
}

// ── 新增 / 编辑弹窗 ──
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref(0)
const formRef = ref<FormInstance>()
const optionRows = ref<{ label: string; value: string }[]>([])
const defaultForm = { group: '', key: '', name: '', type: 'string', value: '', is_public: false, sort: 0, remark: '' }
const form = reactive({ ...defaultForm })
const formRules = {
  key: [{ required: true, message: '请输入配置键', trigger: 'blur' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
}

const validOptionRows = computed(() => optionRows.value.filter((o) => o.value !== ''))

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
  await formRef.value?.validate()

  let options = ''
  if (form.type === 'select') {
    const rows = optionRows.value.filter((o) => o.value !== '')
    if (rows.length === 0) {
      ElMessage.warning('请至少添加一个选项')
      return
    }
    options = JSON.stringify(rows.map((o) => ({ label: o.label || o.value, value: o.value })))
  }

  const payload = { ...form, options }
  submitLoading.value = true
  try {
    if (dialogMode.value === 'edit') {
      await updateConfig(editingId.value, payload)
    } else {
      await createConfig(payload)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

onMounted(fetchData)
</script>
