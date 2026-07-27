import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PilotRuntimeConfig } from './bridge'
import PilotBootstrapApp from './PilotBootstrapApp.vue'
import { BrowserMockPilotBridge } from './mockBridge'

const config: PilotRuntimeConfig = {
  workspaceId: 'field-workspace',
  api: { baseUrl: 'https://api.example.test/v1' },
  websocket: { url: 'wss://api.example.test/ws' },
  requiredModules: ['flight_status', 'alarm_feed'],
}

describe('PilotBootstrapApp', () => {
  it('renders the touch-first shell only after the Mock Bridge is ready', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: { bridge: new BrowserMockPilotBridge(), config },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="pilot-ready-shell"]').text()).toContain('现场巡检')
    expect(wrapper.get('nav[aria-label="Pilot 主导航"]').findAll('button')).toHaveLength(4)
    expect(wrapper.get('button[aria-pressed="true"]').text()).toContain('主页')
    expect(wrapper.text()).toContain('Mock Bridge 演示')
  })

  it('keeps the shell unavailable and exposes retry when the Mock Bridge is absent', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: { bridge: new BrowserMockPilotBridge({ available: false }), config },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="pilot-ready-shell"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="pilot-startup-state"]').text()).toContain('未检测到')
    expect(wrapper.get('button').text()).toContain('重新检测 Pilot 环境')
  })

  it('shows a non-retryable field instruction when the license is rejected', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge({
          licenseResult: { accepted: false, reason: 'LICENSE_REJECTED' },
        }),
        config,
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="pilot-startup-state"]').text()).toContain('许可未通过验证')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.text()).toContain('联系已授权的现场管理员')
  })

  it('marks local navigation with an accessible pressed state', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: { bridge: new BrowserMockPilotBridge(), config },
    })
    await flushPromises()
    const alerts = wrapper.get('button:nth-child(3)')
    await alerts.trigger('click')
    expect(alerts.attributes('aria-pressed')).toBe('true')
  })
})
