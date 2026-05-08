import type { LucideIcon } from 'lucide-react'
import {
  Activity,
  BookText,
  Bot,
  Cpu,
  Database,
  LayoutDashboard,
  ListTree,
  Network,
  Plug,
  Server,
  Settings,
  ShieldCheck,
  Sparkles,
  TerminalSquare,
} from 'lucide-react'

export type ShellNavGroup = {
  label: string
  items: readonly { id: string; label: string; icon: LucideIcon }[]
}

export const shellNavGroups: readonly ShellNavGroup[] = [
  {
    label: 'Overview',
    items: [{ id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard }],
  },
  {
    label: 'Discover',
    items: [
      { id: 'agents', label: 'Agents', icon: Server },
      { id: 'inventory', label: 'Inventory', icon: Database },
      { id: 'activity', label: 'AI Activity', icon: Bot },
    ],
  },
  {
    label: 'Analyze',
    items: [
      { id: 'evidence', label: 'Evidence Graph', icon: Network },
      { id: 'detections', label: 'Detection Packs', icon: Sparkles },
      { id: 'activity-log', label: 'Findings', icon: Activity },
    ],
  },
  {
    label: 'Control',
    items: [
      { id: 'controls', label: 'Controls', icon: ShieldCheck },
      { id: 'simulation', label: 'Policy Simulation', icon: TerminalSquare },
    ],
  },
  {
    label: 'Operate',
    items: [
      { id: 'event-feed', label: 'Event Feed', icon: ListTree },
      { id: 'monitoring', label: 'Monitoring', icon: Cpu },
      { id: 'reports', label: 'Reports', icon: BookText },
    ],
  },
  {
    label: 'Configure',
    items: [
      { id: 'connectors', label: 'Connectors', icon: Plug },
      { id: 'settings', label: 'Settings', icon: Settings },
    ],
  },
] as const

/** Target path or special `home:` panel key for dashboard shell query params. */
export function targetForNavItem(itemId: string): string {
  switch (itemId) {
    case 'dashboard':
      return 'home:dashboard'
    case 'agents':
      return 'home:agents'
    case 'inventory':
      return 'home:inventory'
    case 'activity':
      return '/discover/activity'
    case 'evidence':
      return '/analyze/evidence'
    case 'detections':
      return '/detections'
    case 'activity-log':
      return '/analyze/findings'
    case 'controls':
      return '/control/controls'
    case 'simulation':
      return '/control/simulation'
    case 'event-feed':
      return '/operate/events'
    case 'monitoring':
      return '/operate/monitoring'
    case 'reports':
      return '/operate/reports'
    case 'connectors':
      return '/configure/connectors'
    case 'settings':
      return '/configure/settings'
    default:
      return '/'
  }
}
