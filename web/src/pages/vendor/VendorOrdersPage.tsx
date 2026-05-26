import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { StatusBadge } from '../../components/ui/Badge'
import { DataTable, type Column, type TableFilter } from '../../components/ui/DataTable'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import { formatDate, formatMoney } from '../../lib/format'
import type { Order, PaginatedResponse } from '../../types/api'

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'pending', label: 'Pending' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'processing', label: 'Processing' },
  { value: 'shipped', label: 'Shipped' },
  { value: 'delivered', label: 'Delivered' },
  { value: 'cancelled', label: 'Cancelled' },
]

export function VendorOrdersPage() {
  const navigate = useNavigate()
  const { vendorId } = useVendor()
  const [statusFilter, setStatusFilter] = useState('')

  const query = vendorId
    ? `/api/v1/orders?vendor_id=${vendorId}&per_page=100${statusFilter ? `&status=${statusFilter}` : ''}`
    : null

  const { data, loading, reload } = useApi<PaginatedResponse<Order>>(query, [vendorId, statusFilter])

  const columns: Column<Order>[] = [
    {
      key: 'id',
      header: 'Order',
      render: (o) => (
        <span className="font-mono text-xs text-orange-600">{o.id.slice(0, 8)}…</span>
      ),
    },
    { key: 'pnr', header: 'PNR', render: (o) => o.pnr ?? '—', sortable: true },
    {
      key: 'passenger',
      header: 'Passenger',
      render: (o) => (
        <div>
          <p className="font-medium text-gray-900">{o.passenger_name ?? o.customer_name ?? '—'}</p>
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
            <p>{o.train_number}</p>
            <p className="text-xs text-gray-500">{o.train_name}</p>
          </div>
        ) : (
          '—'
        ),
    },
    {
      key: 'total',
      header: 'Amount',
      render: (o) => formatMoney(o.total_cents),
      sortable: true,
    },
    {
      key: 'status',
      header: 'Status',
      render: (o) => <StatusBadge status={o.status} />,
      sortable: true,
    },
    {
      key: 'created',
      header: 'Placed at',
      render: (o) => formatDate(o.created_at),
      sortable: true,
    },
  ]

  const filters: TableFilter[] = [
    {
      id: 'status',
      label: 'Status',
      options: STATUS_OPTIONS,
      value: statusFilter,
      onChange: setStatusFilter,
    },
  ]

  return (
    <div>
      <PageHeader
        title="Orders"
        description="Manage incoming train food orders"
        actions={
          <button
            type="button"
            onClick={() => reload()}
            className="text-sm font-medium text-orange-600 hover:text-orange-500"
          >
            Refresh
          </button>
        }
      />

      <Card padding="none">
        <div className="p-6">
          <CardHeader
            title="All orders"
            description={`${data?.meta.total ?? 0} total — click a row to see items ordered`}
          />
          <DataTable
            columns={columns}
            data={data?.data ?? []}
            rowKey={(o) => o.id}
            onRowClick={(o) => navigate(`/vendor/orders/${o.id}`)}
            filters={filters}
            searchKeys={[
              'pnr',
              'passenger_name',
              'customer_name',
              'train_number',
              'status',
              (o) => o.id,
            ]}
            searchPlaceholder="Search by PNR, passenger, train…"
            loading={loading}
            pageSize={10}
            emptyTitle="No orders found"
            emptyDescription="Try changing filters or check back when new orders arrive."
          />
        </div>
      </Card>
    </div>
  )
}
