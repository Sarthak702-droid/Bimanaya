import { createFileRoute, Outlet, Link, useRouterState } from '@tanstack/react-router'
import { ThemeToggle } from '../components/ThemeToggle'
import { Bell, UserCircle, BarChart3, Heart, Users } from 'lucide-react'

export const Route = createFileRoute('/admin')({
  component: AdminLayout,
})

const adminNavItems = [
  { label: 'Overview', to: '/admin', icon: BarChart3 },
  { label: 'System Health', to: '/admin/health', icon: Heart },
  { label: 'Users', to: '/admin/users', icon: Users },
]

function AdminLayout() {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <nav
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '12px 24px',
          borderBottom: '1px solid var(--outline-variant)',
          background: 'var(--surface-container-lowest)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
          <span style={{ fontFamily: 'var(--font-headline)', fontWeight: 800, fontSize: 18, color: 'var(--on-surface)' }}>
            BimaNyaya Admin
          </span>
          <div style={{ display: 'flex', gap: 4 }}>
            {adminNavItems.map((item) => {
              const isActive = currentPath === item.to
              const Icon = item.icon
              return (
                <Link
                  key={item.label}
                  to={item.to}
                  style={{
                    textDecoration: 'none',
                    padding: '6px 14px',
                    fontSize: 14,
                    fontWeight: isActive ? 600 : 400,
                    color: isActive ? 'var(--on-surface)' : 'var(--on-surface-variant)',
                    borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                  }}
                >
                  <Icon size={16} />
                  {item.label}
                </Link>
              )
            })}
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ThemeToggle />
          <button className="btn-ghost"><Bell size={20} /></button>
          <button className="btn-ghost"><UserCircle size={20} /></button>
        </div>
      </nav>
      <main style={{ flex: 1, padding: '32px 40px', background: 'var(--surface)' }}>
        <Outlet />
      </main>
    </div>
  )
}
