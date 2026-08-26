<template>
  <div class="page-shell">
    <n-card class="page-panel" :bordered="true">
      <template #header>
        <div class="page-toolbar">
          <div>
            <h1 class="page-title">角色管理</h1>
            <span class="muted-text">配置角色、状态与菜单权限范围</span>
          </div>
          <n-button v-if="canAdd" type="primary" @click="openDialog()">
            <template #icon><n-icon :component="AddOutline" /></template>
            新增角色
          </n-button>
        </div>
      </template>

      <n-data-table
        remote
        striped
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :scroll-x="900"
      />
      <div class="table-footer">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[10, 20, 50]"
          show-size-picker
          :prefix="({ itemCount }) => `共 ${itemCount} 条`"
          @update:page="fetchData"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </n-card>

    <n-modal
      v-model:show="dialogVisible"
      preset="card"
      :title="form.id ? '编辑角色' : '新增角色'"
      :style="modalStyle"
      :mask-closable="false"
    >
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="84">
        <n-form-item label="角色名称" path="name">
          <n-input v-model:value="form.name" placeholder="如：内容运营" />
        </n-form-item>
        <n-form-item label="角色编码" path="code">
          <n-input v-model:value="form.code" :disabled="Boolean(form.id)" placeholder="如：operator" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="form.sort" :min="0" />
        </n-form-item>
        <n-form-item label="状态">
          <n-radio-group v-model:value="form.status">
            <n-space>
              <n-radio :value="1">正常</n-radio>
              <n-radio :value="0">禁用</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="备注">
          <n-input v-model:value="form.remark" type="textarea" :autosize="{ minRows: 3, maxRows: 6 }" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="modal-actions">
          <n-button @click="dialogVisible = false">取消</n-button>
          <n-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</n-button>
        </div>
      </template>
    </n-modal>

    <n-modal
      v-model:show="menuDialogVisible"
      preset="card"
      title="分配菜单权限"
      :style="modalStyle"
      :mask-closable="false"
    >
      <div class="permission-summary">
        已选择 <strong>{{ checkedMenuIds.length }}</strong> 项菜单与按钮权限
      </div>
      <div class="tree-panel">
        <n-tree
          block-line
          cascade
          checkable
          default-expand-all
          :data="menuTreeData"
          :checked-keys="checkedMenuIds"
          :key-field="'id'"
          :label-field="'name'"
          :children-field="'children'"
          check-strategy="all"
          @update:checked-keys="handleCheckedKeysUpdate"
        />
      </div>
      <template #footer>
        <div class="modal-actions">
          <n-button @click="menuDialogVisible = false">取消</n-button>
          <n-button type="primary" :loading="menuSubmitLoading" @click="handleMenuSubmit">保存权限</n-button>
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
  NIcon,
  NInput,
  NInputNumber,
  NModal,
  NPagination,
  NRadio,
  NRadioGroup,
  NSpace,
  NTag,
  NTree,
  type DataTableColumns,
  type FormInst,
  type FormRules,
  type TreeOption,
} from 'naive-ui'
import { AddOutline, CreateOutline, ShieldCheckmarkOutline, TrashOutline } from '@vicons/ionicons5'
import { createRole, deleteRole, getRoleById, getRoleList, setRoleMenus, updateRole } from '@/api/role'
import { getMenuTree } from '@/api/menu'
import TableActionButton from '@/components/TableActionButton.vue'
import { useUserStore } from '@/store/user'
import { dialog, message } from '@/utils/ui'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:role:add'))
const canEdit = computed(() => userStore.hasPermission('system:role:edit'))
const canDelete = computed(() => userStore.hasPermission('system:role:delete'))
const loading = ref(false)
const submitLoading = ref(false)
const menuSubmitLoading = ref(false)
const dialogVisible = ref(false)
const menuDialogVisible = ref(false)
const formRef = ref<FormInst | null>(null)
const tableData = ref<any[]>([])
const menuTreeData = ref<TreeOption[]>([])
const checkedMenuIds = ref<Array<string | number>>([])
const currentRoleId = ref(0)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const modalStyle = { width: 'min(540px, calc(100vw - 32px))' }

