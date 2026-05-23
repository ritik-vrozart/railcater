import { Outlet } from 'react-router-dom'

export function AuthLayout() {
  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <div className="grid min-h-screen lg:grid-cols-2">
        <aside className="relative hidden overflow-hidden bg-gradient-to-br from-slate-900 via-slate-900 to-amber-950 lg:flex lg:flex-col lg:justify-between lg:p-12">
          <div>
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-amber-400">
              RailCater
            </p>
            <h1 className="mt-6 max-w-md text-4xl font-bold leading-tight text-white">
              Train food ordering, simplified for passengers and vendors.
            </h1>
            <p className="mt-4 max-w-sm text-slate-400">
              Order meals by PNR, track delivery at your station, and manage catering operations
              from one platform.
            </p>
          </div>
          <ul className="space-y-3 text-sm text-slate-400">
            <li className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
              PNR-based ordering at upcoming stations
            </li>
            <li className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
              Live delivery windows and train delays
            </li>
            <li className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
              Vendor menus and kitchen workflows
            </li>
          </ul>
        </aside>

        <main className="flex items-center justify-center px-6 py-12">
          <div className="w-full max-w-md">
            <div className="mb-8 lg:hidden">
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-amber-400">
                RailCater
              </p>
              <h2 className="mt-2 text-2xl font-bold">Welcome back</h2>
            </div>
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
