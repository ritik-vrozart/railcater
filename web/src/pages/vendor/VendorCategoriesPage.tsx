import { useState } from 'react'
import { CategoryFormModal } from '../../components/vendor/CategoryFormModal'
import { Badge } from '../../components/ui/Badge'
import { ButtonLight } from '../../components/ui/ButtonLight'
import { Card, CardHeader } from '../../components/ui/Card'
import { DataTable, type Column } from '../../components/ui/DataTable'
import { PageHeader } from '../../components/ui/PageHeader'
import { useVendor } from '../../context/VendorContext'
import { useApi } from '../../hooks/useApi'
import type { MenuCategory } from '../../types/api'

export function VendorCategoriesPage() {
  const { vendorId } = useVendor()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<MenuCategory | null>(null)

  const path = vendorId ? `/api/v1/vendors/${vendorId}/menu/categories` : null
  const { data, loading, reload } = useApi<{ data: MenuCategory[] }>(path, [vendorId])

  const columns: Column<MenuCategory>[] = [
    { key: 'name', header: 'Category', sortable: true },
    {
      key: 'food_type',
      header: 'Type',
      render: (c) => (
        <Badge variant={c.food_type === 'veg' ? 'success' : 'danger'}>
          {c.food_type === 'veg' ? 'Veg' : 'Non-veg'}
        </Badge>
      ),
    },
    {
      key: 'description',
      header: 'Description',
      render: (c) => <span className="text-gray-500">{c.description ?? '—'}</span>,
    },
    { key: 'sort_order', header: 'Order', sortable: true },
    {
      key: 'is_active',
      header: 'Status',
      render: (c) => (
        <Badge variant={c.is_active ? 'success' : 'default'}>{c.is_active ? 'Active' : 'Inactive'}</Badge>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (c) => (
        <ButtonLight variant="ghost" size="sm" className="!w-auto" onClick={() => { setEditing(c); setModalOpen(true) }}>
          Edit
        </ButtonLight>
      ),
    },
  ]

  return (
    <div>
      <PageHeader
        title="Categories"
        description="Master list — create categories before adding menu items"
        actions={
          <ButtonLight
            onClick={() => {
              setEditing(null)
              setModalOpen(true)
            }}
            className="!w-auto"
          >
            + Add category
          </ButtonLight>
        }
      />

      <Card padding="none">
        <div className="p-6">
          <CardHeader title="Category master" description={`${(data?.data ?? []).length} categories`} />
          <DataTable
            columns={columns}
            data={data?.data ?? []}
            rowKey={(c) => c.id}
            searchKeys={['name', 'description', 'food_type']}
            searchPlaceholder="Search categories…"
            loading={loading}
            emptyTitle="No categories"
            emptyDescription="Create Meals, Snacks, Beverages, etc. before adding menu items."
          />
        </div>
      </Card>

      <CategoryFormModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSaved={reload}
        category={editing}
      />
    </div>
  )
}
