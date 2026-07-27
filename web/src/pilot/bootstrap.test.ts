import { describe, expect, it } from 'vitest'
import type { PilotRuntimeConfig, PilotStartupState } from './bridge'
import { bootstrapPilot } from './bootstrap'
import { BrowserMockPilotBridge } from './mockBridge'

const config: PilotRuntimeConfig = {
  workspaceId: 'workspace-1',
  api: { baseUrl: 'https://api.example.test/v1' },
  websocket: { url: 'wss://api.example.test/ws' },
  requiredModules: ['flight_status', 'alarm_feed'],
}

describe('bootstrapPilot', () => {
  it('reaches ready through the browser Mock Bridge', async () => {
    const bridge = new BrowserMockPilotBridge()
    const states: PilotStartupState[] = []

    await expect(bootstrapPilot({ bridge, config, onState: (state) => states.push(state) })).resolves.toEqual({
      phase: 'ready',
      bridge: 'mock',
    })
    expect(bridge.recordedCalls()).toEqual([
      'verifyLicense',
      'setWorkspace:workspace-1',
      'configureApi:https://api.example.test/v1',
      'configureWebSocket:wss://api.example.test/ws',
    ])
    expect(states.map((state) => state.phase)).toEqual([
      'verifying_license',
      'configuring',
      'loading_modules',
      'ready',
    ])
  })

  it('reports an unavailable bridge without invoking configuration', async () => {
    const bridge = new BrowserMockPilotBridge({ available: false })
    await expect(bootstrapPilot({ bridge, config })).resolves.toEqual({
      phase: 'failed',
      code: 'BRIDGE_UNAVAILABLE',
      retryable: true,
    })
    expect(bridge.recordedCalls()).toEqual([])
  })

  it('reports a rejected license as a non-retryable startup state', async () => {
    const bridge = new BrowserMockPilotBridge({
      licenseResult: { accepted: false, reason: 'LICENSE_REJECTED' },
    })
    await expect(bootstrapPilot({ bridge, config })).resolves.toEqual({
      phase: 'failed',
      code: 'LICENSE_REJECTED',
      retryable: false,
    })
    expect(bridge.recordedCalls()).toEqual(['verifyLicense'])
  })

  it('normalizes Mock Bridge configuration failures without raw error details', async () => {
    const bridge = new BrowserMockPilotBridge({ rejectWebSocketConfiguration: true })
    const result = await bootstrapPilot({ bridge, config })
    expect(result).toEqual({
      phase: 'failed',
      code: 'CONFIGURATION_REJECTED',
      retryable: true,
    })
    expect(result).not.toHaveProperty('message')
    expect(result).not.toHaveProperty('stack')
  })
})
