import { describe, expect, it, vi } from 'vitest'
import { createPilotDiagnosticController, createBrowserMockDiagnosticRunner } from './diagnostics'

describe('PilotDiagnosticController', () => {
  it('requires explicit consent before preparing a redacted receipt', async () => {
    const prepare = vi.fn().mockResolvedValue({
      receiptId: 'receipt-1',
      itemCount: 2,
      redactedFields: ['credentials', 'filesystem_path'],
      generatedAt: 'ignored',
    })
    const controller = createPilotDiagnosticController({ prepare })

    await controller.accept()
    expect(prepare).not.toHaveBeenCalled()
    expect(controller.state.value.phase).toBe('idle')

    controller.begin()
    expect(controller.state.value.phase).toBe('consent_required')
    await controller.accept()

    expect(prepare).toHaveBeenCalledTimes(1)
    expect(controller.state.value).toMatchObject({
      phase: 'ready',
      receipt: { receiptId: 'receipt-1', itemCount: 2 },
    })
  })

  it('cancels an in-flight preparation and ignores a late result', async () => {
    let resolve: ((value: { receiptId: string; itemCount: number; redactedFields: string[]; generatedAt: string }) => void) | undefined
    const controller = createPilotDiagnosticController({
      prepare: () =>
        new Promise((done) => {
          resolve = done
        }),
    })

    controller.begin()
    const accepted = controller.accept()
    expect(controller.state.value.phase).toBe('preparing')
    controller.cancel()
    expect(controller.state.value.phase).toBe('cancelled')
    resolve?.({ receiptId: 'late', itemCount: 1, redactedFields: [], generatedAt: 'late' })
    await accepted

    expect(controller.state.value.phase).toBe('cancelled')
  })

  it('redacts invalid receipt metadata and exposes a stable failure state', async () => {
    const controller = createPilotDiagnosticController({
      prepare: vi.fn().mockResolvedValue({
        receiptId: 'C:\\private\\diagnostics\\raw.log',
        itemCount: -4,
        redactedFields: ['raw_logs', '/var/log', 'credentials'],
        generatedAt: 'raw',
      }),
    }, () => Date.parse('2026-07-28T02:00:00Z'))

    controller.begin()
    await controller.accept()

    expect(controller.state.value).toMatchObject({
      phase: 'ready',
      receipt: {
        itemCount: 0,
        redactedFields: ['raw_logs', 'credentials'],
      },
    })
    if (controller.state.value.phase === 'ready') {
      expect(controller.state.value.receipt.receiptId).toMatch(/^redacted-diagnostic-\d+$/)
    }
  })

  it('supports retry after a failure without exposing the raw error', async () => {
    const prepare = vi
      .fn()
      .mockRejectedValueOnce(new Error('filesystem path and token leaked'))
      .mockResolvedValueOnce({
        receiptId: 'receipt-2',
        itemCount: 0,
        redactedFields: ['credentials'],
        generatedAt: 'ignored',
      })
    const controller = createPilotDiagnosticController({ prepare })

    controller.begin()
    await controller.accept()
    expect(controller.state.value).toEqual({ phase: 'failed', reason: 'PREPARATION_FAILED' })
    controller.retry()
    await controller.accept()
    expect(controller.state.value.phase).toBe('ready')
  })

  it('provides a browser-safe mock runner with no path or upload behavior', async () => {
    const controller = createPilotDiagnosticController(
      createBrowserMockDiagnosticRunner(() => Date.parse('2026-07-28T02:00:00Z')),
    )

    controller.begin()
    await controller.accept()

    expect(controller.state.value).toMatchObject({
      phase: 'ready',
      receipt: { itemCount: 0, redactedFields: ['credentials', 'filesystem_path', 'raw_logs'] },
    })
  })
})
