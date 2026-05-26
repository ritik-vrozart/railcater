import { useEffect, useMemo, useState } from 'react'
import { Alert } from '../../components/ui/Alert'
import { ButtonLight } from '../../components/ui/ButtonLight'
import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { useAuth } from '../../context/AuthContext'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import { apiRequest, ApiError } from '../../lib/api'
import type { DailyMenu, MenuItem } from '../../types/api'

function todayISO(): string {
  return new Date().toISOString().slice(0, 10)
}

export function VendorDailyMenuPage() {
  const { token } = useAuth()
  const { vendorId } = useVendor()
  const [date, setDate] = useState(todayISO())
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const menuPath = vendorId ? `/api/v1/vendors/${vendorId}/menu?active_only=false` : null
  const dailyPath = vendorId ? `/api/v1/vendors/${vendorId}/daily-menu?date=${date}` : null

  const { data: menuRes, loading: menuLoading } = useApi<{ data: MenuItem[] }>(menuPath, [vendorId])
  const { data: dailyMenu, loading: dailyLoading, reload } = useApi<DailyMenu>(dailyPath, [
    vendorId,
    date,
  ])

  const masterItems = menuRes?.data ?? []

  const dailyItems = dailyMenu?.items ?? []

  useEffect(() => {
    if (!dailyMenu) return
    const ids = new Set(
      dailyItems.filter((i) => i.is_available).map((i) => i.menu_item_id),
    )
    if (ids.size > 0) {
      setSelected(ids)
      return
    }
    if (masterItems.length > 0 && selected.size === 0) {
      setSelected(new Set(masterItems.filter((m) => m.is_active).map((m) => m.id)))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dailyMenu?.id, dailyItems, masterItems.length])

  const availableCount = useMemo(() => selected.size, [selected])

  async function handleSave() {
    if (!token || !vendorId) return
    setError('')
    setSuccess('')
    setSaving(true)
    try {
      const items = masterItems.map((m) => ({
        menu_item_id: m.id,
        is_available: selected.has(m.id),
      }))
      await apiRequest(
        `/api/v1/vendors/${vendorId}/daily-menu?date=${date}`,
        { method: 'PUT', body: JSON.stringify({ items }) },
        token,
      )
      setSuccess("Today's menu saved. Passengers will see these items.")
      reload()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function selectAll() {
    setSelected(new Set(masterItems.filter((m) => m.is_active).map((m) => m.id)))
  }

  function clearAll() {
    setSelected(new Set())
  }

  const loading = menuLoading || dailyLoading

  return (
    <div>
      <PageHeader
        title="Today's menu"
        description="Choose which items are available for passengers today (water, meals, snacks, etc.)"
        actions={
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        }
      />

      {error ? <Alert message={error} /> : null}
      {success ? (
        <div className="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          {success}
        </div>
      ) : null}

      <Card>
        <CardHeader
          title={`Menu for ${date}`}
          description={`${availableCount} of ${masterItems.length} items available today`}
        />
        <div className="mb-4 flex gap-2">
          <ButtonLight variant="secondary" size="sm" onClick={selectAll}>
            Select all active
          </ButtonLight>
          <ButtonLight variant="secondary" size="sm" onClick={clearAll}>
            Clear all
          </ButtonLight>
          <ButtonLight size="sm" onClick={handleSave} disabled={saving || !vendorId}>
            {saving ? 'Saving…' : "Save today's menu"}
          </ButtonLight>
        </div>

        {loading ? (
          <p className="text-sm text-gray-500">Loading…</p>
        ) : masterItems.length === 0 ? (
          <p className="text-sm text-gray-500">
            No master menu items yet. Add items under Menu first, then publish them for today.
          </p>
        ) : (
          <ul className="divide-y divide-gray-100 rounded-xl border border-gray-200">
            {masterItems.map((item) => (
              <li key={item.id} className="flex items-center gap-4 px-4 py-3">
                <input
                  type="checkbox"
                  checked={selected.has(item.id)}
                  onChange={() => toggle(item.id)}
                  className="h-4 w-4 rounded border-gray-300 text-orange-600 focus:ring-orange-500"
                />
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-gray-900">{item.name}</p>
                  <p className="text-xs text-gray-500">
                    {item.is_veg ? 'Veg' : 'Non-veg'} · stock {item.total_stock ?? 0}
                  </p>
                </div>
                {!item.is_active ? (
                  <span className="text-xs text-amber-600">Inactive in master menu</span>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}
