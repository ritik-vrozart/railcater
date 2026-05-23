import type { PortionCode, PortionFormRow } from '../types/api'

export const PORTION_OPTIONS: { code: PortionCode; label: string; sortOrder: number }[] = [
  { code: 'quarter', label: 'Quarter', sortOrder: 1 },
  { code: 'half', label: 'Half', sortOrder: 2 },
  { code: 'full', label: 'Full', sortOrder: 3 },
  { code: 'single', label: 'Single / Piece', sortOrder: 4 },
]

export function defaultPortionRows(): PortionFormRow[] {
  return PORTION_OPTIONS.map((p) => ({
    portion: p.code,
    label: p.label,
    priceRupees: '',
    stockQuantity: '',
    enabled: p.code === 'full',
  }))
}

export function portionsFromItem(portions: { portion: PortionCode; label: string; price_cents: number; stock_quantity: number }[]): PortionFormRow[] {
  const map = new Map(portions.map((p) => [p.portion, p]))
  return PORTION_OPTIONS.map((opt) => {
    const existing = map.get(opt.code)
    return {
      portion: opt.code,
      label: existing?.label ?? opt.label,
      priceRupees: existing ? String(existing.price_cents / 100) : '',
      stockQuantity: existing ? String(existing.stock_quantity) : '',
      enabled: !!existing,
    }
  })
}
