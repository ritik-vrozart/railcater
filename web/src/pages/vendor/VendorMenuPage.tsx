import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { MenuItemFormModal } from '../../components/vendor/MenuItemFormModal'
import { Badge } from '../../components/ui/Badge'
import { ButtonLight } from '../../components/ui/ButtonLight'
import { Card, CardHeader } from '../../components/ui/Card'
import { PageHeader } from '../../components/ui/PageHeader'
import { DataTable, type Column, type TableFilter } from '../../components/ui/DataTable'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import { formatMoney } from '../../lib/format'
import type { MenuItem } from '../../types/api'

export function VendorMenuPage() {
  const { vendorId } = useVendor()
  const [vegFilter, setVegFilter] = useState('')
  const [activeFilter, setActiveFilter] = useState('')
  const [modalOpen, setModalOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<MenuItem | null>(null)

  const path = vendorId ? `/api/v1/vendors/${vendorId}/menu?active_only=false` : null
  const { data: menuRes, loading, reload } = useApi<{ data: MenuItem[] }>(path, [vendorId])

  const items = useMemo(() => {
    let list = menuRes?.data ?? []
    if (vegFilter === 'veg') list = list.filter((i) => i.is_veg)
    if (vegFilter === 'nonveg') list = list.filter((i) => !i.is_veg)
    if (activeFilter === 'active') list = list.filter((i) => i.is_active)
    if (activeFilter === 'inactive') list = list.filter((i) => !i.is_active)
    return list
  }, [menuRes, vegFilter, activeFilter])

  const columns: Column<MenuItem>[] = [
    {
      key: 'name',
      header: 'Item',
      sortable: true,
      render: (i) => (
        <div className="flex items-center gap-3">
          {i.image_url ? (
            <img src={i.image_url} alt="" className="h-10 w-10 rounded-lg object-cover" />
          ) : (
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 text-xs text-gray-400">
              —
            </div>
          )}
          <div>
            <p className="font-medium text-gray-900">{i.name}</p>
            <p className="text-xs text-gray-500">{i.category ?? '—'}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'portions',
      header: 'Portions & price',
      render: (i) => (
        <div className="space-y-0.5 text-xs">
          {(i.portions ?? []).map((p) => (
            <div key={p.id} className="text-gray-600">
              <span className="font-medium capitalize">{p.portion}</span>: {formatMoney(p.price_cents)}
              <span className="text-gray-400"> · stock {p.stock_quantity}</span>
            </div>
          ))}
        </div>
      ),
    },
    {
      key: 'stock',
      header: 'Total stock',
      render: (i) => <span className="font-medium">{i.total_stock ?? 0}</span>,
      sortable: true,
    },
    {
      key: 'veg',
      header: 'Type',
      render: (i) => (
        <Badge variant={i.is_veg ? 'success' : 'danger'}>{i.is_veg ? 'Veg' : 'Non-veg'}</Badge>
      ),
    },
    {
      key: 'active',
      header: 'Status',
      render: (i) => (
        <Badge variant={i.is_active ? 'success' : 'default'}>{i.is_active ? 'Active' : 'Inactive'}</Badge>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (i) => (
        <ButtonLight variant="ghost" size="sm" className="!w-auto" onClick={() => { setEditingItem(i); setModalOpen(true) }}>
          Edit
        </ButtonLight>
      ),
    },
  ]

  const filters: TableFilter[] = [
    {
      id: 'veg',
      label: 'Food type',
      options: [
        { value: '', label: 'All' },
        { value: 'veg', label: 'Vegetarian' },
        { value: 'nonveg', label: 'Non-vegetarian' },
      ],
      value: vegFilter,
      onChange: setVegFilter,
    },
    {
      id: 'active',
      label: 'Status',
      options: [
        { value: '', label: 'All' },
        { value: 'active', label: 'Active' },
        { value: 'inactive', label: 'Inactive' },
      ],
      value: activeFilter,
      onChange: setActiveFilter,
    },
  ]

  return (
    <div>
      <PageHeader
        title="Menu"
        description="Dishes with quarter / half / full portions and stock"
        actions={
          <div className="flex gap-2">
            <Link to="/vendor/categories">
              <ButtonLight variant="secondary" className="!w-auto" type="button">
                Categories
              </ButtonLight>
            </Link>
            <ButtonLight onClick={() => { setEditingItem(null); setModalOpen(true) }} className="!w-auto">
              + Add item
            </ButtonLight>
          </div>
        }
      />

      <Card padding="none">
        <div className="p-6">
          <CardHeader title="Menu items" description={`${items.length} items`} />
          <DataTable
            columns={columns}
            data={items}
            rowKey={(i) => i.id}
            filters={filters}
            searchKeys={['name', 'category', 'description']}
            searchPlaceholder="Search menu…"
            loading={loading}
            pageSize={10}
            emptyTitle="No menu items"
            emptyDescription="Create categories first, then add items with portions and stock."
          />
        </div>
      </Card>

      <MenuItemFormModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSaved={reload}
        item={editingItem}
      />
    </div>
  )
}
