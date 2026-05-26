import { useMemo, useState } from 'react'
import { DateRangeToolbar } from '../../components/admin/DateRangeToolbar'
import { Card, CardHeader } from '../../components/ui/Card'
import { DataTable, type TableFilter } from '../../components/ui/DataTable'
import { PageHeader } from '../../components/ui/PageHeader'
import { useApi } from '../../hooks/useApi'
import { adminOrderColumns } from '../../lib/adminTables'
import { buildDateQuery, rangeForPreset, type DatePreset, type DateRange } from '../../lib/dateRange'
import type { Order, PaginatedResponse } from '../../types/api'

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'preparing', label: 'Preparing' },
  { value: 'ready', label: 'Ready' },
  { value: 'dispatched', label: 'Dispatched' },
  { value: 'delivered', label: 'Delivered' },
  { value: 'cancelled', label: 'Cancelled' },
]

export function AdminOrdersPage() {
  const [range, setRange] = useState<DateRange>(() => rangeForPreset('30d'))
  const [, setPreset] = useState<DatePreset>('30d')
  const [statusFilter, setStatusFilter] = useState('')

  const query = buildDateQuery(range, {
    per_page: 100,
    status: statusFilter || undefined,
  })

  const { data, loading, error } = useApi<PaginatedResponse<Order> & { date_from?: string; date_to?: string }>(
    `/api/v1/admin/orders?${query}`,
    [range.from, range.to, statusFilter],
  )

  const orders = data?.data ?? []
  const columns = useMemo(() => adminOrderColumns(), [])

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
    <div className="space-y-6">
      <PageHeader
        title="All orders"
        description="Train orders across every pantry — filter by date and status"
      />

      <DateRangeToolbar
        value={range}
        onChange={(r, p) => {
          setRange(r)
          setPreset(p)
        }}
      />

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      <Card padding="none">
        <div className="p-6">
          <CardHeader
            title="Order log"
            description={
              data?.meta
                ? `${data.meta.total} orders · ${data.date_from ?? range.from} to ${data.date_to ?? range.to}`
                : undefined
            }
          />
          <DataTable
            columns={columns}
            data={orders}
            rowKey={(o) => o.id}
            filters={filters}
            searchPlaceholder="Search PNR, passenger, pantry, train…"
            searchKeys={[
              'pnr',
              'passenger_name',
              'customer_name',
              'vendor_name',
              'train_number',
              'status',
            ]}
            loading={loading}
            pageSize={20}
            emptyTitle="No orders"
            emptyDescription="Try a wider date range or different status filter."
          />
        </div>
      </Card>
    </div>
  )
}
