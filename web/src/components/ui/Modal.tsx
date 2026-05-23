import { useEffect, type ReactNode } from 'react'
import { ButtonLight } from './ButtonLight'

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
}: {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children: ReactNode
  footer?: ReactNode
}) {
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button
        type="button"
        className="absolute inset-0 bg-black/40"
        aria-label="Close"
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-w-2xl rounded-xl bg-white shadow-xl">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">{title}</h2>
          {description ? <p className="mt-1 text-sm text-gray-500">{description}</p> : null}
        </div>
        <div className="max-h-[70vh] overflow-y-auto px-6 py-4">{children}</div>
        {footer ? (
          <div className="flex justify-end gap-2 border-t border-gray-200 px-6 py-4">{footer}</div>
        ) : null}
      </div>
    </div>
  )
}

export function ModalFooter({
  onCancel,
  onSubmit,
  submitLabel = 'Save',
  loading,
  formId,
}: {
  onCancel: () => void
  onSubmit?: () => void
  submitLabel?: string
  loading?: boolean
  /** Links submit button to a form outside the footer (HTML5 form attribute). */
  formId?: string
}) {
  return (
    <>
      <ButtonLight type="button" variant="secondary" onClick={onCancel} disabled={loading}>
        Cancel
      </ButtonLight>
      <ButtonLight
        type="submit"
        form={formId}
        onClick={formId ? undefined : onSubmit}
        loading={loading}
      >
        {submitLabel}
      </ButtonLight>
    </>
  )
}
