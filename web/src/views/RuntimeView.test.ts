import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Pinia } from 'pinia'
import RuntimeView from './RuntimeView.vue'
import { useOperationsStore } from '../state/operations'

describe('RuntimeView', () => {
  let pinia: Pinia
  beforeEach(() => { pinia = createPinia(); setActivePinia(pinia) })

  it('shows browser evidence while preserving the management-plane capacity boundary', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    const wrapper = mount(RuntimeView, { global: { plugins: [pinia], stubs: { RouterLink: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('实时连接正常')
    expect(wrapper.text()).toContain('容量指标不暴露给租户 Web 会话')
    expect(wrapper.text()).toContain('MQTT PUBACK 不代表设备执行成功')
    expect(wrapper.text()).not.toContain('执行容量管理操作')
  })
})
