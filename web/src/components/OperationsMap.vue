<script setup lang="ts">
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'
import type { Device } from '../types/contracts'

const props = defineProps<{
  devices: Device[]
  selectedId?: string
  attentionDeviceIds?: string[]
}>()
const emit = defineEmits<{ select: [device: Device] }>()

const positionedDevices = computed(() =>
  props.devices.map((device, index) => ({
    ...device,
    source: device,
    left: 14 + ((device.longitude ?? 121.47) - 121.46) * 9000,
    top: 72 - ((device.latitude ?? 31.22) - 31.21) * 7000,
    fallbackLeft: 22 + (index % 3) * 28,
    fallbackTop: 28 + Math.floor(index / 3) * 25,
  })),
)

function position(device: (typeof positionedDevices.value)[number]) {
  const left = Number.isFinite(device.left) && device.left >= 4 && device.left <= 92 ? device.left : device.fallbackLeft
  const top = Number.isFinite(device.top) && device.top >= 8 && device.top <= 88 ? device.top : device.fallbackTop
  return { left: `${left}%`, top: `${top}%` }
}

function statusLabel(device: Device) {
  if (device.status === 'OFFLINE' || device.online === false) return '离线'
  const last = device.server_time ? Date.now() - new Date(device.server_time).getTime() : 0
  return last > 60_000 ? '过期' : '在线'
}

function isAttentionDevice(device: Device) {
  return props.attentionDeviceIds?.includes(device.id) ?? false
}
</script>

<template>
  <section class="map-panel map-panel--overview" aria-labelledby="map-title">
    <div class="map-panel__header">
      <div>
        <p class="eyebrow">SPATIAL WORKSPACE</p>
        <h2 id="map-title">运营区域</h2>
      </div>
      <div class="map-panel__meta">
        <span><AppIcon name="layers" :size="15" />默认图层</span>
        <span>设备列表提供键盘等价入口</span>
      </div>
    </div>
    <div class="map-canvas map-canvas--v2" aria-label="设备实时地图，设备列表提供键盘等价入口" aria-describedby="map-caption">
      <div class="map-canvas__terrain" aria-hidden="true">
        <span class="map-contour map-contour--one"></span>
        <span class="map-contour map-contour--two"></span>
        <span class="map-road map-road--one"></span>
        <span class="map-road map-road--two"></span>
        <span class="map-water"></span>
        <span class="map-label map-label--north">浦东运营区</span>
        <span class="map-label map-label--south">Dock 运营边界</span>
      </div>
      <button
        v-for="device in positionedDevices"
        :key="device.id"
        class="map-device"
        :class="{
          'map-device--selected': selectedId === device.id,
          'map-device--offline': statusLabel(device) !== '在线',
          'map-device--attention': isAttentionDevice(device),
        }"
        :style="position(device)"
        type="button"
        :aria-label="`${device.serial_number}，${statusLabel(device)}，${device.battery_percent ?? '无'}% 电量`"
        @click="emit('select', device.source)"
      >
        <span class="map-device__halo" aria-hidden="true"></span>
        <span class="map-device__attention" aria-hidden="true"></span>
        <span class="map-device__shape" aria-hidden="true"><AppIcon :name="device.device_type === 'GATEWAY' ? 'dock' : 'aircraft'" :size="15" /></span>
        <span class="map-device__label">{{ device.serial_number }}</span>
      </button>
      <div class="map-legend" aria-hidden="true"><span><i class="map-legend__symbol"></i>在线</span><span><i class="map-legend__ring"></i>待关注</span></div>
      <div class="map-canvas__scale" aria-hidden="true">500 m</div>
    </div>
    <p id="map-caption" class="map-panel__caption">当前视野 {{ devices.length }} 个对象 · 位置接收与视觉刷新分离</p>
  </section>
</template>
