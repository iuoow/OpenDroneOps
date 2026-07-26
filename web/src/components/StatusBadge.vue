<script setup lang="ts">
defineProps<{
  label: string
  tone?: 'success' | 'warning' | 'danger' | 'offline' | 'info' | 'neutral'
}>()
</script>

<template>
  <span class="status-badge" :class="`status-badge--${tone ?? 'neutral'}`">
    <span class="status-badge__mark" aria-hidden="true"></span>
    {{ label }}
  </span>
</template>
