import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Pinia } from 'pinia'
import ReplayView from './ReplayView.vue'
import { useOperationsStore } from '../state/operations'

describe('ReplayView', () => {
  let pinia: Pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('keeps historical evidence separate and lets an event locate the shared playback time', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/app/:workspaceId/replay/:deviceId?', component: ReplayView }] })
    await router.push('/app/demo/replay/aircraft-01?hours=2')
    await router.isReady()
    const wrapper = mount(ReplayView, { global: { plugins: [pinia, router], stubs: { RouterLink: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('这是只读历史证据')
    expect(wrapper.text()).toContain('证据边界')
    expect(wrapper.get('.replay-event-card').text()).toContain('指令更新')
    const slider = wrapper.get('input[type="range"]').element as HTMLInputElement
    expect(slider.value).toBe('0')

    await wrapper.get('.replay-event-card').trigger('click')
    await flushPromises()
    expect(Number(slider.value)).toBeGreaterThan(0)
    expect(wrapper.text()).toContain('指令更新')
  })
})
