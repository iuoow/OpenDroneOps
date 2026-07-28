import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { ApiClient } from '../api/client'
import { RealtimeClient, type RealtimeStatus } from '../realtime/websocket'
import type { Alarm, Command, Device, DomainEvent, WebSocketEnvelope } from '../types/contracts'

const demoDevices: Device[] = [
  {
    id: 'aircraft-01',
    workspace_id: 'demo',
    vendor: 'DJI',
    serial_number: 'SIM-AIR-001',
    product_model: 'M350 RTK',
    device_type: 'AIRCRAFT',
    status: 'ONLINE',
    online: true,
    state_version: 42,
    latitude: 31.2304,
    longitude: 121.4737,
    altitude: 86,
    battery_percent: 74,
    mode: 'MISSION',
    server_time: new Date(Date.now() - 2_000).toISOString(),
  },
  {
    id: 'aircraft-02',
    workspace_id: 'demo',
    vendor: 'DJI',
    serial_number: 'SIM-AIR-002',
    product_model: 'M30T',
    device_type: 'AIRCRAFT',
    status: 'ONLINE',
    online: true,
    state_version: 18,
    latitude: 31.2242,
    longitude: 121.4811,
    altitude: 44,
    battery_percent: 16,
    mode: 'RETURN_TO_HOME',
    server_time: new Date(Date.now() - 18_000).toISOString(),
  },
  {
    id: 'dock-01',
    workspace_id: 'demo',
    vendor: 'DJI',
    serial_number: 'SIM-DOCK-001',
    product_model: 'Dock 2',
    device_type: 'GATEWAY',
    status: 'ONLINE',
    online: true,
    state_version: 9,
    latitude: 31.2269,
    longitude: 121.4682,
    battery_percent: null,
    mode: 'READY',
    server_time: new Date(Date.now() - 4_000).toISOString(),
  },
]

const demoAlarms: Alarm[] = [
  {
    id: 'alarm-demo-01',
    workspace_id: 'demo',
    device_id: 'aircraft-02',
    dedup_key: 'device:aircraft-02:battery.low',
    alarm_type: 'battery.low',
    severity: 'CRITICAL',
    status: 'OPEN',
    first_occurred_at: new Date(Date.now() - 9 * 60_000).toISOString(),
    last_occurred_at: new Date(Date.now() - 18_000).toISOString(),
    occurrence_count: 7,
    details: { battery_percent: 16, recommendation: '确认返航状态与电池余量' },
  },
]

const demoCommands: Command[] = [
  {
    id: 'command-demo-01',
    workspace_id: 'demo',
    target_device_id: 'aircraft-01',
    gateway_device_id: 'dock-01',
    method: 'sim_status_refresh',
    status: 'SUCCEEDED',
    risk_level: 'LOW',
    idempotency_key: 'demo-command-001',
    dji_tid: 'tid-demo-001',
    dji_bid: 'bid-demo-001',
    parameters: { refresh: true },
    requested_by: 'operator.demo',
    created_at: new Date(Date.now() - 4 * 60_000).toISOString(),
    completed_at: new Date(Date.now() - 3 * 60_000).toISOString(),
    updated_at: new Date(Date.now() - 3 * 60_000).toISOString(),
  },
]

const demoEvents: DomainEvent[] = [
  {
    event_id: 'event-demo-01',
    event_type: 'command.updated',
    schema_version: '1.0',
    workspace_id: 'demo',
    device_id: 'aircraft-01',
    occurred_at: new Date(Date.now() - 3 * 60_000).toISOString(),
    payload: { status: 'SUCCEEDED', method: 'sim_status_refresh' },
  },
  {
    event_id: 'event-demo-02',
    event_type: 'alarm.updated',
    schema_version: '1.0',
    workspace_id: 'demo',
    device_id: 'aircraft-02',
    occurred_at: new Date(Date.now() - 18_000).toISOString(),
    payload: { alarm_type: 'battery.low', severity: 'CRITICAL' },
  },
]

