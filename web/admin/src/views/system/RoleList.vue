<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>角色管理</span>
          <el-button v-if="userStore.hasPermission('system:role:add')" type="primary" @click="openDialog()">新增角色</el-button>
        </div>
      </template>
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="角色名称" width="160" />
        <el-table-column prop="code" label="角色编码" width="160" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" />
        <el-table-column v-if="userStore.hasPermission('system:role:edit') || userStore.hasPermission('system:role:delete')" label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button v-if="userStore.hasPermission('system:role:edit')" link type="primary" @click="openDialog(row)">编辑</el-button>
            <el-button v-if="userStore.hasPermission('system:role:edit')" link type="primary" @click="openMenuDialog(row)">分配菜单</el-button>
            <el-popconfirm v-if="userStore.hasPermission('system:role:delete')" title="确认删除？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top: 16px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑角色' : '新增角色'" width="500px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="auto">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="角色编码" prop="code">
          <el-input v-model="form.code" :disabled="!!form.id" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">正常</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="menuDialogVisible" title="分配菜单权限" width="500px" destroy-on-close>
      <el-tree
        ref="menuTreeRef"
        :data="menuTreeData"
        show-checkbox
        check-strictly
        default-expand-all
        node-key="id"
        :default-checked-keys="checkedMenuIds"
        :props="{ label: 'name', children: 'children' }"
        @check="handleTreeCheck"
      />
      <template #footer>
        <el-button @click="menuDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="menuSubmitLoading" :disabled="!menuTreeData.length" @click="handleMenuSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { getRoleList, createRole, updateRole, deleteRole, setRoleMenus, getRoleById } from '@/api/role'
import { getMenuTree } from '@/api/menu'
import { useUserStore } from '@/store/user'

const userStore = useUserStore()

const loading = ref(false)
const submitLoading = ref(false)
const menuSubmitLoading = ref(false)
const dialogVisible = ref(false)
const menuDialogVisible = ref(false)
const formRef = ref<FormInstance>()
const menuTreeRef = ref<any>()
const tableData = ref<any[]>([])
const menuTreeData = ref<any[]>([])
const checkedMenuIds = ref<number[]>([])
const currentRoleId = ref(0)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

const defaultForm = { id: 0, name: '', code: '', sort: 0, status: 1, remark: '' }
const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }],
}

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

function openDialog(row?: any) {
  Object.assign(form, defaultForm)
  if (row) {
    Object.assign(form, { id: row.id, name: row.name, code: row.code, sort: row.sort, status: row.status, remark: row.remark })
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitLoading.value = true
  try {
    if (form.id) {
      await updateRole(form.id, { name: form.name, sort: form.sort, status: form.status, remark: form.remark })
    } else {
      await createRole(form)
    }
    ElMessage.success('操作成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id: number) {
  await deleteRole(id)
  ElMessage.success('删除成功')
  fetchData()
}

function collectAllIds(menus: any[]): number[] {
  const ids: number[] = []
  for (const m of menus) {
    ids.push(m.id)
    if (m.children?.length) ids.push(...collectAllIds(m.children))
  }
  return ids
}

// Build a parentId map from tree data for upward traversal
function buildParentMap(tree: any[], parentId: number | null = null, map: Map<number, number | null> = new Map()) {
  for (const node of tree) {
    map.set(node.id, parentId)
    if (node.children?.length) buildParentMap(node.children, node.id, map)
  }
  return map
}

// Build a children map for downward traversal
function buildChildrenMap(tree: any[], map: Map<number, number[]> = new Map()) {
  for (const node of tree) {
    const childIds = (node.children || []).map((c: any) => c.id)
    map.set(node.id, childIds)
    if (node.children?.length) buildChildrenMap(node.children, map)
  }
  return map
}

function getAllDescendants(id: number, childrenMap: Map<number, number[]>): number[] {
  const ids: number[] = []
  for (const childId of childrenMap.get(id) || []) {
    ids.push(childId)
    ids.push(...getAllDescendants(childId, childrenMap))
  }
  return ids
}

function handleTreeCheck(node: any, data: { checkedKeys: number[] }) {
  const tree = menuTreeRef.value
  if (!tree) return

  const isChecked = data.checkedKeys.includes(node.id)
  const parentMap = buildParentMap(menuTreeData.value)
  const childrenMap = buildChildrenMap(menuTreeData.value)

  if (isChecked) {
    // Check all ancestors
    let pid = parentMap.get(node.id)
    while (pid != null) {
      tree.setChecked(pid, true, false)
      pid = parentMap.get(pid)
    }
  } else {
    // Uncheck all descendants
    for (const descId of getAllDescendants(node.id, childrenMap)) {
      tree.setChecked(descId, false, false)
    }
  }
}

// Bumped on every open so a slow response for a previously clicked role
// cannot fill the dialog of the current one.
let menuDialogRequest = 0

async function openMenuDialog(row: any) {
  // Never show a stale or empty tree: confirming it would overwrite this
  // role's menus (and its API permissions) with the wrong set.
  const requestId = ++menuDialogRequest
  currentRoleId.value = row.id
  menuTreeData.value = []
  checkedMenuIds.value = []
  try {
    const [treeRes, roleRes]: any[] = await Promise.all([getMenuTree(), getRoleById(row.id)])
    if (requestId !== menuDialogRequest) return
    menuTreeData.value = treeRes.data || []
    checkedMenuIds.value = collectAllIds((roleRes.data?.menus || []) as any[])
    menuDialogVisible.value = true
  } catch {
    // request.ts has already shown the error; keep the dialog closed.
  }
}

async function handleMenuSubmit() {
  menuSubmitLoading.value = true
  try {
    const checked = menuTreeRef.value?.getCheckedKeys() || []
    await setRoleMenus(currentRoleId.value, checked as number[])
    ElMessage.success('菜单分配成功')
    menuDialogVisible.value = false
  } finally {
    menuSubmitLoading.value = false
  }
}

onMounted(() => fetchData())
</script>
