import { useState } from 'react'
import { ButtonLight } from '../ui/ButtonLight'
import { useAuth } from '../../context/AuthContext'
import { apiRequest } from '../../lib/api'
import type { Order, PaymentMethod, PaymentStatus } from '../../types/api'

export function OrderPayment({
  order,
  onUpdated,
}: {
  order: Order
  onUpdated: () => void
}) {
  const { token } = useAuth()
  const [status, setStatus] = useState<PaymentStatus>(
    (order.payment_status as PaymentStatus) || 'pending',
  )
  const [method, setMethod] = useState<PaymentMethod | ''>(
    (order.payment_method as PaymentMethod) || '',
  )
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!token) return
    if (status === 'paid' && !method) {
      setError('Paid hone par Cash ya UPI choose karein.')
      return
    }
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      await apiRequest<Order>(
        `/api/v1/orders/${order.id}/payment`,
        {
          method: 'PATCH',
          body: JSON.stringify({
            payment_status: status,
            payment_method: status === 'paid' ? method : null,
          }),
        },
        token,
      )
      setMessage(status === 'paid' ? `Payment marked — ${method.toUpperCase()}` : 'Payment pending.')
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
        Order total: <strong>₹{(order.total_cents / 100).toFixed(2)}</strong> — mark jab passenger ne pay kar diya ho.
      </p>

      <div className="flex flex-wrap gap-4">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name="payment_status"
            checked={status === 'pending'}
            onChange={() => {
              setStatus('pending')
              setMethod('')
            }}
            className="text-orange-600"
          />
          Pending
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name="payment_status"
            checked={status === 'paid'}
            onChange={() => setStatus('paid')}
            className="text-orange-600"
          />
          Paid
        </label>
      </div>

      {status === 'paid' ? (
        <div className="flex flex-wrap gap-4">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="radio"
              name="payment_method"
              checked={method === 'cash'}
              onChange={() => setMethod('cash')}
              className="text-orange-600"
            />
            Cash
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="radio"
              name="payment_method"
              checked={method === 'upi'}
              onChange={() => setMethod('upi')}
              className="text-orange-600"
            />
            UPI
          </label>
        </div>
      ) : null}

      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {message ? <p className="text-sm text-green-700">{message}</p> : null}

      <ButtonLight type="submit" loading={saving} className="!w-auto">
        Save payment
      </ButtonLight>
    </form>
  )
}
