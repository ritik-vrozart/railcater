import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { StatCard } from '../../components/ui/StatCard'
import { StatusBadge } from '../../components/ui/Badge'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import { formatDate, formatMoney } from '../../lib/format'
import type { Order, PaginatedResponse } from '../../types/api'
import { DataTable, type Column } from '../../components/ui/DataTable'

export function VendorOverviewPage() {
  const navigate = useNavigate()
  const { vendor, vendorId } = useVendor()
  const ordersPath = vendorId ? `/api/v1/orders?vendor_id=${vendorId}&per_page=100` : null
  const { data: ordersRes, loading } = useApi<PaginatedResponse<Order>>(ordersPath, [vendorId])

  const orders = ordersRes?.data ?? []

  const stats = useMemo(() => {
    const today = new Date().toDateString()
    const todayOrders = orders.filter((o) => new Date(o.created_at).toDateString() === today)
    const revenue = orders
      .filter((o) => o.status !== 'cancelled')
      .reduce((s, o) => s + o.total_cents, 0)
    const pending = orders.filter((o) => o.status === 'pending' || o.status === 'confirmed').length
    return {
      total: orders.length,
      today: todayOrders.length,
      revenue,
      pending,
    }
  }, [orders])

  const recentColumns: Column<Order>[] = [
    {
      key: 'pnr',
      header: 'PNR',
      render: (o) => o.pnr ?? '—',
    },
    {
      key: 'passenger',
      header: 'Passenger',
      render: (o) => o.passenger_name ?? o.customer_name ?? '—',
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
    },
    {
      key: 'created',
      header: 'Placed',
      render: (o) => formatDate(o.created_at),
      sortable: true,
    },
  ]

  return (
    <div>
      <PageHeader
        title="Overview"
        description={vendor ? `Welcome back — ${vendor.name}` : 'Loading vendor…'}
      />

      <div className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Total orders" value={stats.total} hint="All time for your vendor" />
        <StatCard label="Today's orders" value={stats.today} />
        <StatCard label="Revenue" value={formatMoney(stats.revenue)} hint="Excluding cancelled" />
        <StatCard label="Pending / confirmed" value={stats.pending} />
      </div>

      <Card>
        <CardHeader title="Recent orders" description="Latest train food orders for your pantry" />
        <DataTable
          columns={recentColumns}
          data={orders.slice(0, 20)}
          rowKey={(o) => o.id}
          onRowClick={(o) => navigate(`/vendor/orders/${o.id}`)}
          searchKeys={['pnr', 'passenger_name', 'status']}
          searchPlaceholder="Search orders…"
          loading={loading}
          pageSize={5}
          emptyTitle="No orders yet"
          emptyDescription="Train orders assigned to your vendor will appear here."
        />
      </Card>
    </div>
  )
}
