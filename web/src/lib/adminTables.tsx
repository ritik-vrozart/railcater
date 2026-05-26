import { Badge } from '../components/ui/Badge'
import { StatusBadge } from '../components/ui/Badge'
import type { Column } from '../components/ui/DataTable'
import { formatDate, formatMoney } from './format'
import type { Order, VendorDetail } from '../types/api'

export function pantryColumns(): Column<VendorDetail>[] {
  return [
    {
      key: 'name',
      header: 'Pantry',
      sortable: true,
      render: (p) => (
        <div>
          <p className="font-medium text-gray-900">{p.name}</p>
          <p className="text-xs text-gray-400">{p.code}</p>
        </div>
      ),
    },
    {
      key: 'train',
      header: 'Train',
      render: (p) => {
        const trains = p.trains ?? []
        if (trains.length === 0) return <span className="text-gray-400">—</span>
        return (
          <div className="space-y-0.5">
            {trains.map((t) => (
              <p key={t.train_id} className="text-gray-800">
                <span className="font-medium">{t.train_number}</span>
                <span className="text-gray-500"> — {t.train_name}</span>
              </p>
            ))}
          </div>
        )
      },
    },
    {
      key: 'admin_email',
      header: 'Manager email',
      sortable: true,
      render: (p) => <span className="text-gray-600">{p.admin_email ?? '—'}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      render: (p) => (
        <Badge variant={p.is_active && p.is_approved ? 'success' : 'warning'}>
          {p.is_active && p.is_approved ? 'Active' : 'Inactive'}
        </Badge>
      ),
    },
    {
      key: 'period_orders',
      header: 'Orders (period)',
      className: 'text-right',
      sortable: true,
      render: (p) => <span className="font-medium">{p.period_orders ?? 0}</span>,
    },
    {
      key: 'period_revenue',
      header: 'Revenue (period)',
      className: 'text-right',
      sortable: true,
      render: (p) => (
        <span className="font-medium">{formatMoney(p.period_revenue_cents ?? 0)}</span>
      ),
    },
    {
      key: 'total_orders',
      header: 'All-time orders',
      className: 'text-right',
      render: (p) => <span className="text-gray-600">{p.total_orders ?? 0}</span>,
    },
    {
      key: 'total_revenue',
      header: 'All-time revenue',
      className: 'text-right',
      render: (p) => (
        <span className="text-gray-600">{formatMoney(p.total_revenue_cents ?? 0)}</span>
      ),
    },
  ]
}

export function adminOrderColumns(): Column<Order>[] {
  return [
    {
      key: 'id',
      header: 'Order',
      render: (o) => (
        <span className="font-mono text-xs text-amber-700">{o.id.slice(0, 8)}…</span>
      ),
    },
    {
      key: 'pantry',
      header: 'Pantry',
      sortable: true,
      render: (o) => o.vendor_name ?? '—',
    },
    { key: 'pnr', header: 'PNR', render: (o) => o.pnr ?? '—', sortable: true },
    {
      key: 'passenger',
      header: 'Passenger',
      render: (o) => (
        <div>
          <p className="font-medium">{o.passenger_name ?? o.customer_name ?? '—'}</p>
          {o.coach ? (
            <p className="text-xs text-gray-500">
              {o.coach} / {o.berth}
            </p>
          ) : null}
        </div>
      ),
    },
    {
      key: 'train',
      header: 'Train',
      render: (o) =>
        o.train_number ? (
          <div>
            <p className="font-medium">{o.train_number}</p>
            <p className="text-xs text-gray-500">{o.train_name}</p>
          </div>
        ) : (
          '—'
        ),
    },
    {
      key: 'total',
      header: 'Amount',
      className: 'text-right',
      sortable: true,
      render: (o) => formatMoney(o.total_cents),
    },
    {
      key: 'status',
      header: 'Status',
      render: (o) => <StatusBadge status={o.status} />,
    },
    {
      key: 'created_at',
      header: 'Placed',
      sortable: true,
      render: (o) => <span className="text-gray-600">{formatDate(o.created_at)}</span>,
    },
  ]
}
