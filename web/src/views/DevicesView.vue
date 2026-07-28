<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Alarm, Device } from '../types/contracts'

const store = useOperationsStore()
const route = useRoute()
const router = useRouter()
const query = ref('')
const filter = ref<'ALL' | 'ONLINE' | 'STALE' | 'OFFLINE'>('ALL')
const typeFilter = ref<'ALL' | Device['device_type']>('ALL')
const selectedId = ref(String(route.query.device ?? store.deviceList[0]?.id ?? ''))
const selected = computed<Device | undefined>(() => store.devices[selectedId.value ?? ''])

const devices = computed(() => store.deviceList
  .filter((device) => {
    const matchesQuery = `${device.serial_number} ${device.product_model ?? ''} ${device.device_type}`.toLowerCase().includes(query.value.trim().toLowerCase())
    const matchesStatus = filter.value === 'ALL' || deviceHealth(device) === filter.value
    const matchesType = typeFilter.value === 'ALL' || device.device_type === typeFilter.value
    return matchesQuery && matchesStatus && matchesType
  })
  .sort((left, right) => attentionRank(right) - attentionRank(left) || left.serial_number.localeCompare(right.serial_number)),
)
const onlineCount = computed(() => store.deviceList.filter((device) => deviceHealth(device) === 'ONLINE').length)
const staleCount = computed(() => store.deviceList.filter((device) => deviceHealth(device) === 'STALE').length)
const offlineCount = computed(() => store.deviceList.filter((device) => deviceHealth(device) === 'OFFLINE').length)
const selectedAlarms = computed(() => store.activeAlarms.filter((alarm) => alarm.device_id === selected.value?.id))
const selectedCommands = computed(() => store.commandList.filter((command) => command.target_device_id === selected.value?.id).slice(0, 3))

function deviceHealth(device: Device) {
  if (device.online === false || device.status === 'OFFLINE') return 'OFFLINE'
  if (!device.server_time) return device.status === 'ONLINE' ? 'STALE' : 'OFFLINE'
  const age = Date.now() - new Date(device.server_time).getTime()
  return Number.isFinite(age) && age > 60_000 ? 'STALE' : 'ONLINE'
}

function healthLabel(device: Device) {
  const health = deviceHealth(device)
  return health === 'ONLINE' ? '在线' : health === 'STALE' ? '数据过期' : device.status === 'REGISTERED' ? '待上线' : '离线'
}

function healthTone(device: Device) {
  const health = deviceHealth(device)
  return health === 'ONLINE' ? 'success' : health === 'STALE' ? 'warning' : 'offline'
}

function attentionRank(device: Device) {
  const alarm = store.activeAlarms.find((item) => item.device_id === device.id)
  if (alarm?.severity === 'CRITICAL') return 5
  if (alarm) return 4
  if (deviceHealth(device) === 'OFFLINE') return 3
  if (deviceHealth(device) === 'STALE') return 2
  if ((device.battery_percent ?? 100) < 20) return 1
  return 0
}

function deviceFact(device: Device) {
  const health = deviceHealth(device)
  if (health === 'OFFLINE') return device.status === 'REGISTERED' ? '设备已注册，但尚无在线遥测证据。' : '设备当前离线；请结合最后上报时间判断其状态。'
  if (health === 'STALE') return '设备曾在线，但遥测数据可能已过期；不应将其当作实时状态。'
  if ((device.battery_percent ?? 100) < 20) return '设备在线，但电量偏低；请优先核对活动告警与当前模式。'
  return '设备在线，遥测仍在新鲜度窗口内。'
}

function alarmLabel(alarm: Alarm) {
  return alarm.alarm_type === 'battery.low' ? '低电量' : alarm.alarm_type
}

function severityTone(severity: Alarm['severity']) {
  return severity === 'CRITICAL' ? 'danger' : severity === 'WARNING' ? 'warning' : 'info'
}

function dateTime(value?: string) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString('zh-CN', { hour12: false })
}

function commandLabel(status: string) {
  const labels: Record<string, string> = { SUCCEEDED: '结果已确认', PUBLISHED: '已发布', ACCEPTED: '设备已接收', EXECUTING: '执行中', PUBLISH_PENDING: '等待发布', TIMEOUT: '已超时', FAILED: '执行失败', REJECTED: '已拒绝', CANCELED: '已取消', CREATED: '已创建', VALIDATED: '已校验' }
  return labels[status] ?? status
}

