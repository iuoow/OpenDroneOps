<script setup lang="ts">
import { computed, ref } from 'vue'
import StatusBadge from '../components/StatusBadge.vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import { useOperationsStore } from '../state/operations'
import type { Alarm } from '../types/contracts'

const store = useOperationsStore()
const filter = ref<'ALL' | 'OPEN' | 'ACKNOWLEDGED'>('ALL')
const selectedId = ref(store.activeAlarms[0]?.id)
const filtered = computed(() => store.alarmList.filter((alarm) => filter.value === 'ALL' || alarm.status === filter.value))
const selected = computed<Alarm | undefined>(() => store.alarms[selectedId.value ?? ''])
function tone(severity: Alarm['severity']) { return severity === 'CRITICAL' ? 'danger' : severity === 'WARNING' ? 'warning' : 'info' }
function deviceName(deviceId: string) { return store.devices[deviceId]?.serial_number ?? deviceId }
</script>

<template>
  <div class="page">
    <div class="page-heading"><div><p class="eyebrow">RELIABLE OPERATIONS / ALARMS</p><h1>告警中心</h1><p class="page-heading__summary">确认接手不等于问题已恢复；每次去重更新都保留次数和时间。</p></div><span class="page-heading__count text-danger">{{ store.activeAlarms.length }} 活动</span></div>
    <div class="split-panel">
      <section class="panel alarm-queue" aria-labelledby="alarm-queue-title">
        <div class="section-heading"><div><h2 id="alarm-queue-title">告警队列</h2><p class="muted">按严重程度排序</p></div><select v-model="filter" class="compact-select" aria-label="告警状态筛选"><option value="ALL">全部</option><option value="OPEN">未确认</option><option value="ACKNOWLEDGED">已确认</option></select></div>
        <div v-if="filtered.length" class="alarm-list">
          <button v-for="alarm in filtered" :key="alarm.id" class="alarm-row" :class="{ 'alarm-row--selected': selectedId === alarm.id }" type="button" @click="selectedId = alarm.id">
            <span class="alarm-row__severity" :class="`alarm-row__severity--${alarm.severity.toLowerCase()}`">{{ alarm.severity }}</span>
            <span class="alarm-row__body"><strong>{{ alarm.alarm_type }}</strong><small>{{ deviceName(alarm.device_id) }} · ×{{ alarm.occurrence_count }}</small></span>
            <span class="alarm-row__status">{{ alarm.status }}</span>
          </button>
        </div>
        <div v-else class="empty-state"><strong>当前没有活动告警</strong><span>最近同步：{{ store.lastSyncAt ? new Date(store.lastSyncAt).toLocaleTimeString('zh-CN') : '—' }}</span></div>
      </section>
      <aside class="panel alarm-detail" aria-labelledby="alarm-detail-title">
        <div v-if="selected">
          <div class="section-heading"><div><p class="eyebrow">ALARM DETAIL</p><h2 id="alarm-detail-title">{{ selected.alarm_type }}</h2></div><StatusBadge :label="selected.severity" :tone="tone(selected.severity)" /></div>
          <p class="detail-lead">{{ selected.details?.recommendation ?? '检查设备状态、数据新鲜度和关联事件。' }}</p>
          <dl class="detail-list"><div><dt>设备</dt><dd>{{ deviceName(selected.device_id) }}</dd></div><div><dt>状态</dt><dd>{{ selected.status }}</dd></div><div><dt>首次发生</dt><dd>{{ new Date(selected.first_occurred_at).toLocaleString('zh-CN') }}</dd></div><div><dt>最近发生</dt><dd>{{ new Date(selected.last_occurred_at).toLocaleString('zh-CN') }}</dd></div><div><dt>重复次数</dt><dd class="tabular">{{ selected.occurrence_count }}</dd></div></dl>
          <div class="detail-context"><FreshnessIndicator :timestamp="store.devices[selected.device_id]?.server_time" :online="store.devices[selected.device_id]?.online" /></div>
          <div class="context-actions"><button v-if="selected.status === 'OPEN'" class="button button--primary" type="button" @click="store.acknowledgeAlarm(selected)">确认接手此告警</button><RouterLink class="button button--secondary" :to="`/app/${store.workspaceId}/overview`">查看设备上下文</RouterLink></div>
        </div>
        <div v-else class="empty-state">从左侧选择告警</div>
      </aside>
    </div>
  </div>
</template>
