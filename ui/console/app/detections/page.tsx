'use client'

import { useEffect, useMemo, useState } from 'react'
import type React from 'react'
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle,
  Clock,
  FileCheck2,
  Fingerprint,
  RefreshCw,
  ShieldCheck,
  ShieldQuestion,
  XCircle
} from 'lucide-react'

type CandidateStatus =
  | 'draft'
  | 'validating'
  | 'validation_failed'
  | 'ready_for_review'
  | 'approved'
  | 'rejected'
  | 'signed'

interface Candidate {
  id: string
  research_item_id: string
  title: string
  description?: string
  status: CandidateStatus
  created_at_ms: number
  updated_at_ms: number
  reject_reason?: string
  pack_id: string
  pack_version: string
  min_agent_version: string
  supported_os: string[]
  author?: string
  source?: string
  last_validation_id?: string
  signed_pack_id?: string
}

interface ValidationRun {
  id: string
  candidate_id: string
  started_at_ms: number
  ended_at_ms: number
  success: boolean
  device_id?: string
  events_fetched: number
  matched_rules: number
  details?: string
  errors?: string
}

interface SignedPack {
  id: string
  candidate_id: string
  created_at_ms: number
  signature_algorithm: string
  key_id: string
  pack_bytes: number
  pack_id?: string
  pack_version?: string
  sha256?: string
}

interface SignerInfo {
  algorithm: string
  key_id: string
  public_key_b64: string
  sign_message: string
}

interface AgentRolloutStatus {
  agent_uid: string
  device_id?: string
  reported_agent_version?: string
  rollout_state: string
  active_pack_id?: string
  active_pack_version?: string
  previous_pack_id?: string
  previous_pack_version?: string
  last_check_at_ms?: number
  last_applied_at_ms?: number
  last_rejected_at_ms?: number
  last_rejected_pack_id?: string
  last_rejected_reason?: string
  reason_detail?: string
  reason_codes?: string[]
  signature_status?: string
  hash_status?: string
  schema_status?: string
  compatibility_status?: string
  computed_stale?: boolean
}

interface RolloutStatus {
  pack_id: string
  agents_reported: number
  count_applied: number
  count_rejected: number
  count_incompatible: number
  count_expired: number
  stale_threshold_ms: number
  agents: AgentRolloutStatus[]
}

const statusLabels: Record<CandidateStatus, string> = {
  draft: 'Draft',
  validating: 'Validating',
  validation_failed: 'Validation failed',
  ready_for_review: 'Ready for review',
  approved: 'Approved',
  rejected: 'Rejected',
  signed: 'Signed'
}

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, options)
  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `Request failed: ${response.status}`)
  }
  return response.json()
}

function formatTime(value?: number) {
  if (!value) return 'n/a'
  return new Date(value).toLocaleString()
}

function shortHash(value?: string) {
  if (!value) return 'n/a'
  return value.length > 18 ? `${value.slice(0, 10)}...${value.slice(-8)}` : value
}

function statusBadge(status: CandidateStatus) {
  switch (status) {
    case 'signed':
    case 'approved':
    case 'ready_for_review':
      return <span className="badge badge-success">{statusLabels[status]}</span>
    case 'validation_failed':
    case 'rejected':
      return <span className="badge badge-danger">{statusLabels[status]}</span>
    case 'validating':
      return <span className="badge badge-info">{statusLabels[status]}</span>
    default:
      return <span className="badge badge-warning">{statusLabels[status]}</span>
  }
}

function rolloutBadge(state?: string, stale?: boolean) {
  if (stale) return <span className="badge badge-warning">Stale</span>
  switch (state) {
    case 'applied':
      return <span className="badge badge-success">Applied</span>
    case 'rejected':
    case 'rollback':
      return <span className="badge badge-danger">{state === 'rollback' ? 'Rollback' : 'Rejected'}</span>
    case 'incompatible':
    case 'expired':
    case 'not_checked':
      return <span className="badge badge-warning">{state.replace('_', ' ')}</span>
    default:
      return <span className="badge badge-warning">No status</span>
  }
}

