<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Alarm } from '../types/contracts'

const store = useOperationsStore()
const filter = ref<'ALL' | 'OPEN' | 'ACKNOWLEDGED'>('ALL')
const selectedId = ref(store.activeAlarms[0]?.id)
const filtered = computed(() => store.alarmList.filter((alarm) => filter.value === 'ALL' || alarm.status === filter.value))
const selected = computed<Alarm | undefined>(() => store.alarms[selectedId.value ?? ''])
const openCount = computed(() => store.activeAlarms.filter((alarm) => alarm.status === 'OPEN').length)
const acknowledgedCount = computed(() => store.activeAlarms.filter((alarm) => alarm.status === 'ACKNOWLEDGED').length)

function tone(severity: Alarm['severity']) {
  return severity === 'CRITICAL' ? 'danger' : severity === 'WARNING' ? 'warning' : 'info'
}

function deviceName(deviceId: string) {
  return store.devices[deviceId]?.serial_number ?? deviceId
}

function alarmLabel(alarm: Alarm) {
  return alarm.alarm_type === 'battery.low' ? '低电量' : alarm.alarm_type
}

function statusLabel(status: Alarm['status']) {
  return status === 'OPEN' ? '待接手' : status === 'ACKNOWLEDGED' ? '已接手' : '已恢复'
}

function recommendation(alarm: Alarm) {
  const value = alarm.details?.recommendation
  return typeof value === 'string' ? value : '检查设备状态、数据新鲜度和关联事件。'
}

function dateTime(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString('zh-CN', { hour12: false })
}

function selectAlarm(alarm: Alarm) {
  selectedId.value = alarm.id
}
</script>

