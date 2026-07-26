import type { TrajectoryPoint } from '../types/contracts'

export const MAX_REPLAY_POINTS = 5000
export const MAX_REPLAY_WINDOW_MS = 24 * 60 * 60 * 1000

export function clampIndex(index: number, length: number) {
  return Math.max(0, Math.min(Math.max(length - 1, 0), Math.round(index)))
}

export function nextPlaybackIndex(index: number, length: number, speed: number) {
  if (length < 2) return 0
  return clampIndex(index + Math.max(speed, 0.5), length)
}

export function downsample(points: TrajectoryPoint[], maxPoints = 1000) {
  if (points.length <= maxPoints) return points
  const stride = Math.ceil(points.length / maxPoints)
  return points.filter((_, index) => index % stride === 0 || index === points.length - 1)
}

export function formatPlaybackTime(value?: string) {
  if (!value) return '暂无时间'
  const date = new Date(value)
  return Number.isNaN(date.getTime())
    ? '暂无时间'
    : date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
