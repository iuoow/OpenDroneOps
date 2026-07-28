import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import OperationsMap from './OperationsMap.vue'
import type { Device } from '../types/contracts'

const device: Device = {
  id: 'aircraft-01',
  workspace_id: 'demo',
  vendor: 'DJI',
  serial_number: 'SIM-AIR-001',
  product_model: 'M350 RTK',
  device_type: 'AIRCRAFT',
  status: 'ONLINE',
  online: true,
  state_version: 42,
  latitude: 31.2304,
  longitude: 121.4737,
  battery_percent: 74,
  server_time: new Date().toISOString(),
}

describe('OperationsMap', () => {
  it('keeps attention, selection, and keyboard-equivalent device labels visible', async () => {
    const wrapper = mount(OperationsMap, {
      props: {
        devices: [device],
        selectedId: device.id,
        attentionDeviceIds: [device.id],
      },
    })

    const marker = wrapper.get('button.map-device')
    expect(marker.classes()).toContain('map-device--selected')
    expect(marker.classes()).toContain('map-device--attention')
    expect(marker.attributes('aria-label')).toContain('SIM-AIR-001')

    await marker.trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([device])
  })
})
