<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiClient } from '../api/client'
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
const events = computed<DomainEvent[]>(() =>
  store.events.filter((event) => event.device_id === selectedDeviceId.value).sort((a, b) => a.occurred_at.localeCompare(b.occurred_at)),
)
const path = computed(() => {
  if (!points.value.length) return ''
  const lats = points.value.map((point) => point.latitude)
  const lngs = points.value.map((point) => point.longitude)
  const minLat = Math.min(...lats)
  const maxLat = Math.max(...lats)
  const minLng = Math.min(...lngs)
  const maxLng = Math.max(...lngs)
  const latSpan = maxLat - minLat || 0.001
  const lngSpan = maxLng - minLng || 0.001
  return points.value.map((point) => {
    const x = 40 + ((point.longitude - minLng) / lngSpan) * 920
    const y = 460 - ((point.latitude - minLat) / latSpan) * 400
    return `${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
})
const currentPosition = computed(() => {
  if (!playbackPoint.value || !points.value.length) return ''
  const first = points.value[0]
  const last = points.value.at(-1) ?? first
  const x = 40 + ((playbackPoint.value.longitude - first.longitude) / ((last.longitude - first.longitude) || 0.001)) * 920
  const y = 460 - ((playbackPoint.value.latitude - first.latitude) / ((last.latitude - first.latitude) || 0.001)) * 400
  return `${Math.max(20, Math.min(980, x)).toFixed(1)},${Math.max(20, Math.min(480, y)).toFixed(1)}`
})
const hasHistory = computed(() => points.value.length > 0)

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
    const page = await api.getTrajectory(store.workspaceId, selectedDeviceId.value, {
      from: from.value,
      to: to.value,
      limit: 5000,
    })
    points.value = page.items.length > 1000 ? await simplifyInWorker(page.items) : page.items
    if (page.truncated) error.value = '结果已按服务端上限截断，可缩短时间范围后继续查询。'
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
    worker.onmessage = (event) => {
      worker.terminate()
      resolve(event.data as TrajectoryPoint[])
    }
    worker.postMessage(items)
  })
}

function togglePlayback() {
  playing.value = !playing.value
}

function returnRealtime() {
  void router.push({ path: `/app/${store.workspaceId}/overview`, query: { device: selectedDeviceId.value } })
}

function timeLabel(value?: string) {
  return formatPlaybackTime(value)
}

watch([selectedDeviceId, rangeHours], () => {
  void loadTrack()
  void router.replace({ query: { ...route.query, device: selectedDeviceId.value, hours: String(rangeHours.value) } })
})
watch(playing, (value) => {
  if (!value) {
    if (timer) window.clearInterval(timer)
    timer = undefined
    return
  }
  timer = window.setInterval(() => {
    const next = nextPlaybackIndex(selectedIndex.value, points.value.length, speed.value)
    selectedIndex.value = next
    if (next >= points.value.length - 1) playing.value = false
  }, 250)
})
onMounted(() => void loadTrack())
onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="page page--replay">
    <div class="page-heading">
      <div>
        <p class="eyebrow">HISTORY / REPLAY</p>
        <h1>轨迹回放</h1>
        <p class="page-heading__summary">历史模式与实时状态严格分离；地图、指标和事件共享同一 playbackTime。</p>
      </div>
      <button class="button button--secondary" type="button" @click="returnRealtime">返回实时</button>
    </div>

    <div class="replay-toolbar panel">
      <label class="field"><span>设备</span><select :value="selectedDeviceId" @change="setDevice(aircraft.find((item) => item.id === ($event.target as HTMLSelectElement).value)!)"><option v-for="device in aircraft" :key="device.id" :value="device.id">{{ device.serial_number }}</option></select></label>
      <label class="field"><span>时间范围</span><select v-model.number="rangeHours"><option :value="0.5">最近 30 分钟</option><option :value="2">最近 2 小时</option><option :value="6">最近 6 小时</option><option :value="24">最近 24 小时</option></select></label>
      <div class="replay-toolbar__state"><StatusBadge :label="loading ? '加载中' : hasHistory ? '历史模式' : '无轨迹'" :tone="loading ? 'warning' : hasHistory ? 'info' : 'offline'" /><span class="muted">{{ selectedDevice?.serial_number ?? '未选择设备' }}</span></div>
    </div>

    <div v-if="error" class="global-notice" role="alert">{{ error }}</div>
    <div class="replay-grid">
      <section class="replay-map panel" aria-labelledby="replay-map-title">
        <div class="section-heading"><div><p class="eyebrow">MAP / TRAJECTORY</p><h2 id="replay-map-title">路径与当前位置</h2></div><span class="muted">{{ points.length }} 点{{ points.length > 1000 ? '（已降采样）' : '' }}</span></div>
        <svg class="trajectory-canvas" viewBox="0 0 1000 520" role="img" aria-labelledby="replay-map-title replay-map-caption">
          <title id="replay-map-caption">设备历史轨迹地图</title>
          <path v-if="path" :d="`M ${path.replaceAll(' ', ' L ')}`" class="trajectory-path" />
          <circle v-if="currentPosition" :cx="Number(currentPosition.split(',')[0])" :cy="Number(currentPosition.split(',')[1])" r="10" class="trajectory-current" />
          <text v-if="!hasHistory" x="500" y="260" text-anchor="middle" class="trajectory-empty">当前时间范围无轨迹</text>
        </svg>
        <p class="muted">时间：<time :datetime="playbackPoint?.occurred_at">{{ timeLabel(playbackPoint?.occurred_at) }}</time></p>
      </section>

      <aside class="replay-side panel" aria-label="回放指标">
        <div class="section-heading"><div><p class="eyebrow">TELEMETRY</p><h2>当前时刻</h2></div><span class="code-text">{{ playbackPoint?.id ?? '—' }}</span></div>
        <dl class="metric-grid">
          <div><dt>高度</dt><dd>{{ playbackPoint?.altitude?.toFixed(1) ?? '—' }}<small v-if="playbackPoint?.altitude != null"> m</small></dd></div>
          <div><dt>速度</dt><dd>{{ playbackPoint?.speed?.toFixed(1) ?? '—' }}<small v-if="playbackPoint?.speed != null"> m/s</small></dd></div>
          <div><dt>电量</dt><dd>{{ playbackPoint?.battery_percent?.toFixed(0) ?? '—' }}<small v-if="playbackPoint?.battery_percent != null">%</small></dd></div>
          <div><dt>航向</dt><dd>{{ playbackPoint?.heading?.toFixed(0) ?? '—' }}<small v-if="playbackPoint?.heading != null">°</small></dd></div>
        </dl>
        <div class="context-section"><div class="section-heading section-heading--compact"><h3>事件同步</h3><span>{{ events.length }}</span></div><ol class="replay-events"><li v-for="event in events.slice(0, 5)" :key="event.event_id"><strong>{{ event.event_type }}</strong><time :datetime="event.occurred_at">{{ timeLabel(event.occurred_at) }}</time></li><li v-if="!events.length" class="muted">该时间范围没有关键事件</li></ol></div>
      </aside>
    </div>

    <section class="replay-controls panel" aria-label="播放控制">
      <div class="replay-controls__buttons"><button class="button button--primary" type="button" @click="togglePlayback">{{ playing ? '暂停' : '播放' }}</button><button v-for="value in [0.5, 1, 2, 4]" :key="value" class="button button--secondary" :class="{ 'button--selected': speed === value }" type="button" @click="speed = value">{{ value }}×</button></div>
      <label class="replay-slider"><span class="sr-only">回放时间</span><input v-model.number="selectedIndex" type="range" min="0" :max="Math.max(points.length - 1, 0)" :disabled="!hasHistory" /><span>{{ timeLabel(playbackPoint?.occurred_at) }}</span></label>
      <span class="muted">{{ selectedIndex + 1 }} / {{ points.length || 0 }}</span>
    </section>
  </div>
</template>
