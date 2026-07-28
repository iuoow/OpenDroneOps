import { computed, readonly, ref, type ComputedRef, type DeepReadonly, type Ref } from 'vue'
import { ApiClient } from '../api/client'
import {
  RealtimeClient,
  type RealtimeOptions,
  type RealtimeStatus,
} from '../realtime/websocket'
import type { Alarm, Device, WebSocketEnvelope } from '../types/contracts'

export interface PilotSnapshot {
  devices: Device[]
  alarms: Alarm[]
  cursor?: string
}

export interface PilotSnapshotClient {
  getPilotSnapshot(workspaceId: string): Promise<PilotSnapshot>
}

export interface PilotRealtimeConnection {
  connect(): void
  disconnect(): void
  setCursor(cursor: string): void
}

export interface PilotReadModel {
  readonly devices: ComputedRef<readonly Device[]>
  readonly activeAlarms: ComputedRef<readonly Alarm[]>
  readonly currentDevice: ComputedRef<Device | undefined>
  readonly currentAlarm: ComputedRef<Alarm | undefined>
  readonly connection: DeepReadonly<Ref<RealtimeStatus>>
  readonly connectionDetail: DeepReadonly<Ref<string>>
  readonly loading: DeepReadonly<Ref<boolean>>
  readonly error: DeepReadonly<Ref<string>>
  readonly lastSyncAt: DeepReadonly<Ref<string | null>>
  readonly stale: ComputedRef<boolean>
  hydrate(): Promise<void>
  reconnect(): Promise<void>
  stop(): void
}

export interface PilotReadModelOptions {
  workspaceId: string
  apiBaseUrl: string
  websocketUrl: string
  demo?: boolean
  client?: PilotSnapshotClient
  realtimeFactory?: (options: RealtimeOptions) => PilotRealtimeConnection
  now?: () => number
  staleAfterMs?: number
}

export function createPilotReadModel(options: PilotReadModelOptions): PilotReadModel {
  const now = options.now ?? Date.now
  const staleAfterMs = options.staleAfterMs ?? 30_000
  const client = options.client ?? new ApiClient({ baseUrl: options.apiBaseUrl })
  const realtimeFactory = options.realtimeFactory ?? ((input) => new RealtimeClient(input))
  const deviceMap = ref<Record<string, Device>>({})
  const alarmMap = ref<Record<string, Alarm>>({})
  const cursor = ref('')
  const connection = ref<RealtimeStatus>('idle')
  const connectionDetail = ref('')
  const loading = ref(false)
  const error = ref('')
  const lastSyncAt = ref<string | null>(null)
  const clock = ref(now())
  let realtime: PilotRealtimeConnection | undefined
  let freshnessTimer: number | undefined

  const devices = computed(() =>
    Object.values(deviceMap.value).sort((left, right) => devicePriority(right) - devicePriority(left)),
  )
  const activeAlarms = computed(() =>
    Object.values(alarmMap.value)
      .filter((alarm) => alarm.status !== 'RESOLVED')
      .sort((left, right) => {
        const severity = severityRank(right.severity) - severityRank(left.severity)
        return severity || right.last_occurred_at.localeCompare(left.last_occurred_at)
      }),
  )
  const currentDevice = computed(() => devices.value[0])
  const currentAlarm = computed(() => activeAlarms.value[0])
  const stale = computed(() => {
    if (!lastSyncAt.value) return false
    return clock.value - Date.parse(lastSyncAt.value) > staleAfterMs
  })

  async function hydrate() {
    loading.value = true
    error.value = ''
    try {
      const snapshot = options.demo ? createDemoPilotSnapshot(options.workspaceId, now()) : await client.getPilotSnapshot(options.workspaceId)
      deviceMap.value = Object.fromEntries(snapshot.devices.map((device) => [device.id, device]))
      alarmMap.value = Object.fromEntries(snapshot.alarms.map((alarm) => [alarm.id, alarm]))
      cursor.value = snapshot.cursor ?? ''
      markSynchronized()
      startFreshnessClock()
      startRealtime()
    } catch {
      error.value = '无法加载只读现场数据'
      connection.value = 'disconnected'
      connectionDetail.value = '快照加载失败'
    } finally {
      loading.value = false
    }
  }

  async function reconnect() {
    stop()
    await hydrate()
  }

  function startRealtime() {
    realtime?.disconnect()
    if (options.demo) {
      realtime = undefined
      connection.value = 'connected'
      connectionDetail.value = '浏览器演示数据'
      return
    }
    realtime = realtimeFactory({
      workspaceId: options.workspaceId,
      url: websocketUrlWithWorkspace(options.websocketUrl, options.workspaceId),
      cursor: cursor.value,
      channels: ['telemetry', 'state', 'alarm'],
      onStatus(status, detail) {
        connection.value = status
        connectionDetail.value = detail ?? ''
      },
      onEvent: applyEvent,
    })
    realtime.connect()
  }

  function applyEvent(event: WebSocketEnvelope) {
    if (event.workspace_id !== options.workspaceId) return
    cursor.value = event.sequence ? `${event.occurred_at}:${event.sequence}` : event.event_id
    realtime?.setCursor(cursor.value)
    if (event.type.startsWith('device.') && typeof event.data.id === 'string') {
      applyDevice(event.data as unknown as Device)
    } else if (event.type.startsWith('alarm.') && typeof event.data.id === 'string') {
      const alarm = event.data as unknown as Alarm
      alarmMap.value[alarm.id] = alarm
    }
    markSynchronized()
    if (connection.value === 'recovering' || connection.value === 'disconnected') {
      connection.value = 'connected'
      connectionDetail.value = '实时数据已恢复'
    }
  }

  function applyDevice(next: Device) {
    const current = deviceMap.value[next.id]
    if (
      current?.state_version !== undefined &&
      next.state_version !== undefined &&
      next.state_version <= current.state_version
    ) {
      return
    }
    deviceMap.value[next.id] = { ...current, ...next }
  }

  function markSynchronized() {
    const timestamp = now()
    clock.value = timestamp
    lastSyncAt.value = new Date(timestamp).toISOString()
  }

  function startFreshnessClock() {
    if (freshnessTimer !== undefined) return
    freshnessTimer = window.setInterval(() => {
      clock.value = now()
    }, 5_000)
  }

  function stop() {
    realtime?.disconnect()
    realtime = undefined
    if (freshnessTimer !== undefined) {
      window.clearInterval(freshnessTimer)
      freshnessTimer = undefined
    }
  }

  return {
    devices,
    activeAlarms,
    currentDevice,
    currentAlarm,
    connection: readonly(connection),
    connectionDetail: readonly(connectionDetail),
    loading: readonly(loading),
    error: readonly(error),
    lastSyncAt: readonly(lastSyncAt),
    stale,
    hydrate,
    reconnect,
    stop,
  }
}

