import { describe, expect, it } from 'vitest'
import { createPilotDraftStore, type PilotDraftStorage } from './drafts'

function memoryStorage(): PilotDraftStorage {
  const values = new Map<string, string>()
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  }
}

describe('PilotDraftStore', () => {
  it('persists only a workspace-scoped, non-sensitive field note', () => {
    const storage = memoryStorage()
    const first = createPilotDraftStore('workspace-1', storage, () => Date.parse('2026-07-28T01:00:00Z'))
    const draft = first.save({ deviceId: 'aircraft-1', body: '  检查北侧风况，等待云端恢复。  ' })
    const second = createPilotDraftStore('workspace-1', storage)

    expect(draft.body).toBe('检查北侧风况，等待云端恢复。')
    expect(second.list()).toEqual([draft])
    expect(JSON.stringify(second.list())).not.toMatch(/token|secret|diagnostic|日志路径/i)
  })

  it('rejects credentials and diagnostic paths', () => {
    const store = createPilotDraftStore('workspace-1', memoryStorage())

    expect(() => store.save({ body: 'token=do-not-store' })).toThrow('现场草稿仅支持')
    expect(() => store.save({ body: '日志路径 C:\\pilot\\diagnostics\\latest.log' })).toThrow('现场草稿仅支持')
  })

  it('removes drafts explicitly and never submits them', () => {
    const store = createPilotDraftStore('workspace-1', memoryStorage())
    const draft = store.save({ body: '重新确认作业区域' })

    store.remove(draft.id)

    expect(store.list()).toEqual([])
    expect(store).not.toHaveProperty('submit')
    expect(store).not.toHaveProperty('retry')
  })
})
