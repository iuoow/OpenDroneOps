<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import OperationsMap from '../components/OperationsMap.vue'
import StatusBadge from '../components/StatusBadge.vue'
import Timeline from '../components/Timeline.vue'
import { useOperationsStore } from '../state/operations'
import type { Alarm, Device } from '../types/contracts'

const store = useOperationsStore()
const route = useRoute()
const router = useRouter()
const selectedId = ref(String(route.query.device ?? store.deviceList[0]?.id ?? ''))
const attentionCollapsed = ref(false)

const selectedDevice = computed<Device | undefined>(() => store.devices[selectedId.value ?? ''])
const selectedAlarms = computed(() => store.activeAlarms.filter((alarm) => alarm.device_id === selectedDevice.value?.id))
const attentionAlarms = computed(() => store.activeAlarms.slice(0, 4))
const attentionDeviceIds = computed(() => attentionAlarms.value.map((alarm) => alarm.device_id))
const railDevices = computed(() =>
  [...store.deviceList].sort((left, right) => {
    if (left.online === false && right.online !== false) return -1
    if (right.online === false && left.online !== false) return 1
    return (left.battery_percent ?? 101) - (right.battery_percent ?? 101)
  }),
)

function selectDevice(device: Device) {
  selectedId.value = device.id
  void router.replace({ query: { ...route.query, device: device.id } })
}

function selectAlarm(alarm: Alarm) {
  const device = store.devices[alarm.device_id]
  if (device) selectDevice(device)
}

function severityTone(severity: string) {
  return severity === 'CRITICAL' ? 'danger' : severity === 'WARNING' ? 'warning' : 'info'
}

function deviceTone(device: Device) {
  if (device.online === false || device.status === 'OFFLINE') return 'offline'
  if ((device.battery_percent ?? 100) < 20) return 'warning'
  return 'success'
}

function devicePresence(device: Device) {
  return device.online === false || device.status === 'OFFLINE' ? '离线' : '在线'
}

function alarmLabel(alarm: Alarm) {
  return alarm.alarm_type === 'battery.low' ? '低电量' : alarm.alarm_type
}
</script>

