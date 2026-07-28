<script setup lang="ts">
import { computed } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import StatusBadge from '../components/StatusBadge.vue'
import { useOperationsStore } from '../state/operations'
import type { Command } from '../types/contracts'

const store = useOperationsStore()
const pendingStatuses: Command['status'][] = ['CREATED', 'VALIDATED', 'PUBLISH_PENDING', 'PUBLISHED', 'ACCEPTED', 'EXECUTING']
const pendingCommands = computed(() => store.commandList.filter((command) => pendingStatuses.includes(command.status)))
const terminalCommands = computed(() => store.commandList.filter((command) => !pendingStatuses.includes(command.status)))
const browserTone = computed(() => store.connection === 'connected' ? 'success' : store.connection === 'recovering' || store.connection === 'connecting' ? 'warning' : 'offline')
const browserLabel = computed(() => store.connection === 'connected' ? '实时连接正常' : store.connection === 'recovering' ? '正在恢复实时数据' : store.connection === 'connecting' ? '正在建立连接' : '实时连接未建立')
const commandRisk = computed(() => pendingCommands.value.some((command) => command.status === 'PUBLISHED' || command.status === 'ACCEPTED' || command.status === 'EXECUTING'))

function commandLabel(status: Command['status']) {
  const labels: Record<Command['status'], string> = { CREATED: '已创建', VALIDATED: '已校验', REJECTED: '已拒绝', PUBLISH_PENDING: '等待发布', PUBLISHED: '已发布', ACCEPTED: '设备已接收', EXECUTING: '执行中', SUCCEEDED: '结果已确认', FAILED: '执行失败', TIMEOUT: '已超时', CANCELED: '已取消' }
  return labels[status]
}

function timeLabel(value?: string | null) {
  if (!value) return '尚未完成快照同步'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '时间不可用' : date.toLocaleString('zh-CN', { hour12: false })
}
</script>

<template>
  <div class="page runtime-v2">
    <div class="page-heading runtime-v2__heading">
      <div><p class="eyebrow">OPERATIONS / RUNTIME EVIDENCE</p><h1>系统运行</h1><p class="page-heading__summary">区分浏览器实时连接、平台管理面与设备遥测；这里显示可由当前 Web 会话确认的事实，不伪造 Broker、Outbox 或容量指标。</p></div>
      <div class="runtime-summary"><div><strong :class="browserTone === 'success' ? 'text-success' : browserTone === 'warning' ? 'text-warning' : 'text-danger'">{{ store.connection === 'connected' ? '正常' : '关注' }}</strong><span>浏览器实时链路</span></div><div><strong>{{ pendingCommands.length }}</strong><span>处理中指令</span></div><div><strong>{{ store.activeAlarms.length }}</strong><span>活动告警</span></div></div>
    </div>

    <div class="runtime-v2__workspace">
      <section class="runtime-connection" aria-labelledby="runtime-connection-title">
        <div class="runtime-panel__header"><div><p class="eyebrow">CONNECTION EVIDENCE</p><h2 id="runtime-connection-title">实时连接</h2></div><StatusBadge :label="browserLabel" :tone="browserTone" /></div>
        <div class="runtime-connection__fact"><AppIcon name="overview" :size="21" /><div><strong>{{ browserLabel }}</strong><span>{{ store.connectionDetail || '未提供额外连接详情。' }}</span></div></div>
        <ol class="runtime-connection__steps" aria-label="实时恢复状态">
          <li class="runtime-connection__step runtime-connection__step--complete"><span>1</span><div><strong>保留最后快照</strong><small>最新成功同步：{{ timeLabel(store.lastSyncAt) }}</small></div></li>
          <li class="runtime-connection__step" :class="{ 'runtime-connection__step--complete': store.connection === 'connected', 'runtime-connection__step--current': store.connection === 'recovering' || store.connection === 'connecting' }"><span>2</span><div><strong>连接与 Cursor 恢复</strong><small>{{ store.connection === 'connected' ? '实时增量已恢复。' : '等待连接状态更新；不会把旧快照误标为实时。' }}</small></div></li>
          <li class="runtime-connection__step" :class="{ 'runtime-connection__step--complete': store.connection === 'connected' }"><span>3</span><div><strong>按版本合并增量</strong><small>设备、告警和指令仍可从 REST 快照与事件恢复。</small></div></li>
        </ol>
      </section>

      <section class="runtime-command-flow" aria-labelledby="runtime-command-title">
        <div class="runtime-panel__header"><div><p class="eyebrow">COMMAND WORKFLOW</p><h2 id="runtime-command-title">发布与结果证据</h2></div><RouterLink class="button button--secondary" :to="`/app/${store.workspaceId}/commands`">查看指令</RouterLink></div>
        <div class="runtime-command-flow__summary"><span class="runtime-command-flow__signal" :class="{ 'runtime-command-flow__signal--attention': commandRisk }"><AppIcon name="commands" :size="20" /></span><div><strong>{{ pendingCommands.length ? `${pendingCommands.length} 条指令正在推进` : '当前没有处理中指令' }}</strong><p>{{ commandRisk ? '发布或设备接收仍不等于设备执行成功。' : '最终设备结果会在指令中心保留为可恢复记录。' }}</p></div></div>
        <div v-if="pendingCommands.length" class="runtime-command-flow__list"><div v-for="command in pendingCommands.slice(0, 4)" :key="command.id"><StatusBadge :label="commandLabel(command.status)" :tone="command.status === 'ACCEPTED' || command.status === 'EXECUTING' ? 'warning' : 'info'" /><span><strong>{{ command.method }}</strong><small>{{ store.devices[command.target_device_id]?.serial_number ?? command.target_device_id }}</small></span></div></div>
        <div v-else class="runtime-command-flow__empty"><span>{{ terminalCommands.length }} 条最近指令已进入终态。</span><span>Outbox 领取、发布和重试细节仅在管理面审计与运行日志中查看。</span></div>
      </section>

      <section class="runtime-management" aria-labelledby="runtime-management-title">
        <div class="runtime-panel__header"><div><p class="eyebrow">MANAGEMENT PLANE</p><h2 id="runtime-management-title">容量与队列边界</h2></div><span class="runtime-management__private">受限数据</span></div>
        <div class="runtime-management__notice"><AppIcon name="operations" :size="20" /><div><strong>容量指标不暴露给租户 Web 会话</strong><p>`/capacity` 与 Prometheus 指标仅由 loopback 管理面、受控终端或监控代理访问；本页不会请求或缓存它们。</p></div></div>
        <dl class="runtime-management__grid"><div><dt>MQTT 摄入队列</dt><dd>有界分片与热点隔离</dd><small>关注 shard queue limit、hot key backpressure</small></div><div><dt>WebSocket 投递</dt><dd>有界写队列</dd><small>慢客户端被隔离，告警与指令可经快照恢复</small></div><div><dt>Outbox</dt><dd>事务性发布边界</dd><small>MQTT PUBACK 不代表设备执行成功</small></div><div><dt>容量告警</dt><dd>按速率观察</dd><small>累计计数自进程启动后持续累加</small></div></dl>
        <p class="runtime-management__footnote">需要运行级别诊断时，请按已批准的管理面 Runbook 使用 `/capacity` 或 Prometheus；不要从公开应用接口绕过访问边界。</p>
      </section>
    </div>
  </div>
</template>
