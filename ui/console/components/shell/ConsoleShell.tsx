'use client'

import { LogOut, Search, UserCircle } from 'lucide-react'
import { useRouter } from 'next/navigation'
import type { ReactNode } from 'react'
import { shellNavGroups, targetForNavItem } from './shellNav'

export type HealthTone = 'emerald' | 'amber' | 'slate'

export type BreadcrumbSegment = { label: string; href?: string }

type ConsoleShellProps = {
  children: ReactNode
  activeNavId: string
  breadcrumbs: BreadcrumbSegment[]
  health: { label: string; tone: HealthTone; text: string }
  onLogout: () => void
  /** Aggregate AI / platform health label (optional). */
  aiHealthSummary?: string | null
}

export function ConsoleShell({
  children,
  activeNavId,
  breadcrumbs,
  health,
  onLogout,
  aiHealthSummary,
}: ConsoleShellProps) {
  const router = useRouter()

  function goNav(itemId: string) {
    const target = targetForNavItem(itemId)
    if (target.startsWith('home:')) {
      const panel = target.slice('home:'.length)
      if (panel === 'dashboard') router.replace('/', { scroll: false })
      else if (panel === 'agents') router.replace('/?panel=agents', { scroll: false })
      else if (panel === 'inventory') router.replace('/?panel=inventory', { scroll: false })
      return
    }
    router.push(target)
  }

  const healthDot =
    health.tone === 'emerald'
      ? 'bg-emerald-500'
      : health.tone === 'amber'
        ? 'bg-amber-400'
        : 'bg-slate-400'

  return (
    <div className="h-screen flex flex-col bg-gray-50 font-sans">
      {/* ── Header ─────────────────────────────────────────────── */}
      <header className="flex-shrink-0 bg-white border-b border-gray-200 shadow-sm z-10">
        <div className="flex items-center justify-between h-16 px-5">
          {/* Logo / wordmark */}
          <button
            type="button"
            onClick={() => goNav('dashboard')}
            className="flex items-center gap-3 border-0 bg-transparent cursor-pointer shrink-0"
          >
            <div className="w-9 h-9 rounded-lg bg-slate-950 flex items-center justify-center overflow-hidden">
              <img src="/aegisflux-shield.png" alt="AegisFlux" className="w-full h-full object-cover" />
            </div>
            <span className="text-xl font-extrabold tracking-tight text-slate-950 leading-none">
              Aegis<span className="text-blue-600">Flux</span>
            </span>
          </button>

          {/* Right actions */}
          <div className="flex items-center gap-2">
            {/* Search */}
            <button
              type="button"
              className="hidden sm:inline-flex items-center gap-2 h-8 px-3 rounded-lg bg-gray-100 text-slate-500 text-sm hover:bg-gray-200 transition-colors"
            >
              <Search className="h-3.5 w-3.5" />
              <span>Search</span>
              <kbd className="ml-1 rounded border border-gray-300 bg-white px-1.5 py-0.5 text-[10px] font-semibold text-gray-400 leading-none">
                ⌘K
              </kbd>
            </button>

            {/* Health status */}
            <div className="flex items-center gap-2 pl-3 ml-1 border-l border-gray-200">
              <span className={`h-2 w-2 rounded-full shrink-0 ${healthDot}`} />
              <span className="text-sm font-semibold text-slate-700">{health.label}</span>
              {aiHealthSummary ? (
                <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-0.5 text-[11px] font-semibold text-slate-500">
                  AI {aiHealthSummary}
                </span>
              ) : null}
            </div>

            {/* User */}
            <div className="flex items-center gap-2 pl-3 ml-1 border-l border-gray-200">
              <UserCircle className="h-5 w-5 text-slate-400" />
              <span className="text-sm font-semibold text-slate-700">operator</span>
              <span className="rounded border border-blue-200 bg-blue-50 px-1.5 py-0.5 text-[11px] font-bold text-blue-700 leading-none">
                Admin
              </span>
              <button
                type="button"
                onClick={onLogout}
                className="ml-0.5 h-7 w-7 flex items-center justify-center rounded-md text-slate-400 hover:text-slate-600 hover:bg-gray-100 transition-colors"
                title="Sign out"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* ── Body ──────────────────────────────────────────────── */}
      <div className="flex flex-1 min-h-0">
        {/* Sidebar */}
        <aside className="w-64 flex-shrink-0 bg-white border-r border-gray-200 flex flex-col overflow-y-auto">
          <nav className="p-3 flex-1">
            {shellNavGroups.map((group) => (
              <div key={group.label}>
                <div className="px-3 pt-4 pb-1">
                  <span className="text-[11px] font-semibold text-gray-400 uppercase tracking-wider select-none">
                    {group.label}
                  </span>
                </div>
                {group.items.map((item) => {
                  const Icon = item.icon
                  const active = item.id === activeNavId
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => goNav(item.id)}
                      className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors mb-0.5 ${
                        active
                          ? 'bg-[#1e3a5f] text-white'
                          : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                      }`}
                    >
                      <Icon className="h-4 w-4 shrink-0" />
                      <span className="truncate">{item.label}</span>
                    </button>
                  )
                })}
              </div>
            ))}
          </nav>
        </aside>

        {/* Main content */}
        <div className="flex-1 flex flex-col min-w-0 min-h-0 overflow-hidden">
          {/* Breadcrumb */}
          <nav className="flex-shrink-0 flex items-center gap-1.5 px-6 py-2 text-sm text-gray-500 border-b border-gray-100 bg-white">
            <span>AegisFlux</span>
            {breadcrumbs.map((segment, index) => (
              <span key={`${segment.label}-${index}`} className="flex items-center gap-1.5">
                <span className="text-gray-300">/</span>
                {segment.href ? (
                  <a
                    href={segment.href}
                    className="font-medium text-blue-600 hover:text-blue-800 transition-colors"
                  >
                    {segment.label}
                  </a>
                ) : (
                  <span className="font-medium text-gray-900">{segment.label}</span>
                )}
              </span>
            ))}
          </nav>

          {/* Scrollable content area */}
          <main className="flex-1 overflow-y-auto min-h-0">
            {children}
          </main>
        </div>
      </div>
    </div>
  )
}
