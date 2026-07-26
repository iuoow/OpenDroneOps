import { describe, expect, it, vi } from 'vitest'
import { ApiClient, ApiError } from './client'

describe('ApiClient', () => {
  it('adds workspace and idempotency headers to command requests', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ id: 'command-1' }), {
        status: 200,
        headers: { 'content-type': 'application/json', 'X-Request-ID': 'request-1' },
      }),
    )
    const client = new ApiClient({ baseUrl: '/api/v1', fetcher })
    await client.createCommand('workspace-1', {
      target_device_id: 'device-1',
      method: 'sim_status_refresh',
      parameters: { refresh: true },
      idempotencyKey: 'idem-123456',
    })

    expect(fetcher).toHaveBeenCalledWith(
      '/api/v1/commands',
      expect.objectContaining({
        method: 'POST',
        headers: expect.any(Headers),
      }),
    )
    const headers = (fetcher.mock.calls[0][1] as RequestInit).headers as Headers
    expect(headers.get('X-Workspace-ID')).toBe('workspace-1')
    expect(headers.get('Idempotency-Key')).toBe('idem-123456')
  })

  it('surfaces request id and server message on errors', async () => {
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ message: '权限不足' }), {
        status: 403,
        headers: { 'X-Request-ID': 'request-denied' },
      }),
    )
    const client = new ApiClient({ baseUrl: '/api/v1', fetcher })
    await expect(client.acknowledgeAlarm('workspace-1', 'alarm-1')).rejects.toEqual(
      new ApiError('权限不足', 403, 'request-denied'),
    )
  })
})
