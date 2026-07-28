import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Pinia } from 'pinia'
import AlarmsView from './AlarmsView.vue'
import { useOperationsStore } from '../state/operations'

describe('AlarmsView', () => {
  let pinia: Pinia
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('keeps acknowledgement distinct from automatic recovery in the incident detail', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    const wrapper = mount(AlarmsView, { global: { plugins: [pinia], stubs: { RouterLink: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('待接手')
    expect(wrapper.get('button.button--primary').text()).toContain('确认接手此告警')
    await wrapper.get('button.button--primary').trigger('click')
    await flushPromises()

    expect(store.activeAlarms[0].status).toBe('ACKNOWLEDGED')
    expect(wrapper.text()).toContain('已接手')
    expect(wrapper.text()).toContain('不由接手操作自动完成')
    expect(wrapper.text()).not.toContain('规则已确认恢复')
  })
})
