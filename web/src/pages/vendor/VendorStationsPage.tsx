import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { DataTable, type Column } from '../../components/ui/DataTable'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import type { Station } from '../../types/api'

export function VendorStationsPage() {
  const { vendor } = useVendor()
  const { data: stationsRes, loading } = useApi<{ data: Station[] }>('/api/v1/stations', [])

  // Demo: seeded vendor serves JP, KOTA, BPL — show all stations with served flag
  const servedCodes = new Set(['JP', 'KOTA', 'BPL'])

  const rows = (stationsRes?.data ?? []).map((s) => ({
    ...s,
    served: servedCodes.has(s.code),
  }))

  type Row = Station & { served: boolean }

  const columns: Column<Row>[] = [
    { key: 'code', header: 'Code', sortable: true },
    { key: 'name', header: 'Station', sortable: true },
    { key: 'city', header: 'City', sortable: true },
    { key: 'state', header: 'State', sortable: true },
    {
      key: 'served',
      header: 'Your coverage',
      render: (s) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
            s.served ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-500'
          }`}
        >
          {s.served ? 'Serving' : 'Not active'}
        </span>
      ),
    },
  ]

  return (
    <div>
      <PageHeader
        title="Stations"
        description={`Stations where ${vendor?.name ?? 'your vendor'} delivers food`}
      />

      <Card padding="none">
        <div className="p-6">
          <CardHeader
            title="Station coverage"
            description="Stations linked to your vendor account"
          />
          <DataTable
            columns={columns}
            data={rows}
            rowKey={(s) => s.id}
            searchKeys={['code', 'name', 'city', 'state']}
            searchPlaceholder="Search stations…"
            loading={loading}
            emptyTitle="No stations"
          />
        </div>
      </Card>
    </div>
  )
}
