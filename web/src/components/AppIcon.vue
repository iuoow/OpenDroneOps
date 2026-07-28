<script setup lang="ts">
withDefaults(defineProps<{
  name: 'overview' | 'devices' | 'alarms' | 'commands' | 'replay' | 'operations' | 'aircraft' | 'dock' | 'layers' | 'pin' | 'refresh' | 'more' | 'close'
  size?: number
}>(), { size: 18 })
</script>

<template>
  <svg class="app-icon" :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="name === 'overview'">
      <rect x="3.5" y="3.5" width="6.5" height="6.5" rx="1" /><rect x="14" y="3.5" width="6.5" height="6.5" rx="1" /><rect x="3.5" y="14" width="6.5" height="6.5" rx="1" /><rect x="14" y="14" width="6.5" height="6.5" rx="1" />
    </template>
    <template v-else-if="name === 'devices' || name === 'aircraft'">
      <path d="M3 12h18M12 3v18M6 7.5 12 12l6-4.5M6 16.5 12 12l6 4.5" /><circle cx="12" cy="12" r="2.4" />
    </template>
    <template v-else-if="name === 'dock'">
      <rect x="4" y="5" width="16" height="14" rx="2" /><path d="M8 9h8M9 15h6" />
    </template>
    <template v-else-if="name === 'alarms'">
      <path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />
    </template>
    <template v-else-if="name === 'commands'">
      <path d="M5 12h12M13 7l5 5-5 5" /><path d="M5 5v14" />
    </template>
    <template v-else-if="name === 'replay'">
      <path d="M3.5 12a8.5 8.5 0 1 0 2.5-6" /><path d="M3.5 5v4h4" /><path d="m10 8 5 4-5 4z" />
    </template>
    <template v-else-if="name === 'operations'">
      <circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.04 2.04-.06-.06a1.7 1.7 0 0 0-1.88-.34 1.7 1.7 0 0 0-1.03 1.56v.1H12v-2.9h.1A1.7 1.7 0 0 0 13.66 16.3a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.04-2.04.06.06A1.7 1.7 0 0 0 17.24 12c.67-.28 1.1-.94 1.1-1.66v-.1h2.9v.1A1.7 1.7 0 0 0 22.8 12c0 .67-.4 1.28-1.03 1.56" />
    </template>
    <template v-else-if="name === 'layers'">
      <path d="m12 3 8 4.5-8 4.5-8-4.5zM4 12l8 4.5 8-4.5M4 16.5l8 4.5 8-4.5" />
    </template>
    <template v-else-if="name === 'pin'">
      <path d="M12 21s6-5.1 6-11a6 6 0 1 0-12 0c0 5.9 6 11 6 11" /><circle cx="12" cy="10" r="2" />
    </template>
    <template v-else-if="name === 'refresh'">
      <path d="M20 11a8 8 0 0 0-14.8-4.2L3 9m1-5v5h5M4 13a8 8 0 0 0 14.8 4.2L21 15m-1 5v-5h-5" />
    </template>
    <template v-else-if="name === 'more'">
      <circle cx="5" cy="12" r="1" fill="currentColor" /><circle cx="12" cy="12" r="1" fill="currentColor" /><circle cx="19" cy="12" r="1" fill="currentColor" />
    </template>
    <path v-else d="m6 6 12 12M18 6 6 18" />
  </svg>
</template>
