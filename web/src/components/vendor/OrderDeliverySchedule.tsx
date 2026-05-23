import { useState } from 'react'
import { ButtonLight } from '../ui/ButtonLight'
import { useAuth } from '../../context/AuthContext'
import { apiRequest } from '../../lib/api'
import { formatDate } from '../../lib/format'
import type { Order } from '../../types/api'

function toDatetimeLocal(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fromDatetimeLocal(value: string): string {
  return new Date(value).toISOString()
}

export function OrderDeliverySchedule({
  order,
  onUpdated,
}: {
  order: Order
  onUpdated: () => void
}) {
  const { token } = useAuth()
  const [start, setStart] = useState(toDatetimeLocal(order.delivery_window_start))
  const [end, setEnd] = useState(toDatetimeLocal(order.delivery_window_end))
  const [notify, setNotify] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!token || !start) return
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      await apiRequest<Order>(
        `/api/v1/orders/${order.id}/delivery`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            delivery_window_start: fromDatetimeLocal(start),
            delivery_window_end: end ? fromDatetimeLocal(end) : undefined,
            notify_customer: notify,
          }),
        },
        token,
      )
      setMessage(notify ? 'Saved — customer notified on WhatsApp.' : 'Delivery time saved.')
      onUpdated()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSave} className="space-y-4">
      <p className="text-sm text-gray-600">
        Set when food will reach the passenger. Saving sends a WhatsApp message like
        &quot;Your order will be delivered by 10 PM&quot;.
      </p>

      {order.customer_phone ? (
        <p className="text-xs text-gray-500">
          WhatsApp: {order.customer_phone}
          {order.delivery_notified_at
            ? ` · Last notified ${formatDate(order.delivery_notified_at)}`
            : null}
        </p>
      ) : (
        <p className="text-sm text-amber-700">No customer phone — WhatsApp notify will be skipped.</p>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-medium text-gray-700">Delivery from</span>
          <input
            type="datetime-local"
            required
            value={start}
            onChange={(e) => setStart(e.target.value)}
            className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>
        <label className="block text-sm">
          <span className="font-medium text-gray-700">Delivery until (optional)</span>
          <input
            type="datetime-local"
            value={end}
            onChange={(e) => setEnd(e.target.value)}
            className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        </label>
      </div>

      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={notify}
          onChange={(e) => setNotify(e.target.checked)}
          className="rounded border-gray-300 text-orange-600"
        />
        Notify customer on WhatsApp
      </label>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {message ? <p className="text-sm text-green-700">{message}</p> : null}

      <ButtonLight type="submit" loading={saving} className="!w-auto">
        Save & notify
      </ButtonLight>
    </form>
  )
}
