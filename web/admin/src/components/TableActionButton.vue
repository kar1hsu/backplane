<template>
  <n-tooltip :delay="250">
    <template #trigger>
      <n-button
        quaternary
        circle
        size="small"
        :type="type"
        :aria-label="label"
        class="table-action-button"
        @click="$emit('click', $event)"
      >
        <template #icon>
          <n-icon :component="icon" />
        </template>
      </n-button>
    </template>
    {{ label }}
  </n-tooltip>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { NButton, NIcon, NTooltip, type ButtonProps } from 'naive-ui'

withDefaults(defineProps<{
  label: string
  icon: Component
  type?: ButtonProps['type']
}>(), {
  type: 'default',
})

defineEmits<{ click: [event: MouseEvent] }>()
</script>

<style scoped>
.table-action-button {
  width: 30px;
  height: 30px;
}
</style>
