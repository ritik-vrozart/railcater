import { useCallback, useEffect, useState } from 'react'
import { apiRequest, ApiError } from '../lib/api'
import { useAuth } from '../context/AuthContext'

export function useApi<T>(path: string | null, deps: unknown[] = []) {
  const { token } = useAuth()
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(!!path)

  const reload = useCallback(async () => {
    if (!path || !token) return
    setLoading(true)
    setError(null)
    try {
      const result = await apiRequest<T>(path, {}, token)
      setData(result)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Request failed')
    } finally {
      setLoading(false)
    }
  }, [path, token])

  useEffect(() => {
    reload()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [reload, ...deps])

  return { data, error, loading, reload }
}
