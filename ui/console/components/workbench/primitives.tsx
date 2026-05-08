'use client'

import type { ReactNode } from 'react'
import { useMemo, useState } from 'react'
import { Copy, X } from 'lucide-react'
import { formatJson } from '@/shared/formatting'

export function WorkbenchHeader({
  title,
  subtitle,
  actions,
}: {
  title: string
  subtitle?: string
  actions?: ReactNode
}) {
  return (
    <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 className="text-2xl font-bold text-slate-950">{title}</h1>
        {subtitle ? <p className="mt-1 text-sm text-slate-500">{subtitle}</p> : null}
      </div>
      {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
    </div>
  )
}

export function SummaryStrip({ children }: { children: ReactNode }) {
  return <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">{children}</div>
}

export function KpiTile({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white px-3 py-3 shadow-sm">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-xl font-semibold text-slate-950">{value}</p>
    </div>
  )
}

export function FilterBar({ children }: { children: ReactNode }) {
  return <div className="mb-4 flex flex-wrap gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-sm">{children}</div>
}

export function BoundedTable({
  headers,
  rows,
  maxRows = 120,
}: {
  headers: string[]
  rows: ReactNode[][]
  maxRows?: number
}) {
  const visibleRows = rows.slice(0, maxRows)
  const hiddenCount = Math.max(rows.length - visibleRows.length, 0)

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200">
      <table className="w-full table-fixed text-sm">
        <thead className="bg-slate-50">
          <tr>
            {headers.map((header) => (
              <th key={header} className="truncate px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white">
          {visibleRows.map((row, idx) => (
            <tr key={idx}>
              {row.map((cell, cellIdx) => (
                <td key={cellIdx} className="ux-table-cell px-3 py-2 align-top">{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
        {hiddenCount > 0 ? (
          <tfoot className="bg-slate-50">
            <tr>
              <td colSpan={headers.length} className="px-3 py-2 text-xs text-slate-500">
                Showing first {visibleRows.length} of {rows.length} rows. Use filters to narrow the list.
              </td>
            </tr>
          </tfoot>
        ) : null}
      </table>
    </div>
  )
}

export function FormattedValue({
  value,
  fullValue,
  mono = true,
}: {
  value: string
  fullValue?: string
  mono?: boolean
}) {
  return (
    <span
      className={`inline-block max-w-full truncate ${mono ? 'font-mono text-xs' : 'text-sm'}`}
      title={fullValue || value}
    >
      {value}
    </span>
  )
}

export function CopyValueButton({ value, label = 'Copy value' }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      type="button"
      aria-label={label}
      className="inline-flex h-6 w-6 items-center justify-center rounded border border-slate-200 bg-white text-slate-500 hover:bg-slate-50"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value)
          setCopied(true)
          window.setTimeout(() => setCopied(false), 1200)
        } catch {
          setCopied(false)
        }
      }}
      title={copied ? 'Copied' : label}
    >
      <Copy className="h-3.5 w-3.5" />
    </button>
  )
}

export function EmptyState({ title, message }: { title: string; message: string }) {
  return (
    <div className="rounded-lg border border-dashed border-slate-300 bg-white px-4 py-10 text-center">
      <p className="text-sm font-semibold text-slate-700">{title}</p>
      <p className="mt-1 text-sm text-slate-500">{message}</p>
    </div>
  )
}

export function DetailModal({
  open,
  title,
  detail,
  onClose,
}: {
  open: boolean
  title: string
  detail: unknown
  onClose: () => void
}) {
  const rendered = useMemo(() => formatJson(detail), [detail])
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4">
      <div className="max-h-[80vh] w-full max-w-3xl overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
          <h3 className="text-sm font-semibold text-slate-900">{title}</h3>
          <button
            type="button"
            className="inline-flex h-7 w-7 items-center justify-center rounded border border-slate-200 text-slate-500 hover:bg-slate-50"
            onClick={onClose}
            aria-label="Close detail"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[calc(80vh-52px)] overflow-auto p-4">
          <pre className="overflow-auto whitespace-pre-wrap rounded bg-slate-900 p-3 font-mono text-xs text-slate-100">{rendered}</pre>
        </div>
      </div>
    </div>
  )
}
