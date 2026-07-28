import { describe, expect, it, vi } from 'vitest'
import type { RealtimeOptions } from '../realtime/websocket'
import type { Alarm, Device } from '../types/contracts'
import { createPilotReadModel, type PilotRealtimeConnection } from './readModel'

const device: Device = {
  id: 'aircraft-1',
  workspace_id: 'workspace-1',
  vendor: 'DJI',
  serial_number: 'SIM-001',
  product_model: 'M350 RTK',
  device_type: 'AIRCRAFT',
  status: 'ONLINE',
  online: true,
  state_version: 7,
  battery_percent: 82,
  server_time: '2026-07-28T00:00:00.000Z',
}

const alarm: Alarm = {
  id: 'alarm-1',
  workspace_id: 'workspace-1',
  device_id: 'aircraft-1',
  dedup_key: 'device:aircraft-1:wind.warning',
  alarm_type: 'wind.warning',
  severity: 'WARNING',
  status: 'OPEN',
  first_occurred_at: '2026-07-28T00:00:00.000Z',
  last_occurred_at: '2026-07-28T00:00:05.000Z',
  occurrence_count: 1,
}

function setup() {
  let realtimeOptions: RealtimeOptions | undefined
  const realtime: PilotRealtimeConnection = {
    connect: vi.fn(),
    disconnect: vi.fn(),
    setCursor: vi.fn(),
  }
  const client = {
    getPilotSnapshot: vi.fn().mockResolvedValue({
      devices: [device],
      alarms: [alarm],
      cursor: 'snapshot-cursor',
    }),
  }
  let now = Date.parse('2026-07-28T00:00:10.000Z')
  const model = createPilotReadModel({
    workspaceId: 'workspace-1',
    apiBaseUrl: 'https://api.example.test/v1',
    websocketUrl: 'wss://api.example.test/ws',
    client,
    realtimeFactory(options) {
      realtimeOptions = options
      return realtime
    },
    now: () => now,
  })
  return { model, client, realtime, getRealtimeOptions: () => realtimeOptions, setNow: (value: number) => (now = value) }
}

describe('PilotReadModel', () => {
  it('loads only the read-only device/alarm snapshot and starts scoped realtime', async () => {
    const context = setup()
    await context.model.hydrate()

    expect(context.client.getPilotSnapshot).toHaveBeenCalledWith('workspace-1')
    expect(context.model.currentDevice.value?.id).toBe('aircraft-1')
    expect(context.model.currentAlarm.value?.id).toBe('alarm-1')
    expect(context.getRealtimeOptions()?.channels).toEqual(['telemetry', 'state', 'alarm'])
    expect(context.getRealtimeOptions()?.url).toContain('workspace_id=workspace-1')
    expect(context.model).not.toHaveProperty('acknowledgeAlarm')
    expect(context.model).not.toHaveProperty('createCommand')
    context.model.stop()
  })

  it('applies newer device events and restores connection after recovery', async () => {
    const context = setup()
    await context.model.hydrate()
    context.getRealtimeOptions()?.onStatus?.('recovering', '恢复游标中')
    context.getRealtimeOptions()?.onEvent?.({
      event_id: 'event-8',
      type: 'device.updated',
      schema_version: '1.0',
      workspace_id: 'workspace-1',
      aggregate_id: 'aircraft-1',
      occurred_at: '2026-07-28T00:00:11.000Z',
      sequence: 8,
      data: { ...device, state_version: 8, battery_percent: 79 },
    })

    expect(context.model.currentDevice.value?.battery_percent).toBe(79)
    expect(context.model.connection.value).toBe('connected')
    expect(context.model.connectionDetail.value).toBe('实时数据已恢复')
    expect(context.realtime.setCursor).toHaveBeenCalledWith('2026-07-28T00:00:11.000Z:8')
    context.model.stop()
  })

  it('supports an explicit reconnect without submitting or mutating field data', async () => {
    const context = setup()
    await context.model.hydrate()

    await context.model.reconnect()

    expect(context.realtime.disconnect).toHaveBeenCalled()
    expect(context.client.getPilotSnapshot).toHaveBeenCalledTimes(2)
    expect(context.model).not.toHaveProperty('submitDraft')
    context.model.stop()
  })

  it('ignores older device versions and cross-workspace events', async () => {
    const context = setup()
    await context.model.hydrate()
    context.getRealtimeOptions()?.onEvent?.({
      event_id: 'event-old',
      type: 'device.updated',
      schema_version: '1.0',
      workspace_id: 'workspace-1',
      occurred_at: '2026-07-28T00:00:11.000Z',
      data: { ...device, state_version: 6, battery_percent: 20 },
    })
    context.getRealtimeOptions()?.onEvent?.({
      event_id: 'event-other',
      type: 'alarm.updated',
      schema_version: '1.0',
      workspace_id: 'workspace-2',
      occurred_at: '2026-07-28T00:00:12.000Z',
      data: { ...alarm, id: 'other-alarm', workspace_id: 'workspace-2' },
    })

    expect(context.model.currentDevice.value?.battery_percent).toBe(82)
    expect(context.model.activeAlarms.value).toHaveLength(1)
    context.model.stop()
  })

  it('marks data stale when no snapshot or event refresh arrives', async () => {
    vi.useFakeTimers()
    const context = setup()
    await context.model.hydrate()
    context.setNow(Date.parse('2026-07-28T00:00:45.001Z'))
    vi.advanceTimersByTime(5_000)
    expect(context.model.stale.value).toBe(true)
    context.model.stop()
    vi.useRealTimers()
  })

  it('exposes a stable read-only error when snapshot loading fails', async () => {
    const model = createPilotReadModel({
      workspaceId: 'workspace-1',
      apiBaseUrl: 'https://api.example.test/v1',
      websocketUrl: 'wss://api.example.test/ws',
      client: { getPilotSnapshot: vi.fn().mockRejectedValue(new Error('secret upstream detail')) },
    })
    await model.hydrate()
    expect(model.error.value).toBe('无法加载只读现场数据')
    expect(model.connection.value).toBe('disconnected')
    expect(model.error.value).not.toContain('secret')
    model.stop()
  })
})
