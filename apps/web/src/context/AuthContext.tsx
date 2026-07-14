import React, { createContext, useContext, useState, useEffect } from 'react'

export type UserRole = 'POLICYHOLDER' | 'REVIEWER' | 'ADMIN'

export interface UserProfile {
  id: string
  name: string
  email: string
  role: UserRole
  avatarUrl?: string
}

interface AuthContextType {
  user: UserProfile | null
  isSignedIn: boolean
  signIn: (role: UserRole, customEmail?: string, customName?: string) => void
  signOut: () => void
  showLoginModal: boolean
  setShowLoginModal: (show: boolean) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

const DEFAULT_USERS: Record<UserRole, UserProfile> = {
  POLICYHOLDER: {
    id: 'user_policyholder_rajesh',
    name: 'Rajesh Kumar',
    email: 'rajesh.kumar@gmail.com',
    role: 'POLICYHOLDER',
    avatarUrl: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?auto=format&fit=crop&q=80&w=100',
  },
  REVIEWER: {
    id: 'user_reviewer_amit',
    name: 'Amit Sharma',
    email: 'amit.reviewer@bimanyaya.in',
    role: 'REVIEWER',
    avatarUrl: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?auto=format&fit=crop&q=80&w=100',
  },
  ADMIN: {
    id: 'user_admin_siddharth',
    name: 'Siddharth Mehta',
    email: 'siddharth.admin@bimanyaya.in',
    role: 'ADMIN',
    avatarUrl: 'https://images.unsplash.com/photo-1519085360753-af0119f7cbe7?auto=format&fit=crop&q=80&w=100',
  },
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [showLoginModal, setShowLoginModal] = useState(false)

  // Load user from localStorage on mount
  useEffect(() => {
    const savedUser = localStorage.getItem('bn_user')
    if (savedUser) {
      try {
        setUser(JSON.parse(savedUser))
      } catch (e) {
        localStorage.removeItem('bn_user')
      }
    } else {
      // Default auto-login as Policyholder for convenience, but can switch
      const defaultUser = DEFAULT_USERS.POLICYHOLDER
      setUser(defaultUser)
      localStorage.setItem('bn_user', JSON.stringify(defaultUser))
    }
  }, [])

  const signIn = (role: UserRole, customEmail?: string, customName?: string) => {
    const profile: UserProfile = {
      id: `user_${role.toLowerCase()}_${Date.now()}`,
      name: customName || DEFAULT_USERS[role].name,
      email: customEmail || DEFAULT_USERS[role].email,
      role: role,
      avatarUrl: DEFAULT_USERS[role].avatarUrl,
    }
    setUser(profile)
    localStorage.setItem('bn_user', JSON.stringify(profile))
    setShowLoginModal(false)
  }

  const signOut = () => {
    setUser(null)
    localStorage.removeItem('bn_user')
    window.location.href = '/'
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        isSignedIn: !!user,
        signIn,
        signOut,
        showLoginModal,
        setShowLoginModal,
      }}
    >
      {children}
      {showLoginModal && <LoginModal onClose={() => setShowLoginModal(false)} />}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

/* ── Login Modal Component ── */
function LoginModal({ onClose }: { onClose: () => void }) {
  const { signIn } = useAuth()
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState<UserRole>('POLICYHOLDER')

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        backgroundColor: 'rgba(5, 8, 7, 0.85)',
        backdropFilter: 'blur(12px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
      }}
    >
      <div
        className="glass-card animate-fade-in"
        style={{
          maxWidth: 460,
          width: '100%',
          padding: '32px 40px',
          background: 'var(--surface-container-low)',
          border: '1px solid var(--glass-border)',
          borderRadius: 24,
          boxShadow: '0 20px 40px rgba(0,0,0,0.5), 0 0 40px rgba(77, 255, 163, 0.05)',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h3 style={{ margin: 0, fontFamily: 'var(--font-headline)', fontSize: 22, fontWeight: 700, color: 'var(--on-surface)' }}>
            Sign In to <span style={{ color: 'var(--accent)' }}>BimaNyaya</span>
          </h3>
          <button
            onClick={onClose}
            style={{
              background: 'transparent',
              border: 'none',
              color: 'var(--on-surface-variant)',
              fontSize: 20,
              cursor: 'pointer',
            }}
          >
            ✕
          </button>
        </div>

        <p style={{ color: 'var(--on-surface-variant)', fontSize: 14, marginBottom: 24, lineHeight: 1.5 }}>
          Simulating Clerk secure authentication. Select a workspace profile below to test specific role capabilities.
        </p>

        {/* Profile Selector cards */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 24 }}>
          {(['POLICYHOLDER', 'REVIEWER', 'ADMIN'] as UserRole[]).map((r) => {
            const defUser = DEFAULT_USERS[r]
            return (
              <button
                key={r}
                onClick={() => signIn(r)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 16,
                  width: '100%',
                  padding: 16,
                  background: 'var(--surface-container)',
                  border: '1px solid var(--outline-variant)',
                  borderRadius: 12,
                  cursor: 'pointer',
                  textAlign: 'left',
                  transition: 'all 0.2s ease',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.borderColor = 'var(--accent)'
                  e.currentTarget.style.transform = 'translateY(-2px)'
                  e.currentTarget.style.boxShadow = '0 4px 12px rgba(77, 255, 163, 0.1)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.borderColor = 'var(--outline-variant)'
                  e.currentTarget.style.transform = 'none'
                  e.currentTarget.style.boxShadow = 'none'
                }}
              >
                <img
                  src={defUser.avatarUrl}
                  alt={defUser.name}
                  style={{ width: 40, height: 40, borderRadius: '50%', objectFit: 'cover', border: '2px solid var(--outline-variant)' }}
                />
                <div>
                  <div style={{ fontWeight: 600, fontSize: 15, color: 'var(--on-surface)' }}>{defUser.name}</div>
                  <div style={{ fontSize: 12, color: 'var(--on-surface-variant)' }}>{defUser.email}</div>
                  <span
                    style={{
                      display: 'inline-block',
                      marginTop: 4,
                      fontSize: 10,
                      fontWeight: 700,
                      letterSpacing: '0.05em',
                      padding: '2px 8px',
                      borderRadius: 4,
                      background: r === 'POLICYHOLDER' ? 'rgba(77, 255, 163, 0.15)' : (r === 'REVIEWER' ? 'rgba(102, 224, 255, 0.15)' : 'rgba(255, 107, 107, 0.15)'),
                      color: r === 'POLICYHOLDER' ? '#4dffa3' : (r === 'REVIEWER' ? '#66e0ff' : '#ff6b6b'),
                    }}
                  >
                    {r}
                  </span>
                </div>
              </button>
            )
          })}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', margin: '20px 0', color: 'var(--on-surface-variant)', fontSize: 12 }}>
          <div style={{ flex: 1, height: 1, background: 'var(--outline-variant)' }} />
          <span style={{ padding: '0 12px' }}>OR CUSTOM EMAIL</span>
          <div style={{ flex: 1, height: 1, background: 'var(--outline-variant)' }} />
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginBottom: 20 }}>
          <input
            type="text"
            className="input"
            placeholder="Full Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            style={{ width: '100%', padding: '10px 14px', borderRadius: 8, border: '1px solid var(--outline-variant)', background: 'var(--surface-container)', color: 'var(--on-surface)' }}
          />
          <input
            type="email"
            className="input"
            placeholder="Email Address"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            style={{ width: '100%', padding: '10px 14px', borderRadius: 8, border: '1px solid var(--outline-variant)', background: 'var(--surface-container)', color: 'var(--on-surface)' }}
          />
          <select
            className="input"
            value={role}
            onChange={(e) => setRole(e.target.value as UserRole)}
            style={{ width: '100%', padding: '10px 14px', borderRadius: 8, border: '1px solid var(--outline-variant)', background: 'var(--surface-container)', color: 'var(--on-surface)' }}
          >
            <option value="POLICYHOLDER">Policyholder Workspace</option>
            <option value="REVIEWER">Reviewer Workspace</option>
            <option value="ADMIN">System Administrator</option>
          </select>
        </div>

        <button
          onClick={() => {
            if (email) signIn(role, email, name || undefined)
          }}
          disabled={!email}
          className="btn btn-primary"
          style={{ width: '100%', justifyContent: 'center', opacity: email ? 1 : 0.6 }}
        >
          Sign In Custom Profile
        </button>
      </div>
    </div>
  )
}
