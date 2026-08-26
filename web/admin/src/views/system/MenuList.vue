<template>
  <div class="page-shell">
    <n-card class="page-panel" :bordered="true">
      <template #header>
        <div class="page-toolbar">
          <div>
            <h1 class="page-title">菜单管理</h1>
            <span class="muted-text">维护目录、页面、按钮权限与路由映射</span>
          </div>
          <n-button v-if="canAdd" type="primary" @click="openDialog()">
            <template #icon><n-icon :component="AddOutline" /></template>
            新增菜单
          </n-button>
        </div>
      </template>

      <n-data-table
        striped
        default-expand-all
        :columns="columns"
        :data="treeData"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :scroll-x="1260"
      />
    </n-card>

    <n-modal
      v-model:show="dialogVisible"
      preset="card"
      :title="form.id ? '编辑菜单' : '新增菜单'"
      :style="modalStyle"
      :mask-closable="false"
    >
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="86">
        <n-form-item label="上级菜单">
          <n-tree-select
            v-model:value="form.parent_id"
            clearable
            default-expand-all
            :options="parentOptions"
            key-field="id"
            label-field="name"
            children-field="children"
            placeholder="顶级菜单"
          />
        </n-form-item>
        <n-form-item label="菜单类型">
          <n-radio-group v-model:value="form.type">
            <n-space>
              <n-radio :value="0">目录</n-radio>
              <n-radio :value="1">菜单</n-radio>
              <n-radio :value="2">按钮</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="菜单名称" path="name">
          <n-input v-model:value="form.name" placeholder="显示名称" />
        </n-form-item>
        <n-form-item v-if="form.type !== 2" label="路由路径">
          <n-input v-model:value="form.path" placeholder="如：/system/user" />
        </n-form-item>
        <n-form-item v-if="form.type === 1" label="组件路径">
          <n-input v-model:value="form.component" placeholder="如：system/user/index" />
        </n-form-item>
        <n-form-item v-if="form.type !== 2" label="图标">
          <n-input v-model:value="form.icon" placeholder="后端图标标识，如 User" />
        </n-form-item>
        <n-form-item v-if="form.type === 2" label="权限标识">
          <n-input v-model:value="form.permission" placeholder="如：system:user:add" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="form.sort" :min="0" />
        </n-form-item>
        <n-grid cols="1 s:2" responsive="screen" :x-gap="16">
          <n-form-item-gi label="显示状态">
            <n-radio-group v-model:value="form.visible">
              <n-space>
                <n-radio :value="1">显示</n-radio>
                <n-radio :value="0">隐藏</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item-gi>
          <n-form-item-gi label="菜单状态">
            <n-radio-group v-model:value="form.status">
              <n-space>
                <n-radio :value="1">正常</n-radio>
                <n-radio :value="0">禁用</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item-gi>
        </n-grid>
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
  NForm,
  NFormItem,
  NFormItemGi,
  NGrid,
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NRadio,
  NRadioGroup,
  NSpace,
  NTag,
  NTreeSelect,
  type DataTableColumns,
  type FormInst,
  type FormRules,
  type TreeSelectOption,
} from 'naive-ui'
import { AddOutline, CreateOutline, TrashOutline } from '@vicons/ionicons5'
import { createMenu, deleteMenu, getMenuTree, updateMenu } from '@/api/menu'
import TableActionButton from '@/components/TableActionButton.vue'
import { useUserStore } from '@/store/user'
import { resolveIcon } from '@/utils/icons'
import { dialog, message } from '@/utils/ui'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:menu:add'))
const canEdit = computed(() => userStore.hasPermission('system:menu:edit'))
const canDelete = computed(() => userStore.hasPermission('system:menu:delete'))
const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInst | null>(null)
const treeData = ref<any[]>([])
const modalStyle = { width: 'min(620px, calc(100vw - 32px))' }

const parentOptions = computed<TreeSelectOption[]>(() => [
  { id: 0, name: '顶级菜单', children: treeData.value },
])
const defaultForm = {
  id: 0,
  parent_id: 0,
  name: '',
  path: '',
  component: '',
  icon: '',
  sort: 0,
  type: 1,
  permission: '',
  visible: 1,
  status: 1,
}
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  name: { required: true, message: '请输入菜单名称', trigger: ['input', 'blur'] },
}

