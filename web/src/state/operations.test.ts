import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { useOperationsStore } from './operations'

describe('operations store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('hydrates the demo snapshot and exposes actionable summaries', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')

    expect(store.deviceList).toHaveLength(3)
    expect(store.onlineCount).toBe(3)
    expect(store.criticalCount).toBe(1)
    expect(store.activeAlarms[0].dedup_key).toContain('aircraft-02')
    expect(store.connection).toBe('connected')
  })

  it('reconciles device state by version and applies command events', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    const current = store.devices['aircraft-01']

    store.applyDevice({ ...current, state_version: 10, battery_percent: 20 })
    expect(store.devices['aircraft-01'].state_version).toBe(42)
    store.applyDevice({ ...current, state_version: 43, battery_percent: 20 })
    expect(store.devices['aircraft-01'].battery_percent).toBe(20)

    store.applyEvent({
      event_id: 'command-event-1',
      type: 'command.updated',
      schema_version: '1.0',
      workspace_id: 'demo',
      aggregate_id: 'aircraft-01',
      occurred_at: new Date().toISOString(),
      data: { id: 'command-demo-01', status: 'EXECUTING', method: 'sim_status_refresh' },
    })
    expect(store.commands['command-demo-01'].status).toBe('EXECUTING')
    expect(store.events[0].event_id).toBe('command-event-1')
  })

  it('keeps acknowledgement separate from resolution', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    await store.acknowledgeAlarm(store.activeAlarms[0])

    expect(store.activeAlarms[0].status).toBe('ACKNOWLEDGED')
    expect(store.activeAlarms[0].resolved_at).toBeUndefined()
  })
})
