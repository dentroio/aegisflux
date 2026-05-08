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
import {
  BoundedTable,
  DetailModal,
  EmptyState,
  FilterBar,
  KpiTile,
  SummaryStrip,
  WorkbenchHeader,
} from '@/components/workbench/primitives'
import { formatHash, formatRelativeAge } from '@/shared/formatting'

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
  const [view, setView] = useState<'queue' | 'signed' | 'rollout'>('queue')
  const [query, setQuery] = useState('')
  const [detailModal, setDetailModal] = useState<{ title: string; payload: unknown } | null>(null)
  const [rawMode, setRawMode] = useState(false)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [actingID, setActingID] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

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

  const queueRows = useMemo(() => {
    const needle = query.trim().toLowerCase()
    const source = candidates.filter((candidate) =>
      !needle || JSON.stringify(candidate).toLowerCase().includes(needle)
    )
    return source
  }, [candidates, query])

  const signedRows = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return signedPacks.filter((pack) => !needle || JSON.stringify(pack).toLowerCase().includes(needle))
  }, [signedPacks, query])

  const rolloutRows = useMemo(() => {
    const flattened = Object.entries(rollouts).flatMap(([packID, rollout]) =>
      rollout.agents.map((agent) => ({ packID, rollout, agent }))
    )
    const needle = query.trim().toLowerCase()
    return flattened.filter((row) => !needle || JSON.stringify(row).toLowerCase().includes(needle))
  }, [rollouts, query])

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="py-6">
            <WorkbenchHeader
              title="Detection Packs"
              subtitle="Primary task: work the candidate queue, then verify signed and rollout state."
              actions={
                <>
                  <a href="/" className="btn btn-secondary h-9 px-3">
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Dashboard
                  </a>
                  <button onClick={refresh} disabled={refreshing} className="btn btn-secondary h-9 px-3">
                    <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                  </button>
                </>
              }
            />
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

        <SummaryStrip>
          <KpiTile label="Candidates" value={candidates.length} />
          <KpiTile label="Ready for review" value={counts.ready_for_review || 0} />
          <KpiTile label="Signed packs" value={signedPacks.length} />
          <KpiTile label="Agents reporting" value={rolloutTotals.agents} />
          <KpiTile label="Applied" value={rolloutTotals.applied} />
          <KpiTile label="Needs attention" value={rolloutTotals.rejected + rolloutTotals.incompatible + rolloutTotals.expired + rolloutTotals.stale} />
        </SummaryStrip>

        <FilterBar>
          <button type="button" onClick={() => setView('queue')} className={`rounded-full border px-3 py-1 text-xs font-semibold ${view === 'queue' ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-gray-50 text-gray-700'}`}>Candidates</button>
          <button type="button" onClick={() => setView('signed')} className={`rounded-full border px-3 py-1 text-xs font-semibold ${view === 'signed' ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-gray-50 text-gray-700'}`}>Signed packs</button>
          <button type="button" onClick={() => setView('rollout')} className={`rounded-full border px-3 py-1 text-xs font-semibold ${view === 'rollout' ? 'border-primary-600 bg-primary-600 text-white' : 'border-gray-200 bg-gray-50 text-gray-700'}`}>Rollout</button>
          <input value={query} onChange={(e) => setQuery(e.target.value)} className="input h-9 min-w-[240px] max-w-md" placeholder="Filter id, pack, status, device..." />
          <div className="ml-auto text-xs text-slate-500">Signer: {shortHash(signerInfo?.key_id)}</div>
        </FilterBar>

        <section className="card p-5">
          {loading ? (
            <EmptyState title="Loading detections" message="Collecting candidate, signed-pack, and rollout data." />
          ) : view === 'queue' ? (
            queueRows.length === 0 ? (
              <EmptyState title="No candidates" message="No detection candidates match this filter." />
            ) : (
              <BoundedTable
                headers={['Title', 'Status', 'Pack', 'Updated', 'Actions', 'Detail']}
                rows={queueRows.map((candidate) => ([
                  <div key={`${candidate.id}-title`}>
                    <p className="text-sm font-semibold text-gray-900">{candidate.title}</p>
                    <p className="text-xs text-gray-500">{candidate.description || 'No description'}</p>
                  </div>,
                  <div key={`${candidate.id}-status`}>{statusBadge(candidate.status)}</div>,
                  <span key={`${candidate.id}-pack`} className="font-mono text-xs">{formatHash(candidate.pack_id)}@{candidate.pack_version}</span>,
                  <span key={`${candidate.id}-updated`} className="text-xs text-gray-600" title={formatTime(candidate.updated_at_ms)}>{formatRelativeAge(candidate.updated_at_ms)}</span>,
                  <div key={`${candidate.id}-actions`} className="flex flex-wrap gap-1">
                    <ActionButton label="Validate" disabled={!['draft', 'validation_failed', 'ready_for_review'].includes(candidate.status)} busy={actingID === `${candidate.id}:validate`} onClick={() => runAction(candidate, 'validate')} />
                    <ActionButton label="Approve" disabled={candidate.status !== 'ready_for_review'} busy={actingID === `${candidate.id}:approve`} onClick={() => runAction(candidate, 'approve')} />
                    <ActionButton label="Sign" disabled={candidate.status !== 'approved'} busy={actingID === `${candidate.id}:sign`} onClick={() => runAction(candidate, 'sign')} />
                    <ActionButton label="Reject" danger disabled={!['draft', 'validating', 'validation_failed', 'ready_for_review', 'approved'].includes(candidate.status)} busy={actingID === `${candidate.id}:reject`} onClick={() => runAction(candidate, 'reject')} />
                  </div>,
                  <button key={`${candidate.id}-detail`} className="text-xs font-semibold text-primary-700" onClick={async () => {
                    let runs: ValidationRun[] = []
                    try {
                      const data = await fetchJSON<{ items: ValidationRun[] }>(`/api/detection/candidates/${encodeURIComponent(candidate.id)}/validations`)
                      runs = data.items || []
                    } catch {
                      runs = []
                    }
                    setRawMode(false)
                    setDetailModal({ title: `Candidate: ${candidate.title}`, payload: { candidate, validations: runs } })
                  }}>View detail</button>,
                ]))}
              />
            )
          ) : view === 'signed' ? (
            signedRows.length === 0 ? (
              <EmptyState title="No signed packs" message="No signed detection packs are available." />
            ) : (
              <BoundedTable
                headers={['Pack', 'Artifact', 'Hash', 'Created', 'Detail']}
                rows={signedRows.map((pack) => ([
                  <div key={`${pack.id}-pack`}>
                    <p className="text-sm font-semibold text-gray-900">{pack.pack_id || pack.id}</p>
                    <p className="text-xs text-gray-500">{pack.pack_version || 'unversioned'}</p>
                  </div>,
                  <span key={`${pack.id}-artifact`} className="font-mono text-xs">{formatHash(pack.id)}</span>,
                  <span key={`${pack.id}-hash`} className="font-mono text-xs">{shortHash(pack.sha256)}</span>,
                  <span key={`${pack.id}-created`} className="text-xs text-gray-600" title={formatTime(pack.created_at_ms)}>{formatRelativeAge(pack.created_at_ms)}</span>,
                  <button key={`${pack.id}-detail`} className="text-xs font-semibold text-primary-700" onClick={() => {
                    setRawMode(false)
                    setDetailModal({ title: `Signed pack: ${pack.pack_id || pack.id}`, payload: pack })
                  }}>View detail</button>,
                ]))}
              />
            )
          ) : rolloutRows.length === 0 ? (
            <EmptyState title="No rollout reports" message="No agents have reported signed pack rollout yet." />
          ) : (
            <BoundedTable
              headers={['Pack', 'Agent', 'State', 'Trust', 'Last check', 'Detail']}
              rows={rolloutRows.map(({ packID, agent }) => ([
                <span key={`${packID}-${agent.agent_uid}-pack`} className="font-mono text-xs">{formatHash(packID)}</span>,
                <div key={`${packID}-${agent.agent_uid}-agent`}>
                  <p className="text-sm text-gray-900">{agent.device_id || agent.agent_uid}</p>
                  <p className="font-mono text-xs text-gray-500">{formatHash(agent.agent_uid)}</p>
                </div>,
                <div key={`${packID}-${agent.agent_uid}-state`}>{rolloutBadge(agent.rollout_state, agent.computed_stale)}</div>,
                <span key={`${packID}-${agent.agent_uid}-trust`} className="text-xs text-gray-600">sig={agent.signature_status || 'n/a'} hash={agent.hash_status || 'n/a'} schema={agent.schema_status || 'n/a'}</span>,
                <span key={`${packID}-${agent.agent_uid}-check`} className="text-xs text-gray-600" title={formatTime(agent.last_check_at_ms)}>{agent.last_check_at_ms ? formatRelativeAge(agent.last_check_at_ms) : 'n/a'}</span>,
                <button key={`${packID}-${agent.agent_uid}-detail`} className="text-xs font-semibold text-primary-700" onClick={() => {
                  setRawMode(false)
                  setDetailModal({ title: `Rollout: ${agent.device_id || agent.agent_uid}`, payload: { pack_id: packID, agent } })
                }}>View detail</button>,
              ]))}
            />
          )}
        </section>
      </main>
      <div className="fixed bottom-4 right-4 z-10">
        {detailModal ? (
          <button
            type="button"
            className="btn btn-secondary h-8 px-2 text-xs"
            onClick={() => setRawMode((v) => !v)}
          >
            {rawMode ? 'Show summary' : 'Show raw'}
          </button>
        ) : null}
      </div>
      <DetailModal
        open={Boolean(detailModal)}
        title={detailModal?.title || 'Detail'}
        detail={rawMode ? detailModal?.payload || {} : summarizeDetail(detailModal?.payload)}
        onClose={() => {
          setDetailModal(null)
          setRawMode(false)
        }}
      />
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
  onClick: () => void
}) {
  return (
    <button
      onClick={(event) => {
        event.stopPropagation()
        onClick()
      }}
      disabled={disabled || busy}
      className={`btn h-7 px-2 text-xs ${danger ? 'btn-danger' : 'btn-secondary'}`}
    >
      {busy ? <RefreshCw className="mr-2 h-4 w-4 animate-spin" /> : null}
      {label}
    </button>
  )
}

function summarizeDetail(payload: unknown) {
  if (!payload || typeof payload !== 'object') return payload
  const data = payload as Record<string, any>
  if (data.candidate) {
    return {
      candidate: {
        id: data.candidate.id,
        title: data.candidate.title,
        status: data.candidate.status,
        pack: `${data.candidate.pack_id}@${data.candidate.pack_version}`,
        updated: formatTime(data.candidate.updated_at_ms),
      },
      validation_runs: Array.isArray(data.validations) ? data.validations.length : 0,
    }
  }
  if (data.agent) {
    return {
      pack_id: data.pack_id,
      agent_uid: data.agent.agent_uid,
      device_id: data.agent.device_id,
      state: data.agent.rollout_state,
      trust: {
        signature: data.agent.signature_status,
        hash: data.agent.hash_status,
        schema: data.agent.schema_status,
      },
      reason: data.agent.reason_detail || data.agent.last_rejected_reason || 'n/a',
    }
  }
  return payload
}
