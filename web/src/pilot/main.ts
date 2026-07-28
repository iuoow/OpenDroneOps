import { createApp } from 'vue'
import PilotBootstrapApp from './PilotBootstrapApp.vue'
import { type PilotRuntimeConfig } from './bridge'
import { BrowserMockPilotBridge } from './mockBridge'
import { createBrowserMockDiagnosticRunner, createPilotDiagnosticController } from './diagnostics'
import { createPilotDraftStore } from './drafts'
import { createPilotReadModel } from './readModel'
import {
  assertPilotRuntimeAllowed,
  createUnapprovedPilotEvidence,
  evaluatePilotReadiness,
} from './readiness'
import './styles.css'

const config: PilotRuntimeConfig = {
  workspaceId: import.meta.env.VITE_PILOT_WORKSPACE_ID || 'demo',
  api: {
    baseUrl: import.meta.env.VITE_PILOT_API_BASE_URL || 'http://127.0.0.1:8080/api/v1',
  },
  websocket: {
    url: import.meta.env.VITE_PILOT_WS_URL || 'ws://127.0.0.1:8080/ws',
  },
  requiredModules: ['flight_status', 'alarm_feed', 'field_notes'],
}

const bridge = new BrowserMockPilotBridge()
const readiness = evaluatePilotReadiness({
  target: 'browser_mock',
  evidence: createUnapprovedPilotEvidence(),
  capabilities: { readOnly: true, commands: false, drc: false },
})
assertPilotRuntimeAllowed(bridge.kind, readiness)

createApp(PilotBootstrapApp, {
  bridge,
  config,
  readModel: createPilotReadModel({
    workspaceId: config.workspaceId,
    apiBaseUrl: config.api.baseUrl,
    websocketUrl: config.websocket.url,
    demo: import.meta.env.VITE_DEMO_MODE !== 'false',
  }),
  draftStore: createPilotDraftStore(config.workspaceId),
  diagnostics: createPilotDiagnosticController(createBrowserMockDiagnosticRunner()),
  readiness,
}).mount('#pilot-app')
