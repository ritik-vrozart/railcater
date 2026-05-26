import { NavLink } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { formatRole } from '../../lib/format'

const navItems = [
  { to: '/admin', label: 'Overview', icon: '◉', end: true },
  { to: '/admin/pantries', label: 'All pantries', icon: '☰' },
  { to: '/admin/orders', label: 'All orders', icon: '◎' },
  { to: '/admin/invite', label: 'Invite pantry', icon: '＋' },
]

export function AdminSidebar() {
  const { user } = useAuth()

  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-gray-200 bg-slate-900 text-white">
      <div className="shrink-0 border-b border-slate-700 px-5 py-5">
        <p className="text-xs font-bold uppercase tracking-widest text-amber-400">RailCater</p>
        <p className="mt-1 text-sm font-semibold">Department head</p>
        <p className="text-xs text-slate-400">Train catering control</p>
      </div>

      <nav className="min-h-0 flex-1 space-y-0.5 overflow-y-auto p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                isActive
                  ? 'bg-amber-500/20 text-amber-300'
                  : 'text-slate-300 hover:bg-slate-800 hover:text-white'
              }`
            }
          >
            <span className="text-base opacity-70">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>

      <div className="shrink-0 border-t border-slate-700 p-4">
        <p className="truncate text-sm font-medium">{user?.name}</p>
        <p className="truncate text-xs text-slate-400">{user?.email}</p>
        <span className="mt-2 inline-block rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-300">
          {user?.role ? formatRole(user.role) : ''}
        </span>
      </div>
    </aside>
  )
}