<template>
  <div class="page page--overview overview-v2">
    <div class="page-heading page-heading--overview overview-v2__heading">
      <div>
        <p class="eyebrow">OPERATIONS / LIVE</p>
        <h1>实时态势</h1>
        <p class="page-heading__summary">聚焦正在发生的风险、空间位置与证据新鲜度。</p>
      </div>
      <div class="summary-strip" aria-label="态势摘要">
        <div><strong>{{ store.onlineCount }}/{{ store.deviceList.length }}</strong><span>在线设备</span></div>
        <div><strong class="text-danger">{{ store.criticalCount }}</strong><span>严重告警</span></div>
        <div><strong>{{ store.activeAlarms.length }}</strong><span>待关注</span></div>
      </div>
    </div>

    <div class="overview-v2__workspace" :class="{ 'overview-v2__workspace--rail-collapsed': attentionCollapsed }">
      <aside class="attention-rail" :class="{ 'attention-rail--collapsed': attentionCollapsed }" aria-label="关注队列">
        <div class="attention-rail__header">
          <div v-if="!attentionCollapsed">
            <p class="eyebrow">ATTENTION</p>
            <h2>需要关注</h2>
          </div>
          <button class="icon-button" type="button" :aria-label="attentionCollapsed ? '展开关注队列' : '收起关注队列'" @click="attentionCollapsed = !attentionCollapsed">
            <AppIcon :name="attentionCollapsed ? 'more' : 'close'" />
          </button>
        </div>
        <template v-if="!attentionCollapsed">
          <div v-if="attentionAlarms.length" class="attention-rail__alarms">
            <button v-for="alarm in attentionAlarms" :key="alarm.id" class="attention-alarm" :class="`attention-alarm--${severityTone(alarm.severity)}`" type="button" @click="selectAlarm(alarm)">
              <StatusBadge :label="alarm.severity === 'CRITICAL' ? '严重' : '警告'" :tone="severityTone(alarm.severity)" />
              <strong>{{ alarmLabel(alarm) }}</strong>
              <span>{{ store.devices[alarm.device_id]?.serial_number ?? '未知设备' }} · {{ alarm.occurrence_count }} 次</span>
            </button>
          </div>
          <div v-else class="attention-rail__empty">
            <AppIcon name="alarms" :size="20" />
            <span>当前没有待处理告警</span>
          </div>

          <div class="attention-rail__section-heading">
            <span>设备列表</span>
            <span>{{ store.deviceList.length }}</span>
          </div>
          <div class="attention-device-list" aria-label="设备等价列表">
            <button v-for="device in railDevices" :key="device.id" class="attention-device" :class="{ 'attention-device--selected': selectedId === device.id }" type="button" @click="selectDevice(device)">
              <span class="attention-device__icon" :class="`attention-device__icon--${deviceTone(device)}`"><AppIcon :name="device.device_type === 'GATEWAY' ? 'dock' : 'aircraft'" :size="17" /></span>
              <span><strong>{{ device.serial_number }}</strong><small>{{ device.mode ?? '暂无模式' }}</small></span>
              <span class="attention-device__battery">{{ device.battery_percent ?? '—' }}<small v-if="device.battery_percent !== null">%</small></span>
            </button>
          </div>
        </template>
      </aside>

      <OperationsMap :devices="store.deviceList" :selected-id="selectedId" :attention-device-ids="attentionDeviceIds" @select="selectDevice" />

      <aside class="context-panel context-panel--preview" aria-labelledby="context-title">
        <div class="section-heading">
          <div>
            <p class="eyebrow">PREVIEW</p>
            <h2 id="context-title">设备预览</h2>
          </div>
          <span class="muted">{{ selectedDevice ? '已选择' : '未选择' }}</span>
        </div>
        <div v-if="selectedDevice" class="device-context">
          <div class="device-context__heading">
            <div>
              <span class="device-type">{{ selectedDevice.device_type }}</span>
              <h3>{{ selectedDevice.serial_number }}</h3>
              <p>{{ selectedDevice.product_model ?? '未知型号' }}</p>
            </div>
            <StatusBadge :label="devicePresence(selectedDevice)" :tone="deviceTone(selectedDevice)" />
          </div>
          <FreshnessIndicator :timestamp="selectedDevice.server_time" :online="selectedDevice.online" />
          <dl class="metric-grid">
            <div><dt>电量</dt><dd>{{ selectedDevice.battery_percent ?? '—' }}<small v-if="selectedDevice.battery_percent !== null">%</small></dd></div>
            <div><dt>高度</dt><dd>{{ selectedDevice.altitude ?? '—' }}<small v-if="selectedDevice.altitude !== null"> m</small></dd></div>
            <div><dt>模式</dt><dd>{{ selectedDevice.mode ?? '—' }}</dd></div>
            <div><dt>状态版本</dt><dd>{{ selectedDevice.state_version ?? '—' }}</dd></div>
          </dl>
          <div class="context-section context-section--health">
            <div class="section-heading section-heading--compact"><h3>健康与告警</h3><span>{{ selectedAlarms.length }}</span></div>
            <div v-if="selectedAlarms.length" class="alarm-mini-list">
              <button v-for="alarm in selectedAlarms" :key="alarm.id" class="alarm-mini" type="button" @click="selectAlarm(alarm)">
                <StatusBadge :label="alarm.severity === 'CRITICAL' ? '严重' : '警告'" :tone="severityTone(alarm.severity)" />
                <span>{{ alarmLabel(alarm) }}</span>
                <strong>×{{ alarm.occurrence_count }}</strong>
              </button>
            </div>
            <p v-else class="muted">当前设备没有活动告警</p>
          </div>
          <div class="context-actions">
            <RouterLink class="button button--primary" :to="`/app/${store.workspaceId}/replay/${selectedDevice.id}`">查看轨迹</RouterLink>
            <button class="button button--secondary" type="button" @click="store.createStatusRefresh(selectedDevice)"><AppIcon name="refresh" :size="15" />刷新状态</button>
            <button class="button button--quiet" type="button" aria-label="更多安全操作"><AppIcon name="more" :size="18" /></button>
          </div>
        </div>
        <div v-else class="empty-state">从地图或左侧设备列表选择对象以查看证据。</div>
      </aside>
    </div>
    <Timeline :events="store.events" />
  </div>
</template>