function selectDevice(device: Device) {
  selectedId.value = device.id
  void router.replace({ query: { ...route.query, device: device.id } })
}
</script>

<template>
  <div class="page devices-v2">
    <div class="page-heading devices-v2__heading">
      <div>
        <p class="eyebrow">INVENTORY / DEVICES</p>
        <h1>设备管理</h1>
        <p class="page-heading__summary">从设备清单直接判断在线状态、遥测新鲜度、风险线索与可继续调查的上下文。</p>
      </div>
      <div class="device-summary" aria-label="设备状态摘要">
        <div><strong class="text-success">{{ onlineCount }}</strong><span>在线</span></div>
        <div><strong class="text-warning">{{ staleCount }}</strong><span>数据过期</span></div>
        <div><strong>{{ offlineCount }}</strong><span>离线或待上线</span></div>
      </div>
    </div>

    <div class="devices-v2__workspace">
      <section class="device-queue-v2" aria-labelledby="device-queue-title">
        <div class="device-queue-v2__header">
          <div><p class="eyebrow">FLEET INVENTORY</p><h2 id="device-queue-title">设备清单</h2></div>
          <span class="muted">{{ devices.length }} / {{ store.deviceList.length }}</span>
        </div>
        <div class="device-toolbar">
          <label class="device-search"><AppIcon name="devices" :size="16" /><span class="sr-only">搜索设备</span><input v-model="query" placeholder="搜索序列号、型号或类型" /></label>
          <label class="device-type-select"><span class="sr-only">设备类型</span><select v-model="typeFilter"><option value="ALL">全部类型</option><option value="AIRCRAFT">飞行器</option><option value="GATEWAY">Dock / 网关</option><option value="UNKNOWN">未知类型</option></select></label>
        </div>
        <div class="device-filter" role="tablist" aria-label="设备状态筛选">
          <button v-for="item in ([['ALL', '全部'], ['ONLINE', '在线'], ['STALE', '过期'], ['OFFLINE', '离线']] as const)" :key="item[0]" class="device-filter__item" :class="{ 'device-filter__item--active': filter === item[0] }" type="button" role="tab" :aria-selected="filter === item[0]" @click="filter = item[0]">{{ item[1] }}<span>{{ item[0] === 'ALL' ? store.deviceList.length : item[0] === 'ONLINE' ? onlineCount : item[0] === 'STALE' ? staleCount : offlineCount }}</span></button>
        </div>
        <div v-if="devices.length" class="device-list-v2">
          <button v-for="device in devices" :key="device.id" class="device-card" :class="[{ 'device-card--selected': selectedId === device.id }, `device-card--${healthTone(device)}`]" type="button" @click="selectDevice(device)">
            <span class="device-card__icon"><AppIcon :name="device.device_type === 'GATEWAY' ? 'dock' : 'aircraft'" :size="18" /></span>
            <span class="device-card__body">
              <span class="device-card__meta"><StatusBadge :label="healthLabel(device)" :tone="healthTone(device)" /><span>{{ device.device_type === 'GATEWAY' ? 'DOCK / 网关' : '飞行器' }}</span></span>
              <strong>{{ device.serial_number }}</strong>
              <small>{{ device.product_model ?? '未知型号' }} · {{ device.mode ?? '暂无模式' }}</small>
            </span>
            <span class="device-card__metrics"><span>{{ device.battery_percent ?? '—' }}<small v-if="device.battery_percent !== null">%</small></span><small>电量</small></span>
          </button>
        </div>
        <div v-else class="empty-state device-queue-v2__empty"><AppIcon name="devices" :size="28" /><strong>{{ store.deviceList.length ? '没有符合筛选条件的设备' : '尚未接入设备' }}</strong><span>{{ store.deviceList.length ? '调整搜索或状态筛选，继续查找设备。' : '可启动协议模拟器，或按接入指南连接受支持设备。' }}</span></div>
      </section>

      <aside class="device-detail-v2" aria-labelledby="device-detail-title">
        <template v-if="selected">
          <div class="device-detail-v2__header" :class="`device-detail-v2__header--${healthTone(selected)}`">
            <div class="device-detail-v2__title"><span class="device-detail-v2__signal"><AppIcon :name="selected.device_type === 'GATEWAY' ? 'dock' : 'aircraft'" :size="22" /></span><div><p class="eyebrow">DEVICE CONTEXT</p><h2 id="device-detail-title">{{ selected.serial_number }}</h2><p>{{ selected.product_model ?? '未知型号' }} · {{ selected.device_type === 'GATEWAY' ? 'Dock / 网关' : '飞行器' }}</p></div></div>
            <StatusBadge :label="healthLabel(selected)" :tone="healthTone(selected)" />
          </div>

          <section class="device-state-note" :class="`device-state-note--${healthTone(selected)}`" aria-label="设备当前事实"><span>当前事实</span><p>{{ deviceFact(selected) }}</p></section>

          <section class="device-evidence" aria-labelledby="device-evidence-title">
            <div class="section-heading section-heading--compact"><h3 id="device-evidence-title">最新证据</h3><span>状态与遥测分开呈现</span></div>
            <dl class="device-evidence__grid">
              <div><dt>遥测新鲜度</dt><dd><FreshnessIndicator :timestamp="selected.server_time" :online="selected.online" /></dd></div>
              <div><dt>最后上报</dt><dd>{{ dateTime(selected.server_time ?? selected.updated_at) }}</dd></div>
              <div><dt>电量</dt><dd>{{ selected.battery_percent ?? '—' }}<small v-if="selected.battery_percent !== null">%</small></dd></div>
              <div><dt>当前模式</dt><dd>{{ selected.mode ?? '—' }}</dd></div>
              <div><dt>高度</dt><dd>{{ selected.altitude ?? '—' }}<small v-if="selected.altitude !== null"> m</small></dd></div>
              <div><dt>状态版本</dt><dd>{{ selected.state_version ?? '—' }}</dd></div>
            </dl>
          </section>

          <section class="device-related" aria-labelledby="device-related-title">
            <div class="section-heading section-heading--compact"><h3 id="device-related-title">关联风险与指令</h3><span>{{ selectedAlarms.length }} 告警 · {{ selectedCommands.length }} 指令</span></div>
            <div v-if="selectedAlarms.length" class="device-related__alarms"><RouterLink v-for="alarm in selectedAlarms" :key="alarm.id" class="device-related__row" :to="`/app/${store.workspaceId}/alarms`"><StatusBadge :label="alarm.severity === 'CRITICAL' ? '严重' : alarm.severity === 'WARNING' ? '警告' : '信息'" :tone="severityTone(alarm.severity)" /><span><strong>{{ alarmLabel(alarm) }}</strong><small>最近 {{ dateTime(alarm.last_occurred_at) }}</small></span></RouterLink></div>
            <p v-else class="device-related__empty">当前没有活动告警。</p>
            <div v-if="selectedCommands.length" class="device-related__commands"><RouterLink v-for="command in selectedCommands" :key="command.id" class="device-related__row" :to="`/app/${store.workspaceId}/commands`"><AppIcon name="commands" :size="16" /><span><strong>{{ command.method }}</strong><small>{{ commandLabel(command.status) }} · {{ dateTime(command.updated_at ?? command.created_at) }}</small></span></RouterLink></div>
          </section>

          <div class="device-detail-v2__actions"><RouterLink class="button button--primary" :to="{ path: `/app/${store.workspaceId}/overview`, query: { device: selected.id } }"><AppIcon name="pin" :size="15" />查看实时态势</RouterLink><RouterLink v-if="selected.device_type === 'AIRCRAFT'" class="button button--secondary" :to="`/app/${store.workspaceId}/replay/${selected.id}`"><AppIcon name="replay" :size="15" />查看轨迹</RouterLink><span class="device-action-hint">未提供飞行、Dock 或批量控制操作。</span></div>
        </template>
        <div v-else class="empty-state"><AppIcon name="devices" :size="30" /><strong>从左侧选择设备</strong><span>最新遥测、风险线索和后续调查入口会显示在这里。</span></div>
      </aside>
    </div>
  </div>
</template>