function createDemoPilotSnapshot(workspaceId: string, timestamp: number): PilotSnapshot {
  return {
    cursor: 'pilot-demo-1',
    devices: [
      {
        id: 'pilot-aircraft-01',
        workspace_id: workspaceId,
        vendor: 'DJI',
        serial_number: 'SIM-PILOT-AIR-001',
        product_model: 'M350 RTK (Simulated)',
        device_type: 'AIRCRAFT',
        status: 'ONLINE',
        online: true,
        state_version: 14,
        latitude: 31.2304,
        longitude: 121.4737,
        altitude: 82,
        battery_percent: 74,
        mode: 'MISSION_SIMULATION',
        server_time: new Date(timestamp - 2_000).toISOString(),
      },
    ],
    alarms: [
      {
        id: 'pilot-alarm-01',
        workspace_id: workspaceId,
        device_id: 'pilot-aircraft-01',
        dedup_key: 'pilot-demo:wind.warning',
        alarm_type: 'wind.warning',
        severity: 'WARNING',
        status: 'OPEN',
        first_occurred_at: new Date(timestamp - 4 * 60_000).toISOString(),
        last_occurred_at: new Date(timestamp - 12_000).toISOString(),
        occurrence_count: 2,
        details: { summary: '模拟风速告警，仅供只读 UI 验证' },
      },
    ],
  }
}

function devicePriority(device: Device) {
  const online = device.online || device.status === 'ONLINE' ? 100 : 0
  const aircraft = device.device_type === 'AIRCRAFT' ? 10 : 0
  return online + aircraft
}

function severityRank(severity: Alarm['severity']) {
  return severity === 'CRITICAL' ? 3 : severity === 'WARNING' ? 2 : 1
}

function websocketUrlWithWorkspace(endpoint: string, workspaceId: string) {
  const url = new URL(endpoint)
  url.searchParams.set('workspace_id', workspaceId)
  return url.toString()
}
