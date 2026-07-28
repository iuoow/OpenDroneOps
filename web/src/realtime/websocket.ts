import type { WebSocketEnvelope } from '../types/contracts'

export type RealtimeStatus = 'idle' | 'connecting' | 'connected' | 'recovering' | 'disconnected'
export type RealtimeChannel = 'telemetry' | 'state' | 'alarm' | 'command'

export interface RealtimeOptions {
  workspaceId: string
  url?: string
  cursor?: string
  channels?: readonly RealtimeChannel[]
  WebSocketImpl?: typeof WebSocket
  onStatus?: (status: RealtimeStatus, detail?: string) => void
  onEvent?: (event: WebSocketEnvelope) => void
}

export class RealtimeClient {
  private readonly options: RealtimeOptions
  private readonly WebSocketImpl: typeof WebSocket
  private socket?: WebSocket
  private stopped = true
  private reconnectAttempt = 0
  private cursor = ''
  private recovering = false
  private reconnectTimer?: number

  constructor(options: RealtimeOptions) {
    this.options = options
    this.WebSocketImpl = options.WebSocketImpl ?? WebSocket
    this.cursor = options.cursor ?? ''
  }

  connect() {
    this.stopped = false
    this.open()
  }

  disconnect() {
    this.stopped = true
    if (this.reconnectTimer !== undefined) {
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = undefined
    }
    this.socket?.close()
    this.socket = undefined
    this.options.onStatus?.('disconnected', '已主动断开')
  }

  setCursor(cursor: string) {
    this.cursor = cursor
  }

  private open() {
    if (this.stopped) return
    this.options.onStatus?.(this.reconnectAttempt ? 'recovering' : 'connecting')
    const url = this.options.url ?? this.defaultUrl()
    const socket = new this.WebSocketImpl(url)
    this.socket = socket
    socket.addEventListener('open', () => {
      this.reconnectAttempt = 0
      this.recovering = Boolean(this.cursor)
      this.options.onStatus?.(this.recovering ? 'recovering' : 'connected')
      this.sendSubscription()
    })
    socket.addEventListener('message', (message) => this.handleMessage(message.data))
    socket.addEventListener('error', () => this.options.onStatus?.('disconnected', '实时连接错误'))
    socket.addEventListener('close', () => {
      if (this.socket === socket) this.socket = undefined
      if (!this.stopped) this.scheduleReconnect()
    })
  }

  private handleMessage(raw: unknown) {
    try {
      const event = JSON.parse(String(raw)) as WebSocketEnvelope
      if (!event.event_id || !event.type) return
      if (this.recovering) {
        this.recovering = false
        this.options.onStatus?.('connected', '已恢复实时增量')
      }
      this.options.onEvent?.(event)
    } catch { /* Invalid events are ignored; a valid socket remains connected. */ }
  }

  private sendSubscription() {
    if (!this.socket || this.socket.readyState !== this.WebSocketImpl.OPEN) return
    this.socket.send(
      JSON.stringify({
        type: 'subscription.set',
        request_id: crypto.randomUUID(),
        data: {
          channels: this.options.channels ?? ['telemetry', 'state', 'alarm', 'command'],
          cursor: this.cursor || undefined,
        },
      }),
    )
  }

  private scheduleReconnect() {
    this.reconnectAttempt += 1
    const base = Math.min(15_000, 500 * 2 ** Math.min(this.reconnectAttempt - 1, 5))
    const delay = Math.round(base * (0.8 + Math.random() * 0.4))
    this.options.onStatus?.('disconnected', `正在重连（第 ${this.reconnectAttempt} 次）`)
    this.reconnectTimer = window.setTimeout(() => this.open(), delay)
  }

  private defaultUrl() {
    const explicit = import.meta.env.VITE_WS_URL as string | undefined
    if (explicit) return `${explicit}?workspace_id=${encodeURIComponent(this.options.workspaceId)}`
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/api/v1/ws?workspace_id=${encodeURIComponent(this.options.workspaceId)}`
  }
}
