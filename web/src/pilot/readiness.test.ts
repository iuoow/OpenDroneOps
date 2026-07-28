import { describe, expect, it } from 'vitest'
import { evaluatePilotReadiness, type PilotGateEvidence } from './readiness'

const approvedEvidence: PilotGateEvidence = {
  supportedModel: 'approved',
  credentialsLicense: 'approved',
  authorizedLab: 'approved',
  securityPrivacy: 'approved',
  operatingSop: 'approved',
  commandDrcApproval: 'approved',
}

const readOnlyRequest = { readOnly: true, commands: false, drc: false }

describe('evaluatePilotReadiness', () => {
  it('allows the browser Mock Bridge only for read-only capabilities', () => {
    expect(
      evaluatePilotReadiness({
        target: 'browser_mock',
        evidence: {
          supportedModel: 'missing',
          credentialsLicense: 'missing',
          authorizedLab: 'missing',
          securityPrivacy: 'missing',
          operatingSop: 'missing',
          commandDrcApproval: 'missing',
        },
        capabilities: readOnlyRequest,
      }),
    ).toEqual({ allowed: true, mode: 'mock', blockers: [] })
  })

  it('blocks mock command and DRC requests', () => {
    const decision = evaluatePilotReadiness({
      target: 'browser_mock',
      evidence: approvedEvidence,
      capabilities: { readOnly: true, commands: true, drc: false },
    })

    expect(decision).toEqual({
      allowed: false,
      mode: 'blocked',
      blockers: ['MOCK_CONTROL_NOT_ALLOWED'],
    })
  })

  it('reports every missing real read-only prerequisite without leaking details', () => {
    const decision = evaluatePilotReadiness({
      target: 'pilot2',
      evidence: {
        supportedModel: 'missing',
        credentialsLicense: 'draft',
        authorizedLab: 'submitted',
        securityPrivacy: 'rejected',
        operatingSop: 'missing',
        commandDrcApproval: 'missing',
      },
      capabilities: readOnlyRequest,
    })

    expect(decision).toEqual({
      allowed: false,
      mode: 'blocked',
      blockers: [
        'SUPPORTED_MODEL_NOT_APPROVED',
        'CREDENTIALS_LICENSE_NOT_APPROVED',
        'AUTHORIZED_LAB_NOT_APPROVED',
        'SECURITY_PRIVACY_NOT_APPROVED',
        'OPERATING_SOP_NOT_APPROVED',
      ],
    })
    expect(JSON.stringify(decision)).not.toMatch(/token|secret|path|serial/i)
  })

  it('allows real Pilot 2 read-only integration only after the five baseline approvals', () => {
    const decision = evaluatePilotReadiness({
      target: 'pilot2',
      evidence: { ...approvedEvidence, commandDrcApproval: 'missing' },
      capabilities: readOnlyRequest,
    })

    expect(decision).toEqual({ allowed: true, mode: 'pilot2_read_only', blockers: [] })
  })

  it('keeps controls blocked until the separate Command/DRC approval exists', () => {
    const decision = evaluatePilotReadiness({
      target: 'pilot2',
      evidence: { ...approvedEvidence, commandDrcApproval: 'submitted' },
      capabilities: { readOnly: true, commands: true, drc: true },
    })

    expect(decision).toEqual({
      allowed: false,
      mode: 'blocked',
      blockers: ['COMMAND_DRC_NOT_APPROVED'],
    })
  })

  it('allows control mode only with every approved gate', () => {
    const decision = evaluatePilotReadiness({
      target: 'pilot2',
      evidence: approvedEvidence,
      capabilities: { readOnly: true, commands: true, drc: true },
    })

    expect(decision).toEqual({ allowed: true, mode: 'pilot2_control', blockers: [] })
  })
})
