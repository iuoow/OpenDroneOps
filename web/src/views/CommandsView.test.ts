import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Pinia } from 'pinia'
import CommandsView from './CommandsView.vue'
import { useOperationsStore } from '../state/operations'

describe('CommandsView', () => {
  let pinia: Pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('keeps device confirmation distinct from transport publication and creates only the low-risk refresh', async () => {
    const store = useOperationsStore()
    await store.hydrate('demo')
    const wrapper = mount(CommandsView, { global: { plugins: [pinia], stubs: { RouterLink: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('设备结果已确认成功。')
    expect(wrapper.text()).toContain('仅低风险')
    expect(wrapper.get('[data-testid="command-status-refresh"]').text()).toContain('发送低风险状态刷新')

    await wrapper.get('[data-testid="command-status-refresh"]').trigger('click')
    await flushPromises()

    expect(store.commandList).toHaveLength(2)
    expect(store.commandList[0].method).toBe('sim_status_refresh')
    expect(store.commandList[0].risk_level).toBe('LOW')
    expect(store.commandList[0].status).toBe('PUBLISH_PENDING')
    expect(wrapper.text()).toContain('已创建低风险状态刷新指令；正在等待发布与设备证据。')
    expect(wrapper.text()).toContain('尚未获得 MQTT 发布确认。')
    expect(wrapper.text()).not.toContain('设备结果已确认成功。设备')
  })
})
