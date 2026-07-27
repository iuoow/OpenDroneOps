import { createApp } from 'vue'
import PilotBootstrapApp from './PilotBootstrapApp.vue'
import { type PilotRuntimeConfig } from './bridge'
import { BrowserMockPilotBridge } from './mockBridge'
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

createApp(PilotBootstrapApp, {
  bridge: new BrowserMockPilotBridge(),
  config,
}).mount('#pilot-app')
