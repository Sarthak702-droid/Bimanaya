import { Link } from '@tanstack/react-router'
import { Bell, Languages, Menu, X, LogOut, LogIn, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { Logo } from '../Logo'
import { ThemeToggle } from '../ThemeToggle'
import { useAuth } from '../../context/AuthContext'

export function TopNavBar() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const { user, isSignedIn, signOut, setShowLoginModal } = useAuth()
  const [profileDropdownOpen, setProfileDropdownOpen] = useState(false)

  return (
    <nav
      style={{
        position: 'sticky',
        top: 0,
        zIndex: 50,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        width: '100%',
        padding: '16px 40px',
        backgroundColor: 'rgba(6, 10, 9, 0.75)',
        borderBottom: '1px solid var(--outline-variant)',
        backdropFilter: 'blur(20px)',
      }}
    >
      {/* Left: Logo + Nav Links */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
        <Link to="/" style={{ display: 'flex', alignItems: 'center', textDecoration: 'none' }}>
          <Logo size="md" />
        </Link>

        {/* Desktop Nav */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 32,
          }}
          className="desktop-nav"
        >
          {[
            { label: 'Dashboard', to: '/dashboard' as const },
            { label: 'Claims Queue', to: (user?.role === 'REVIEWER' || user?.role === 'ADMIN' ? '/reviewer' : '/dashboard') as const },
            { label: 'Ombudsman Flow', to: '/dashboard' as const },
            { label: 'Legal Aid', to: '/dashboard' as const },
          ].map((item) => (
            <Link
              key={item.label}
              to={item.to}
              style={{
                color: 'var(--on-surface-variant)',
                textDecoration: 'none',
                fontFamily: 'var(--font-headline)',
                fontSize: 14,
                fontWeight: 500,
                letterSpacing: '0.05em',
                textTransform: 'uppercase',
                transition: 'all 0.2s ease',
              }}
              activeProps={{ style: { color: 'var(--accent)', fontWeight: 600 } }}
              onMouseEnter={(e) => {
                e.currentTarget.style.color = 'var(--accent)'
                e.currentTarget.style.textShadow = '0 0 8px var(--accent-light)'
              }}
              onMouseLeave={(e) => {
                if (window.location.pathname !== item.to) {
                  e.currentTarget.style.color = 'var(--on-surface-variant)'
                  e.currentTarget.style.textShadow = 'none'
                }
              }}
            >
              {item.label}
            </Link>
          ))}
        </div>
      </div>

      {/* Right: Actions */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <div className="desktop-nav" style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <ThemeToggle />
          <button className="btn-ghost" title="Notifications" style={{ color: 'var(--on-surface-variant)', cursor: 'pointer' }}>
            <Bell size={18} />
          </button>
          <button className="btn-ghost" title="Language" style={{ color: 'var(--on-surface-variant)', cursor: 'pointer' }}>
            <Languages size={18} />
          </button>

          {/* User auth state */}
          {isSignedIn ? (
            <div style={{ position: 'relative' }}>
              <button
                onClick={() => setProfileDropdownOpen(!profileDropdownOpen)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  background: 'rgba(234, 255, 244, 0.05)',
                  border: '1px solid var(--outline-variant)',
                  padding: '6px 12px',
                  borderRadius: 100,
                  cursor: 'pointer',
                  color: 'var(--on-surface)',
                  transition: 'all 0.2s ease',
                }}
                onMouseEnter={(e) => (e.currentTarget.style.borderColor = 'var(--accent)')}
                onMouseLeave={(e) => (e.currentTarget.style.borderColor = 'var(--outline-variant)')}
              >
                <img
                  src={user.avatarUrl}
                  alt={user.name}
                  style={{ width: 24, height: 24, borderRadius: '50%', objectFit: 'cover' }}
                />
                <span style={{ fontSize: 13, fontWeight: 600 }}>{user.name}</span>
                <span
                  style={{
                    fontSize: 9,
                    fontWeight: 800,
                    letterSpacing: '0.05em',
                    background: user.role === 'POLICYHOLDER' ? 'rgba(77, 255, 163, 0.15)' : 'rgba(255, 107, 107, 0.15)',
                    color: user.role === 'POLICYHOLDER' ? '#4dffa3' : '#ff6b6b',
                    padding: '1px 6px',
                    borderRadius: 4,
                  }}
                >
                  {user.role}
                </span>
              </button>

              {profileDropdownOpen && (
                <div
                  style={{
                    position: 'absolute',
                    top: 'calc(100% + 8px)',
                    right: 0,
                    width: 240,
                    background: 'var(--surface-container-high)',
                    border: '1px solid var(--glass-border)',
                    borderRadius: 16,
                    padding: 8,
                    boxShadow: 'var(--shadow-lg)',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 4,
                  }}
                >
                  <div style={{ padding: '8px 12px', borderBottom: '1px solid var(--outline-variant)', marginBottom: 4 }}>
                    <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--on-surface)' }}>{user.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--on-surface-variant)', wordBreak: 'break-all' }}>{user.email}</div>
                  </div>
                  <button
                    onClick={() => {
                      setProfileDropdownOpen(false)
                      setShowLoginModal(true)
                    }}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px 12px',
                      background: 'transparent',
                      border: 'none',
                      borderRadius: 8,
                      color: 'var(--on-surface)',
                      fontSize: 13,
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(77, 255, 163, 0.08)')}
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    <ShieldCheck size={16} style={{ color: 'var(--accent)' }} />
                    Switch User / Role
                  </button>
                  <button
                    onClick={() => {
                      setProfileDropdownOpen(false)
                      signOut()
                    }}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px 12px',
                      background: 'transparent',
                      border: 'none',
                      borderRadius: 8,
                      color: 'var(--error)',
                      fontSize: 13,
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(255, 107, 107, 0.08)')}
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    <LogOut size={16} />
                    Sign Out
                  </button>
                </div>
              )}
            </div>
          ) : (
            <button
              onClick={() => setShowLoginModal(true)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                background: 'transparent',
                border: '1px solid var(--accent)',
                padding: '8px 20px',
                borderRadius: 100,
                cursor: 'pointer',
                color: 'var(--accent)',
                fontSize: 13,
                fontWeight: 600,
                transition: 'all 0.2s ease',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = 'var(--accent)'
                e.currentTarget.style.color = 'var(--on-primary)'
                e.currentTarget.style.boxShadow = '0 0 14px var(--accent-light)'
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = 'transparent'
                e.currentTarget.style.color = 'var(--accent)'
                e.currentTarget.style.boxShadow = 'none'
              }}
            >
              <LogIn size={16} />
              Sign In (Clerk)
            </button>
          )}
        </div>

        <Link
          to="/dashboard/cases/new"
          className="btn btn-primary desktop-nav"
          style={{
            boxShadow: '0 0 14px var(--accent-light)',
          }}
        >
          Check My Claim
        </Link>

        {/* Mobile menu button */}
        <button
          className="btn-ghost mobile-only"
          onClick={() => setMobileOpen(!mobileOpen)}
          style={{ display: 'none', color: 'var(--on-surface)' }}
        >
          {mobileOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </div>

      {/* Mobile Nav Overlay */}
      {mobileOpen && (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            top: 64,
            backgroundColor: 'var(--background)',
            zIndex: 49,
            padding: '24px',
            display: 'flex',
            flexDirection: 'column',
            gap: 16,
          }}
        >
          {['Dashboard', 'Claims Queue', 'Ombudsman Flow', 'Legal Aid'].map((label) => (
            <Link
              key={label}
              to="/dashboard"
              onClick={() => setMobileOpen(false)}
              style={{
                color: 'var(--on-surface)',
                textDecoration: 'none',
                fontSize: 18,
                fontWeight: 500,
                padding: '12px 0',
                borderBottom: '1px solid var(--outline-variant)',
              }}
            >
              {label}
            </Link>
          ))}
          <Link
            to="/dashboard/cases/new"
            className="btn btn-primary"
            onClick={() => setMobileOpen(false)}
            style={{ marginTop: 16 }}
          >
            Check My Claim
          </Link>
        </div>
      )}

      <style>{`
        @media (max-width: 768px) {
          .desktop-nav { display: none !important; }
          .mobile-only { display: flex !important; }
        }
        @media (min-width: 769px) {
          .mobile-only { display: none !important; }
        }
      `}</style>
    </nav>
  )
}