export const useOperationsStore = defineStore('operations', () => {
  const api = new ApiClient()
  const workspaceId = ref('demo')
  const devices = ref<Record<string, Device>>({})
  const alarms = ref<Record<string, Alarm>>({})
  const commands = ref<Record<string, Command>>({})
  const events = ref<DomainEvent[]>([])
  const cursor = ref('')
  const connection = ref<RealtimeStatus>('idle')
  const connectionDetail = ref('')
  const loading = ref(false)
  const error = ref('')
  const lastSyncAt = ref<string | null>(null)
  const sourceMode = computed(() => isDemoMode() ? 'demo' : 'api')
  let realtime: RealtimeClient | undefined

  const deviceList = computed(() => Object.values(devices.value))
  const alarmList = computed(() =>
    Object.values(alarms.value).sort((a, b) => severityRank(b.severity) - severityRank(a.severity)),
  )
  const commandList = computed(() =>
    Object.values(commands.value).sort((a, b) => b.created_at.localeCompare(a.created_at)),
  )
  const activeAlarms = computed(() => alarmList.value.filter((alarm) => alarm.status !== 'RESOLVED'))
  const onlineCount = computed(() => deviceList.value.filter((device) => device.online || device.status === 'ONLINE').length)
  const criticalCount = computed(
    () => activeAlarms.value.filter((alarm) => alarm.severity === 'CRITICAL').length,
  )

  async function hydrate(id = workspaceId.value) {
    workspaceId.value = id
    loading.value = true
    error.value = ''
    try {
      if (isDemoMode()) {
        setSnapshot({ devices: demoDevices, alarms: demoAlarms, commands: demoCommands, events: demoEvents, cursor: 'demo-1' })
      } else {
        setSnapshot(await api.getSnapshot(id))
      }
      lastSyncAt.value = new Date().toISOString()
      startRealtime()
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '无法加载 Operations 快照'
    } finally {
      loading.value = false
    }
  }

  function setSnapshot(snapshot: {
    devices: Device[]
    alarms: Alarm[]
    commands: Command[]
    events: DomainEvent[]
    cursor?: string
  }) {
    devices.value = Object.fromEntries(snapshot.devices.map((device) => [device.id, device]))
    alarms.value = Object.fromEntries(snapshot.alarms.map((alarm) => [alarm.id, alarm]))
    commands.value = Object.fromEntries(snapshot.commands.map((command) => [command.id, command]))
    events.value = snapshot.events.slice(0, 100)
    cursor.value = snapshot.cursor ?? ''
  }

  function startRealtime() {
    realtime?.disconnect()
    if (isDemoMode()) {
      connection.value = 'connected'
      connectionDetail.value = '演示数据源'
      return
    }
    realtime = new RealtimeClient({
      workspaceId: workspaceId.value,
      cursor: cursor.value,
      onStatus(status, detail) {
        connection.value = status
        connectionDetail.value = detail ?? ''
      },
      onEvent: applyEvent,
    })
    realtime.connect()
  }

  function applyEvent(event: WebSocketEnvelope) {
    cursor.value = event.sequence ? `${event.occurred_at}:${event.sequence}` : event.event_id
    realtime?.setCursor(cursor.value)
    const data = event.data
    if (event.type.startsWith('alarm.') && typeof data.id === 'string') {
      const alarm = data as unknown as Alarm
      alarms.value[alarm.id] = alarm
    } else if (event.type === 'command.updated' && typeof data.id === 'string') {
      const command = data as unknown as Command
      commands.value[command.id] = command
    } else if (event.type.startsWith('device.') && typeof data.id === 'string') {
      applyDevice(data as unknown as Device)
    }
    events.value = [
      {
        event_id: event.event_id,
        event_type: event.type,
        schema_version: event.schema_version,
        workspace_id: event.workspace_id,
        device_id: event.aggregate_id,
        occurred_at: event.occurred_at,
        payload: data,
      },
      ...events.value,
    ].slice(0, 100)
  }

  function applyDevice(next: Device) {
    const current = devices.value[next.id]
    if (current?.state_version !== undefined && next.state_version !== undefined && next.state_version <= current.state_version) {
      return
    }
    devices.value[next.id] = { ...current, ...next }
  }

  async function acknowledgeAlarm(alarm: Alarm) {
    if (isDemoMode()) {
      alarms.value[alarm.id] = { ...alarm, status: 'ACKNOWLEDGED', acknowledged_by: 'operator.demo', acknowledged_at: new Date().toISOString() }
      return
    }
    const updated = await api.acknowledgeAlarm(workspaceId.value, alarm.id)
    alarms.value[updated.id] = updated
  }

  async function createStatusRefresh(device: Device) {
    const idempotencyKey = `ui-${crypto.randomUUID()}`
    if (isDemoMode()) {
      const command: Command = {
        id: `command-${crypto.randomUUID()}`,
        workspace_id: workspaceId.value,
        target_device_id: device.id,
        method: 'sim_status_refresh',
        status: 'PUBLISH_PENDING',
        risk_level: 'LOW',
        idempotency_key: idempotencyKey,
        parameters: { refresh: true },
        requested_by: 'operator.demo',
        created_at: new Date().toISOString(),
      }
      commands.value[command.id] = command
      return command
    }
    const command = await api.createCommand(workspaceId.value, {
      target_device_id: device.id,
      method: 'sim_status_refresh',
      parameters: { refresh: true },
      idempotencyKey,
    })
    commands.value[command.id] = command
    return command
  }

  function stopRealtime() {
    realtime?.disconnect()
    realtime = undefined
  }

  return {
    workspaceId,
    devices,
    alarms,
    commands,
    events,
    cursor,
    connection,
    connectionDetail,
    loading,
    error,
    lastSyncAt,
    sourceMode,
    deviceList,
    alarmList,
    commandList,
    activeAlarms,
    onlineCount,
    criticalCount,
    hydrate,
    setSnapshot,
    applyEvent,
    applyDevice,
    acknowledgeAlarm,
    createStatusRefresh,
    stopRealtime,
  }
})

function severityRank(severity: Alarm['severity']) {
  return severity === 'CRITICAL' ? 3 : severity === 'WARNING' ? 2 : 1
}

function isDemoMode() {
  return import.meta.env.VITE_DEMO_MODE !== 'false'
}