<template>
  <div class="page alarms-v2">
    <div class="page-heading alarms-v2__heading">
      <div>
        <p class="eyebrow">INCIDENT DESK / ALARMS</p>
        <h1>告警中心</h1>
        <p class="page-heading__summary">先确认谁在处理，再查看证据与安全下一步；接手不代表问题已恢复。</p>
      </div>
      <div class="alarm-summary" aria-label="告警处置摘要">
        <div><strong class="text-danger">{{ openCount }}</strong><span>待接手</span></div>
        <div><strong>{{ acknowledgedCount }}</strong><span>已接手</span></div>
        <div><strong>{{ store.activeAlarms.length }}</strong><span>活动告警</span></div>
      </div>
    </div>

    <div class="alarms-v2__workspace">
      <section class="alarm-queue-v2" aria-labelledby="alarm-queue-title">
        <div class="alarm-queue-v2__header">
          <div>
            <p class="eyebrow">TRIAGE QUEUE</p>
            <h2 id="alarm-queue-title">处置队列</h2>
          </div>
          <span class="muted">严重度优先</span>
        </div>
        <div class="alarm-filter" role="tablist" aria-label="告警状态筛选">
          <button v-for="item in ([['ALL', '全部'], ['OPEN', '待接手'], ['ACKNOWLEDGED', '已接手']] as const)" :key="item[0]" class="alarm-filter__item" :class="{ 'alarm-filter__item--active': filter === item[0] }" type="button" role="tab" :aria-selected="filter === item[0]" @click="filter = item[0]">
            {{ item[1] }}<span>{{ item[0] === 'ALL' ? store.activeAlarms.length : item[0] === 'OPEN' ? openCount : acknowledgedCount }}</span>
          </button>
        </div>
        <div v-if="filtered.length" class="alarm-list-v2">
          <button v-for="alarm in filtered" :key="alarm.id" class="alarm-card" :class="[`alarm-card--${tone(alarm.severity)}`, { 'alarm-card--selected': selectedId === alarm.id }]" type="button" @click="selectAlarm(alarm)">
            <span class="alarm-card__icon"><AppIcon name="alarms" :size="18" /></span>
            <span class="alarm-card__body">
              <span class="alarm-card__meta"><StatusBadge :label="alarm.severity === 'CRITICAL' ? '严重' : alarm.severity === 'WARNING' ? '警告' : '信息'" :tone="tone(alarm.severity)" /><span>{{ statusLabel(alarm.status) }}</span></span>
              <strong>{{ alarmLabel(alarm) }}</strong>
              <small>{{ deviceName(alarm.device_id) }} · 最近 {{ dateTime(alarm.last_occurred_at) }}</small>
            </span>
            <span class="alarm-card__count">×{{ alarm.occurrence_count }}</span>
          </button>
        </div>
        <div v-else class="empty-state alarm-queue-v2__empty"><AppIcon name="alarms" :size="28" /><strong>当前筛选下没有活动告警</strong><span>最近同步：{{ dateTime(store.lastSyncAt ?? undefined) }}</span></div>
      </section>

      <aside class="alarm-detail-v2" aria-labelledby="alarm-detail-title">
        <template v-if="selected">
          <div class="alarm-detail-v2__header" :class="`alarm-detail-v2__header--${tone(selected.severity)}`">
            <div class="alarm-detail-v2__title">
              <span class="alarm-detail-v2__signal"><AppIcon name="alarms" :size="22" /></span>
              <div><p class="eyebrow">INCIDENT EVIDENCE</p><h2 id="alarm-detail-title">{{ alarmLabel(selected) }}</h2><p>{{ deviceName(selected.device_id) }} · {{ statusLabel(selected.status) }}</p></div>
            </div>
            <StatusBadge :label="selected.severity === 'CRITICAL' ? '严重' : selected.severity === 'WARNING' ? '警告' : '信息'" :tone="tone(selected.severity)" />
          </div>

          <section class="incident-recommendation" aria-label="建议处置">
            <span class="incident-recommendation__label">建议处置</span>
            <p>{{ recommendation(selected) }}</p>
          </section>

          <section class="incident-evidence" aria-labelledby="evidence-title">
            <div class="section-heading section-heading--compact"><h3 id="evidence-title">事件证据</h3><span>实时证据</span></div>
            <dl class="incident-evidence__grid">
              <div><dt>首次发生</dt><dd>{{ dateTime(selected.first_occurred_at) }}</dd></div>
              <div><dt>最近发生</dt><dd>{{ dateTime(selected.last_occurred_at) }}</dd></div>
              <div><dt>重复次数</dt><dd class="tabular">{{ selected.occurrence_count }}</dd></div>
              <div><dt>设备遥测</dt><dd><FreshnessIndicator :timestamp="store.devices[selected.device_id]?.server_time" :online="store.devices[selected.device_id]?.online" /></dd></div>
            </dl>
          </section>

          <section class="incident-handling" aria-labelledby="handling-title">
            <div class="section-heading section-heading--compact"><h3 id="handling-title">处置状态</h3><span>{{ statusLabel(selected.status) }}</span></div>
            <ol class="incident-steps" aria-label="告警处置状态">
              <li class="incident-steps__item incident-steps__item--complete"><span>1</span><div><strong>已发现</strong><small>{{ dateTime(selected.first_occurred_at) }}</small></div></li>
              <li class="incident-steps__item" :class="{ 'incident-steps__item--complete': selected.status === 'ACKNOWLEDGED' || selected.status === 'RESOLVED', 'incident-steps__item--current': selected.status === 'OPEN' }"><span>2</span><div><strong>确认接手</strong><small>{{ selected.acknowledged_at ? dateTime(selected.acknowledged_at) : '等待授权操作员接手' }}</small></div></li>
              <li class="incident-steps__item" :class="{ 'incident-steps__item--complete': selected.status === 'RESOLVED', 'incident-steps__item--current': selected.status === 'ACKNOWLEDGED' }"><span>3</span><div><strong>等待规则恢复</strong><small>{{ selected.resolved_at ? dateTime(selected.resolved_at) : '不由接手操作自动完成' }}</small></div></li>
            </ol>
          </section>

          <div class="alarm-detail-v2__actions">
            <button v-if="selected.status === 'OPEN'" class="button button--primary" type="button" @click="store.acknowledgeAlarm(selected)">确认接手此告警</button>
            <span v-else class="handling-owner"><AppIcon name="overview" :size="15" />{{ selected.status === 'ACKNOWLEDGED' ? `${selected.acknowledged_by ?? '操作员'} 已接手` : '规则已确认恢复' }}</span>
            <RouterLink class="button button--secondary" :to="`/app/${store.workspaceId}/overview?device=${selected.device_id}`"><AppIcon name="pin" :size="15" />查看设备上下文</RouterLink>
          </div>
        </template>
        <div v-else class="empty-state"><AppIcon name="alarms" :size="30" /><strong>从左侧选择告警</strong><span>详细证据和处置状态会在这里显示。</span></div>
      </aside>
    </div>
  </div>
</template>
