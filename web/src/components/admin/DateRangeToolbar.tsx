import { useState } from 'react'
import {
  type DatePreset,
  type DateRange,
  formatRangeLabel,
  rangeForPreset,
} from '../../lib/dateRange'

const PRESETS: { id: DatePreset; label: string }[] = [
  { id: 'today', label: 'Today' },
  { id: '7d', label: 'Last 7 days' },
  { id: '30d', label: 'Last 30 days' },
  { id: 'month', label: 'This month' },
]

interface DateRangeToolbarProps {
  value: DateRange
  onChange: (range: DateRange, preset: DatePreset) => void
}

export function DateRangeToolbar({ value, onChange }: DateRangeToolbarProps) {
  const [preset, setPreset] = useState<DatePreset>('30d')
  const [customFrom, setCustomFrom] = useState(value.from)
  const [customTo, setCustomTo] = useState(value.to)

  function applyPreset(id: DatePreset) {
    setPreset(id)
    const range = rangeForPreset(id)
    setCustomFrom(range.from)
    setCustomTo(range.to)
    onChange(range, id)
  }

  function applyCustom() {
    if (!customFrom || !customTo || customTo < customFrom) return
    setPreset('custom')
    onChange({ from: customFrom, to: customTo }, 'custom')
  }

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-gray-900">Date range</p>
          <p className="text-xs text-gray-500">{formatRangeLabel(value)}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {PRESETS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => applyPreset(p.id)}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                preset === p.id
                  ? 'bg-amber-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-end gap-3 border-t border-gray-100 pt-4">
        <label className="text-sm">
          <span className="mb-1 block text-xs font-medium text-gray-500">From</span>
          <input
            type="date"
            value={customFrom}
            onChange={(e) => setCustomFrom(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>
        <label className="text-sm">
          <span className="mb-1 block text-xs font-medium text-gray-500">To</span>
          <input
            type="date"
            value={customTo}
            onChange={(e) => setCustomTo(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>
        <button
          type="button"
          onClick={applyCustom}
          className="rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-800 hover:bg-gray-50"
        >
          Apply
        </button>
      </div>
    </div>
  )
}