function typeTag(row: any) {
  const types: Record<number, { label: string; type: 'warning' | 'info' | 'default' }> = {
    0: { label: '目录', type: 'warning' },
    1: { label: '菜单', type: 'info' },
    2: { label: '按钮', type: 'default' },
  }
  const item = types[row.type] || types[2]
  return h(NTag, { size: 'small', bordered: false, type: item.type }, { default: () => item.label })
}

const columns = computed<DataTableColumns<any>>(() => {
  const items: DataTableColumns<any> = [
    { title: '菜单名称', key: 'name', width: 210 },
    {
      title: '图标',
      key: 'icon',
      width: 70,
      align: 'center',
      render: (row) => row.icon ? h(NIcon, { component: resolveIcon(row.icon), size: 18 }) : '-',
    },
    {
      title: '路由路径',
      key: 'path',
      minWidth: 170,
      ellipsis: { tooltip: true },
      render: (row) => row.path ? h('span', { class: 'code-text' }, row.path) : '-',
    },
    {
      title: '组件路径',
      key: 'component',
      minWidth: 170,
      ellipsis: { tooltip: true },
      render: (row) => row.component ? h('span', { class: 'code-text' }, row.component) : '-',
    },
    {
      title: '权限标识',
      key: 'permission',
      width: 180,
      ellipsis: { tooltip: true },
      render: (row) => row.permission ? h('span', { class: 'code-text' }, row.permission) : '-',
    },
    { title: '类型', key: 'type', width: 82, render: typeTag },
    { title: '排序', key: 'sort', width: 72 },
    {
      title: '状态',
      key: 'status',
      width: 84,
      render: (row) =>
        h(NTag, { size: 'small', bordered: false, type: row.status === 1 ? 'success' : 'error' }, {
          default: () => (row.status === 1 ? '正常' : '禁用'),
        }),
    },
  ]

  if (canAdd.value || canEdit.value || canDelete.value) {
    items.push({
      title: '操作',
      key: 'actions',
      width: 128,
      fixed: 'right',
      render: (row) =>
        h('div', { class: 'table-actions' }, [
            canAdd.value
              ? h(TableActionButton, {
                  label: '新增子项',
                  icon: AddOutline,
                  type: 'primary',
                  onClick: () => openDialog(undefined, row.id),
                })
              : null,
            canEdit.value
              ? h(TableActionButton, {
                  label: '编辑',
                  icon: CreateOutline,
                  onClick: () => openDialog(row),
                })
              : null,
            canDelete.value
              ? h(TableActionButton, {
                  label: '删除',
                  icon: TrashOutline,
                  type: 'error',
                  onClick: () => confirmDelete(row.id),
                })
              : null,
          ]),
    })
  }
  return items
})

async function fetchData() {
  loading.value = true
  try {
    const res: any = await getMenuTree()
    treeData.value = res.data || []
  } finally {
    loading.value = false
  }
}

function openDialog(row?: any, parentId?: number) {
  Object.assign(form, defaultForm)
  if (row) {
    Object.assign(form, {
      id: row.id,
      parent_id: row.parent_id,
      name: row.name,
      path: row.path,
      component: row.component,
      icon: row.icon,
      sort: row.sort,
      type: row.type,
      permission: row.permission,
      visible: row.visible,
      status: row.status,
    })
  } else if (parentId !== undefined) {
    form.parent_id = parentId
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitLoading.value = true
  try {
    if (form.id) await updateMenu(form.id, form)
    else await createMenu(form)
    message.success('操作成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

function confirmDelete(id: number) {
  dialog.warning({
    title: '删除菜单',
    content: '确认删除该菜单项？请先确保没有依赖的子项。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteMenu(id)
      message.success('删除成功')
      fetchData()
    },
  })
}

onMounted(fetchData)
</script>

<style scoped>
.muted-text {
  display: block;
  margin-top: 3px;
  font-size: 12px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
