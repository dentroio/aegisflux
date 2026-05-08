'use client'

import { Bell, BookText, Bot, LogOut, Moon, Search, UserCircle } from 'lucide-react'
import { useRouter } from 'next/navigation'
import type { ReactNode } from 'react'
import { shellNavGroups, targetForNavItem } from './shellNav'
import { shellStyles as sh } from './consoleShellStyles'

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

  return (
    <div className="min-h-screen bg-gray-50 text-slate-900" style={sh.page}>
      <div style={sh.appShell}>
        <header className="border-b border-gray-200 bg-white shadow-sm" style={sh.header}>
          <div className="flex h-16 items-center justify-between px-5" style={sh.headerInner}>
            <button type="button" onClick={() => goNav('dashboard')} style={{ ...sh.headerLogo, border: 0, background: 'none', cursor: 'pointer' }}>
              <div style={sh.headerLogoMark}>
                <img src="/aegisflux-shield.png" alt="AegisFlux" style={sh.logoImage} />
              </div>
              <div>
                <div style={sh.wordmarkText}>
                  Aegis<span style={sh.wordmarkFlux}>Flux</span>
                </div>
              </div>
            </button>

            <div className="flex items-center gap-3" style={sh.headerActions}>
              <button type="button" style={sh.headerSearch} title="Search">
                <Search className="h-4 w-4" />
                <span>Search</span>
                <span style={sh.keycap}>Cmd K</span>
              </button>
              <button type="button" style={sh.iconButton} title="Notifications">
                <Bell className="h-5 w-5" />
              </button>
              <button type="button" style={sh.iconButton} title="Theme">
                <Moon className="h-5 w-5" />
              </button>
              <button type="button" style={sh.iconButton} title="Documentation">
                <BookText className="h-5 w-5" />
              </button>
              <button type="button" style={sh.iconButton} title="AI Assistant">
                <Bot className="h-5 w-5" />
              </button>
              <div style={sh.headerStatus}>
                <FreshDot active={health.tone === 'emerald'} />
                <span style={{ fontSize: 14, fontWeight: 650, color: health.tone === 'emerald' ? '#334155' : '#92400e' }}>
                  {health.label}
                </span>
                {aiHealthSummary ? (
                  <span style={sh.aiHealthChip} title="Configured AI providers">
                    AI {aiHealthSummary}
                  </span>
                ) : null}
              </div>
              <div style={sh.userBadge}>
                <UserCircle className="h-5 w-5 text-slate-500" />
                <span style={{ fontSize: 14, fontWeight: 650, color: '#334155' }}>operator</span>
                <span style={sh.rolePill}>Admin</span>
                <button type="button" onClick={onLogout} style={{ ...sh.iconButton, width: 28, height: 28 }} title="Sign out">
                  <LogOut className="h-4 w-4 text-slate-400" />
                </button>
              </div>
            </div>
          </div>
        </header>

        <div style={sh.bodyShell}>
          <aside style={sh.sidebar}>
            <nav style={sh.sideNav}>
              {shellNavGroups.map((group) => (
                <div key={group.label}>
                  <div style={sh.sideGroupLabel}>{group.label}</div>
                  {group.items.map((item) => {
                    const Icon = item.icon
                    const navActive = item.id === activeNavId
                    return (
                      <button
                        key={item.id}
                        type="button"
                        onClick={() => goNav(item.id)}
                        style={{
                          ...sh.sideButton,
                          ...(navActive ? sh.sideButtonActive : sh.sideButtonMuted),
                        }}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        <span className="truncate">{item.label}</span>
                      </button>
                    )
                  })}
                </div>
              ))}
            </nav>
            <div style={sh.sideFooter}>
              <div style={sh.sideFooterCard}>
                <div style={{ fontSize: 13, fontWeight: 750, color: '#075985' }}>Observe-only</div>
                <div style={{ marginTop: 4, fontSize: 12, lineHeight: 1.45, color: '#0369a1' }}>
                  Controls remain staged until approval and rollback are ready.
                </div>
              </div>
            </div>
          </aside>

          <div style={sh.contentArea}>
            <div style={sh.breadcrumb}>
              <span>AegisFlux</span>
              {breadcrumbs.map((segment, index) => (
                <span className="flex min-w-0 items-center gap-1" key={`${segment.label}-${index}`}>
                  <span>/</span>
                  {segment.href ? (
                    <a href={segment.href} className="min-w-0 truncate font-semibold hover:text-blue-900" style={{ color: '#2563eb' }}>
                      {segment.label}
                    </a>
                  ) : (
                    <span className="min-w-0 truncate font-semibold text-slate-900" style={{ color: '#0f172a' }}>
                      {segment.label}
                    </span>
                  )}
                </span>
              ))}
            </div>
            <div style={sh.scrollArea}>{children}</div>
          </div>
        </div>
      </div>
    </div>
  )
}

function FreshDot({ active }: { active: boolean }) {
  return <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ background: active ? '#10b981' : '#f59e0b' }} />
}
