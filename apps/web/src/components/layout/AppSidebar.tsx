import { Link, useRouterState } from '@tanstack/react-router'
import {
  LayoutGrid,
  Waves,
  FileText,
  Scale,
  HelpCircle,
  Settings,
  RefreshCw,
  LogOut,
  Plus,
  Users,
  ShieldCheck,
  FileSearch,
} from 'lucide-react'
import { useAuth } from '../../context/AuthContext'

export function AppSidebar() {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname
  const { user, signOut, setShowLoginModal } = useAuth()

  const isReviewer = user?.role === 'REVIEWER' || user?.role === 'ADMIN'

  const navItems = isReviewer
    ? [
        { icon: LayoutGrid, label: 'Claims Overview', to: '/reviewer' },
        { icon: Waves, label: 'Cases Queue', to: '/reviewer/queue' },
        { icon: FileSearch, label: 'Policy Vault', to: '/dashboard/vault' },
        { icon: Users, label: 'Manage Users', to: '/admin' },
        { icon: Settings, label: 'System Settings', to: '/dashboard/settings' },
      ]
    : [
        { icon: LayoutGrid, label: 'Overview', to: '/dashboard' },
        { icon: Waves, label: 'My Claims', to: '/dashboard/claims' },
        { icon: FileText, label: 'Policy Vault', to: '/dashboard/vault' },
        { icon: Scale, label: 'Legal Aid', to: '/dashboard/legal-aid' },
        { icon: HelpCircle, label: 'Support', to: '/dashboard/support' },
        { icon: Settings, label: 'Settings', to: '/dashboard/settings' },
      ]

  const firstLetter = user?.name ? user.name.charAt(0) : 'B'

  return (
    <aside className="sidebar" style={{ background: 'var(--surface-container-low)', borderRight: '1px solid var(--outline-variant)' }}>
      {/* Header */}
      <div style={{ padding: '20px', borderBottom: '1px solid var(--outline-variant)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {user?.avatarUrl ? (
            <img
              src={user.avatarUrl}
              alt={user.name}
              style={{ width: 40, height: 40, borderRadius: '50%', objectFit: 'cover', border: '2px solid var(--accent)' }}
            />
          ) : (
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                background: 'var(--accent-light)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--accent)',
                fontWeight: 700,
                fontSize: 16,
                fontFamily: 'var(--font-headline)',
              }}
            >
              {firstLetter}
            </div>
          )}
          <div>
            <div
              style={{
                fontFamily: 'var(--font-headline)',
                fontWeight: 700,
                fontSize: 15,
                color: 'var(--on-surface)',
              }}
            >
              {user?.name || 'BimaNyaya Portal'}
            </div>
            <div style={{ fontSize: 12, color: 'var(--accent)', fontWeight: 500, display: 'flex', alignItems: 'center', gap: 4 }}>
              <ShieldCheck size={12} />
              {user?.role === 'POLICYHOLDER' ? 'Policyholder View' : (user?.role === 'REVIEWER' ? 'Reviewer View' : 'Admin View')}
            </div>
          </div>
        </div>
      </div>

      {/* New Dispute Button */}
      {!isReviewer && (
        <div style={{ padding: '16px 16px 8px' }}>
          <Link
            to="/dashboard/cases/new"
            className="btn btn-primary"
            style={{ width: '100%', justifyContent: 'center', gap: 8, boxShadow: '0 0 12px var(--accent-light)' }}
          >
            <Plus size={16} />
            New Dispute
          </Link>
        </div>
      )}

      {/* Nav Items */}
      <nav style={{ flex: 1, paddingTop: 16 }}>
        {navItems.map((item) => {
          const isActive = currentPath === item.to || 
            (item.to !== '/dashboard' && item.to !== '/reviewer' && currentPath.startsWith(item.to))
          const Icon = item.icon

          return (
            <Link
              key={item.to}
              to={item.to as any}
              className={`sidebar-item ${isActive ? 'active' : ''}`}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                padding: '12px 20px',
                color: isActive ? 'var(--accent)' : 'var(--on-surface-variant)',
                background: isActive ? 'var(--accent-light)' : 'transparent',
                textDecoration: 'none',
                fontSize: 14,
                fontWeight: 500,
                borderLeft: isActive ? '3px solid var(--accent)' : '3px solid transparent',
              }}
            >
              <Icon size={18} />
              {item.label}
            </Link>
          )
        })}
      </nav>

      {/* Footer Actions */}
      <div style={{ borderTop: '1px solid var(--outline-variant)', padding: '8px 0' }}>
        <button
          onClick={() => setShowLoginModal(true)}
          className="sidebar-item"
          style={{
            width: '100%',
            border: 'none',
            background: 'none',
            cursor: 'pointer',
            fontFamily: 'var(--font-body)',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '12px 20px',
            color: 'var(--on-surface-variant)',
            textAlign: 'left',
          }}
        >
          <RefreshCw size={18} />
          Switch Role
        </button>
        <button
          onClick={signOut}
          className="sidebar-item"
          style={{
            width: '100%',
            border: 'none',
            background: 'none',
            cursor: 'pointer',
            fontFamily: 'var(--font-body)',
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '12px 20px',
            color: 'var(--error)',
            textAlign: 'left',
          }}
        >
          <LogOut size={18} />
          Sign Out
        </button>
      </div>
    </aside>
  )
}
