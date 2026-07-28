import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import FreshnessIndicator from './FreshnessIndicator.vue'

describe('FreshnessIndicator', () => {
  it('labels missing or invalid telemetry as unavailable instead of presenting a false fresh value', () => {
    const missing = mount(FreshnessIndicator)
    const invalid = mount(FreshnessIndicator, { props: { timestamp: 'not-a-timestamp' } })
    const offline = mount(FreshnessIndicator, { props: { timestamp: 'not-a-timestamp', online: false } })

    expect(missing.text()).toContain('暂无遥测数据')
    expect(invalid.text()).toContain('暂无遥测数据')
    expect(invalid.classes()).toContain('freshness--unavailable')
    expect(offline.text()).toContain('设备离线')
  })
})
