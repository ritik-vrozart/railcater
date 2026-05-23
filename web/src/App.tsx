import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthLayout } from './components/AuthLayout'
import { DashboardLayout } from './components/dashboard/DashboardLayout'
import { ProtectedRoute } from './components/ProtectedRoute'
import { AuthProvider, useAuth } from './context/AuthContext'
import { VendorProvider } from './context/VendorContext'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { VendorCategoriesPage } from './pages/vendor/VendorCategoriesPage'
import { VendorMenuPage } from './pages/vendor/VendorMenuPage'
import { VendorOrderDetailPage } from './pages/vendor/VendorOrderDetailPage'
import { VendorOrdersPage } from './pages/vendor/VendorOrdersPage'
import { VendorOverviewPage } from './pages/vendor/VendorOverviewPage'
import { VendorStationsPage } from './pages/vendor/VendorStationsPage'

function GuestOnly({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-300">
        Loading…
      </div>
    )
  }
  if (user) return <Navigate to="/" replace />
  return children
}

function HomeRedirect() {
  const { user, isLoading } = useAuth()
  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-50 text-gray-500">
        Loading…
      </div>
    )
  }
  if (user?.role === 'vendor_admin') {
    return <Navigate to="/vendor" replace />
  }
  return <DashboardPage />
}

function VendorRoutes() {
  return (
    <VendorProvider>
      <Routes>
        <Route element={<DashboardLayout />}>
          <Route index element={<VendorOverviewPage />} />
          <Route path="orders" element={<VendorOrdersPage />} />
          <Route path="orders/:orderId" element={<VendorOrderDetailPage />} />
          <Route path="categories" element={<VendorCategoriesPage />} />
          <Route path="menu" element={<VendorMenuPage />} />
          <Route path="stations" element={<VendorStationsPage />} />
        </Route>
      </Routes>
    </VendorProvider>
  )
}

function AppRoutes() {
  return (
    <Routes>
      <Route
        element={
          <GuestOnly>
            <AuthLayout />
          </GuestOnly>
        }
      >
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>

      <Route
        path="/"
        element={
          <ProtectedRoute>
            <HomeRedirect />
          </ProtectedRoute>
        }
      />

      <Route
        path="/vendor/*"
        element={
          <ProtectedRoute>
            <VendorRoutes />
          </ProtectedRoute>
        }
      />

      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  )
}
