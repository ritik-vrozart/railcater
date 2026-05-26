import { useState } from 'react'
import { apiRequest, ApiError } from '../../lib/api'
import { useAuth } from '../../context/AuthContext'
import type { Order } from '../../types/api'
import { Alert } from '../ui/Alert'
import { ButtonLight } from '../ui/ButtonLight'
import { StatusBadge } from '../ui/Badge'

const TRAIN_FLOW: { status: string; label: string; description: string }[] = [
  { status: 'confirmed', label: 'Confirmed', description: 'Order received from passenger' },
  { status: 'preparing', label: 'Preparing', description: 'Pantry is preparing your order' },
  { status: 'ready', label: 'Ready', description: 'Packed and ready to dispatch' },
  { status: 'dispatched', label: 'Dispatched', description: 'Sent from pantry toward seat' },
  { status: 'delivered', label: 'Delivered', description: 'Handed to passenger' },
]

function nextAction(current: string): { status: string; label: string } | null {
  switch (current) {
    case 'confirmed':
      return { status: 'preparing', label: 'Start preparing' }
    case 'preparing':
      return { status: 'ready', label: 'Mark ready' }
    case 'ready':
      return { status: 'dispatched', label: 'Dispatch order' }
    case 'dispatched':
      return { status: 'delivered', label: 'Mark delivered' }
    default:
      return null
  }
}

export function OrderStatusWorkflow({
  order,
  onUpdated,
}: {
  order: Order
  onUpdated: () => void
}) {
  const { token } = useAuth()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const action = nextAction(order.status)
  const isCancelled = order.status === 'cancelled'
  const isDone = order.status === 'delivered'

  async function updateStatus(status: string) {
    if (!token) return
    setError('')
    setLoading(true)
    try {
      await apiRequest(
        `/api/v1/orders/${order.id}/status`,
        { method: 'PATCH', body: JSON.stringify({ status }) },
        token,
      )
      onUpdated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to update status')
    } finally {
      setLoading(false)
    }
  }

  if (order.source !== 'train') {
    return (
      <p className="text-sm text-gray-500">
        Current status: <StatusBadge status={order.status} />
      </p>
    )
  }

  const currentIdx = TRAIN_FLOW.findIndex((s) => s.status === order.status)

  return (
    <div className="space-y-4">
      {error ? <Alert message={error} /> : null}

      <ol className="space-y-3">
        {TRAIN_FLOW.map((step, idx) => {
          const done = currentIdx > idx || order.status === step.status && isDone
          const active = order.status === step.status
          return (
            <li
              key={step.status}
              className={`flex gap-3 rounded-lg border px-4 py-3 ${
                active
                  ? 'border-orange-300 bg-orange-50'
                  : done
                    ? 'border-emerald-200 bg-emerald-50/50'
                    : 'border-gray-200 bg-white'
              }`}
            >
              <span
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
                  done
                    ? 'bg-emerald-600 text-white'
                    : active
                      ? 'bg-orange-600 text-white'
                      : 'bg-gray-200 text-gray-500'
                }`}
              >
                {done ? '✓' : idx + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="font-medium text-gray-900">{step.label}</p>
                <p className="text-xs text-gray-500">{step.description}</p>
              </div>
              {active && !isDone && !isCancelled ? (
                <StatusBadge status={order.status} />
              ) : null}
            </li>
          )
        })}
      </ol>

      {!isCancelled && !isDone && action ? (
        <ButtonLight onClick={() => updateStatus(action.status)} disabled={loading}>
          {loading ? 'Updating…' : action.label}
        </ButtonLight>
      ) : null}

      {isDone ? (
        <p className="text-sm font-medium text-emerald-700">Order completed — passenger notified on WhatsApp.</p>
      ) : null}
      {isCancelled ? (
        <p className="text-sm text-red-600">This order was cancelled.</p>
      ) : null}
    </div>
  )
}
