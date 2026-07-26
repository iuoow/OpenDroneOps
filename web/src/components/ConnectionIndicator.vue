<script setup lang="ts">
import StatusBadge from './StatusBadge.vue'

defineProps<{
  status: 'idle' | 'connecting' | 'connected' | 'recovering' | 'disconnected'
  detail?: string
}>()

const labels = {
  idle: '未连接',
  connecting: '连接中',
  connected: '实时连接正常',
  recovering: '正在恢复',
  disconnected: '实时连接已断开',
}
</script>

<template>
  <div class="connection-indicator" role="status" aria-live="polite">
    <StatusBadge
      :label="labels[status]"
      :tone="status === 'connected' ? 'success' : status === 'recovering' || status === 'connecting' ? 'warning' : 'offline'"
    />
    <span v-if="detail" class="connection-indicator__detail">{{ detail }}</span>
  </div>
</template>
