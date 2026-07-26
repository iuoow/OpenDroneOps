import { describe, expect, it } from 'vitest'
import { clampIndex, downsample, nextPlaybackIndex } from './playback'

describe('trajectory playback helpers', () => {
  it('keeps playback inside the loaded point window', () => {
    expect(clampIndex(-2, 3)).toBe(0)
    expect(clampIndex(9, 3)).toBe(2)
    expect(nextPlaybackIndex(1, 3, 4)).toBe(2)
    expect(nextPlaybackIndex(1, 0, 4)).toBe(0)
  })

  it('preserves endpoints when simplifying a large track', () => {
    const points = Array.from({ length: 2501 }, (_, index) => ({ id: String(index) }))
    const result = downsample(points as never, 1000)
    expect(result.length).toBeLessThanOrEqual(1001)
    expect(result[0].id).toBe('0')
    expect(result.at(-1)?.id).toBe('2500')
  })
})
