export function formatMoney(cents: number): string {
  return new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency: 'INR',
    maximumFractionDigits: 0,
  }).format(cents / 100)
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('en-IN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(iso))
}

export function formatRole(role: string): string {
  const labels: Record<string, string> = {
    super_admin: 'Department head',
    vendor_admin: 'Pantry manager',
    passenger: 'Passenger',
  }
  return labels[role] ?? role.replace(/_/g, ' ')
}

export function formatStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1)
}
