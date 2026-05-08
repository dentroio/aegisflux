export function truncateMiddle(value: string, lead = 8, tail = 6): string {
  if (value.length <= lead + tail + 3) return value
  return `${value.slice(0, lead)}...${value.slice(-tail)}`
}

export function formatAgentId(value: string): string {
  return truncateMiddle(value, 10, 8)
}

export function formatHostname(value: string): string {
  return value.length > 40 ? `${value.slice(0, 37)}...` : value
}

export function formatIpOrCidr(value: string): string {
  return value.trim()
}

export function formatMac(value: string): string {
  const cleaned = value.toLowerCase().replace(/-/g, ':')
  return cleaned
}

export function formatHash(value: string): string {
  return truncateMiddle(value, 12, 10)
}

export function formatPath(value: string, compact = true): string {
  if (!compact) return value
  return value.length > 56 ? `${value.slice(0, 53)}...` : value
}

export function formatCommandLine(value: string, compact = true): string {
  if (!compact) return value
  return value.length > 72 ? `${value.slice(0, 69)}...` : value
}

export function formatJson(value: unknown, spacing = 2): string {
  try {
    return JSON.stringify(value, null, spacing)
  } catch {
    return String(value)
  }
}

export function formatDateTime(value: number | string | Date): string {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return 'Invalid date'
  return date.toLocaleString()
}

export function formatRelativeAge(value: number | string | Date): string {
  const ts = value instanceof Date ? value.getTime() : new Date(value).getTime()
  if (Number.isNaN(ts)) return 'n/a'
  const diffSeconds = Math.max(0, Math.round((Date.now() - ts) / 1000))
  if (diffSeconds < 60) return `${diffSeconds}s ago`
  const mins = Math.round(diffSeconds / 60)
  if (mins < 60) return `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.round(hours / 24)}d ago`
}
