<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiClient } from '../api/client'
import AppIcon from '../components/AppIcon.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Device, DomainEvent, TrajectoryPoint } from '../types/contracts'
import { clampIndex, downsample, formatPlaybackTime, nextPlaybackIndex } from '../replay/playback'

const route = useRoute()
const router = useRouter()
const store = useOperationsStore()
const api = new ApiClient()
const points = ref<TrajectoryPoint[]>([])
const loading = ref(false)
const error = ref('')
const playing = ref(false)
const speed = ref(1)
const selectedIndex = ref(0)
const rangeHours = ref(Number(route.query.hours ?? 2))
let timer: number | undefined

const aircraft = computed(() => store.deviceList.filter((device) => device.device_type === 'AIRCRAFT'))
const selectedDeviceId = computed(() => String(route.params.deviceId ?? route.query.device ?? aircraft.value[0]?.id ?? ''))
const selectedDevice = computed<Device | undefined>(() => store.devices[selectedDeviceId.value])
const to = computed(() => new Date().toISOString())
const from = computed(() => new Date(Date.now() - rangeHours.value * 60 * 60 * 1000).toISOString())
const playbackPoint = computed(() => points.value[clampIndex(selectedIndex.value, points.value.length)])
const hasHistory = computed(() => points.value.length > 0)
const bounds = computed(() => {
  if (!points.value.length) return undefined
  const lats = points.value.map((point) => point.latitude)
  const lngs = points.value.map((point) => point.longitude)
  return { minLat: Math.min(...lats), maxLat: Math.max(...lats), minLng: Math.min(...lngs), maxLng: Math.max(...lngs) }
})
const path = computed(() => points.value.map((point) => {
  const position = project(point)
  return `${position.x.toFixed(1)},${position.y.toFixed(1)}`
}).join(' '))
const currentPosition = computed(() => playbackPoint.value ? project(playbackPoint.value) : undefined)
const pointWindow = computed(() => ({ from: points.value[0]?.occurred_at, to: points.value.at(-1)?.occurred_at }))
const events = computed<DomainEvent[]>(() => store.events
  .filter((event) => event.device_id === selectedDeviceId.value)
  .filter((event) => !pointWindow.value.from || !pointWindow.value.to || (event.occurred_at >= pointWindow.value.from && event.occurred_at <= pointWindow.value.to))
  .sort((a, b) => a.occurred_at.localeCompare(b.occurred_at)))
const eventMarkers = computed(() => events.value.map((event) => ({ event, position: project(points.value[nearestPointIndex(event.occurred_at)]) })))
const activeEvent = computed(() => events.value.find((event) => nearestPointIndex(event.occurred_at) === selectedIndex.value))

function project(point?: TrajectoryPoint) {
  if (!point || !bounds.value) return { x: 500, y: 260 }
  const latSpan = bounds.value.maxLat - bounds.value.minLat || 0.001
  const lngSpan = bounds.value.maxLng - bounds.value.minLng || 0.001
  return {
    x: 40 + ((point.longitude - bounds.value.minLng) / lngSpan) * 920,
    y: 460 - ((point.latitude - bounds.value.minLat) / latSpan) * 400,
  }
}

function nearestPointIndex(timestamp: string) {
  if (!points.value.length) return 0
  const target = new Date(timestamp).getTime()
  if (Number.isNaN(target)) return 0
  return points.value.reduce((nearest, point, index) => Math.abs(new Date(point.occurred_at).getTime() - target) < Math.abs(new Date(points.value[nearest].occurred_at).getTime() - target) ? index : nearest, 0)
}

function setDevice(device: Device) {
  void router.replace({ params: { ...route.params, deviceId: device.id }, query: { ...route.query, device: device.id } })
}

