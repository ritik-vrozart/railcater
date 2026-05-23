export function Alert({
  message,
  variant = 'light',
}: {
  message: string
  variant?: 'light' | 'dark'
}) {
  const styles =
    variant === 'dark'
      ? 'border-red-500/40 bg-red-500/10 text-red-200'
      : 'border-red-200 bg-red-50 text-red-700'
  return (
    <div className={`rounded-lg border px-4 py-3 text-sm ${styles}`}>{message}</div>
  )
}
