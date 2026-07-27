import {
  initialPilotStartupState,
  reducePilotStartup,
  validatePilotRuntimeConfig,
  type PilotBridgeAdapter,
  type PilotRuntimeConfig,
  type PilotStartupEvent,
  type PilotStartupState,
} from './bridge'

export interface PilotBootstrapDependencies {
  bridge: PilotBridgeAdapter
  config: PilotRuntimeConfig
  onState?: (state: PilotStartupState) => void
}

/**
 * Starts only the safe Foundation lifecycle. It publishes normalized state to
 * the UI and deliberately discards raw adapter failures.
 */
export async function bootstrapPilot(
  dependencies: PilotBootstrapDependencies,
): Promise<PilotStartupState> {
  let state = initialPilotStartupState()
  const transition = (event: PilotStartupEvent) => {
    state = reducePilotStartup(state, event)
    dependencies.onState?.(state)
  }

  if (!dependencies.bridge.isAvailable()) {
    transition({ type: 'BRIDGE_UNAVAILABLE' })
    return state
  }

  transition({ type: 'BRIDGE_DETECTED', bridge: dependencies.bridge.kind })
  let license
  try {
    license = await dependencies.bridge.verifyLicense()
  } catch {
    transition({ type: 'UNEXPECTED_FAILURE' })
    return state
  }
  if (!license.accepted) {
    transition({ type: 'LICENSE_REJECTED' })
    return state
  }

  transition({ type: 'LICENSE_VERIFIED', bridge: dependencies.bridge.kind })
  if (!validatePilotRuntimeConfig(dependencies.config).valid) {
    transition({ type: 'CONFIGURATION_REJECTED' })
    return state
  }
  try {
    await dependencies.bridge.setWorkspace(dependencies.config.workspaceId)
    await dependencies.bridge.configureApi(dependencies.config.api)
    await dependencies.bridge.configureWebSocket(dependencies.config.websocket)
  } catch {
    transition({ type: 'CONFIGURATION_REJECTED' })
    return state
  }

  transition({ type: 'MODULES_LOADING', bridge: dependencies.bridge.kind })
  transition({ type: 'MODULES_READY', bridge: dependencies.bridge.kind })
  return state
}