function demoPoints(device: Device) {
  const now = Date.now()
  return Array.from({ length: 60 }, (_, index) => {
    const angle = index / 8
    return {
      id: `demo-track-${device.id}-${index}`,
      workspace_id: device.workspace_id,
      device_id: device.id,
      occurred_at: new Date(now - (59 - index) * 60_000).toISOString(),
      received_at: new Date(now - (59 - index) * 60_000 + 1_000).toISOString(),
      latitude: (device.latitude ?? 31.23) + Math.sin(angle) * 0.003,
      longitude: (device.longitude ?? 121.47) + Math.cos(angle) * 0.004,
      altitude: (device.altitude ?? 60) + Math.sin(angle * 1.7) * 12,
      speed: 8 + Math.abs(Math.sin(angle)) * 4,
      heading: (angle * 57.3 + 90) % 360,
      battery_percent: Math.max(18, (device.battery_percent ?? 80) - index * 0.25),
    }
  })
}

async function loadTrack() {
  if (!selectedDeviceId.value) return
  loading.value = true
  error.value = ''
  playing.value = false
  selectedIndex.value = 0
  try {
    if (import.meta.env.VITE_DEMO_MODE !== 'false') {
      points.value = demoPoints(selectedDevice.value ?? aircraft.value[0])
      return
    }
    const page = await api.getTrajectory(store.workspaceId, selectedDeviceId.value, { from: from.value, to: to.value, limit: 5000 })
    points.value = page.items.length > 1000 ? await simplifyInWorker(page.items) : page.items
    if (page.truncated) error.value = '结果已按服务端上限截断；缩短时间范围后可继续调查。'
  } catch (cause) {
    points.value = []
    error.value = cause instanceof Error ? cause.message : '轨迹查询失败'
  } finally {
    loading.value = false
  }
}

function simplifyInWorker(items: TrajectoryPoint[]) {
  if (typeof Worker === 'undefined') return Promise.resolve(downsample(items))
  return new Promise<TrajectoryPoint[]>((resolve) => {
    const worker = new Worker(new URL('../replay/trajectory.worker.ts', import.meta.url), { type: 'module' })
    worker.onmessage = (event) => { worker.terminate(); resolve(event.data as TrajectoryPoint[]) }
    worker.postMessage(items)
  })
}

function togglePlayback() { playing.value = !playing.value }
function returnRealtime() { void router.push({ path: `/app/${store.workspaceId}/overview`, query: { device: selectedDeviceId.value } }) }
function timeLabel(value?: string) { return formatPlaybackTime(value) }
function eventLabel(event: DomainEvent) { return event.event_type === 'alarm.updated' ? '告警更新' : event.event_type === 'command.updated' ? '指令更新' : event.event_type === 'device.updated' ? '设备状态更新' : event.event_type }
function eventTone(event: DomainEvent) { return event.event_type.startsWith('alarm.') ? 'danger' : event.event_type.startsWith('command.') ? 'warning' : 'info' }
function selectEvent(event: DomainEvent) { selectedIndex.value = nearestPointIndex(event.occurred_at); playing.value = false }
function jumpEvent(direction: -1 | 1) {
  const current = playbackPoint.value?.occurred_at ?? ''
  const candidates = direction < 0 ? [...events.value].reverse().filter((event) => event.occurred_at < current) : events.value.filter((event) => event.occurred_at > current)
  if (candidates[0]) selectEvent(candidates[0])
}

watch([selectedDeviceId, rangeHours], () => {
  void loadTrack()
  void router.replace({ query: { ...route.query, device: selectedDeviceId.value, hours: String(rangeHours.value) } })
})
watch(playing, (value) => {
  if (!value) { if (timer) window.clearInterval(timer); timer = undefined; return }
  timer = window.setInterval(() => {
    const next = nextPlaybackIndex(selectedIndex.value, points.value.length, speed.value)
    selectedIndex.value = next
    if (next >= points.value.length - 1) playing.value = false
  }, 250)
})
onMounted(() => void loadTrack())
onUnmounted(() => { if (timer) window.clearInterval(timer) })
</script>

