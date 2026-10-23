<template>
  <n-result status="404" title="页面不存在" description="该地址没有对应的页面，可能是菜单路径配置有误">
    <template #footer>
      <n-button type="primary" @click="router.replace('/dashboard')">返回控制台</n-button>
    </template>
  </n-result>
</template>

<script setup lang="ts">
import { NButton, NResult } from 'naive-ui'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
