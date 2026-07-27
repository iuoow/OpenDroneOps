<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { PilotBridgeAdapter, PilotRuntimeConfig, PilotStartupState } from './bridge'
import { bootstrapPilot } from './bootstrap'

const props = defineProps<{
  bridge: PilotBridgeAdapter
  config: PilotRuntimeConfig
}>()

const state = ref<PilotStartupState>({ phase: 'detecting' })
const activeNavigation = ref('home')
const startedAt = new Intl.DateTimeFormat('zh-CN', {
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
}).format(new Date())

const navigation = [
  { id: 'home', label: '主页', short: 'HOME' },
  { id: 'device', label: '设备', short: 'DEV' },
  { id: 'alerts', label: '告警', short: 'ALT' },
  { id: 'more', label: '更多', short: 'MORE' },
]

const startupSteps = [
  { phase: 'detecting', label: '检测 Pilot 环境' },
  { phase: 'verifying_license', label: '验证 Pilot 许可' },
  { phase: 'configuring', label: '配置安全连接' },
  { phase: 'loading_modules', label: '加载现场模块' },
]

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
      return failureText(state.value.code)
  }
})

const currentStep = computed(() => {
  const index = startupSteps.findIndex((step) => step.phase === state.value.phase)
  return index < 0 ? startupSteps.length : index
})

function failureText(code: Extract<PilotStartupState, { phase: 'failed' }>['code']) {
  switch (code) {
    case 'BRIDGE_UNAVAILABLE':
      return '未检测到可用的 Pilot Bridge'
    case 'LICENSE_REJECTED':
      return 'Pilot 许可未通过验证'
    case 'CONFIGURATION_REJECTED':
      return '现场连接配置不可用'
    case 'REQUIRED_MODULE_UNAVAILABLE':
      return '所需现场模块不可用'
    case 'UNEXPECTED':
      return 'Pilot Shell 启动遇到意外问题'
  }
}

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
  <main class="pilot-shell" aria-labelledby="pilot-shell-title">
    <header class="pilot-shell__header">
      <div>
        <p class="pilot-shell__eyebrow">WORKSPACE</p>
        <h1 id="pilot-shell-title">{{ config.workspaceId }}</h1>
      </div>
      <div class="pilot-shell__connection" aria-label="连接状态">
        <span class="pilot-shell__connection-item">
          <span aria-hidden="true" class="pilot-shell__status-dot" :class="{ 'is-ready': state.phase === 'ready' }"></span>
          云端 {{ state.phase === 'ready' ? '已连接' : '初始化中' }}
        </span>
        <span class="pilot-shell__connection-item">
          <span aria-hidden="true" class="pilot-shell__status-dot" :class="{ 'is-ready': state.phase === 'ready' }"></span>
          Pilot {{ bridge.kind === 'mock' ? 'Mock' : '已检测' }}
        </span>
        <time :datetime="new Date().toISOString()">{{ startedAt }}</time>
      </div>
    </header>

    <section v-if="state.phase !== 'ready'" class="pilot-shell__startup" aria-describedby="pilot-startup-status">
      <p class="pilot-shell__eyebrow">PILOT STARTUP</p>
      <h2>{{ state.phase === 'failed' ? '需要处理后才能继续' : '正在准备现场工作区' }}</h2>
      <p id="pilot-startup-status" class="pilot-shell__status-copy" data-testid="pilot-startup-state" role="status" aria-live="polite">
        {{ statusText }}
      </p>
      <ol class="pilot-shell__steps" aria-label="Pilot 启动步骤">
        <li
          v-for="(step, index) in startupSteps"
          :key="step.phase"
          :class="{ 'is-current': index === currentStep, 'is-complete': index < currentStep }"
          :aria-current="index === currentStep ? 'step' : undefined"
        >
          <span aria-hidden="true">{{ index < currentStep ? '✓' : index + 1 }}</span>
          {{ step.label }}
        </li>
      </ol>
      <button
        v-if="state.phase === 'failed' && state.retryable"
        class="pilot-shell__primary-action"
        type="button"
        @click="start"
      >
        重新检测 Pilot 环境
      </button>
      <p v-else-if="state.phase === 'failed'" class="pilot-shell__hint">
        此状态不能通过重复请求恢复，请联系已授权的现场管理员。
      </p>
    </section>

    <section v-else class="pilot-shell__home" data-testid="pilot-ready-shell" aria-label="Pilot 现场主页">
      <div class="pilot-shell__task-card">
        <p class="pilot-shell__eyebrow">CURRENT TASK</p>
        <h2>现场巡检 · Mock 024</h2>
        <p>当前为浏览器 Mock Bridge 演示。实时设备与告警数据将在后续只读任务中接入。</p>
        <div class="pilot-shell__map-placeholder" role="img" aria-label="任务区域预览占位图">
          <span>任务区域</span>
          <span class="pilot-shell__map-marker" aria-hidden="true">OD</span>
          <span class="pilot-shell__map-grid" aria-hidden="true"></span>
        </div>
      </div>

      <div class="pilot-shell__summary-grid" aria-label="当前设备与告警摘要">
        <article>
          <p class="pilot-shell__eyebrow">CURRENT DEVICE</p>
          <h3>等待实时设备数据</h3>
          <p>Task 17 将通过既有快照和 WebSocket 契约填充此区域。</p>
        </article>
        <article>
          <p class="pilot-shell__eyebrow">ACTIVE ALERT</p>
          <h3>等待实时告警数据</h3>
          <p>仅展示只读现场告警；不会在 Pilot Shell 中确认或关闭告警。</p>
        </article>
      </div>
    </section>

    <nav v-if="state.phase === 'ready'" class="pilot-shell__navigation" aria-label="Pilot 主导航">
      <button
        v-for="item in navigation"
        :key="item.id"
        type="button"
        :aria-pressed="activeNavigation === item.id"
        :class="{ 'is-active': activeNavigation === item.id }"
        @click="activeNavigation = item.id"
      >
        <span aria-hidden="true">{{ item.short }}</span>
        {{ item.label }}
      </button>
    </nav>
  </main>
</template>
