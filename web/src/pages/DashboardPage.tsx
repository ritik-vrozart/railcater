import { useNavigate } from 'react-router-dom'
import { Button } from '../components/ui/Button'
import { useAuth } from '../context/AuthContext'

export function DashboardPage() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <header className="border-b border-slate-800 bg-slate-900/50 px-6 py-4">
        <div className="mx-auto flex max-w-5xl items-center justify-between">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-amber-400">RailCater</p>
            <h1 className="text-xl font-bold">Dashboard</h1>
          </div>
          <Button variant="secondary" className="!w-auto px-5" onClick={handleLogout}>
            Sign out
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-6 py-10">
        <div className="rounded-2xl border border-slate-800 bg-slate-900/60 p-8">
          <p className="text-slate-400">Signed in as</p>
          <h2 className="mt-1 text-2xl font-semibold">{user?.name}</h2>
          <p className="mt-1 text-slate-400">{user?.email}</p>
          <span className="mt-4 inline-block rounded-full bg-amber-500/15 px-3 py-1 text-xs font-medium uppercase tracking-wide text-amber-300">
            {user?.role.replace('_', ' ')}
          </span>
          <p className="mt-6 text-sm text-slate-500">
            More screens (PNR ordering, vendor panel) will be added in the next steps.
          </p>
        </div>
      </main>
    </div>
  )
}
