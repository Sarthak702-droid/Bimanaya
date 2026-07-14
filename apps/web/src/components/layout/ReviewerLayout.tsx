import { Link, useRouterState } from '@tanstack/react-router'
import { Bell, UserCircle, Search } from 'lucide-react'
import { ThemeToggle } from '../ThemeToggle'
import type { ReactNode } from 'react'

const reviewerNavItems = [
  { label: 'Dashboard', to: '/reviewer' },
  { label: 'Grievances', to: '/reviewer/queue' },
  { label: 'Reviews', to: '/reviewer' },
  { label: 'Experts', to: '/reviewer' },
]

export function ReviewerLayout({ children }: { children: ReactNode }) {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Nav */}
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
          <span
            style={{
              fontFamily: 'var(--font-headline)',
              fontWeight: 800,
              fontSize: 18,
              color: 'var(--on-surface)',
            }}
          >
            BimaNyaya
          </span>
          <div style={{ display: 'flex', gap: 8 }}>
            {reviewerNavItems.map((item) => {
              const isActive = currentPath === item.to
              return (
                <Link
                  key={item.label}
                  to={item.to}
                  style={{
                    textDecoration: 'none',
                    padding: '6px 12px',
                    fontSize: 14,
                    fontWeight: isActive ? 600 : 400,
                    color: isActive ? 'var(--on-surface)' : 'var(--on-surface-variant)',
                    borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                    transition: 'all 0.2s ease',
                  }}
                >
                  {item.label}
                </Link>
              )
            })}
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '6px 12px',
              background: 'var(--surface-container)',
              borderRadius: 'var(--radius-lg)',
              border: '1px solid var(--outline-variant)',
            }}
          >
            <Search size={16} style={{ color: 'var(--on-surface-variant)' }} />
            <input
              type="text"
              placeholder="Search..."
              style={{
                border: 'none',
                background: 'transparent',
                outline: 'none',
                color: 'var(--on-surface)',
                fontSize: 14,
                width: 160,
                fontFamily: 'var(--font-body)',
              }}
            />
          </div>
          <ThemeToggle />
          <button className="btn-ghost">
            <Bell size={20} />
          </button>
          <button className="btn-ghost">
            <UserCircle size={20} />
          </button>
        </div>
      </nav>

      {/* Content */}
      <main style={{ flex: 1, padding: '32px 40px', background: 'var(--surface)' }}>
        {children}
      </main>

      {/* Footer */}
      <footer
        style={{
          padding: '16px 40px',
          borderTop: '1px solid var(--outline-variant)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: 12,
          color: 'var(--on-surface-variant)',
          background: 'var(--surface-container-lowest)',
        }}
      >
        <span>© 2024 BimaNyaya Insurance Systems. Evidence-Based Design Protocol.</span>
        <div style={{ display: 'flex', gap: 24 }}>
          <a href="#" style={{ color: 'var(--on-surface-variant)', textDecoration: 'none' }}>
            Legal Compliance
          </a>
          <a href="#" style={{ color: 'var(--on-surface-variant)', textDecoration: 'none' }}>
            Data Privacy
          </a>
          <a href="#" style={{ color: 'var(--on-surface-variant)', textDecoration: 'none' }}>
            Audit Logs
          </a>
        </div>
      </footer>
    </div>
  )
}
