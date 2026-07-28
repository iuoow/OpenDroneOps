import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PilotRuntimeConfig } from './bridge'
import PilotBootstrapApp from './PilotBootstrapApp.vue'
import { BrowserMockPilotBridge } from './mockBridge'
import { createBrowserMockDiagnosticRunner, createPilotDiagnosticController } from './diagnostics'
import { createPilotDraftStore } from './drafts'
import { createPilotReadModel } from './readModel'

const config: PilotRuntimeConfig = {
  workspaceId: 'field-workspace',
  api: { baseUrl: 'https://api.example.test/v1' },
  websocket: { url: 'wss://api.example.test/ws' },
  requiredModules: ['flight_status', 'alarm_feed'],
}

const makeReadModel = () =>
  createPilotReadModel({
    workspaceId: config.workspaceId,
    apiBaseUrl: config.api.baseUrl,
    websocketUrl: config.websocket.url,
    demo: true,
    now: () => Date.parse('2026-07-28T00:00:10Z'),
  })

const makeDraftStore = () => {
  const values = new Map<string, string>()
  return createPilotDraftStore(config.workspaceId, {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  })
}

const makeDiagnostics = () =>
  createPilotDiagnosticController(createBrowserMockDiagnosticRunner(() => Date.parse('2026-07-28T00:00:30Z')))

describe('PilotBootstrapApp', () => {
  it('renders the touch-first shell only after the Mock Bridge is ready', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge(),
        config,
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="pilot-ready-shell"]').text()).toContain('现场巡检')
    expect(wrapper.get('nav[aria-label="Pilot 主导航"]').findAll('button')).toHaveLength(4)
    expect(wrapper.get('button[aria-pressed="true"]').text()).toContain('主页')
    expect(wrapper.text()).toContain('Mock Bridge 演示')
  })

  it('keeps the shell unavailable and exposes retry when the Mock Bridge is absent', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge({ available: false }),
        config,
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
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
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="pilot-startup-state"]').text()).toContain('许可未通过验证')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.text()).toContain('联系已授权的现场管理员')
  })

  it('marks local navigation with an accessible pressed state', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge(),
        config,
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
    })
    await flushPromises()
    const alerts = wrapper.findAll('nav button')[2]
    expect(alerts).toBeDefined()
    await alerts.trigger('click')
    expect(alerts.attributes('aria-pressed')).toBe('true')
    expect(wrapper.text()).toContain('现场告警')
  })

  it('saves, retries, and discards a local draft without submission controls', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge(),
        config,
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
    })
    await flushPromises()

    await wrapper.get('[data-testid="pilot-draft-body"]').setValue('北侧风况变化，待网络恢复后复核')
    await wrapper.get('[data-testid="pilot-save-draft"]').trigger('click')

    expect(wrapper.get('[data-testid="pilot-draft-list"]').text()).toContain('北侧风况变化')
    expect(wrapper.text()).toContain('不会自动提交')
    expect(wrapper.find('button[data-testid="pilot-submit-draft"]').exists()).toBe(false)

    const draftActions = wrapper.findAll('.pilot-shell__draft-item-actions button')
    await draftActions[0].trigger('click')
    expect((wrapper.get('[data-testid="pilot-draft-body"]').element as HTMLTextAreaElement).value).toContain(
      '北侧风况变化',
    )
    await draftActions[1].trigger('click')
    expect(wrapper.find('[data-testid="pilot-draft-list"]').exists()).toBe(false)
  })

  it('keeps diagnostics consent-first and cancellable in the More view', async () => {
    const wrapper = mount(PilotBootstrapApp, {
      props: {
        bridge: new BrowserMockPilotBridge(),
        config,
        readModel: makeReadModel(),
        draftStore: makeDraftStore(),
        diagnostics: makeDiagnostics(),
      },
    })
    await flushPromises()

    await wrapper.findAll('nav button')[3].trigger('click')
    expect(wrapper.get('[data-testid="pilot-diagnostic-status"]').text()).toContain('尚未开始')
    await wrapper.get('[data-testid="pilot-diagnostic-begin"]').trigger('click')
    expect(wrapper.get('[data-testid="pilot-diagnostic-status"]').text()).toContain('等待你的同意')
    expect(wrapper.find('[data-testid="pilot-diagnostic-accept"]').exists()).toBe(true)
    await wrapper.get('[data-testid="pilot-diagnostic-cancel"]').trigger('click')
    expect(wrapper.get('[data-testid="pilot-diagnostic-status"]').text()).toContain('已取消')
    expect(wrapper.text()).not.toContain('C:\\')
  })
})
