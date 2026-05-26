import { Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { ButtonLight } from '../ui/ButtonLight'
import { AdminSidebar } from './AdminSidebar'

export function AdminLayout() {
  const { logout } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50">
      <AdminSidebar />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <header className="z-10 flex h-14 shrink-0 items-center justify-between border-b border-gray-200 bg-white px-6">
          <p className="text-sm text-gray-500">Super admin — train department</p>
          <ButtonLight variant="secondary" size="sm" onClick={handleLogout}>
            Sign out
          </ButtonLight>
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