export default function DetectionPacksPage() {
  const [candidates, setCandidates] = useState<Candidate[]>([])
  const [signedPacks, setSignedPacks] = useState<SignedPack[]>([])
  const [signerInfo, setSignerInfo] = useState<SignerInfo | null>(null)
  const [rollouts, setRollouts] = useState<Record<string, RolloutStatus>>({})
  const [selectedCandidateID, setSelectedCandidateID] = useState<string>('')
  const [validations, setValidations] = useState<ValidationRun[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [actingID, setActingID] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const selectedCandidate = useMemo(
    () => candidates.find((candidate) => candidate.id === selectedCandidateID) || candidates[0],
    [candidates, selectedCandidateID]
  )

  const refresh = async () => {
    try {
      setRefreshing(true)
      setError(null)
      const [candidateData, signedData, signerData] = await Promise.all([
        fetchJSON<{ items: Candidate[] }>('/api/detection/candidates'),
        fetchJSON<{ items: SignedPack[] }>('/api/detection/signed-packs'),
        fetchJSON<SignerInfo>('/api/detection/signer-info')
      ])

      const candidateItems = candidateData.items || []
      const signedItems = signedData.items || []
      setCandidates(candidateItems)
      setSignedPacks(signedItems)
      setSignerInfo(signerData)
      setSelectedCandidateID((current) => current || candidateItems[0]?.id || '')

      const packIDs = Array.from(new Set(signedItems.map((pack) => pack.pack_id).filter(Boolean))) as string[]
      const rolloutEntries = await Promise.all(
        packIDs.map(async (packID) => {
          try {
            const status = await fetchJSON<RolloutStatus>(`/api/detection-packs/${encodeURIComponent(packID)}/rollout-status`)
            return [packID, status] as const
          } catch {
            return [packID, null] as const
          }
        })
      )
      const rolloutMap: Record<string, RolloutStatus> = {}
      rolloutEntries.forEach(([packID, status]) => {
        if (status) {
          rolloutMap[packID] = status
        }
      })
      setRollouts(rolloutMap)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load detection packs')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, 30000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    if (!selectedCandidate?.id) {
      setValidations([])
      return
    }
    fetchJSON<{ items: ValidationRun[] }>(`/api/detection/candidates/${encodeURIComponent(selectedCandidate.id)}/validations`)
      .then((data) => setValidations(data.items || []))
      .catch(() => setValidations([]))
  }, [selectedCandidate?.id])

  const runAction = async (candidate: Candidate, action: 'validate' | 'approve' | 'reject' | 'sign') => {
    try {
      setActingID(`${candidate.id}:${action}`)
      setError(null)
      const body = action === 'reject' ? JSON.stringify({ reason: 'Rejected from Detection Packs console' }) : JSON.stringify({})
      await fetchJSON(`/api/detection/candidates/${encodeURIComponent(candidate.id)}/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body
      })
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} candidate`)
    } finally {
      setActingID(null)
    }
  }

  const counts = candidates.reduce<Record<string, number>>((acc, candidate) => {
    acc[candidate.status] = (acc[candidate.status] || 0) + 1
    return acc
  }, {})

  const rolloutTotals = Object.values(rollouts).reduce(
    (acc, rollout) => {
      acc.agents += rollout.agents_reported
      acc.applied += rollout.count_applied
      acc.rejected += rollout.count_rejected
      acc.incompatible += rollout.count_incompatible
      acc.expired += rollout.count_expired
      acc.stale += rollout.agents.filter((agent) => agent.computed_stale).length
      return acc
    },
    { agents: 0, applied: 0, rejected: 0, incompatible: 0, expired: 0, stale: 0 }
  )

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col gap-4 py-6 md:flex-row md:items-center md:justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="flex items-center text-gray-600 hover:text-gray-900">
                <ArrowLeft className="h-5 w-5 mr-2" />
                Back to Dashboard
              </a>
              <div className="hidden h-6 w-px bg-gray-300 md:block" />
              <div className="flex items-center gap-3">
                <ShieldCheck className="h-8 w-8 text-primary-600" />
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">Detection Packs</h1>
                  <p className="text-sm text-gray-500">Candidate review, signing, and lab rollout status</p>
                </div>
              </div>
            </div>
            <button onClick={refresh} disabled={refreshing} className="btn btn-secondary px-4 py-2">
              <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {error && (
          <div className="mb-6 rounded-md border border-danger-200 bg-danger-50 p-4">
            <div className="flex gap-3">
              <AlertTriangle className="h-5 w-5 text-danger-500" />
              <p className="text-sm text-danger-800">{error}</p>
            </div>
          </div>
        )}

        <div className="mb-8 grid grid-cols-2 gap-4 lg:grid-cols-6">
          <Metric icon={<ShieldQuestion className="h-5 w-5" />} label="Candidates" value={candidates.length} />
          <Metric icon={<Clock className="h-5 w-5" />} label="Review" value={counts.ready_for_review || 0} />
          <Metric icon={<CheckCircle className="h-5 w-5" />} label="Signed" value={signedPacks.length} />
          <Metric icon={<FileCheck2 className="h-5 w-5" />} label="Agents" value={rolloutTotals.agents} />
          <Metric icon={<CheckCircle className="h-5 w-5" />} label="Applied" value={rolloutTotals.applied} />
          <Metric icon={<XCircle className="h-5 w-5" />} label="Needs attention" value={rolloutTotals.rejected + rolloutTotals.incompatible + rolloutTotals.expired + rolloutTotals.stale} />
        </div>

        <div className="grid grid-cols-1 gap-8 xl:grid-cols-3">
          <section className="card xl:col-span-2">
            <div className="border-b border-gray-200 px-6 py-4">
              <h2 className="text-lg font-semibold text-gray-900">Candidate Pipeline</h2>
              <p className="text-sm text-gray-500">Validate, approve, reject, and sign detection-pack candidates.</p>
            </div>
            <div className="divide-y divide-gray-200">
              {loading ? (
                <p className="p-6 text-sm text-gray-500">Loading candidates...</p>
              ) : candidates.length === 0 ? (
                <p className="p-6 text-sm text-gray-500">No detection candidates are available.</p>
              ) : candidates.map((candidate) => (
                <div
                  key={candidate.id}
                  onClick={() => setSelectedCandidateID(candidate.id)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault()
                      setSelectedCandidateID(candidate.id)
                    }
                  }}
                  role="button"
                  tabIndex={0}
                  className={`block w-full cursor-pointer px-6 py-5 text-left transition-colors hover:bg-gray-50 ${selectedCandidate?.id === candidate.id ? 'bg-primary-50' : ''}`}
                >
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="font-semibold text-gray-900">{candidate.title}</h3>
                        {statusBadge(candidate.status)}
                      </div>
                      <p className="mt-1 text-sm text-gray-500">{candidate.description || 'No description provided.'}</p>
                      <div className="mt-3 flex flex-wrap gap-2 text-xs text-gray-500">
                        <span className="font-mono">{candidate.pack_id}@{candidate.pack_version}</span>
                        <span>min agent {candidate.min_agent_version}</span>
                        <span>{candidate.supported_os.join(', ')}</span>
                        <span>updated {formatTime(candidate.updated_at_ms)}</span>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <ActionButton label="Validate" disabled={!['draft', 'validation_failed', 'ready_for_review'].includes(candidate.status)} busy={actingID === `${candidate.id}:validate`} onClick={(event) => { event.stopPropagation(); runAction(candidate, 'validate') }} />
                      <ActionButton label="Approve" disabled={candidate.status !== 'ready_for_review'} busy={actingID === `${candidate.id}:approve`} onClick={(event) => { event.stopPropagation(); runAction(candidate, 'approve') }} />
                      <ActionButton label="Sign" disabled={candidate.status !== 'approved'} busy={actingID === `${candidate.id}:sign`} onClick={(event) => { event.stopPropagation(); runAction(candidate, 'sign') }} />
                      <ActionButton label="Reject" danger disabled={!['draft', 'validating', 'validation_failed', 'ready_for_review', 'approved'].includes(candidate.status)} busy={actingID === `${candidate.id}:reject`} onClick={(event) => { event.stopPropagation(); runAction(candidate, 'reject') }} />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </section>

          <aside className="space-y-8">
            <section className="card p-6">
              <div className="mb-4 flex items-center gap-3">
                <Fingerprint className="h-5 w-5 text-primary-600" />
                <h2 className="text-lg font-semibold text-gray-900">Signer</h2>
              </div>
              <dl className="space-y-3 text-sm">
                <InfoRow label="Algorithm" value={signerInfo?.algorithm || 'n/a'} />
                <InfoRow label="Key ID" value={signerInfo?.key_id || 'n/a'} mono />
                <InfoRow label="Public key" value={shortHash(signerInfo?.public_key_b64)} mono />
              </dl>
            </section>

            <section className="card p-6">
              <h2 className="mb-4 text-lg font-semibold text-gray-900">Selected Candidate</h2>
              {selectedCandidate ? (
                <div className="space-y-4">
                  <div>
                    <p className="font-medium text-gray-900">{selectedCandidate.title}</p>
                    <p className="mt-1 font-mono text-xs text-gray-500">{selectedCandidate.id}</p>
                  </div>
                  <dl className="space-y-3 text-sm">
                    <InfoRow label="Status" value={statusLabels[selectedCandidate.status]} />
                    <InfoRow label="Pack" value={`${selectedCandidate.pack_id}@${selectedCandidate.pack_version}`} mono />
                    <InfoRow label="Signed artifact" value={selectedCandidate.signed_pack_id || 'n/a'} mono />
                    <InfoRow label="Rejected reason" value={selectedCandidate.reject_reason || 'n/a'} />
                  </dl>
                  <div>
                    <h3 className="mb-2 text-sm font-semibold text-gray-900">Validations</h3>
                    {validations.length === 0 ? (
                      <p className="text-sm text-gray-500">No validation runs recorded.</p>
                    ) : (
                      <div className="space-y-3">
                        {validations.map((run) => (
                          <div key={run.id} className="rounded-md border border-gray-200 p-3">
                            <div className="flex items-center justify-between gap-3">
                              {run.success ? <span className="badge badge-success">Passed</span> : <span className="badge badge-danger">Failed</span>}
                              <span className="text-xs text-gray-500">{formatTime(run.ended_at_ms || run.started_at_ms)}</span>
                            </div>
                            <p className="mt-2 text-xs text-gray-600">
                              {run.events_fetched} events, {run.matched_rules} matched rules
                            </p>
                            {(run.details || run.errors) && (
                              <p className="mt-1 break-words text-xs text-gray-500">{run.details || run.errors}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ) : (
                <p className="text-sm text-gray-500">No candidate selected.</p>
              )}
            </section>
          </aside>
        </div>

        <section className="mt-8 card">
          <div className="border-b border-gray-200 px-6 py-4">
            <h2 className="text-lg font-semibold text-gray-900">Signed Packs and Rollout</h2>
            <p className="text-sm text-gray-500">Controller visibility for lab agents reporting pack health.</p>
          </div>
          <div className="divide-y divide-gray-200">
            {signedPacks.length === 0 ? (
              <p className="p-6 text-sm text-gray-500">No signed detection packs are available.</p>
            ) : signedPacks.map((pack) => {
              const rollout = pack.pack_id ? rollouts[pack.pack_id] : undefined
              return (
                <div key={pack.id} className="p-6">
                  <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="font-semibold text-gray-900">{pack.pack_id || pack.id}</h3>
                        <span className="badge badge-info">{pack.pack_version || 'unversioned'}</span>
                      </div>
                      <p className="mt-2 font-mono text-xs text-gray-500">artifact {pack.id}</p>
                      <p className="mt-1 font-mono text-xs text-gray-500">sha256 {shortHash(pack.sha256)}</p>
                    </div>
                    <div className="grid grid-cols-2 gap-3 text-sm sm:grid-cols-5">
                      <MiniStat label="Agents" value={rollout?.agents_reported || 0} />
                      <MiniStat label="Applied" value={rollout?.count_applied || 0} />
                      <MiniStat label="Rejected" value={rollout?.count_rejected || 0} />
                      <MiniStat label="Incompatible" value={rollout?.count_incompatible || 0} />
                      <MiniStat label="Expired" value={rollout?.count_expired || 0} />
                    </div>
                  </div>

                  {rollout?.agents?.length ? (
                    <div className="mt-5 overflow-x-auto">
                      <table className="min-w-full divide-y divide-gray-200 text-sm">
                        <thead>
                          <tr className="text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                            <th className="py-2 pr-4">Agent</th>
                            <th className="py-2 pr-4">State</th>
                            <th className="py-2 pr-4">Active Pack</th>
                            <th className="py-2 pr-4">Trust</th>
                            <th className="py-2 pr-4">Last Check</th>
                            <th className="py-2 pr-4">Reason</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                          {rollout.agents.map((agent) => (
                            <tr key={agent.agent_uid}>
                              <td className="py-3 pr-4">
                                <p className="font-medium text-gray-900">{agent.device_id || agent.agent_uid}</p>
                                <p className="font-mono text-xs text-gray-500">{agent.agent_uid}</p>
                              </td>
                              <td className="py-3 pr-4">{rolloutBadge(agent.rollout_state, agent.computed_stale)}</td>
                              <td className="py-3 pr-4 font-mono text-xs">{agent.active_pack_id || 'none'} {agent.active_pack_version ? `@ ${agent.active_pack_version}` : ''}</td>
                              <td className="py-3 pr-4 text-xs text-gray-600">sig={agent.signature_status || 'n/a'} hash={agent.hash_status || 'n/a'} schema={agent.schema_status || 'n/a'}</td>
                              <td className="py-3 pr-4 text-xs text-gray-600">{formatTime(agent.last_check_at_ms)}</td>
                              <td className="py-3 pr-4 text-xs text-gray-600">{agent.reason_detail || agent.last_rejected_reason || 'n/a'}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <p className="mt-4 text-sm text-gray-500">No agents have reported this pack yet.</p>
                  )}
                </div>
              )
            })}
          </div>
        </section>
      </main>
    </div>
  )
}

function Metric({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="card p-4">
      <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-md bg-primary-50 text-primary-700">{icon}</div>
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-gray-900">{value}</p>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-gray-200 p-3">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-lg font-semibold text-gray-900">{value}</p>
    </div>
  )
}

function InfoRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</dt>
      <dd className={`mt-1 break-words text-gray-900 ${mono ? 'font-mono text-xs' : ''}`}>{value}</dd>
    </div>
  )
}

function ActionButton({
  label,
  disabled,
  busy,
  danger = false,
  onClick
}: {
  label: string
  disabled: boolean
  busy: boolean
  danger?: boolean
  onClick: (event: React.MouseEvent<HTMLButtonElement>) => void
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled || busy}
      className={`btn px-3 py-2 ${danger ? 'btn-danger' : 'btn-secondary'}`}
    >
      {busy ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : null}
      {label}
    </button>
  )
}
