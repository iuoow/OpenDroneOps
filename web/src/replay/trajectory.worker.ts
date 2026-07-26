import { downsample } from './playback'

self.onmessage = (event: MessageEvent) => {
  const points = Array.isArray(event.data) ? event.data : []
  self.postMessage(downsample(points))
}
