<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { PilotBridgeAdapter, PilotRuntimeConfig, PilotStartupState } from './bridge'
import { bootstrapPilot } from './bootstrap'

const props = defineProps<{
  bridge: PilotBridgeAdapter
  config: PilotRuntimeConfig
}>()

const state = ref<PilotStartupState>({ phase: 'detecting' })
const statusText = computed(() => {
  switch (state.value.phase) {
    case 'detecting':
      return '正在检测 Pilot 环境'
    case 'verifying_license':
      return '正在验证 Pilot 许可'
    case 'configuring':
      return '正在配置安全连接'
    case 'loading_modules':
      return '正在加载现场模块'
    case 'ready':
      return 'Pilot Shell 已准备就绪（Mock Bridge）'
    case 'failed':
      return `Pilot Shell 暂不可用：${state.value.code}`
  }
})

async function start() {
  state.value = { phase: 'detecting' }
  await bootstrapPilot({
    bridge: props.bridge,
    config: props.config,
    onState: (next) => {
      state.value = next
    },
  })
}

onMounted(() => void start())
</script>

<template>
  <main aria-labelledby="pilot-bootstrap-title">
    <h1 id="pilot-bootstrap-title">OpenDroneOps Pilot</h1>
    <p data-testid="pilot-startup-state" role="status">{{ statusText }}</p>
    <button v-if="state.phase === 'failed' && state.retryable" type="button" @click="start">
      重试
    </button>
  </main>
</template>
