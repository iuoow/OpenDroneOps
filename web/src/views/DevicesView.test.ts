import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Pinia } from 'pinia'
import DevicesView from './DevicesView.vue'
import { useOperationsStore } from '../state/operations'

describe('DevicesView', () => {
  let pinia: Pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('filters stale telemetry and preserves the selected device in the URL without adding controls', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    store.devices['dock-01'] = { ...store.devices['dock-01'], server_time: new Date(Date.now() - 90_000).toISOString() }
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/app/:workspaceId/devices', component: DevicesView }] })
    await router.push('/app/demo/devices?device=aircraft-01')
    await router.isReady()
    const wrapper = mount(DevicesView, { global: { plugins: [pinia, router], stubs: { RouterLink: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('设备在线，遥测仍在新鲜度窗口内。')
    await wrapper.findAll('.device-filter__item')[2].trigger('click')
    await flushPromises()
    expect(wrapper.get('.device-list-v2').text()).toContain('SIM-DOCK-001')
    expect(wrapper.get('.device-list-v2').text()).not.toContain('SIM-AIR-001')

    await wrapper.get('.device-card').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.device).toBe('dock-01')
    expect(wrapper.text()).toContain('设备曾在线，但遥测数据可能已过期；不应将其当作实时状态。')
    expect(wrapper.text()).toContain('未提供飞行、Dock 或批量控制操作。')
  })
})