const defaultForm = { id: 0, name: '', code: '', sort: 0, status: 1, remark: '' }
const form = reactive({ ...defaultForm })
const rules: FormRules = {
  name: { required: true, message: '请输入角色名称', trigger: ['input', 'blur'] },
  code: { required: true, message: '请输入角色编码', trigger: ['input', 'blur'] },
}

const columns = computed<DataTableColumns<any>>(() => {
  const items: DataTableColumns<any> = [
    { title: 'ID', key: 'id', width: 72 },
    { title: '角色名称', key: 'name', width: 160 },
    {
      title: '角色编码',
      key: 'code',
      width: 170,
      render: (row) => h('span', { class: 'code-text' }, row.code),
    },
    { title: '排序', key: 'sort', width: 80 },
    {
      title: '状态',
      key: 'status',
      width: 86,
      render: (row) =>
        h(NTag, { size: 'small', bordered: false, type: row.status === 1 ? 'success' : 'error' }, {
          default: () => (row.status === 1 ? '正常' : '禁用'),
        }),
    },
    { title: '备注', key: 'remark', minWidth: 180, ellipsis: { tooltip: true } },
  ]

  if (canEdit.value || canDelete.value) {
    items.push({
      title: '操作',
      key: 'actions',
      width: 128,
      fixed: 'right',
      render: (row) =>
        h('div', { class: 'table-actions' }, [
            canEdit.value
              ? h(TableActionButton, {
                  label: '编辑',
                  icon: CreateOutline,
                  onClick: () => openDialog(row),
                })
              : null,
            canEdit.value
              ? h(TableActionButton, {
                  label: '菜单权限',
                  icon: ShieldCheckmarkOutline,
                  type: 'primary',
                  onClick: () => openMenuDialog(row),
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
    const res: any = await getRoleList({ page: page.value, page_size: pageSize.value })
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

function openDialog(row?: any) {
  Object.assign(form, defaultForm)
  if (row) {
    Object.assign(form, {
      id: row.id,
      name: row.name,
      code: row.code,
      sort: row.sort,
      status: row.status,
      remark: row.remark,
    })
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
    if (form.id) {
      await updateRole(form.id, {
        name: form.name,
        sort: form.sort,
        status: form.status,
        remark: form.remark,
      })
    } else {
      await createRole(form)
    }
    message.success('操作成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

function confirmDelete(id: number) {
  dialog.warning({
    title: '删除角色',
    content: '确认删除该角色？已关联账号可能失去相应权限。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteRole(id)
      message.success('删除成功')
      fetchData()
    },
  })
}

function collectAllIds(menus: any[]): number[] {
  const ids: number[] = []
  for (const menu of menus) {
    ids.push(menu.id)
    if (menu.children?.length) ids.push(...collectAllIds(menu.children))
  }
  return ids
}

async function openMenuDialog(row: any) {
  currentRoleId.value = row.id
  try {
    const [treeRes, roleRes]: any[] = await Promise.all([getMenuTree(), getRoleById(row.id)])
    menuTreeData.value = treeRes.data || []
    checkedMenuIds.value = collectAllIds(roleRes.data?.menus || [])
    menuDialogVisible.value = true
  } catch {
    // The shared interceptor displays request errors.
  }
}

function handleCheckedKeysUpdate(keys: Array<string | number>) {
  checkedMenuIds.value = keys
}

async function handleMenuSubmit() {
  menuSubmitLoading.value = true
  try {
    await setRoleMenus(currentRoleId.value, checkedMenuIds.value.map(Number))
    message.success('菜单权限已保存')
    menuDialogVisible.value = false
  } finally {
    menuSubmitLoading.value = false
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.permission-summary {
  margin-bottom: 12px;
  color: #667085;
  font-size: 13px;
}

.permission-summary strong {
  color: #0f766e;
}

.tree-panel {
  max-height: 440px;
  overflow: auto;
  padding: 10px;
  border: 1px solid #e4e7ec;
  border-radius: 6px;
  background: #fafbfc;
}
</style>
