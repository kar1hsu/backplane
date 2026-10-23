<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>菜单管理</span>
          <el-button v-if="userStore.hasPermission('system:menu:add')" type="primary" @click="openDialog()">新增菜单</el-button>
        </div>
      </template>
      <el-table :data="treeData" v-loading="loading" row-key="id" :tree-props="{ children: 'children' }" stripe>
        <el-table-column prop="name" label="菜单名称" width="200" />
        <el-table-column prop="icon" label="图标" width="80">
          <template #default="{ row }">
            <el-icon v-if="resolveIcon(row.icon)"><component :is="resolveIcon(row.icon)" /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路由路径" />
        <el-table-column prop="component" label="组件路径" />
        <el-table-column prop="permission" label="权限标识" width="160" />
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.type === 0" type="warning">目录</el-tag>
            <el-tag v-else-if="row.type === 1">菜单</el-tag>
            <el-tag v-else type="info">按钮</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="70" />
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '正常' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button v-if="userStore.hasPermission('system:menu:add')" link type="primary" @click="openDialog(undefined, row.id)">新增子项</el-button>
            <el-button v-if="userStore.hasPermission('system:menu:edit')" link type="primary" @click="openDialog(row)">编辑</el-button>
            <el-popconfirm v-if="userStore.hasPermission('system:menu:delete')" title="确认删除？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑菜单' : '新增菜单'" width="580px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="上级菜单">
          <el-tree-select
            v-model="form.parent_id"
            :data="parentOptions"
            :props="{ label: 'name', value: 'id', children: 'children' }"
            check-strictly
            clearable
            :value-on-clear="0"
            placeholder="留空为顶级菜单"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="菜单类型">
          <el-radio-group v-model="form.type">
            <el-radio :value="0">目录</el-radio>
            <el-radio :value="1">菜单</el-radio>
            <el-radio :value="2">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="菜单名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="路由路径">
          <el-input v-model="form.path" />
        </el-form-item>
        <el-form-item label="组件路径" v-if="form.type === 1">
          <el-input v-model="form.component" />
        </el-form-item>
        <el-form-item label="图标">
          <el-input v-model="form.icon" />
        </el-form-item>
        <el-form-item label="权限标识" v-if="form.type === 2">
          <el-input v-model="form.permission" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" />
        </el-form-item>
        <el-row>
          <el-col :span="12">
            <el-form-item label="显示状态">
              <el-radio-group v-model="form.visible">
                <el-radio :value="1">显示</el-radio>
                <el-radio :value="0">隐藏</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="菜单状态">
              <el-radio-group v-model="form.status">
                <el-radio :value="1">正常</el-radio>
                <el-radio :value="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
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
import { getMenuTree, createMenu, updateMenu, deleteMenu } from '@/api/menu'
import { useUserStore } from '@/store/user'
import { resolveIcon } from '@/utils/icons'

const userStore = useUserStore()

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const treeData = ref<any[]>([])

// A menu cannot move under itself, one of its descendants or a button; the
// backend rejects those too, this just keeps them out of the picker.
function toParentOptions(nodes: any[], selfId: number, insideSelf = false): any[] {
  return nodes.map((node) => {
    const blocked = insideSelf || (selfId !== 0 && node.id === selfId)
    return {
      ...node,
      disabled: blocked || node.type === 2,
      children: toParentOptions(node.children || [], selfId, blocked),
    }
  })
}

const parentOptions = computed(() => {
  return [{ id: 0, name: '顶级菜单', children: toParentOptions(treeData.value, form.id) }]
})

const defaultForm = {
  id: 0, parent_id: 0, name: '', path: '', component: '',
  icon: '', sort: 0, type: 1, permission: '', visible: 1, status: 1,
}
const form = reactive({ ...defaultForm })

const rules = {
  name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }],
}

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
  Object.assign(form, { ...defaultForm })
  if (row) {
    Object.assign(form, {
      id: row.id, parent_id: row.parent_id, name: row.name, path: row.path,
      component: row.component, icon: row.icon, sort: row.sort, type: row.type,
      permission: row.permission, visible: row.visible, status: row.status,
    })
  } else if (parentId !== undefined) {
    form.parent_id = parentId
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  await formRef.value?.validate()
  submitLoading.value = true
  try {
    if (form.id) {
      await updateMenu(form.id, form)
    } else {
      await createMenu(form)
    }
    ElMessage.success('操作成功')
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(id: number) {
  await deleteMenu(id)
  ElMessage.success('删除成功')
  fetchData()
}

onMounted(() => fetchData())
</script>
