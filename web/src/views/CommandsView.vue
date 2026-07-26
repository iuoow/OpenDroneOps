<script setup lang="ts">
import { computed, ref } from 'vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Command } from '../types/contracts'

const store = useOperationsStore()
const selectedId = ref(store.commandList[0]?.id)
const selected = computed<Command | undefined>(() => store.commands[selectedId.value ?? ''])
const selectedDevice = computed(() => selected.value ? store.devices[selected.value.target_device_id] : undefined)
const statusTone = (status: Command['status']) => status === 'SUCCEEDED' ? 'success' : status === 'FAILED' || status === 'TIMEOUT' ? 'danger' : status === 'EXECUTING' || status === 'ACCEPTED' ? 'warning' : 'info'
const steps: Command['status'][] = ['CREATED', 'VALIDATED', 'PUBLISH_PENDING', 'PUBLISHED', 'ACCEPTED', 'EXECUTING', 'SUCCEEDED']
const stepLabels: Record<string, string> = { CREATED: '已创建', VALIDATED: '已验证', PUBLISH_PENDING: '等待发布', PUBLISHED: '已发布', ACCEPTED: '设备已接受', EXECUTING: '执行中', SUCCEEDED: '成功' }
function stepState(step: Command['status']) {
  if (!selected.value) return 'pending'
  const current = steps.indexOf(selected.value.status)
  const index = steps.indexOf(step)
  if (selected.value.status === 'FAILED' || selected.value.status === 'TIMEOUT') return index <= current ? 'complete' : 'pending'
  return index < current ? 'complete' : index === current ? 'current' : 'pending'
}
</script>

<template>
  <div class="page">
    <div class="page-heading"><div><p class="eyebrow">RELIABLE OPERATIONS / COMMANDS</p><h1>指令中心</h1><p class="page-heading__summary">发布成功不是业务成功；每一步都展示来源、状态和下一步。</p></div><span class="page-heading__count">{{ store.commandList.length }} 条指令</span></div>
    <div class="split-panel">
      <section class="panel command-list" aria-labelledby="command-list-title">
        <div class="section-heading"><div><h2 id="command-list-title">最近指令</h2><p class="muted">MVP 仅开放低风险状态刷新</p></div><span class="method-chip">sim_status_refresh</span></div>
        <div class="command-list__items">
          <button v-for="command in store.commandList" :key="command.id" class="command-row" :class="{ 'command-row--selected': selectedId === command.id }" type="button" @click="selectedId = command.id">
            <span class="command-row__method">↻</span><span><strong>{{ command.method }}</strong><small>{{ store.devices[command.target_device_id]?.serial_number ?? command.target_device_id }}</small></span><StatusBadge :label="command.status" :tone="statusTone(command.status)" />
          </button>
        </div>
      </section>
      <aside class="panel command-detail" aria-labelledby="command-detail-title">
        <div v-if="selected">
          <div class="section-heading"><div><p class="eyebrow">COMMAND DETAIL</p><h2 id="command-detail-title">{{ selected.method }}</h2></div><StatusBadge :label="selected.status" :tone="statusTone(selected.status)" /></div>
          <p class="detail-lead">目标：{{ selectedDevice?.serial_number ?? selected.target_device_id }} · 风险：{{ selected.risk_level }}</p>
          <ol class="progress-stepper" aria-label="指令进度">
            <li v-for="step in steps" :key="step" :class="`progress-stepper__item progress-stepper__item--${stepState(step)}`"><span class="progress-stepper__dot" aria-hidden="true"></span><span>{{ stepLabels[step] }}</span></li>
          </ol>
          <dl class="detail-list"><div><dt>Command ID</dt><dd class="code-text">{{ selected.id }}</dd></div><div><dt>DJI TID</dt><dd class="code-text">{{ selected.dji_tid ?? '等待发布' }}</dd></div><div><dt>DJI BID</dt><dd class="code-text">{{ selected.dji_bid ?? '等待发布' }}</dd></div><div><dt>幂等 Key</dt><dd class="code-text">{{ selected.idempotency_key }}</dd></div><div><dt>结果说明</dt><dd>{{ selected.result_message ?? '设备结果尚未确认' }}</dd></div></dl>
          <div v-if="selected.status === 'PUBLISHED' || selected.status === 'ACCEPTED' || selected.status === 'EXECUTING'" class="inline-notice inline-notice--warning"><strong>设备结果尚未确认</strong><span>平台不会自动重复发送此操作。</span></div>
        </div>
        <div v-else class="empty-state">从左侧选择指令</div>
      </aside>
    </div>
  </div>
</template>
