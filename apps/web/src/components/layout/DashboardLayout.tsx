import type { ReactNode } from 'react'
import { AppSidebar } from './AppSidebar'
import { ThemeToggle } from '../ThemeToggle'
import { Bell } from 'lucide-react'

interface DashboardLayoutProps {
  children: ReactNode
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <AppSidebar />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Top utility bar */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
            gap: 8,
            padding: '8px 24px',
            borderBottom: '1px solid var(--outline-variant)',
            background: 'var(--surface-container-lowest)',
          }}
        >
          <ThemeToggle />
          <button className="btn-ghost">
            <Bell size={18} />
          </button>
        </div>
        {/* Main Content */}
        <main
          style={{
            flex: 1,
            padding: '32px 40px',
            background: 'var(--surface)',
            overflowY: 'auto',
          }}
        >
          {children}
        </main>
      </div>
    </div>
  )
}