<template>
  <div class="page replay-v2">
    <div class="page-heading replay-v2__heading">
      <div><p class="eyebrow">HISTORY / REPLAY</p><h1>轨迹回放</h1><p class="page-heading__summary">这是只读历史证据：轨迹、事件和遥测都锁定在同一回放时刻，不与实时状态混合。</p></div>
      <div class="replay-mode-summary"><div><strong>历史模式</strong><span>{{ selectedDevice?.serial_number ?? '未选择设备' }}</span></div><button class="button button--secondary" type="button" @click="returnRealtime"><AppIcon name="overview" :size="15" />返回实时</button></div>
    </div>

    <section class="replay-toolbar-v2" aria-label="轨迹查询范围">
      <label class="field"><span>设备</span><select :value="selectedDeviceId" @change="setDevice(aircraft.find((item) => item.id === ($event.target as HTMLSelectElement).value)!)"><option v-for="device in aircraft" :key="device.id" :value="device.id">{{ device.serial_number }}</option></select></label>
      <label class="field"><span>查询范围</span><select v-model.number="rangeHours"><option :value="0.5">最近 30 分钟</option><option :value="2">最近 2 小时</option><option :value="6">最近 6 小时</option><option :value="24">最近 24 小时</option></select></label>
      <div class="replay-toolbar-v2__evidence"><StatusBadge :label="loading ? '加载中' : hasHistory ? '历史证据' : '无轨迹'" :tone="loading ? 'warning' : hasHistory ? 'info' : 'offline'" /><span v-if="hasHistory">{{ timeLabel(pointWindow.from) }} — {{ timeLabel(pointWindow.to) }}</span><span v-else>查询范围内无加载点</span></div>
    </section>

    <div v-if="error" class="global-notice" role="alert">{{ error }}</div>
    <div class="replay-v2__workspace">
      <section class="replay-map-v2" aria-labelledby="replay-map-title">
        <div class="replay-map-v2__header"><div><p class="eyebrow">TRAJECTORY EVIDENCE</p><h2 id="replay-map-title">路径与时间位置</h2></div><div class="replay-map-v2__facts"><span>{{ points.length }} 个证据点</span><span>{{ events.length }} 个关联事件</span></div></div>
        <svg class="trajectory-canvas trajectory-canvas--v2" viewBox="0 0 1000 520" role="img" aria-labelledby="replay-map-title replay-map-caption">
          <title id="replay-map-caption">设备历史轨迹、起止点、当前回放位置和关联事件</title>
          <path v-if="path" :d="`M ${path.replaceAll(' ', ' L ')}`" class="trajectory-path" />
          <circle v-if="points[0]" :cx="project(points[0]).x" :cy="project(points[0]).y" r="8" class="trajectory-start" /><circle v-if="points.at(-1)" :cx="project(points.at(-1)).x" :cy="project(points.at(-1)).y" r="8" class="trajectory-end" />
          <g v-for="marker in eventMarkers" :key="marker.event.event_id" class="trajectory-event" :transform="`translate(${marker.position.x} ${marker.position.y})`" role="button" tabindex="0" :aria-label="`跳转至${eventLabel(marker.event)}：${timeLabel(marker.event.occurred_at)}`" @click="selectEvent(marker.event)" @keydown.enter.prevent="selectEvent(marker.event)" @keydown.space.prevent="selectEvent(marker.event)"><circle r="9" /><path d="M0-4v5M0 4.5h.01" /></g>
          <circle v-if="currentPosition" :cx="currentPosition.x" :cy="currentPosition.y" r="10" class="trajectory-current" />
          <text v-if="!hasHistory" x="500" y="260" text-anchor="middle" class="trajectory-empty">当前时间范围无轨迹</text>
        </svg>
        <div class="replay-map-v2__caption"><span><i class="trajectory-key trajectory-key--start"></i>起点</span><span><i class="trajectory-key trajectory-key--current"></i>回放时刻</span><span><i class="trajectory-key trajectory-key--event"></i>关联事件</span><time :datetime="playbackPoint?.occurred_at">{{ timeLabel(playbackPoint?.occurred_at) }}</time></div>
      </section>

      <aside class="replay-evidence-v2" aria-labelledby="replay-evidence-title">
        <div class="replay-evidence-v2__header"><div><p class="eyebrow">PLAYBACK EVIDENCE</p><h2 id="replay-evidence-title">选中时刻</h2></div><StatusBadge :label="activeEvent ? eventLabel(activeEvent) : '遥测样本'" :tone="activeEvent ? eventTone(activeEvent) : 'info'" /></div>
        <p class="replay-evidence-v2__time">{{ timeLabel(playbackPoint?.occurred_at) }}</p>
        <dl class="replay-evidence-grid"><div><dt>高度</dt><dd>{{ playbackPoint?.altitude?.toFixed(1) ?? '—' }}<small v-if="playbackPoint?.altitude != null"> m</small></dd></div><div><dt>速度</dt><dd>{{ playbackPoint?.speed?.toFixed(1) ?? '—' }}<small v-if="playbackPoint?.speed != null"> m/s</small></dd></div><div><dt>电量</dt><dd>{{ playbackPoint?.battery_percent?.toFixed(0) ?? '—' }}<small v-if="playbackPoint?.battery_percent != null">%</small></dd></div><div><dt>航向</dt><dd>{{ playbackPoint?.heading?.toFixed(0) ?? '—' }}<small v-if="playbackPoint?.heading != null">°</small></dd></div></dl>
        <section class="replay-event-list" aria-labelledby="replay-event-title"><div class="section-heading section-heading--compact"><h3 id="replay-event-title">关联事件</h3><span>点击可定位</span></div><div v-if="events.length" class="replay-event-list__items"><button v-for="event in events.slice(0, 6)" :key="event.event_id" class="replay-event-card" :class="{ 'replay-event-card--active': activeEvent?.event_id === event.event_id }" type="button" @click="selectEvent(event)"><StatusBadge :label="eventLabel(event)" :tone="eventTone(event)" /><span><strong>{{ timeLabel(event.occurred_at) }}</strong><small>{{ event.payload?.alarm_type ?? event.payload?.method ?? event.event_id }}</small></span></button></div><p v-else class="replay-event-list__empty">当前轨迹窗口没有关联事件。</p></section>
        <section class="replay-boundary"><strong>证据边界</strong><span>{{ hasHistory ? `显示已加载的 ${points.length} 个轨迹点；点间状态不会被推断。` : '当前没有可用轨迹点，因此不展示位置或遥测推断。' }}</span></section>
      </aside>
    </div>

    <section class="replay-controls-v2" aria-label="回放控制"><div class="replay-controls-v2__primary"><button class="button button--primary" type="button" :disabled="!hasHistory" @click="togglePlayback"><AppIcon :name="playing ? 'close' : 'replay'" :size="15" />{{ playing ? '暂停回放' : '播放回放' }}</button><button class="button button--secondary" type="button" :disabled="!events.length" @click="jumpEvent(-1)">上一事件</button><button class="button button--secondary" type="button" :disabled="!events.length" @click="jumpEvent(1)">下一事件</button><span class="replay-speed"><button v-for="value in [0.5, 1, 2, 4]" :key="value" :class="{ 'replay-speed__item--active': speed === value }" type="button" @click="speed = value">{{ value }}×</button></span></div><label class="replay-slider"><span class="sr-only">回放时间</span><input v-model.number="selectedIndex" type="range" min="0" :max="Math.max(points.length - 1, 0)" :disabled="!hasHistory" /><span>{{ timeLabel(playbackPoint?.occurred_at) }}</span></label><span class="replay-controls-v2__count">{{ points.length ? selectedIndex + 1 : 0 }} / {{ points.length }}</span></section>
  </div>
</template>
