<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import OperationsMap from '../components/OperationsMap.vue'
import StatusBadge from '../components/StatusBadge.vue'
import Timeline from '../components/Timeline.vue'
import { useOperationsStore } from '../state/operations'
import type { Device } from '../types/contracts'

const store = useOperationsStore()
const route = useRoute()
const router = useRouter()
const selectedId = ref(String(route.query.device ?? store.deviceList[0]?.id ?? ''))
const selectedDevice = computed<Device | undefined>(() => store.devices[selectedId.value ?? ''])

function selectDevice(device: Device) {
  selectedId.value = device.id
  void router.replace({ query: { ...route.query, device: device.id } })
}

function severityTone(severity: string) {
  return severity === 'CRITICAL' ? 'danger' : severity === 'WARNING' ? 'warning' : 'info'
}
</script>

<template>
  <div class="page page--overview">
    <div class="page-heading page-heading--overview">
      <div>
        <p class="eyebrow">OPERATIONS / LIVE</p>
        <h1>实时态势</h1>
        <p class="page-heading__summary">地图负责态势，时间线负责过程，侧栏负责上下文与安全操作。</p>
      </div>
      <div class="summary-strip" aria-label="态势摘要">
        <div><strong>{{ store.onlineCount }}/{{ store.deviceList.length }}</strong><span>在线设备</span></div>
        <div><strong class="text-danger">{{ store.criticalCount }}</strong><span>严重告警</span></div>
        <div><strong>{{ store.activeAlarms.length }}</strong><span>活动告警</span></div>
      </div>
    </div>

    <div class="overview-grid">
      <OperationsMap :devices="store.deviceList" :selected-id="selectedId" @select="selectDevice" />
      <aside class="context-panel" aria-labelledby="context-title">
        <div class="section-heading">
          <div>
            <p class="eyebrow">CONTEXT</p>
            <h2 id="context-title">设备上下文</h2>
          </div>
          <span class="muted">{{ store.deviceList.length }} 对象</span>
        </div>
        <div v-if="selectedDevice" class="device-context">
          <div class="device-context__heading">
            <div>
              <span class="device-type">{{ selectedDevice.device_type }}</span>
              <h3>{{ selectedDevice.serial_number }}</h3>
              <p>{{ selectedDevice.product_model ?? '未知型号' }}</p>
            </div>
            <StatusBadge :label="selectedDevice.online === false ? '离线' : '在线'" :tone="selectedDevice.online === false ? 'offline' : 'success'" />
          </div>
          <FreshnessIndicator :timestamp="selectedDevice.server_time" :online="selectedDevice.online" />
          <dl class="metric-grid">
            <div><dt>电量</dt><dd>{{ selectedDevice.battery_percent ?? '—' }}<small v-if="selectedDevice.battery_percent !== null">%</small></dd></div>
            <div><dt>高度</dt><dd>{{ selectedDevice.altitude ?? '—' }}<small v-if="selectedDevice.altitude !== null"> m</small></dd></div>
            <div><dt>模式</dt><dd>{{ selectedDevice.mode ?? '—' }}</dd></div>
            <div><dt>版本</dt><dd>{{ selectedDevice.state_version ?? '—' }}</dd></div>
          </dl>
          <div class="context-section">
            <div class="section-heading section-heading--compact"><h3>活动告警</h3><span>{{ store.activeAlarms.filter((alarm) => alarm.device_id === selectedDevice?.id).length }}</span></div>
            <div v-if="store.activeAlarms.some((alarm) => alarm.device_id === selectedDevice?.id)" class="alarm-mini-list">
              <div v-for="alarm in store.activeAlarms.filter((item) => item.device_id === selectedDevice?.id)" :key="alarm.id" class="alarm-mini">
                <StatusBadge :label="alarm.severity" :tone="severityTone(alarm.severity)" />
                <span>{{ alarm.alarm_type }}</span>
                <strong>×{{ alarm.occurrence_count }}</strong>
              </div>
            </div>
            <p v-else class="muted">当前设备没有活动告警</p>
          </div>
          <div class="context-actions">
            <button class="button button--primary" type="button" @click="store.createStatusRefresh(selectedDevice)">刷新状态</button>
            <RouterLink class="button button--secondary" :to="`/app/${store.workspaceId}/commands`">查看指令</RouterLink>
          </div>
        </div>
        <div v-else class="empty-state">选择地图或设备列表中的对象</div>
        <div class="device-list" aria-label="设备等价列表">
          <button v-for="device in store.deviceList" :key="device.id" class="device-list__item" :class="{ 'device-list__item--selected': selectedId === device.id }" type="button" @click="selectDevice(device)">
            <span class="device-list__glyph" aria-hidden="true">{{ device.device_type === 'GATEWAY' ? '▣' : '✦' }}</span>
            <span><strong>{{ device.serial_number }}</strong><small>{{ device.mode ?? '未知模式' }}</small></span>
            <span class="device-list__battery">{{ device.battery_percent ?? '—' }}<small v-if="device.battery_percent !== null">%</small></span>
          </button>
        </div>
      </aside>
    </div>
    <Timeline :events="store.events" />
  </div>
</template>
