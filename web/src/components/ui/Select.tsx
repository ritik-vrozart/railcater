import type { SelectHTMLAttributes } from 'react'

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label?: string
}

export function Select({ label, className = '', id, children, ...props }: SelectProps) {
  const selectId = id ?? props.name

  return (
    <div className="space-y-1">
      {label ? (
        <label htmlFor={selectId} className="block text-xs font-medium text-gray-600">
          {label}
        </label>
      ) : null}
      <select
        id={selectId}
        className={`rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-orange-500 focus:outline-none focus:ring-2 focus:ring-orange-500/20 ${className}`}
        {...props}
      >
        {children}
      </select>
    </div>
  )
}
