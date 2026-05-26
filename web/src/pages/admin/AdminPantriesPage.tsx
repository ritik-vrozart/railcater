import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { DateRangeToolbar } from '../../components/admin/DateRangeToolbar'
import { Card, CardHeader } from '../../components/ui/Card'
import { DataTable } from '../../components/ui/DataTable'
import { PageHeader } from '../../components/ui/PageHeader'
import { useApi } from '../../hooks/useApi'
import { pantryColumns } from '../../lib/adminTables'
import { buildDateQuery, rangeForPreset, type DatePreset, type DateRange } from '../../lib/dateRange'
import type { VendorDetail } from '../../types/api'

interface PantriesResponse {
  data: VendorDetail[]
  date_from: string
  date_to: string
}

export function AdminPantriesPage() {
  const [range, setRange] = useState<DateRange>(() => rangeForPreset('30d'))
  const [, setPreset] = useState<DatePreset>('30d')
  const query = buildDateQuery(range)

  const { data, loading, error } = useApi<PantriesResponse>(
    `/api/v1/admin/pantries?${query}`,
    [range.from, range.to],
  )

  const pantries = data?.data ?? []
  const columns = useMemo(() => pantryColumns(), [])

  return (
    <div className="space-y-6">
      <PageHeader
        title="All pantries"
        description="Pantry ↔ train mapping with period and all-time stats"
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

      <Card padding="none">
        <div className="p-6">
          <CardHeader
            title="Pantry directory"
            description={
              data
                ? `Stats for ${data.date_from} to ${data.date_to}`
                : 'Filter by date to see orders and revenue per pantry'
            }
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
            pageSize={20}
            emptyTitle="No pantries"
            emptyDescription="Invite a pantry to get started."
          />
        </div>
      </Card>
    </div>
  )
}
