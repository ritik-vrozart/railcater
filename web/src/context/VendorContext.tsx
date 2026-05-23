import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { apiRequest } from '../lib/api'
import { useAuth } from './AuthContext'
import type { Vendor } from '../types/api'

interface VendorContextValue {
  vendor: Vendor | null
  vendorId: string | null
  loading: boolean
  error: string | null
}

const VendorContext = createContext<VendorContextValue | null>(null)

const DEMO_VENDOR_ID = 'c3000001-0000-4000-8000-000000000001'

export function VendorProvider({ children }: { children: ReactNode }) {
  const { user, token } = useAuth()
  const [vendor, setVendor] = useState<Vendor | null>(null)
  const [resolvedVendorId, setResolvedVendorId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const vendorId = user?.vendor_id ?? resolvedVendorId

  useEffect(() => {
    if (!token || user?.role !== 'vendor_admin') {
      setVendor(null)
      setResolvedVendorId(null)
      setError(null)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)

    async function load() {
      let id = user?.vendor_id ?? null

      if (!id) {
        try {
          const res = await apiRequest<{ data: Vendor[] }>('/api/v1/vendors', {}, token!)
          id = res.data[0]?.id ?? DEMO_VENDOR_ID
          if (!cancelled) setResolvedVendorId(id)
        } catch {
          id = DEMO_VENDOR_ID
          if (!cancelled) setResolvedVendorId(id)
        }
      }

      if (!id) {
        if (!cancelled) {
          setError('No vendor linked to your account. Contact support.')
          setVendor(null)
        }
        return
      }

      try {
        const v = await apiRequest<Vendor>(`/api/v1/vendors/${id}`, {}, token!)
        if (!cancelled) {
          setVendor(v)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load vendor')
          setVendor(null)
        }
      }
    }

    load().finally(() => {
      if (!cancelled) setLoading(false)
    })

    return () => {
      cancelled = true
    }
  }, [token, user?.role, user?.vendor_id])

  const value = useMemo(
    () => ({ vendor, vendorId, loading, error }),
    [vendor, vendorId, loading, error],
  )

  return <VendorContext.Provider value={value}>{children}</VendorContext.Provider>
}

export function useVendor() {
  const ctx = useContext(VendorContext)
  if (!ctx) throw new Error('useVendor must be used within VendorProvider')
  return ctx
}
