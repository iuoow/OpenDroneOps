<script setup lang="ts">
import { computed, ref } from 'vue'
import FreshnessIndicator from '../components/FreshnessIndicator.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'

const store = useOperationsStore()
const query = ref('')
const status = ref('ALL')
const devices = computed(() =>
  store.deviceList.filter((device) => {
    const matchesQuery = `${device.serial_number} ${device.product_model ?? ''}`.toLowerCase().includes(query.value.toLowerCase())
    const matchesStatus = status.value === 'ALL' || device.status === status.value
    return matchesQuery && matchesStatus
  }),
)
</script>

<template>
  <div class="page">
    <div class="page-heading">
      <div><p class="eyebrow">INVENTORY / DEVICES</p><h1>设备管理</h1><p class="page-heading__summary">搜索设备、查看新鲜度和进入实时上下文。</p></div>
      <span class="page-heading__count">{{ devices.length }} 个设备</span>
    </div>
    <section class="panel">
      <div class="toolbar">
        <label class="field field--search"><span class="sr-only">搜索设备</span><span aria-hidden="true">⌕</span><input v-model="query" placeholder="搜索序列号或型号" /></label>
        <label class="field"><span class="sr-only">设备状态</span><select v-model="status"><option value="ALL">全部状态</option><option value="ONLINE">在线</option><option value="OFFLINE">离线</option></select></label>
      </div>
      <div class="table-wrap">
        <table class="data-table">
          <caption class="sr-only">设备清单</caption>
          <thead><tr><th>设备</th><th>类型</th><th>业务状态</th><th>数据新鲜度</th><th>电量</th><th>模式</th><th><span class="sr-only">操作</span></th></tr></thead>
          <tbody>
            <tr v-for="device in devices" :key="device.id">
              <td><strong>{{ device.serial_number }}</strong><small>{{ device.product_model ?? '未知型号' }}</small></td>
              <td>{{ device.device_type }}</td>
              <td><StatusBadge :label="device.status === 'ONLINE' ? '在线' : '离线'" :tone="device.status === 'ONLINE' ? 'success' : 'offline'" /></td>
              <td><FreshnessIndicator :timestamp="device.server_time" :online="device.online" /></td>
              <td class="tabular">{{ device.battery_percent ?? '—' }}<small v-if="device.battery_percent !== null">%</small></td>
              <td>{{ device.mode ?? '—' }}</td>
              <td><RouterLink class="button button--small button--secondary" :to="`/app/${store.workspaceId}/overview`">查看</RouterLink></td>
            </tr>
          </tbody>
        </table>
        <div v-if="!devices.length" class="empty-state">没有符合筛选条件的设备</div>
      </div>
    </section>
  </div>
</template>
