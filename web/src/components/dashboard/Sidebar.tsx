import { NavLink } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { useVendor } from '../../context/VendorContext'
import { formatRole } from '../../lib/format'

const navItems = [
  { to: '/vendor', label: 'Overview', icon: '◉', end: true },
  { to: '/vendor/orders', label: 'Orders', icon: '☰' },
  { to: '/vendor/categories', label: 'Categories', icon: '▦' },
  { to: '/vendor/menu', label: 'Menu', icon: '◎' },
  { to: '/vendor/stations', label: 'Stations', icon: '▣' },
]

export function Sidebar() {
  const { user } = useAuth()
  const { vendor } = useVendor()

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-gray-200 bg-white">
      <div className="border-b border-gray-200 px-5 py-5">
        <p className="text-xs font-bold uppercase tracking-widest text-orange-600">RailCater</p>
        {/* <p className="mt-1 truncate text-sm font-semibold text-gray-900">{vendor?.name ?? 'Vendor panel'}</p> */}
        <p className="truncate text-xs text-gray-500">{vendor?.code}</p>
      </div>

      <nav className="flex-1 space-y-0.5 p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                isActive
                  ? 'bg-orange-50 text-orange-700'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
              }`
            }
          >
            <span className="text-base opacity-70">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-gray-200 p-4">
        <p className="truncate text-sm font-medium text-gray-900">{user?.name}</p>
        <p className="truncate text-xs text-gray-500">{user?.email}</p>
        <span className="mt-2 inline-block rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-gray-600">
          {user?.role ? formatRole(user.role) : ''}
        </span>
      </div>
    </aside>
  )
}
