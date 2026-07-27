export const pilotBridgeErrorCodes = [
  'BRIDGE_UNAVAILABLE',
  'LICENSE_REJECTED',
  'CONFIGURATION_REJECTED',
  'REQUIRED_MODULE_UNAVAILABLE',
  'UNEXPECTED',
] as const

export type PilotBridgeErrorCode = (typeof pilotBridgeErrorCodes)[number]
export type PilotBridgeKind = 'mock' | 'pilot2'
export type PilotModule = 'flight_status' | 'alarm_feed' | 'field_notes' | 'diagnostics'

export type PilotLicenseResult =
  | { accepted: true }
  | { accepted: false; reason: 'LICENSE_REJECTED' | 'LICENSE_UNAVAILABLE' }

export interface PilotApiConfig {
  baseUrl: string
}

export interface PilotWebSocketConfig {
  url: string
}

/**
 * The only permitted application boundary to a future Pilot 2 JS bridge.
 * It deliberately excludes credentials, device control, DRC, and diagnostics.
 */
export interface PilotBridgeAdapter {
  readonly kind: PilotBridgeKind
  isAvailable(): boolean
  verifyLicense(): Promise<PilotLicenseResult>
  setWorkspace(workspaceId: string): Promise<void>
  configureApi(config: PilotApiConfig): Promise<void>
  configureWebSocket(config: PilotWebSocketConfig): Promise<void>
}

export interface PilotRuntimeConfig {
  workspaceId: string
  api: PilotApiConfig
  websocket: PilotWebSocketConfig
  requiredModules: readonly PilotModule[]
}

export type PilotStartupState =
  | { phase: 'detecting' }
  | { phase: 'verifying_license'; bridge: PilotBridgeKind }
  | { phase: 'configuring'; bridge: PilotBridgeKind }
  | { phase: 'loading_modules'; bridge: PilotBridgeKind }
  | { phase: 'ready'; bridge: PilotBridgeKind }
  | { phase: 'failed'; code: PilotBridgeErrorCode; retryable: boolean }

export type PilotStartupEvent =
  | { type: 'BRIDGE_DETECTED'; bridge: PilotBridgeKind }
  | { type: 'BRIDGE_UNAVAILABLE' }
  | { type: 'LICENSE_VERIFIED'; bridge: PilotBridgeKind }
  | { type: 'LICENSE_REJECTED' }
  | { type: 'CONFIGURATION_REJECTED' }
  | { type: 'MODULES_LOADING'; bridge: PilotBridgeKind }
  | { type: 'REQUIRED_MODULE_UNAVAILABLE' }
  | { type: 'MODULES_READY'; bridge: PilotBridgeKind }
  | { type: 'UNEXPECTED_FAILURE' }
  | { type: 'RETRY' }

export function initialPilotStartupState(): PilotStartupState {
  return { phase: 'detecting' }
}

/**
 * A pure state reducer keeps startup ordering and user-safe failure states out
 * of UI components. It never stores raw bridge errors, paths, or credentials.
 */
export function reducePilotStartup(
  state: PilotStartupState,
  event: PilotStartupEvent,
): PilotStartupState {
  switch (event.type) {
    case 'BRIDGE_DETECTED':
      return state.phase === 'detecting'
        ? { phase: 'verifying_license', bridge: event.bridge }
        : state
    case 'BRIDGE_UNAVAILABLE':
      return state.phase === 'detecting'
        ? { phase: 'failed', code: 'BRIDGE_UNAVAILABLE', retryable: true }
        : state
    case 'LICENSE_VERIFIED':
      return state.phase === 'verifying_license' && state.bridge === event.bridge
        ? { phase: 'configuring', bridge: event.bridge }
        : state
    case 'LICENSE_REJECTED':
      return state.phase === 'verifying_license'
        ? { phase: 'failed', code: 'LICENSE_REJECTED', retryable: false }
        : state
    case 'CONFIGURATION_REJECTED':
      return state.phase === 'configuring'
        ? { phase: 'failed', code: 'CONFIGURATION_REJECTED', retryable: true }
        : state
    case 'MODULES_LOADING':
      return state.phase === 'configuring' && state.bridge === event.bridge
        ? { phase: 'loading_modules', bridge: event.bridge }
        : state
    case 'REQUIRED_MODULE_UNAVAILABLE':
      return state.phase === 'loading_modules'
        ? { phase: 'failed', code: 'REQUIRED_MODULE_UNAVAILABLE', retryable: true }
        : state
    case 'MODULES_READY':
      return state.phase === 'loading_modules' && state.bridge === event.bridge
        ? { phase: 'ready', bridge: event.bridge }
        : state
    case 'UNEXPECTED_FAILURE':
      return { phase: 'failed', code: 'UNEXPECTED', retryable: true }
    case 'RETRY':
      return state.phase === 'failed' && state.retryable ? initialPilotStartupState() : state
  }
}

export function validatePilotRuntimeConfig(
  config: PilotRuntimeConfig,
): { valid: true } | { valid: false; code: 'CONFIGURATION_REJECTED' } {
  const hasValidWorkspace = config.workspaceId.trim().length > 0
  const hasValidApiUrl = isHttpUrl(config.api.baseUrl)
  const hasValidWebSocketUrl = isWebSocketUrl(config.websocket.url)
  const hasRequiredModules = config.requiredModules.length > 0
  return hasValidWorkspace && hasValidApiUrl && hasValidWebSocketUrl && hasRequiredModules
    ? { valid: true }
    : { valid: false, code: 'CONFIGURATION_REJECTED' }
}

function isHttpUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function isWebSocketUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'ws:' || url.protocol === 'wss:'
  } catch {
    return false
  }
}
