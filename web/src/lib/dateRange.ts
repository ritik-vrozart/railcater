export type DatePreset = 'today' | '7d' | '30d' | 'month' | 'custom'

export interface DateRange {
  from: string // YYYY-MM-DD
  to: string
}

function pad(n: number): string {
  return String(n).padStart(2, '0')
}

export function toISODate(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function todayISO(): string {
  return toISODate(new Date())
}

export function rangeForPreset(preset: DatePreset): DateRange {
  const now = new Date()
  const to = toISODate(now)
  const start = new Date(now)

  switch (preset) {
    case 'today':
      return { from: to, to }
    case '7d':
      start.setDate(start.getDate() - 6)
      return { from: toISODate(start), to }
    case '30d':
      start.setDate(start.getDate() - 29)
      return { from: toISODate(start), to }
    case 'month':
      start.setDate(1)
      return { from: toISODate(start), to }
    default:
      start.setDate(start.getDate() - 29)
      return { from: toISODate(start), to }
  }
}

export function buildDateQuery(range: DateRange, extra?: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams()
  params.set('from', range.from)
  params.set('to', range.to)
  if (extra) {
    for (const [k, v] of Object.entries(extra)) {
      if (v !== undefined && v !== '') params.set(k, String(v))
    }
  }
  return params.toString()
}

export function formatRangeLabel(range: DateRange): string {
  if (range.from === range.to) return range.from
  return `${range.from} → ${range.to}`
}
