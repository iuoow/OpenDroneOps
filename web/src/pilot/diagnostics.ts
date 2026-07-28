import { readonly, ref, type DeepReadonly, type Ref } from 'vue'

export type PilotDiagnosticPhase =
  | 'idle'
  | 'consent_required'
  | 'preparing'
  | 'ready'
  | 'cancelled'
  | 'failed'

export interface PilotDiagnosticReceipt {
  receiptId: string
  itemCount: number
  redactedFields: readonly string[]
  generatedAt: string
}

export type PilotDiagnosticState =
  | { phase: 'idle' }
  | { phase: 'consent_required' }
  | { phase: 'preparing' }
  | { phase: 'ready'; receipt: PilotDiagnosticReceipt }
  | { phase: 'cancelled' }
  | { phase: 'failed'; reason: 'PREPARATION_FAILED' }

export interface PilotDiagnosticRunner {
  /**
   * A future approved adapter may prepare a redacted summary. It never returns
   * filesystem paths, raw logs, credentials, or upload endpoints.
   */
  prepare(): Promise<PilotDiagnosticReceipt>
}

export interface PilotDiagnosticController {
  readonly state: DeepReadonly<Ref<PilotDiagnosticState>>
  begin(): void
  accept(): Promise<void>
  cancel(): void
  retry(): void
  reset(): void
}

export function createPilotDiagnosticController(
  runner: PilotDiagnosticRunner,
  now: () => number = Date.now,
): PilotDiagnosticController {
  const state = ref<PilotDiagnosticState>({ phase: 'idle' })
  let attempt = 0

  function begin() {
    if (state.value.phase === 'preparing') return
    state.value = { phase: 'consent_required' }
  }

  async function accept() {
    if (state.value.phase !== 'consent_required') return
    const currentAttempt = ++attempt
    state.value = { phase: 'preparing' }
    try {
      const receipt = sanitizeReceipt(await runner.prepare(), now)
      if (currentAttempt !== attempt || state.value.phase !== 'preparing') return
      state.value = { phase: 'ready', receipt }
    } catch {
      if (currentAttempt !== attempt || state.value.phase !== 'preparing') return
      state.value = { phase: 'failed', reason: 'PREPARATION_FAILED' }
    }
  }

  function cancel() {
    if (state.value.phase !== 'consent_required' && state.value.phase !== 'preparing') return
    attempt += 1
    state.value = { phase: 'cancelled' }
  }

  function retry() {
    if (state.value.phase === 'failed' || state.value.phase === 'cancelled' || state.value.phase === 'ready') {
      begin()
    }
  }

  function reset() {
    attempt += 1
    state.value = { phase: 'idle' }
  }

  return {
    state: readonly(state),
    begin,
    accept,
    cancel,
    retry,
    reset,
  }
}

export function createBrowserMockDiagnosticRunner(
  now: () => number = Date.now,
): PilotDiagnosticRunner {
  return {
    async prepare() {
      return {
        receiptId: `mock-diagnostic-${now()}`,
        itemCount: 0,
        redactedFields: ['credentials', 'filesystem_path', 'raw_logs'],
        generatedAt: new Date(now()).toISOString(),
      }
    },
  }
}

function sanitizeReceipt(value: PilotDiagnosticReceipt, now: () => number): PilotDiagnosticReceipt {
  const receiptId = /^[a-z0-9][a-z0-9._-]{2,80}$/i.test(value.receiptId)
    ? value.receiptId
    : `redacted-diagnostic-${now()}`
  const itemCount = Number.isInteger(value.itemCount) && value.itemCount >= 0 ? Math.min(value.itemCount, 100) : 0
  const redactedFields = value.redactedFields
    .filter((field) => /^[a-z0-9_ -]{1,40}$/i.test(field))
    .slice(0, 20)
  return {
    receiptId,
    itemCount,
    redactedFields,
    generatedAt: new Date(now()).toISOString(),
  }
}
