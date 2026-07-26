<script setup lang="ts">
defineProps<{ title?: string; eyebrow?: string }>()
</script>

<template>
  <div class="page page--placeholder">
    <p class="eyebrow">{{ eyebrow ?? 'MODULE' }}</p>
    <h1>{{ title ?? '模块' }}</h1>
    <div class="panel placeholder-panel">
      <span class="placeholder-panel__mark" aria-hidden="true">◌</span>
      <h2>已预留在 Operations Shell</h2>
      <p>该模块将在后续任务接入真实数据查询和恢复流程。当前实时态势、设备、告警和指令主线可用。</p>
    </div>
  </div>
</template>
