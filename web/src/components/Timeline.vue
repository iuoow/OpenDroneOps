<script setup lang="ts">
import type { DomainEvent } from '../types/contracts'

defineProps<{ events: DomainEvent[] }>()

function label(eventType: string) {
  return {
    'alarm.updated': '告警更新',
    'alarm.created': '新告警',
    'alarm.resolved': '告警恢复',
    'command.updated': '指令进度',
    'device.offline': '设备离线',
    'device.online': '设备上线',
  }[eventType] ?? eventType
}

function timeLabel(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '未知时间' : date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <section class="timeline-panel" aria-labelledby="timeline-title">
    <div class="section-heading">
      <div>
        <p class="eyebrow">EVENT STREAM</p>
        <h2 id="timeline-title">最近关键事件</h2>
      </div>
      <span class="muted">遥测已合并</span>
    </div>
    <ol v-if="events.length" class="timeline">
      <li v-for="event in events.slice(0, 6)" :key="event.event_id" class="timeline__item">
        <span class="timeline__line" aria-hidden="true"></span>
        <div>
          <strong>{{ label(event.event_type) }}</strong>
          <p>{{ event.payload?.method ?? event.payload?.alarm_type ?? event.device_id ?? '系统事件' }}</p>
        </div>
        <time :datetime="event.occurred_at">{{ timeLabel(event.occurred_at) }}</time>
      </li>
    </ol>
    <div v-else class="empty-state">当前没有关键事件</div>
  </section>
</template>
