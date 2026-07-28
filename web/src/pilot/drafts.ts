export interface PilotDraft {
  id: string
  workspaceId: string
  deviceId?: string
  body: string
  createdAt: string
  updatedAt: string
}

export interface PilotDraftInput {
  deviceId?: string
  body: string
}

export interface PilotDraftStore {
  list(): readonly PilotDraft[]
  save(input: PilotDraftInput): PilotDraft
  remove(id: string): void
}

export interface PilotDraftStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

const STORAGE_VERSION = 1
const MAX_BODY_LENGTH = 500
const SENSITIVE_INPUT =
  /(password|passwd|secret|token|api[\s_-]?key|authorization|bearer|private[\s_-]?key|credential|日志路径|诊断日志|[a-z]:\\|\/(?:etc|var|tmp|logs?)(?:\/|$))/i

export function createPilotDraftStore(
  workspaceId: string,
  storage: PilotDraftStorage | undefined = browserStorage(),
  now: () => number = Date.now,
): PilotDraftStore {
  const key = `opendroneops.pilot.drafts.v${STORAGE_VERSION}:${workspaceId}`
  let drafts = loadDrafts(storage, key, workspaceId)
  let sequence = 0

  return {
    list() {
      return drafts.map((draft) => ({ ...draft }))
    },
    save(input) {
      const body = normalizeBody(input.body)
      if (!body || body.length > MAX_BODY_LENGTH || SENSITIVE_INPUT.test(body)) {
        throw new Error('现场草稿仅支持不含凭据或诊断路径的简短备注')
      }
      const timestamp = new Date(now()).toISOString()
      const draft: PilotDraft = {
        id: `pilot-draft-${now()}-${sequence++}`,
        workspaceId,
        ...(input.deviceId ? { deviceId: input.deviceId } : {}),
        body,
        createdAt: timestamp,
        updatedAt: timestamp,
      }
      drafts = [draft, ...drafts].slice(0, 20)
      persist(storage, key, drafts)
      return { ...draft }
    },
    remove(id) {
      const next = drafts.filter((draft) => draft.id !== id)
      if (next.length === drafts.length) return
      drafts = next
      persist(storage, key, drafts)
    },
  }
}

function normalizeBody(body: string) {
  return body.replace(/\s+/g, ' ').trim()
}

function loadDrafts(storage: PilotDraftStorage | undefined, key: string, workspaceId: string) {
  if (!storage) return [] as PilotDraft[]
  try {
    const parsed = JSON.parse(storage.getItem(key) ?? '[]') as unknown
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(isDraft)
      .filter((draft) => draft.workspaceId === workspaceId)
      .slice(0, 20)
  } catch {
    storage.removeItem(key)
    return []
  }
}

function persist(storage: PilotDraftStorage | undefined, key: string, drafts: readonly PilotDraft[]) {
  if (!storage) return
  storage.setItem(key, JSON.stringify(drafts))
}

function isDraft(value: unknown): value is PilotDraft {
  if (!value || typeof value !== 'object') return false
  const draft = value as Partial<PilotDraft>
  return (
    typeof draft.id === 'string' &&
    typeof draft.workspaceId === 'string' &&
    (draft.deviceId === undefined || typeof draft.deviceId === 'string') &&
    typeof draft.body === 'string' &&
    typeof draft.createdAt === 'string' &&
    typeof draft.updatedAt === 'string'
  )
}

function browserStorage(): PilotDraftStorage | undefined {
  if (typeof window === 'undefined') return undefined
  try {
    return window.localStorage
  } catch {
    return undefined
  }
}
