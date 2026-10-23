<template>
  <el-result icon="warning" title="页面不存在" sub-title="该地址没有对应的页面，可能是菜单路径配置有误">
    <template #extra>
      <el-button type="primary" @click="router.replace('/dashboard')">返回控制台</el-button>
    </template>
  </el-result>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
