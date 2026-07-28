<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Command } from '../types/contracts'

const store = useOperationsStore()
const filter = ref<'ALL' | 'IN_FLIGHT' | 'TERMINAL'>('ALL')
const selectedId = ref(store.commandList[0]?.id)
const actionFeedback = ref('')
const selected = computed<Command | undefined>(() => store.commands[selectedId.value ?? ''])
const selectedDevice = computed(() => selected.value ? store.devices[selected.value.target_device_id] : undefined)
const inFlightStatuses: Command['status'][] = ['CREATED', 'VALIDATED', 'PUBLISH_PENDING', 'PUBLISHED', 'ACCEPTED', 'EXECUTING']
const terminalStatuses: Command['status'][] = ['SUCCEEDED', 'FAILED', 'TIMEOUT', 'REJECTED', 'CANCELED']
const filtered = computed(() => store.commandList.filter((command) =>
  filter.value === 'ALL' || (filter.value === 'IN_FLIGHT' ? inFlightStatuses.includes(command.status) : terminalStatuses.includes(command.status)),
))
const inFlightCount = computed(() => store.commandList.filter((command) => inFlightStatuses.includes(command.status)).length)
const confirmedCount = computed(() => store.commandList.filter((command) => command.status === 'SUCCEEDED').length)
const canRefresh = computed(() => Boolean(selectedDevice.value && selectedDevice.value.online !== false))

const steps: Command['status'][] = ['CREATED', 'VALIDATED', 'PUBLISH_PENDING', 'PUBLISHED', 'ACCEPTED', 'EXECUTING', 'SUCCEEDED']
const stepLabels: Record<string, string> = {
  CREATED: '已创建',
  VALIDATED: '已校验',
  PUBLISH_PENDING: '等待发布',
  PUBLISHED: '已发布',
  ACCEPTED: '设备已接收',
  EXECUTING: '执行中',
  SUCCEEDED: '结果已确认',
}

function statusTone(status: Command['status']) {
  if (status === 'SUCCEEDED') return 'success'
  if (status === 'FAILED' || status === 'TIMEOUT' || status === 'REJECTED') return 'danger'
  if (status === 'ACCEPTED' || status === 'EXECUTING') return 'warning'
  return 'info'
}

function statusLabel(status: Command['status']) {
  const labels: Record<Command['status'], string> = {
    CREATED: '已创建', VALIDATED: '已校验', REJECTED: '已拒绝', PUBLISH_PENDING: '等待发布',
    PUBLISHED: '已发布', ACCEPTED: '设备已接收', EXECUTING: '执行中', SUCCEEDED: '结果已确认',
    FAILED: '执行失败', TIMEOUT: '已超时', CANCELED: '已取消',
  }
  return labels[status]
}

function statusExplanation(command: Command) {
  const explanations: Record<Command['status'], string> = {
    CREATED: '指令已创建，正在等待平台校验。',
    VALIDATED: '平台校验已通过，正在进入发送队列。',
    REJECTED: '平台在发布前拒绝了此指令；设备未执行。',
    PUBLISH_PENDING: '指令已进入发送队列，尚未获得 MQTT 发布确认。',
    PUBLISHED: '消息已发布，正在等待设备确认；这不等于设备执行成功。',
    ACCEPTED: '设备已接收指令，正在等待执行结果。',
    EXECUTING: '设备正在执行，最终结果尚未确认。',
    SUCCEEDED: '设备结果已确认成功。',
    FAILED: '设备报告执行失败，请结合设备状态和事件判断原因。',
    TIMEOUT: '在截止前未收到最终设备结果；平台不会自动重复发送。',
    CANCELED: '指令已取消；设备结果不会被推断为成功。',
  }
  return explanations[command.status]
}

function stepState(step: Command['status']) {
  if (!selected.value) return 'pending'
  const index = steps.indexOf(step)
  const current = steps.indexOf(selected.value.status)
  if (current < 0) return index === 0 ? 'complete' : 'pending'
  return index < current ? 'complete' : index === current ? 'current' : 'pending'
}

