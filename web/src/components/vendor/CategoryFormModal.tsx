import { useEffect, useState, type FormEvent } from 'react'
import { apiRequest, ApiError } from '../../lib/api'
import { useAuth } from '../../context/AuthContext'
import { useVendor } from '../../context/VendorContext'
import type { FoodType, MenuCategory } from '../../types/api'
import { Alert } from '../ui/Alert'
import { FormField, TextArea, TextInput } from '../ui/FormField'
import { Modal, ModalFooter } from '../ui/Modal'
import { Select } from '../ui/Select'

const FORM_ID = 'category-form'

export function CategoryFormModal({
  open,
  onClose,
  onSaved,
  category,
}: {
  open: boolean
  onClose: () => void
  onSaved: () => void
  category?: MenuCategory | null
}) {
  const { token } = useAuth()
  const { vendorId, loading: vendorLoading } = useVendor()
  const isEdit = !!category

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [foodType, setFoodType] = useState<FoodType>('veg')
  const [sortOrder, setSortOrder] = useState('0')
  const [isActive, setIsActive] = useState(true)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) return
    if (category) {
      setName(category.name)
      setDescription(category.description ?? '')
      setFoodType(category.food_type)
      setSortOrder(String(category.sort_order))
      setIsActive(category.is_active)
    } else {
      setName('')
      setDescription('')
      setFoodType('veg')
      setSortOrder('0')
      setIsActive(true)
    }
    setError('')
  }, [open, category])

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

    setLoading(true)

    const body = {
      name: name.trim(),
      description: description.trim() || null,
      food_type: foodType,
      sort_order: parseInt(sortOrder, 10) || 0,
      ...(isEdit ? { is_active: isActive } : {}),
    }

    try {
      if (isEdit && category) {
        await apiRequest(`/api/v1/vendors/${vendorId}/menu/categories/${category.id}`, {
          method: 'PUT',
          body: JSON.stringify(body),
        }, token)
      } else {
        await apiRequest(`/api/v1/vendors/${vendorId}/menu/categories`, {
          method: 'POST',
          body: JSON.stringify(body),
        }, token)
      }
      onSaved()
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to save category')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit category' : 'Add category'}
      description="Category master — Meals, Snacks, Beverages, etc."
      footer={
        <ModalFooter
          formId={FORM_ID}
          onCancel={onClose}
          submitLabel={isEdit ? 'Update' : 'Create'}
          loading={loading || vendorLoading}
        />
      }
    >
      <form id={FORM_ID} onSubmit={handleSubmit} className="space-y-4">
        {error ? <Alert message={error} /> : null}
        {!vendorId && !vendorLoading ? (
          <Alert message="Vendor account not ready. Refresh the page or sign in again." />
        ) : null}

        <FormField label="Category name">
          <TextInput
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Meals"
            disabled={!vendorId}
          />
        </FormField>

        <FormField label="Description">
          <TextArea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional"
            disabled={!vendorId}
          />
        </FormField>

        <Select
          label="Food type"
          value={foodType}
          onChange={(e) => setFoodType(e.target.value as FoodType)}
          disabled={!vendorId}
        >
          <option value="veg">Vegetarian</option>
          <option value="non_veg">Non-vegetarian</option>
        </Select>

        <FormField label="Sort order" hint="Lower numbers appear first">
          <TextInput
            type="number"
            value={sortOrder}
            onChange={(e) => setSortOrder(e.target.value)}
            disabled={!vendorId}
          />
        </FormField>

        {isEdit ? (
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-gray-300 text-orange-600"
            />
            Active
          </label>
        ) : null}
      </form>
    </Modal>
  )
}
