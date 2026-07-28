export type PilotGateEvidenceStatus = 'missing' | 'draft' | 'submitted' | 'approved' | 'rejected'
export type PilotReadinessMode = 'mock' | 'blocked' | 'pilot2_read_only' | 'pilot2_control'

export interface PilotGateEvidence {
  supportedModel: PilotGateEvidenceStatus
  credentialsLicense: PilotGateEvidenceStatus
  authorizedLab: PilotGateEvidenceStatus
  securityPrivacy: PilotGateEvidenceStatus
  operatingSop: PilotGateEvidenceStatus
  commandDrcApproval: PilotGateEvidenceStatus
}

export interface PilotCapabilityRequest {
  readOnly: boolean
  commands: boolean
  drc: boolean
}

export interface PilotReadinessInput {
  target: 'browser_mock' | 'pilot2'
  evidence: PilotGateEvidence
  capabilities: PilotCapabilityRequest
}

export type PilotReadinessBlocker =
  | 'MOCK_CONTROL_NOT_ALLOWED'
  | 'READ_ONLY_BASELINE_REQUIRED'
  | 'SUPPORTED_MODEL_NOT_APPROVED'
  | 'CREDENTIALS_LICENSE_NOT_APPROVED'
  | 'AUTHORIZED_LAB_NOT_APPROVED'
  | 'SECURITY_PRIVACY_NOT_APPROVED'
  | 'OPERATING_SOP_NOT_APPROVED'
  | 'COMMAND_DRC_NOT_APPROVED'

export interface PilotReadinessDecision {
  allowed: boolean
  mode: PilotReadinessMode
  blockers: readonly PilotReadinessBlocker[]
}

const readOnlyEvidence: Array<[keyof PilotGateEvidence, PilotReadinessBlocker]> = [
  ['supportedModel', 'SUPPORTED_MODEL_NOT_APPROVED'],
  ['credentialsLicense', 'CREDENTIALS_LICENSE_NOT_APPROVED'],
  ['authorizedLab', 'AUTHORIZED_LAB_NOT_APPROVED'],
  ['securityPrivacy', 'SECURITY_PRIVACY_NOT_APPROVED'],
  ['operatingSop', 'OPERATING_SOP_NOT_APPROVED'],
]

export function evaluatePilotReadiness(input: PilotReadinessInput): PilotReadinessDecision {
  if (input.target === 'browser_mock') {
    return input.capabilities.commands || input.capabilities.drc
      ? { allowed: false, mode: 'blocked', blockers: ['MOCK_CONTROL_NOT_ALLOWED'] }
      : { allowed: true, mode: 'mock', blockers: [] }
  }

  const blockers: PilotReadinessBlocker[] = []
  if (!input.capabilities.readOnly) blockers.push('READ_ONLY_BASELINE_REQUIRED')
  for (const [key, blocker] of readOnlyEvidence) {
    if (input.evidence[key] !== 'approved') blockers.push(blocker)
  }
  if ((input.capabilities.commands || input.capabilities.drc) && input.evidence.commandDrcApproval !== 'approved') {
    blockers.push('COMMAND_DRC_NOT_APPROVED')
  }

  if (blockers.length) return { allowed: false, mode: 'blocked', blockers }
  return {
    allowed: true,
    mode: input.capabilities.commands || input.capabilities.drc ? 'pilot2_control' : 'pilot2_read_only',
    blockers: [],
  }
}
