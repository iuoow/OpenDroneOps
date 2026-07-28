<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { PilotBridgeAdapter, PilotRuntimeConfig, PilotStartupState } from './bridge'
import { bootstrapPilot } from './bootstrap'
import type { PilotDraft, PilotDraftStore } from './drafts'
import type { PilotReadModel } from './readModel'

const props = defineProps<{
  bridge: PilotBridgeAdapter
  config: PilotRuntimeConfig
  readModel: PilotReadModel
  draftStore: PilotDraftStore
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
const currentDevice = computed(() => props.readModel.currentDevice.value)
const currentAlarm = computed(() => props.readModel.currentAlarm.value)
const devices = computed(() => props.readModel.devices.value)
const activeAlarms = computed(() => props.readModel.activeAlarms.value)
const dataLoading = computed(() => props.readModel.loading.value)
const dataError = computed(() => props.readModel.error.value)
const dataStale = computed(() => props.readModel.stale.value)
const connection = computed(() => props.readModel.connection.value)
const connectionDetail = computed(() => props.readModel.connectionDetail.value)
const drafts = ref<readonly PilotDraft[]>(props.draftStore.list())
const draftBody = ref('')
const draftError = ref('')
const draftNotice = ref('')
const cloudStatusClass = computed(() => ({
  'is-ready': connection.value === 'connected' && !dataStale.value,
  'is-warning': connection.value === 'connecting' || connection.value === 'recovering' || dataStale.value,
  'is-offline': connection.value === 'disconnected' || Boolean(dataError.value),
}))
const cloudStatusText = computed(() => {
  if (dataError.value) return '快照失败'
  if (connection.value === 'disconnected') return '已断开'
  if (connection.value === 'recovering') return '恢复中'
  if (connection.value === 'connecting') return '连接中'
  if (dataStale.value) return '数据陈旧'
  if (connection.value === 'connected') return '已连接'
  return dataLoading.value ? '加载中' : '待连接'
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
  const result = await bootstrapPilot({
    bridge: props.bridge,
    config: props.config,
    onState: (next) => {
      state.value = next
    },
  })
  if (result.phase === 'ready') await props.readModel.hydrate()
}

onMounted(() => void start())
onUnmounted(() => props.readModel.stop())

function refreshDrafts() {
  drafts.value = props.draftStore.list()
}

function saveDraft() {
  draftError.value = ''
  draftNotice.value = ''
  try {
    props.draftStore.save({ deviceId: currentDevice.value?.id, body: draftBody.value })
    draftBody.value = ''
    draftNotice.value = '草稿已保存在本机，不会自动提交到云端'
    refreshDrafts()
  } catch (error) {
    draftError.value = error instanceof Error ? error.message : '草稿无法保存'
  }
}

function retryDraft(draft: PilotDraft) {
  draftBody.value = draft.body
  draftNotice.value = '草稿已载入编辑区；请确认后重新保存'
  draftError.value = ''
  activeNavigation.value = 'home'
}

function discardDraft(id: string) {
  props.draftStore.remove(id)
  draftNotice.value = '草稿已从本机删除'
  draftError.value = ''
  refreshDrafts()
}

function deviceTitle() {
  return currentDevice.value?.product_model || currentDevice.value?.serial_number || '暂无设备'
}

function alarmTitle() {
  return currentAlarm.value?.alarm_type || '当前无活动告警'
}

function deviceUpdatedAt() {
  const value = currentDevice.value?.server_time || currentDevice.value?.updated_at
  return value ? new Date(value).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' }) : '未知'
}

function batteryLabel() {
  return currentDevice.value?.battery_percent == null ? '未知' : `${currentDevice.value.battery_percent}%`
}

function severityLabel(severity?: string) {
  return severity === 'CRITICAL' ? '严重' : severity === 'WARNING' ? '警告' : '提示'
}
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
          <span aria-hidden="true" class="pilot-shell__status-dot" :class="cloudStatusClass"></span>
          云端 {{ state.phase === 'ready' ? cloudStatusText : '初始化中' }}
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
      <div
        v-if="dataLoading || dataError || dataStale || connection === 'recovering' || connection === 'disconnected'"
        class="pilot-shell__data-state"
        :class="{ 'is-error': dataError || connection === 'disconnected', 'is-warning': dataStale || connection === 'recovering' }"
        data-testid="pilot-data-state"
        role="status"
        aria-live="polite"
      >
        <strong v-if="dataLoading">正在加载只读现场快照</strong>
        <strong v-else-if="dataError">{{ dataError }}</strong>
        <strong v-else-if="connection === 'disconnected'">实时连接已断开</strong>
        <strong v-else-if="connection === 'recovering'">正在从恢复游标继续实时数据</strong>
        <strong v-else>当前数据可能已陈旧</strong>
        <span v-if="connectionDetail">{{ connectionDetail }}</span>
        <button
          v-if="!dataLoading"
          type="button"
          data-testid="pilot-reconnect"
          @click="readModel.reconnect"
        >
          重新连接并刷新
        </button>
      </div>

      <div class="pilot-shell__task-card">
        <p class="pilot-shell__eyebrow">CURRENT TASK</p>
        <h2>现场巡检 · 只读态势</h2>
        <p>当前为浏览器 Mock Bridge 演示。数据来自只读快照与实时事件，不提供任何控制操作。</p>
        <div class="pilot-shell__map-placeholder" role="img" aria-label="任务区域预览占位图">
          <span>任务区域</span>
          <span class="pilot-shell__map-marker" aria-hidden="true">{{ currentDevice ? 'DJI' : 'OD' }}</span>
          <span class="pilot-shell__map-grid" aria-hidden="true"></span>
        </div>
      </div>

      <div v-if="activeNavigation === 'home'" class="pilot-shell__summary-grid" aria-label="当前设备与告警摘要">
        <article>
          <p class="pilot-shell__eyebrow">CURRENT DEVICE</p>
          <h3>{{ deviceTitle() }}</h3>
          <p v-if="currentDevice">
            {{ currentDevice.status }} · 电量 {{ batteryLabel() }} · {{ currentDevice.mode || '模式未知' }}
          </p>
          <p v-else>当前 Workspace 没有可显示的设备。</p>
          <small>更新于 {{ deviceUpdatedAt() }}</small>
        </article>
        <article>
          <p class="pilot-shell__eyebrow">ACTIVE ALERT</p>
          <h3>{{ alarmTitle() }}</h3>
          <p v-if="currentAlarm">
            {{ severityLabel(currentAlarm.severity) }} · 出现 {{ currentAlarm.occurrence_count }} 次
          </p>
          <p v-else>当前没有未解决告警。</p>
          <small>仅供查看；Pilot Shell 不确认或关闭告警。</small>
        </article>
      </div>

      <section v-if="activeNavigation === 'home'" class="pilot-shell__draft-panel" aria-labelledby="pilot-drafts-title">
        <div>
          <p class="pilot-shell__eyebrow">FIELD NOTE</p>
          <h2 id="pilot-drafts-title">现场备注草稿</h2>
          <p class="pilot-shell__draft-copy">
            仅保存在本机，用于断线期间的现场连续性；不会自动提交，也不能包含凭据或诊断路径。
          </p>
        </div>
        <label class="pilot-shell__draft-label" for="pilot-draft-body">备注内容</label>
        <textarea
          id="pilot-draft-body"
          data-testid="pilot-draft-body"
          v-model="draftBody"
          class="pilot-shell__draft-input"
          maxlength="500"
          rows="3"
          placeholder="例如：北侧风况变化，待网络恢复后复核"
        ></textarea>
        <div class="pilot-shell__draft-actions">
          <button type="button" data-testid="pilot-save-draft" @click="saveDraft">保存本机草稿</button>
          <span v-if="draftNotice" class="pilot-shell__draft-notice" role="status">{{ draftNotice }}</span>
          <span v-if="draftError" class="pilot-shell__draft-error" role="alert">{{ draftError }}</span>
        </div>
        <ul v-if="drafts.length" class="pilot-shell__draft-list" data-testid="pilot-draft-list">
          <li v-for="draft in drafts" :key="draft.id">
            <div>
              <strong>{{ draft.body }}</strong>
              <small>{{ new Date(draft.updatedAt).toLocaleString('zh-CN') }}</small>
            </div>
            <div class="pilot-shell__draft-item-actions">
              <button type="button" @click="retryDraft(draft)">重试编辑</button>
              <button type="button" @click="discardDraft(draft.id)">删除</button>
            </div>
          </li>
        </ul>
        <p v-else class="pilot-shell__draft-empty">暂无本机草稿</p>
      </section>

      <section v-else-if="activeNavigation === 'device'" class="pilot-shell__detail-panel" aria-labelledby="pilot-device-title">
        <p class="pilot-shell__eyebrow">DEVICE DETAIL</p>
        <h2 id="pilot-device-title">{{ deviceTitle() }}</h2>
        <dl v-if="currentDevice" class="pilot-shell__detail-grid">
          <div><dt>序列号</dt><dd>{{ currentDevice.serial_number }}</dd></div>
          <div><dt>状态</dt><dd>{{ currentDevice.status }}</dd></div>
          <div><dt>电量</dt><dd>{{ batteryLabel() }}</dd></div>
          <div><dt>模式</dt><dd>{{ currentDevice.mode || '未知' }}</dd></div>
          <div><dt>高度</dt><dd>{{ currentDevice.altitude == null ? '未知' : `${currentDevice.altitude} m` }}</dd></div>
          <div><dt>数据版本</dt><dd>{{ currentDevice.state_version ?? '未知' }}</dd></div>
        </dl>
        <p v-else>当前 Workspace 没有可显示的设备。</p>
        <p class="pilot-shell__readonly-note">只读详情 · 共 {{ devices.length }} 个可见设备</p>
      </section>

      <section v-else-if="activeNavigation === 'alerts'" class="pilot-shell__detail-panel" aria-labelledby="pilot-alerts-title">
        <p class="pilot-shell__eyebrow">ACTIVE ALERTS</p>
        <h2 id="pilot-alerts-title">现场告警（{{ activeAlarms.length }}）</h2>
        <ul v-if="activeAlarms.length" class="pilot-shell__alert-list">
          <li v-for="alarm in activeAlarms" :key="alarm.id">
            <span class="pilot-shell__severity" :data-severity="alarm.severity">{{ severityLabel(alarm.severity) }}</span>
            <div><strong>{{ alarm.alarm_type }}</strong><small>{{ alarm.device_id }} · {{ alarm.status }}</small></div>
            <span>{{ alarm.occurrence_count }} 次</span>
          </li>
        </ul>
        <p v-else>当前没有未解决告警。</p>
        <p class="pilot-shell__readonly-note">仅供查看，不提供确认、关闭或批量操作。</p>
      </section>

      <section v-else class="pilot-shell__detail-panel" aria-labelledby="pilot-more-title">
        <p class="pilot-shell__eyebrow">FOUNDATION STATUS</p>
        <h2 id="pilot-more-title">只读现场模式</h2>
        <p>Bridge：{{ bridge.kind }} · 实时状态：{{ cloudStatusText }}</p>
        <p class="pilot-shell__readonly-note">真实 DJI、诊断上传、第三方应用和设备控制仍未启用。</p>
      </section>
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
