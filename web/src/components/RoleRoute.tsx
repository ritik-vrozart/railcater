import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import type { UserRole } from '../types/auth'

export function RoleRoute({
  children,
  roles,
  fallback = '/',
}: {
  children: React.ReactNode
  roles: UserRole[]
  fallback?: string
}) {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 text-gray-500">
        Loading…
      </div>
    )
  }

  if (!user || !roles.includes(user.role)) {
    return <Navigate to={fallback} replace />
  }

  return children
}
