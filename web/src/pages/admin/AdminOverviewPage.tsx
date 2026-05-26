import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { DateRangeToolbar } from '../../components/admin/DateRangeToolbar'
import { Card, CardHeader } from '../../components/ui/Card'
import { DataTable } from '../../components/ui/DataTable'
import { PageHeader } from '../../components/ui/PageHeader'
import { StatCard } from '../../components/ui/StatCard'
import { useApi } from '../../hooks/useApi'
import { adminOrderColumns, pantryColumns } from '../../lib/adminTables'
import { buildDateQuery, rangeForPreset, type DatePreset, type DateRange } from '../../lib/dateRange'
import { formatMoney } from '../../lib/format'
import type { AdminDashboard, Order, PaginatedResponse } from '../../types/api'

export function AdminOverviewPage() {
  const [range, setRange] = useState<DateRange>(() => rangeForPreset('30d'))
  const [preset, setPreset] = useState<DatePreset>('30d')
  const query = buildDateQuery(range)

  const { data: dash, loading, error } = useApi<AdminDashboard>(
    `/api/v1/admin/dashboard?${query}`,
    [range.from, range.to],
  )

  const { data: ordersRes, loading: ordersLoading } = useApi<PaginatedResponse<Order>>(
    `/api/v1/admin/orders?${query}&per_page=100`,
    [range.from, range.to],
  )

  const pantries = dash?.pantries ?? []
  const orders = ordersRes?.data ?? []

  const columns = useMemo(() => pantryColumns(), [])
  const orderCols = useMemo(() => adminOrderColumns(), [])

  return (
    <div className="space-y-6">
      <PageHeader
        title="Department overview"
        description="Pantries, trains, and sales for the selected period"
        actions={
          <Link
            to="/admin/invite"
            className="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-500"
          >
            Invite pantry
          </Link>
        }
      />

      <DateRangeToolbar
        value={range}
        onChange={(r, p) => {
          setRange(r)
          setPreset(p)
        }}
      />

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      {loading ? (
        <p className="text-sm text-gray-500">Loading dashboard…</p>
      ) : dash ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Total pantries" value={String(dash.total_pantries)} />
            <StatCard label="Active pantries" value={String(dash.active_pantries)} />
            <StatCard
              label="Orders in period"
              value={String(dash.period_orders)}
              hint={preset === 'custom' ? `${range.from} – ${range.to}` : undefined}
            />
            <StatCard
              label="Revenue in period"
              value={formatMoney(dash.period_revenue_cents)}
              hint={`All-time ${formatMoney(dash.total_revenue_cents)}`}
            />
          </div>

          <Card padding="none">
            <div className="p-6">
              <CardHeader
                title="Pantries by train"
                description="Order and revenue counts for the selected date range"
              />
              <DataTable
                columns={columns}
                data={pantries}
                rowKey={(p) => p.id}
                searchPlaceholder="Search pantry, train, email…"
                searchKeys={[
                  'name',
                  'code',
                  (p) => p.admin_email ?? '',
                  (p) => (p.trains ?? []).map((t) => `${t.train_number} ${t.train_name}`).join(' '),
                ]}
                loading={loading}
                pageSize={15}
                emptyTitle="No pantries"
                emptyDescription="Invite a pantry to link it to a train."
              />
            </div>
          </Card>

          <Card padding="none">
            <div className="p-6">
              <CardHeader
                title="Orders in period"
                description="Train food orders across all pantries"
              />
              <DataTable
                columns={orderCols}
                data={orders}
                rowKey={(o) => o.id}
                searchPlaceholder="Search PNR, passenger, pantry…"
                searchKeys={[
                  'pnr',
                  'passenger_name',
                  'customer_name',
                  'vendor_name',
                  'train_number',
                ]}
                loading={ordersLoading}
                pageSize={15}
                emptyTitle="No orders"
                emptyDescription="No orders in this date range."
              />
            </div>
          </Card>
        </>
      ) : null}
    </div>
  )
}
