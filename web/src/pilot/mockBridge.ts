import type {
  PilotApiConfig,
  PilotBridgeAdapter,
  PilotLicenseResult,
  PilotWebSocketConfig,
} from './bridge'

export interface BrowserMockPilotBridgeOptions {
  available?: boolean
  licenseResult?: PilotLicenseResult
  rejectWorkspace?: boolean
  rejectApiConfiguration?: boolean
  rejectWebSocketConfiguration?: boolean
}

/**
 * Browser-only development adapter. It intentionally has no dependency on a
 * Pilot 2 global, browser storage, host logs, or real DJI credentials.
 */
export class BrowserMockPilotBridge implements PilotBridgeAdapter {
  readonly kind = 'mock' as const
  private readonly calls: string[] = []

  constructor(private readonly options: BrowserMockPilotBridgeOptions = {}) {}

  isAvailable(): boolean {
    return this.options.available ?? true
  }

  async verifyLicense(): Promise<PilotLicenseResult> {
    this.calls.push('verifyLicense')
    return this.options.licenseResult ?? { accepted: true }
  }

  async setWorkspace(workspaceId: string): Promise<void> {
    this.calls.push(`setWorkspace:${workspaceId}`)
    if (this.options.rejectWorkspace) throw new Error('mock workspace configuration rejected')
  }

  async configureApi(config: PilotApiConfig): Promise<void> {
    this.calls.push(`configureApi:${config.baseUrl}`)
    if (this.options.rejectApiConfiguration) throw new Error('mock api configuration rejected')
  }

  async configureWebSocket(config: PilotWebSocketConfig): Promise<void> {
    this.calls.push(`configureWebSocket:${config.url}`)
    if (this.options.rejectWebSocketConfiguration) {
      throw new Error('mock websocket configuration rejected')
    }
  }

  recordedCalls(): readonly string[] {
    return this.calls
  }
}
