import { useEffect, useState, type FormEvent } from 'react'
import { apiRequest, ApiError } from '../../lib/api'
import { defaultPortionRows, portionsFromItem, PORTION_OPTIONS } from '../../lib/menuConstants'
import { useAuth } from '../../context/AuthContext'
import { useVendor } from '../../context/VendorContext'
import type { MenuCategory, MenuItem, PortionFormRow } from '../../types/api'
import { Alert } from '../ui/Alert'
import { FormField, TextArea, TextInput } from '../ui/FormField'
import { Modal, ModalFooter } from '../ui/Modal'
import { Select } from '../ui/Select'

const FORM_ID = 'menu-item-form'

export function MenuItemFormModal({
  open,
  onClose,
  onSaved,
  item,
}: {
  open: boolean
  onClose: () => void
  onSaved: () => void
  item?: MenuItem | null
}) {
  const { token } = useAuth()
  const { vendorId } = useVendor()
  const isEdit = !!item

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [imageUrl, setImageUrl] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [isVeg, setIsVeg] = useState(true)
  const [isActive, setIsActive] = useState(true)
  const [portions, setPortions] = useState<PortionFormRow[]>(defaultPortionRows)
  const [categories, setCategories] = useState<MenuCategory[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !vendorId || !token) return

    apiRequest<{ data: MenuCategory[] }>(
      `/api/v1/vendors/${vendorId}/menu/categories?active_only=true`,
      {},
      token,
    )
      .then((res) => setCategories(res.data))
      .catch(() => setCategories([]))

    if (item) {
      setName(item.name)
      setDescription(item.description ?? '')
      setImageUrl(item.image_url ?? '')
      setCategoryId(item.category_id ?? '')
      setIsVeg(item.is_veg)
      setIsActive(item.is_active)
      setPortions(portionsFromItem(item.portions ?? []))
    } else {
      setName('')
      setDescription('')
      setImageUrl('')
      setCategoryId('')
      setIsVeg(true)
      setIsActive(true)
      setPortions(defaultPortionRows())
    }
    setError('')
  }, [open, vendorId, token, item])

  function updatePortion(index: number, patch: Partial<PortionFormRow>) {
    setPortions((rows) => rows.map((r, i) => (i === index ? { ...r, ...patch } : r)))
  }

  function buildPortionsPayload() {
    const enabled = portions.filter((p) => p.enabled)
    if (enabled.length === 0) {
      throw new Error('Enable at least one portion (Quarter / Half / Full) with price and stock')
    }
    return enabled.map((p) => {
      const price = Math.round(parseFloat(p.priceRupees) * 100)
      const stock = parseInt(p.stockQuantity, 10)
      if (Number.isNaN(price) || price < 0) throw new Error(`Invalid price for ${p.label}`)
      if (Number.isNaN(stock) || stock < 0) throw new Error(`Invalid stock for ${p.label}`)
      const meta = PORTION_OPTIONS.find((o) => o.code === p.portion)!
      return {
        portion: p.portion,
        label: p.label.trim() || meta.label,
        price_cents: price,
        stock_quantity: stock,
        sort_order: meta.sortOrder,
        is_active: true,
      }
    })
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')

    if (!token) {
      setError('Session expired. Please sign out and sign in again.')
      return
    }
    if (!vendorId) {
      setError('Vendor profile is still loading. Wait a moment and try again.')
      return
    }
    if (!categoryId) {
      setError('Please select a category (create one under Categories master first)')
      return
    }

    setLoading(true)

    try {
      const portionsPayload = buildPortionsPayload()
      const body = {
        name: name.trim(),
        description: description.trim() || null,
        image_url: imageUrl.trim() || null,
        category_id: categoryId,
        is_veg: isVeg,
        is_active: isActive,
        portions: portionsPayload,
      }

      if (isEdit && item) {
        await apiRequest(`/api/v1/vendors/${vendorId}/menu/${item.id}`, {
          method: 'PUT',
          body: JSON.stringify(body),
        }, token)
      } else {
        await apiRequest(`/api/v1/vendors/${vendorId}/menu`, {
          method: 'POST',
          body: JSON.stringify(body),
        }, token)
      }
      onSaved()
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit menu item' : 'Add menu item'}
      description="Name, category, image, and portion-wise price & stock"
      footer={
        <ModalFooter
          formId={FORM_ID}
          onCancel={onClose}
          submitLabel={isEdit ? 'Update' : 'Create'}
          loading={loading}
        />
      }
    >
      <form id={FORM_ID} onSubmit={handleSubmit} className="space-y-4">
        {error ? <Alert message={error} /> : null}

        <FormField label="Item name">
          <TextInput required value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Paneer Thali" />
        </FormField>

        <FormField label="Description">
          <TextArea value={description} onChange={(e) => setDescription(e.target.value)} />
        </FormField>

        <FormField label="Image URL" hint="Optional — link to food photo">
          <TextInput
            type="url"
            value={imageUrl}
            onChange={(e) => setImageUrl(e.target.value)}
            placeholder="https://..."
          />
        </FormField>

        <Select
          label="Category"
          value={categoryId}
          onChange={(e) => {
            setCategoryId(e.target.value)
            const cat = categories.find((c) => c.id === e.target.value)
            if (cat) setIsVeg(cat.food_type === 'veg')
          }}
          required
        >
          <option value="">Select category</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name} ({c.food_type === 'veg' ? 'Veg' : 'Non-veg'})
            </option>
          ))}
        </Select>

        {categories.length === 0 ? (
          <p className="text-xs text-amber-700">No categories yet — add categories from the Categories page first.</p>
        ) : null}

        <div className="space-y-3 rounded-lg border border-gray-200 bg-gray-50 p-4">
          <p className="text-sm font-semibold text-gray-900">Portions, price & stock</p>
          <p className="text-xs text-gray-500">Enable each size you sell. Set price (₹) and available quantity.</p>

          {portions.map((row, index) => (
            <div key={row.portion} className="rounded-lg border border-gray-200 bg-white p-3">
              <label className="mb-2 flex items-center gap-2 text-sm font-medium text-gray-800">
                <input
                  type="checkbox"
                  checked={row.enabled}
                  onChange={(e) => updatePortion(index, { enabled: e.target.checked })}
                  className="rounded border-gray-300 text-orange-600"
                />
                {row.label}
              </label>
              {row.enabled ? (
                <div className="grid grid-cols-2 gap-2">
                  <FormField label="Price (₹)">
                    <TextInput
                      required
                      type="number"
                      min="0"
                      step="0.01"
                      value={row.priceRupees}
                      onChange={(e) => updatePortion(index, { priceRupees: e.target.value })}
                    />
                  </FormField>
                  <FormField label="Stock qty">
                    <TextInput
                      required
                      type="number"
                      min="0"
                      value={row.stockQuantity}
                      onChange={(e) => updatePortion(index, { stockQuantity: e.target.value })}
                    />
                  </FormField>
                </div>
              ) : null}
            </div>
          ))}
        </div>

        <div className="flex flex-wrap gap-6">
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={isVeg}
              onChange={(e) => setIsVeg(e.target.checked)}
              className="rounded border-gray-300 text-orange-600"
            />
            Vegetarian item
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-gray-300 text-orange-600"
            />
            Active on menu
          </label>
        </div>
      </form>
    </Modal>
  )
}
