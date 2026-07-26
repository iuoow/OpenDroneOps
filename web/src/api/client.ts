import type { Alarm, Command, Device, DomainEvent, TrajectoryPage } from '../types/contracts'

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly requestId?: string,
  ) {
    super(message)
  }
}

export interface ApiClientOptions {
  baseUrl?: string
  fetcher?: typeof fetch
}

export class ApiClient {
  private readonly baseUrl: string
  private readonly fetcher: typeof fetch

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? (import.meta.env.VITE_API_BASE_URL || '/api/v1')
    this.fetcher = options.fetcher ?? fetch
  }

  async getSnapshot(workspaceId: string) {
    const [devices, alarms, commands, events] = await Promise.all([
      this.getPage<Device>('/devices', workspaceId),
      this.getPage<Alarm>('/alarms?status=OPEN', workspaceId),
      this.getPage<Command>('/commands?limit=50', workspaceId),
      this.getPage<DomainEvent>('/events?limit=50', workspaceId),
    ])
    return {
      devices: devices.items,
      alarms: alarms.items,
      commands: commands.items,
      events: events.items,
      cursor: events.next_cursor ?? '',
    }
  }

  async acknowledgeAlarm(workspaceId: string, alarmId: string) {
    return this.request<Alarm>(`/alarms/${encodeURIComponent(alarmId)}/acknowledge`, workspaceId, {
      method: 'POST',
      body: JSON.stringify({}),
    })
  }

  async createCommand(
    workspaceId: string,
    request: {
      target_device_id: string
      method: string
      parameters: Record<string, unknown>
      idempotencyKey: string
    },
  ) {
    return this.request<Command>('/commands', workspaceId, {
      method: 'POST',
      headers: { 'Idempotency-Key': request.idempotencyKey },
      body: JSON.stringify({
        target_device_id: request.target_device_id,
        method: request.method,
        parameters: request.parameters,
      }),
    })
  }

  async getTrajectory(
    workspaceId: string,
    deviceId: string,
    options: { from: string; to: string; limit?: number; cursor?: string },
  ) {
    const query = new URLSearchParams({
      from: options.from,
      to: options.to,
      limit: String(options.limit ?? 500),
    })
    if (options.cursor) query.set('cursor', options.cursor)
    return this.request<TrajectoryPage>(
      `/devices/${encodeURIComponent(deviceId)}/trajectory?${query.toString()}`,
      workspaceId,
    )
  }

  private async getPage<T>(path: string, workspaceId: string) {
    return this.request<{ items: T[]; next_cursor?: string }>(path, workspaceId)
  }

  private async request<T>(path: string, workspaceId: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers)
    headers.set('Accept', 'application/json')
    headers.set('X-Workspace-ID', workspaceId)
    const response = await this.fetcher(`${this.baseUrl}${path}`, { ...init, headers })
    const requestId = response.headers.get('X-Request-ID') ?? undefined
    const text = await response.text()
    let body: unknown
    try {
      body = text ? JSON.parse(text) : undefined
    } catch {
      body = undefined
    }
    if (!response.ok) {
      const message =
        typeof body === 'object' && body !== null && 'message' in body
          ? String((body as { message: unknown }).message)
          : `请求失败（HTTP ${response.status}）`
      throw new ApiError(message, response.status, requestId)
    }
    return body as T
  }
}