function commandTimestamp(command: Command) {
  return command.completed_at ?? command.updated_at ?? command.created_at
}

function dateTime(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString('zh-CN', { hour12: false })
}

async function createStatusRefresh() {
  if (!selectedDevice.value || !canRefresh.value) return
  const command = await store.createStatusRefresh(selectedDevice.value)
  selectedId.value = command.id
  actionFeedback.value = '已创建低风险状态刷新指令；正在等待发布与设备证据。'
}
</script>

<template>
  <div class="page commands-v2">
    <div class="page-heading commands-v2__heading">
      <div>
        <p class="eyebrow">RELIABLE OPERATIONS / COMMANDS</p>
        <h1>指令中心</h1>
        <p class="page-heading__summary">先确认平台和设备各自已确认的事实；发布成功不等于设备执行成功。</p>
      </div>
      <div class="command-summary" aria-label="指令状态摘要">
        <div><strong>{{ inFlightCount }}</strong><span>处理中</span></div>
        <div><strong class="text-success">{{ confirmedCount }}</strong><span>设备已确认</span></div>
        <div><strong>{{ store.commandList.length }}</strong><span>最近指令</span></div>
      </div>
    </div>

    <div class="commands-v2__workspace">
      <section class="command-queue-v2" aria-labelledby="command-queue-title">
        <div class="command-queue-v2__header">
          <div><p class="eyebrow">COMMAND QUEUE</p><h2 id="command-queue-title">指令队列</h2></div>
          <span class="method-chip">仅低风险</span>
        </div>
        <div class="command-filter" role="tablist" aria-label="指令状态筛选">
          <button v-for="item in ([['ALL', '全部'], ['IN_FLIGHT', '处理中'], ['TERMINAL', '已结束']] as const)" :key="item[0]" class="command-filter__item" :class="{ 'command-filter__item--active': filter === item[0] }" type="button" role="tab" :aria-selected="filter === item[0]" @click="filter = item[0]">
            {{ item[1] }}<span>{{ item[0] === 'ALL' ? store.commandList.length : item[0] === 'IN_FLIGHT' ? inFlightCount : store.commandList.length - inFlightCount }}</span>
          </button>
        </div>
        <div v-if="filtered.length" class="command-list-v2">
          <button v-for="command in filtered" :key="command.id" class="command-card" :class="[{ 'command-card--selected': selectedId === command.id }, `command-card--${statusTone(command.status)}`]" type="button" @click="selectedId = command.id">
            <span class="command-card__icon"><AppIcon name="commands" :size="18" /></span>
            <span class="command-card__body">
              <span class="command-card__meta"><StatusBadge :label="statusLabel(command.status)" :tone="statusTone(command.status)" /><span>{{ dateTime(commandTimestamp(command)) }}</span></span>
              <strong>{{ command.method }}</strong>
              <small>{{ store.devices[command.target_device_id]?.serial_number ?? command.target_device_id }} · {{ command.risk_level }} 风险</small>
            </span>
          </button>
        </div>
        <div v-else class="empty-state command-queue-v2__empty"><AppIcon name="commands" :size="28" /><strong>当前筛选下没有指令</strong><span>可切换状态筛选，或从设备上下文创建低风险刷新。</span></div>
      </section>

      <aside class="command-detail-v2" aria-labelledby="command-detail-title">
        <template v-if="selected">
          <div class="command-detail-v2__header" :class="`command-detail-v2__header--${statusTone(selected.status)}`">
            <div class="command-detail-v2__title">
              <span class="command-detail-v2__signal"><AppIcon name="commands" :size="22" /></span>
              <div><p class="eyebrow">COMMAND EVIDENCE</p><h2 id="command-detail-title">{{ selected.method }}</h2><p>{{ selectedDevice?.serial_number ?? selected.target_device_id }} · {{ selected.risk_level }} 风险</p></div>
            </div>
            <StatusBadge :label="statusLabel(selected.status)" :tone="statusTone(selected.status)" />
          </div>

          <section class="command-state-note" :class="`command-state-note--${statusTone(selected.status)}`" aria-label="当前状态说明">
            <span>当前事实</span><p>{{ statusExplanation(selected) }}</p>
          </section>

          <section class="command-evidence" aria-labelledby="command-evidence-title">
            <div class="section-heading section-heading--compact"><h3 id="command-evidence-title">执行证据</h3><span>只显示已确认事实</span></div>
            <dl class="command-evidence__grid">
              <div><dt>目标设备</dt><dd>{{ selectedDevice?.serial_number ?? selected.target_device_id }}</dd></div>
              <div><dt>设备遥测</dt><dd><FreshnessIndicator :timestamp="selectedDevice?.server_time" :online="selectedDevice?.online" /></dd></div>
              <div><dt>创建时间</dt><dd>{{ dateTime(selected.created_at) }}</dd></div>
              <div><dt>最后证据</dt><dd>{{ dateTime(commandTimestamp(selected)) }}</dd></div>
            </dl>
          </section>

          <section class="command-progress" aria-labelledby="command-progress-title">
            <div class="section-heading section-heading--compact"><h3 id="command-progress-title">指令进度</h3><span>{{ statusLabel(selected.status) }}</span></div>
            <ol class="command-steps" aria-label="指令状态进度">
              <li v-for="(step, index) in steps" :key="step" class="command-steps__item" :class="`command-steps__item--${stepState(step)}`">
                <span>{{ index + 1 }}</span>
                <div><strong>{{ stepLabels[step] }}</strong><small>{{ stepState(step) === 'current' ? statusExplanation(selected) : stepState(step) === 'complete' ? '已获得该阶段或后续阶段的证据。' : '等待可恢复的状态或设备证据。' }}</small></div>
              </li>
            </ol>
          </section>

          <details class="command-technical-evidence">
            <summary>查看技术关联与审计标识</summary>
            <dl class="detail-list">
              <div><dt>Command ID</dt><dd class="code-text">{{ selected.id }}</dd></div>
              <div><dt>DJI TID</dt><dd class="code-text">{{ selected.dji_tid ?? '等待发布' }}</dd></div>
              <div><dt>DJI BID</dt><dd class="code-text">{{ selected.dji_bid ?? '等待发布' }}</dd></div>
              <div><dt>幂等 Key</dt><dd class="code-text">{{ selected.idempotency_key }}</dd></div>
              <div><dt>结果说明</dt><dd>{{ selected.result_message ?? '设备结果尚未确认' }}</dd></div>
            </dl>
          </details>

          <div v-if="selected.status === 'PUBLISHED' || selected.status === 'ACCEPTED' || selected.status === 'EXECUTING' || selected.status === 'TIMEOUT'" class="inline-notice inline-notice--warning"><strong>设备结果尚未确认</strong><span>平台不会自动重复发送此操作；请结合设备状态与事件等待或人工判断。</span></div>

          <div class="command-detail-v2__actions">
            <button class="button button--primary" type="button" :disabled="!canRefresh" data-testid="command-status-refresh" @click="createStatusRefresh"><AppIcon name="refresh" :size="15" />发送低风险状态刷新</button>
            <RouterLink class="button button--secondary" :to="`/app/${store.workspaceId}/overview?device=${selected.target_device_id}`"><AppIcon name="pin" :size="15" />查看设备上下文</RouterLink>
            <span v-if="!canRefresh" class="command-action-hint">设备离线，无法创建刷新指令。</span>
            <span v-else class="command-action-hint">仅创建 <code>sim_status_refresh</code>；不包含飞行或设备控制。</span>
          </div>
          <p v-if="actionFeedback" class="command-action-feedback" role="status">{{ actionFeedback }}</p>
        </template>
        <div v-else class="empty-state"><AppIcon name="commands" :size="30" /><strong>从左侧选择指令</strong><span>状态进度、设备证据与安全操作会显示在这里。</span></div>
      </aside>
    </div>
  </div>
</template>
