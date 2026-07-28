<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ timestamp?: string; online?: boolean }>(), { online: undefined })

const seconds = computed(() => {
  if (!props.timestamp) return null
  const timestamp = new Date(props.timestamp).getTime()
  return Number.isFinite(timestamp) ? Math.max(0, Math.round((Date.now() - timestamp) / 1000)) : null
})
const label = computed(() => {
  if (props.online === false) return '设备离线'
  if (seconds.value === null) return '暂无遥测数据'
  if (seconds.value <= 10) return `${seconds.value} 秒前更新`
  if (seconds.value <= 60) return `数据延迟 · ${seconds.value} 秒前`
  return `数据可能已过期 · ${Math.round(seconds.value / 60)} 分钟前`
})
const tone = computed(() => (props.online === false ? 'offline' : seconds.value === null ? 'unavailable' : seconds.value > 60 ? 'warning' : 'success'))
</script>

<template>
  <span class="freshness" :class="`freshness--${tone}`">
    <span class="freshness__icon" aria-hidden="true">◷</span>
    {{ label }}
  </span>
</template>
