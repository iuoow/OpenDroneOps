import { describe, expect, expectTypeOf, it } from 'vitest'
import {
  initialPilotStartupState,
  reducePilotStartup,
  validatePilotRuntimeConfig,
  type PilotBridgeAdapter,
  type PilotLicenseResult,
  type PilotRuntimeConfig,
} from './bridge'

describe('Pilot bridge contract', () => {
  it('keeps the foundation adapter limited to safe startup configuration', () => {
    expectTypeOf<PilotBridgeAdapter['verifyLicense']>().toEqualTypeOf<
      () => Promise<PilotLicenseResult>
    >()
    expectTypeOf<PilotBridgeAdapter['configureApi']>().toEqualTypeOf<
      (config: { baseUrl: string }) => Promise<void>
    >()
    expectTypeOf<PilotBridgeAdapter['configureWebSocket']>().toEqualTypeOf<
      (config: { url: string }) => Promise<void>
    >()
  })

  it('moves through the allowed startup sequence without retaining bridge details', () => {
    const detecting = initialPilotStartupState()
    const verifying = reducePilotStartup(detecting, { type: 'BRIDGE_DETECTED', bridge: 'mock' })
    const configuring = reducePilotStartup(verifying, { type: 'LICENSE_VERIFIED', bridge: 'mock' })
    const loading = reducePilotStartup(configuring, { type: 'MODULES_LOADING', bridge: 'mock' })
    const ready = reducePilotStartup(loading, { type: 'MODULES_READY', bridge: 'mock' })

    expect(ready).toEqual({ phase: 'ready', bridge: 'mock' })
    expect(ready).not.toHaveProperty('error')
    expect(ready).not.toHaveProperty('path')
  })

  it('normalizes unavailable and rejected states into safe retry behavior', () => {
    const unavailable = reducePilotStartup(initialPilotStartupState(), { type: 'BRIDGE_UNAVAILABLE' })
    expect(unavailable).toEqual({
      phase: 'failed',
      code: 'BRIDGE_UNAVAILABLE',
      retryable: true,
    })
    expect(reducePilotStartup(unavailable, { type: 'RETRY' })).toEqual({ phase: 'detecting' })

    const verifying = reducePilotStartup(initialPilotStartupState(), {
      type: 'BRIDGE_DETECTED',
      bridge: 'mock',
    })
    const rejected = reducePilotStartup(verifying, { type: 'LICENSE_REJECTED' })
    expect(rejected).toEqual({
      phase: 'failed',
      code: 'LICENSE_REJECTED',
      retryable: false,
    })
    expect(reducePilotStartup(rejected, { type: 'RETRY' })).toEqual(rejected)
  })

  it('does not permit out-of-order success events', () => {
    const detecting = initialPilotStartupState()
    expect(reducePilotStartup(detecting, { type: 'MODULES_READY', bridge: 'mock' })).toEqual(detecting)
  })

  it('accepts only non-empty workspace and endpoint configuration', () => {
    const valid: PilotRuntimeConfig = {
      workspaceId: 'workspace-1',
      api: { baseUrl: 'https://api.example.test/v1' },
      websocket: { url: 'wss://api.example.test/ws' },
      requiredModules: ['flight_status', 'alarm_feed'],
    }
    expect(validatePilotRuntimeConfig(valid)).toEqual({ valid: true })
    expect(
      validatePilotRuntimeConfig({
        ...valid,
        workspaceId: ' ',
        websocket: { url: 'https://api.example.test/ws' },
      }),
    ).toEqual({ valid: false, code: 'CONFIGURATION_REJECTED' })
  })
})
