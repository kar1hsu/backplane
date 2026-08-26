<template>
  <div class="page-shell">
    <n-card class="page-panel" :bordered="true">
      <template #header>
        <div class="page-toolbar">
          <div>
            <h1 class="page-title">用户管理</h1>
            <span class="muted-text">管理后台账号、角色与启用状态</span>
          </div>
          <n-button v-if="canAdd" type="primary" @click="openDialog()">
            <template #icon><n-icon :component="PersonAddOutline" /></template>
            新增用户
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
        :scroll-x="1040"
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
      :title="form.id ? '编辑用户' : '新增用户'"
      :style="modalStyle"
      :mask-closable="false"
    >
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="76">
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="form.username" :disabled="Boolean(form.id)" placeholder="登录用户名" />
        </n-form-item>
        <n-form-item label="密码" :path="form.id ? undefined : 'password'">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            :placeholder="form.id ? '留空则不修改' : '至少 6 位'"
          />
        </n-form-item>
        <n-form-item label="昵称">
          <n-input v-model:value="form.nickname" placeholder="显示名称" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="form.email" placeholder="name@example.com" />
        </n-form-item>
        <n-form-item label="手机号">
          <n-input v-model:value="form.phone" placeholder="手机号" />
        </n-form-item>
        <n-form-item label="角色">
          <n-select
            v-model:value="form.role_ids"
            multiple
            filterable
            :options="roleOptions"
            placeholder="请选择角色"
          />
        </n-form-item>
        <n-form-item label="状态">
          <n-radio-group v-model:value="form.status">
            <n-space>
              <n-radio :value="1">正常</n-radio>
              <n-radio :value="0">禁用</n-radio>
            </n-space>
          </n-radio-group>
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
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NPagination,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
  NTag,
  type DataTableColumns,
  type FormInst,
  type FormRules,
} from 'naive-ui'
import { CreateOutline, PersonAddOutline, TrashOutline } from '@vicons/ionicons5'
import { createUser, deleteUser, getUserList, updateUser } from '@/api/user'
import { getAllRoles } from '@/api/role'
import TableActionButton from '@/components/TableActionButton.vue'
import { useUserStore } from '@/store/user'
import { dialog, message } from '@/utils/ui'

const userStore = useUserStore()
const canAdd = computed(() => userStore.hasPermission('system:user:add'))
const canEdit = computed(() => userStore.hasPermission('system:user:edit'))
const canDelete = computed(() => userStore.hasPermission('system:user:delete'))
const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInst | null>(null)
const tableData = ref<any[]>([])
const allRoles = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const modalStyle = { width: 'min(520px, calc(100vw - 32px))' }

const defaultForm = {
  id: 0,
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  status: 1,
  role_ids: [] as number[],
}
const form = reactive({ ...defaultForm })
const roleOptions = computed(() => allRoles.value.map((role) => ({ label: role.name, value: role.id })))
const rules = computed<FormRules>(() => ({
  username: { required: true, message: '请输入用户名', trigger: ['input', 'blur'] },
  password: form.id
    ? []
    : [{ required: true, message: '请输入密码', trigger: ['input', 'blur'] }],
}))

const columns = computed<DataTableColumns<any>>(() => {
  const items: DataTableColumns<any> = [
    { title: 'ID', key: 'id', width: 72 },
    { title: '用户名', key: 'username', width: 130 },
    { title: '昵称', key: 'nickname', width: 130, render: (row) => row.nickname || '-' },
    { title: '邮箱', key: 'email', minWidth: 180, ellipsis: { tooltip: true } },
    { title: '手机号', key: 'phone', width: 140, render: (row) => row.phone || '-' },
    {
      title: '角色',
      key: 'roles',
      width: 190,
      render: (row) =>
        h(
          NSpace,
          { size: 4 },
          { default: () => (row.roles || []).map((role: any) => h(NTag, { size: 'small', bordered: false }, { default: () => role.name })) },
        ),
    },
    {
      title: '状态',
      key: 'status',
      width: 86,
      render: (row) =>
        h(
          NTag,
          { size: 'small', bordered: false, type: row.status === 1 ? 'success' : 'error' },
          { default: () => (row.status === 1 ? '正常' : '禁用') },
        ),
    },
  ]

  if (canEdit.value || canDelete.value) {
    items.push({
      title: '操作',
      key: 'actions',
      width: 96,
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
    const res: any = await getUserList({ page: page.value, page_size: pageSize.value })
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

async function fetchRoles() {
  try {
    const res: any = await getAllRoles()
    allRoles.value = res.data || []
  } catch {
    // The shared interceptor displays request errors.
  }
}

function openDialog(row?: any) {
  Object.assign(form, { ...defaultForm, role_ids: [] })
  if (row) {
    Object.assign(form, {
      id: row.id,
      username: row.username,
      nickname: row.nickname,
      email: row.email,
      phone: row.phone,
      status: row.status,
      role_ids: (row.roles || []).map((role: any) => role.id),
      password: '',
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
      const data: any = { ...form }
      if (!data.password) delete data.password
      await updateUser(form.id, data)
    } else {
      await createUser(form)
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
    title: '删除用户',
    content: '确认删除该用户？此操作无法撤销。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await deleteUser(id)
      message.success('删除成功')
      fetchData()
    },
  })
}

onMounted(() => {
  fetchData()
  fetchRoles()
})
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
