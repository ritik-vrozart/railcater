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
  const initial =
    order.expected_delivery_at || order.delivery_window_start || ''
  const [at, setAt] = useState(toDatetimeLocal(initial))
  const [notify, setNotify] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!token || !at) return
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      await apiRequest<Order>(
        `/api/v1/orders/${order.id}/delivery`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            expected_delivery_at: fromDatetimeLocal(at),
            notify_customer: notify,
          }),
        },
        token,
      )
      setMessage(
        notify
          ? 'Saved — customer ko WhatsApp par time bhej diya.'
          : 'Delivery time save ho gaya.',
      )
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
        Passenger ko batayein <strong>kitne baje tak khana seat par milega</strong> (coach{' '}
        {order.coach}/{order.berth}).
      </p>

      {order.customer_phone ? (
        <p className="text-xs text-gray-500">
          WhatsApp: {order.customer_phone}
          {order.delivery_notified_at
            ? ` · Last notified ${formatDate(order.delivery_notified_at)}`
            : null}
        </p>
      ) : (
        <p className="text-sm text-amber-700">No customer phone — WhatsApp notify skip hoga.</p>
      )}

      <label className="block text-sm">
        <span className="font-medium text-gray-700">Khana kab tak milega?</span>
        <input
          type="datetime-local"
          required
          value={at}
          onChange={(e) => setAt(e.target.value)}
          className="mt-1 w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm"
        />
      </label>

      <label className="flex items-center gap-2 text-sm text-gray-700">
        <input
          type="checkbox"
          checked={notify}
          onChange={(e) => setNotify(e.target.checked)}
          className="rounded border-gray-300 text-orange-600"
        />
        Customer ko WhatsApp par bhejein
      </label>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {message ? <p className="text-sm text-green-700">{message}</p> : null}

      <ButtonLight type="submit" loading={saving} className="!w-auto">
        Save & notify
      </ButtonLight>
    </form>
  )
}
