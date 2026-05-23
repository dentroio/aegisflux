'use client'

import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, CheckCircle2, FileText, Shield, Sparkles } from 'lucide-react'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import type { HealthTone } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'

type Privacy = Record<string, boolean>
type SettingConfig = {
  key: keyof Privacy
  label: string
  description: string
}

const outboundControls: SettingConfig[] = [
  {
    key: 'allow_external_ai',
    label: 'Allow external AI requests',
    description: 'Permit provider calls outside the local lab path after redaction is applied.',
  },
]

const identityRedactionControls: SettingConfig[] = [
  {
    key: 'redact_ips',
    label: 'Redact IP addresses and CIDRs',
    description: 'Mask network addresses before outbound AI context is assembled.',
  },
  {
    key: 'redact_macs',
    label: 'Redact MAC addresses',
    description: 'Remove hardware identifiers from prompts and request metadata.',
  },
  {
    key: 'redact_usernames',
    label: 'Redact usernames',
    description: 'Replace local and domain usernames with neutral placeholders.',
  },
  {
    key: 'redact_emails',
    label: 'Redact email addresses',
    description: 'Hide mailbox identifiers found in evidence, paths, and command text.',
  },
  {
    key: 'redact_hostnames',
    label: 'Redact hostnames',
    description: 'Mask endpoint and workload names while retaining investigation shape.',
  },
]

const evidenceRedactionControls: SettingConfig[] = [
  {
    key: 'redact_command_lines',
    label: 'Redact command lines',
    description: 'Strip raw process arguments that may include identifiers or secrets.',
  },
  {
    key: 'redact_file_paths',
    label: 'Redact file paths',
    description: 'Remove user, project, and filesystem details from evidence snippets.',
  },
  {
    key: 'redact_raw_secrets',
    label: 'Block raw secrets and tokens',
    description: 'Prevent obvious credentials from leaving AegisFlux in provider requests.',
  },
]

function ToggleRow({
  setting,
  privacy,
  onSave,
}: {
  setting: SettingConfig
  privacy: Privacy
  onSave: (next: Privacy) => void
}) {
  const checked = Boolean(privacy[setting.key])

  return (
    <label className="flex min-h-[86px] cursor-pointer items-center justify-between gap-4 rounded-lg border border-gray-200 p-3 transition-colors hover:bg-gray-50">
      <span className="min-w-0">
        <span className="block text-sm font-medium text-gray-900">{setting.label}</span>
        <span className="mt-1 block text-xs leading-5 text-gray-500">{setting.description}</span>
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onSave({ ...privacy, [setting.key]: e.target.checked })}
        className="h-4 w-4 flex-none accent-blue-600"
      />
    </label>
  )
}

function SettingsCard({
  icon,
  title,
  description,
  children,
  badge,
}: {
  icon: ReactNode
  title: string
  description: string
  children: ReactNode
  badge?: ReactNode
}) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-6 py-4">
        <div className="flex gap-3">
          <div className="mt-0.5 flex h-9 w-9 flex-none items-center justify-center rounded-lg bg-blue-50 text-blue-600">
            {icon}
          </div>
          <div>
            <h2 className="text-base font-semibold text-gray-900">{title}</h2>
            <p className="mt-0.5 text-sm text-gray-500">{description}</p>
          </div>
        </div>
        {badge}
      </div>
      <div className="p-6">{children}</div>
    </section>
  )
}

export default function SettingsPage() {
  const router = useRouter()
  const [gate, setGate] = useState(false)
  const [privacy, setPrivacy] = useState<Privacy | null>(null)

  useEffect(() => {
    if (!readLabAuthenticated()) {
      router.replace('/')
      return
    }
    setGate(true)
  }, [router])

  useEffect(() => {
    if (!gate) return undefined
    fetch('/api/actions/platform/ai/privacy', { cache: 'no-store' })
      .then((r) => (r.ok ? r.json() : null))
      .then(setPrivacy)
    return undefined
  }, [gate])

  async function save(next: Privacy) {
    await fetch('/api/actions/platform/ai/privacy', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    })
    setPrivacy(next)
  }

  function onLogout() {
    window.localStorage.removeItem('aegisflux.labAuth')
    router.replace('/')
  }

  const health = { label: 'Settings', tone: 'slate' as HealthTone, text: 'Lab privacy' }

  if (!gate || !privacy) return gate ? <div className="p-6 text-sm">Loading…</div> : null

  const allRedactionsOn = [...identityRedactionControls, ...evidenceRedactionControls].every((setting) =>
    Boolean(privacy[setting.key])
  )

  return (
    <ConsoleShell activeNavId="settings" breadcrumbs={[{ label: 'Settings' }]} health={health} onLogout={onLogout}>
      <main className="mx-auto max-w-5xl space-y-6 px-4 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-slate-900">Privacy & audit</h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-600">
              Tune redaction defaults for outbound AI context. Matching records are audited on the Actions API (`/platform/ai/*`).
            </p>
          </div>
          <Link
            href="/configure/connectors/ai"
            className="inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-semibold text-blue-700 shadow-sm transition-colors hover:bg-blue-50"
          >
            <ArrowLeft className="h-4 w-4" />
            AI Providers
          </Link>
        </div>

        <SettingsCard
          icon={<Sparkles className="h-4 w-4" />}
          title="External AI Requests"
          description="Control whether AegisFlux can send governed prompts to configured external providers."
          badge={
            <span
              className={`rounded-full px-2.5 py-1 text-xs font-medium ${
                privacy.allow_external_ai ? 'bg-blue-100 text-blue-800' : 'bg-amber-100 text-amber-800'
              }`}
            >
              {privacy.allow_external_ai ? 'External enabled' : 'Local only'}
            </span>
          }
        >
          <div className="grid gap-3 md:grid-cols-2">
            {outboundControls.map((setting) => (
              <ToggleRow key={setting.key as string} setting={setting} privacy={privacy} onSave={save} />
            ))}
          </div>
        </SettingsCard>

        <SettingsCard
          icon={<Shield className="h-4 w-4" />}
          title="Identity Redaction"
          description="Mask endpoint, user, and network identifiers before provider requests are assembled."
          badge={
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium ${
                allRedactionsOn ? 'bg-green-100 text-green-800' : 'bg-slate-100 text-slate-700'
              }`}
            >
              {allRedactionsOn ? <CheckCircle2 className="h-3.5 w-3.5" /> : null}
              {allRedactionsOn ? 'All redactions on' : 'Review redactions'}
            </span>
          }
        >
          <div className="grid gap-3 md:grid-cols-2">
            {identityRedactionControls.map((setting) => (
              <ToggleRow key={setting.key as string} setting={setting} privacy={privacy} onSave={save} />
            ))}
          </div>
        </SettingsCard>

        <SettingsCard
          icon={<FileText className="h-4 w-4" />}
          title="Evidence Redaction"
          description="Reduce sensitive command, path, and secret exposure in investigation payloads."
        >
          <div className="grid gap-3 md:grid-cols-2">
            {evidenceRedactionControls.map((setting) => (
              <ToggleRow key={setting.key as string} setting={setting} privacy={privacy} onSave={save} />
            ))}
          </div>
          <p className="mt-4 text-xs text-gray-400">
            Updates are saved immediately and apply to future AI provider requests.
          </p>
        </SettingsCard>
      </main>
    </ConsoleShell>
  )
}
