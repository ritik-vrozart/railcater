import { Link, useParams } from 'react-router-dom'
import { OrderDeliverySchedule } from '../../components/vendor/OrderDeliverySchedule'
import { OrderStatusWorkflow } from '../../components/vendor/OrderStatusWorkflow'
import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { StatusBadge } from '../../components/ui/Badge'
import { Alert } from '../../components/ui/Alert'
import { useApi } from '../../hooks/useApi'
import { formatDate, formatMoney } from '../../lib/format'
import type { Order, OrderItem } from '../../types/api'

function itemDisplayName(item: OrderItem): string {
  const name = item.product_name || 'Item'
  if (item.portion_label) return `${name} (${item.portion_label})`
  if (item.portion) return `${name} (${item.portion})`
  return name
}

export function VendorOrderDetailPage() {
  const { orderId } = useParams<{ orderId: string }>()
  const { data: order, loading, error, reload } = useApi<Order>(
    orderId ? `/api/v1/orders/${orderId}` : null,
    [orderId],
  )

  const passenger = order?.passenger_name ?? order?.customer_name
  const items = order?.items ?? []

  return (
    <div>
      <PageHeader
        title="Order details"
        description={orderId ? `Order #${orderId.slice(0, 8)}…` : undefined}
        actions={
          <Link
            to="/vendor/orders"
            className="text-sm font-medium text-orange-600 hover:text-orange-500"
          >
            ← Back to orders
          </Link>
        }
      />

      {loading ? (
        <p className="text-sm text-gray-500">Loading order…</p>
      ) : error ? (
        <Alert message={error} />
      ) : !order ? (
        <Alert message="Order not found" />
      ) : (
        <div className="space-y-6">
          {/* Items ordered — main focus */}
          <Card padding="none">
            <div className="p-6">
              <CardHeader
                title="Items ordered"
                description={`${items.length} item${items.length === 1 ? '' : 's'} · Total ${formatMoney(order.total_cents)}`}
              />
              {items.length === 0 ? (
                <p className="text-sm text-gray-500">No items recorded for this order.</p>
              ) : (
                <ul className="divide-y divide-gray-100 rounded-xl border border-gray-200 bg-white">
                  {items.map((item) => (
                    <li
                      key={item.id}
                      className="flex flex-wrap items-center justify-between gap-4 px-4 py-4 sm:flex-nowrap"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="font-semibold text-gray-900">{itemDisplayName(item)}</p>
                        {item.sku ? (
                          <p className="mt-0.5 text-xs text-gray-400">SKU {item.sku}</p>
                        ) : null}
                      </div>
                      <div className="flex shrink-0 items-center gap-6 text-sm">
                        <span className="rounded-full bg-gray-100 px-3 py-1 font-medium text-gray-700">
                          Qty {item.quantity}
                        </span>
                        <span className="text-gray-500">{formatMoney(item.unit_price_cents)} each</span>
                        <span className="min-w-[5rem] text-right font-semibold text-gray-900">
                          {formatMoney(item.line_total_cents)}
                        </span>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
              {items.length > 0 ? (
                <div className="mt-4 flex justify-end border-t border-gray-100 pt-4">
                  <dl className="text-right text-sm">
                    <dt className="text-gray-500">Order total</dt>
                    <dd className="text-xl font-bold text-gray-900">{formatMoney(order.total_cents)}</dd>
                  </dl>
                </div>
              ) : null}
            </div>
          </Card>

          <Card>
            <CardHeader
              title="Order status"
              description="Update progress — customer gets WhatsApp at each step"
            />
            <OrderStatusWorkflow order={order} onUpdated={reload} />
          </Card>

          <Card>
            <CardHeader
              title="Schedule delivery"
              description="Customer gets WhatsApp when you save (e.g. delivered by 10 PM)"
            />
            <OrderDeliverySchedule order={order} onUpdated={reload} />
          </Card>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader title="Delivery & journey" />
              <dl className="space-y-3 text-sm">
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Status</dt>
                  <dd>
                    <StatusBadge status={order.status} />
                  </dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">PNR</dt>
                  <dd className="font-medium text-gray-900">{order.pnr ?? '—'}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Passenger</dt>
                  <dd className="text-right font-medium text-gray-900">{passenger ?? '—'}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Seat</dt>
                  <dd className="font-medium text-gray-900">
                    {order.coach ? `${order.coach} / ${order.berth}` : '—'}
                  </dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Train</dt>
                  <dd className="text-right font-medium text-gray-900">
                    {order.train_number
                      ? `${order.train_number} — ${order.train_name}`
                      : '—'}
                  </dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Delivery window</dt>
                  <dd className="text-right text-gray-900">
                    {order.delivery_window_start && order.delivery_window_end
                      ? `${formatDate(order.delivery_window_start)} – ${formatDate(order.delivery_window_end)}`
                      : '—'}
                  </dd>
                </div>
              </dl>
            </Card>

            <Card>
              <CardHeader title="Order info" />
              <dl className="space-y-3 text-sm">
                <div>
                  <dt className="text-gray-500">Order ID</dt>
                  <dd className="mt-0.5 break-all font-mono text-xs text-gray-900">{order.id}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Source</dt>
                  <dd className="capitalize text-gray-900">{order.source}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-gray-500">Placed at</dt>
                  <dd className="text-gray-900">{formatDate(order.created_at)}</dd>
                </div>
                {order.notes ? (
                  <div>
                    <dt className="text-gray-500">Notes</dt>
                    <dd className="mt-1 rounded-lg bg-gray-50 px-3 py-2 text-gray-800">{order.notes}</dd>
                  </div>
                ) : null}
              </dl>
            </Card>
          </div>
        </div>
      )}
    </div>
  )
}
