<template>
  <!-- directory with children: sub-menu -->
  <el-sub-menu v-if="item.children && item.children.length" :index="item.path">
    <template #title>
      <el-icon v-if="icon"><component :is="icon" /></el-icon>
      <span>{{ item.name }}</span>
    </template>
    <menu-item v-for="child in item.children" :key="child.id" :item="child" />
  </el-sub-menu>
  <!-- leaf menu item -->
  <el-menu-item v-else :index="item.path">
    <el-icon v-if="icon"><component :is="icon" /></el-icon>
    <template #title>{{ item.name }}</template>
  </el-menu-item>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { resolveIcon } from '@/utils/icons'

const props = defineProps<{ item: any }>()
const icon = computed(() => resolveIcon(props.item.icon))
</script>
