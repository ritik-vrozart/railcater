const variants: Record<string, string> = {
  default: 'bg-gray-100 text-gray-700',
  success: 'bg-emerald-50 text-emerald-700 ring-emerald-600/20',
  warning: 'bg-amber-50 text-amber-800 ring-amber-600/20',
  danger: 'bg-red-50 text-red-700 ring-red-600/20',
  info: 'bg-blue-50 text-blue-700 ring-blue-600/20',
  primary: 'bg-orange-50 text-orange-800 ring-orange-600/20',
}

const statusVariant: Record<string, string> = {
  pending: 'warning',
  confirmed: 'info',
  preparing: 'info',
  ready: 'primary',
  dispatched: 'primary',
  processing: 'info',
  shipped: 'primary',
  delivered: 'success',
  cancelled: 'danger',
}

export function Badge({
  children,
  variant = 'default',
}: {
  children: React.ReactNode
  variant?: keyof typeof variants
}) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${variants[variant]}`}
    >
      {children}
    </span>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const v = (statusVariant[status] ?? 'default') as keyof typeof variants
  return (
    <Badge variant={v}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </Badge>
  )
}
