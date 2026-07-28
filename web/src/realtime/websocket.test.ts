import { describe, expect, it, vi } from 'vitest'
import { RealtimeClient } from './websocket'

class FakeSocket {
  static OPEN = 1
  readyState = 0
  sent: string[] = []
  private listeners = new Map<string, ((event: MessageEvent) => void)[]>()

  addEventListener(type: string, listener: (event: MessageEvent) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener])
  }

  send(payload: string) {
    this.sent.push(payload)
  }

  close() {
    this.readyState = 3
    this.emit('close', {})
  }

  open() {
    this.readyState = 1
    this.emit('open', {})
  }

  message(payload: unknown) {
    this.emit('message', { data: JSON.stringify(payload) })
  }

  private emit(type: string, event: unknown) {
    for (const listener of this.listeners.get(type) ?? []) listener(event as MessageEvent)
  }
}

describe('RealtimeClient', () => {
  it('subscribes with the recovery cursor and forwards envelopes', () => {
    const statuses: string[] = []
    const events: unknown[] = []
    let socket: FakeSocket | undefined
    const client = new RealtimeClient({
      workspaceId: 'workspace-1',
      url: 'ws://localhost/ws?workspace_id=workspace-1',
      cursor: 'cursor-7',
      WebSocketImpl: class {
        static OPEN = 1
        constructor() {
          socket = new FakeSocket()
          return socket as unknown as WebSocket
        }
      } as unknown as typeof WebSocket,
      onStatus: (status) => statuses.push(status),
      onEvent: (event) => events.push(event),
    })

    client.connect()
    socket?.open()
    expect(statuses).toContain('recovering')
    expect(JSON.parse(socket?.sent[0] ?? '{}').data.cursor).toBe('cursor-7')
    socket?.message({ event_id: 'event-1', type: 'alarm.updated', data: {} })
    expect(events).toHaveLength(1)
    client.disconnect()
  })

  it('uses bounded reconnect backoff after a close', () => {
    vi.useFakeTimers()
    const sockets: FakeSocket[] = []
    const client = new RealtimeClient({
      workspaceId: 'workspace-1',
      url: 'ws://localhost/ws',
      WebSocketImpl: class {
        static OPEN = 1
        constructor() {
          const socket = new FakeSocket()
          sockets.push(socket)
          return socket as unknown as WebSocket
        }
      } as unknown as typeof WebSocket,
    })
    client.connect()
    sockets[0].close()
    vi.advanceTimersByTime(20_000)
    expect(sockets.length).toBeGreaterThan(1)
    client.disconnect()
    vi.useRealTimers()
  })

  it('allows Pilot consumers to subscribe without the command channel', () => {
    let socket: FakeSocket | undefined
    const client = new RealtimeClient({
      workspaceId: 'workspace-1',
      channels: ['telemetry', 'state', 'alarm'],
      WebSocketImpl: class {
        static OPEN = 1
        constructor() {
          socket = new FakeSocket()
          return socket as unknown as WebSocket
        }
      } as unknown as typeof WebSocket,
    })

    client.connect()
    socket?.open()
    expect(JSON.parse(socket?.sent[0] ?? '{}').data.channels).toEqual([
      'telemetry',
      'state',
      'alarm',
    ])
    client.disconnect()
  })
})
