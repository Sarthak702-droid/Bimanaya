import React, { createContext, useContext, useState, useEffect } from 'react'
import { useUser, useAuth as useClerkAuth } from '@clerk/clerk-react'

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
  isLoaded: boolean
  signOut: () => void
  showLoginModal: boolean
  setShowLoginModal: (show: boolean) => void
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null)
  const [showLoginModal, setShowLoginModal] = useState(false)
  const { user: clerkUser, isLoaded: isClerkLoaded, isSignedIn: isClerkSignedIn } = useUser()
  const { signOut: clerkSignOut } = useClerkAuth()

  useEffect(() => {
    if (!isClerkLoaded) return

    if (isClerkSignedIn && clerkUser) {
      const userRole = (clerkUser.publicMetadata?.role as UserRole) || 'POLICYHOLDER'
      setUser({
        id: clerkUser.id,
        name: clerkUser.fullName || clerkUser.username || 'Clerk User',
        email: clerkUser.primaryEmailAddress?.emailAddress || '',
        role: userRole,
        avatarUrl: clerkUser.imageUrl,
      })
      return
    }

    // No Clerk session — never treat localStorage / demo users as authenticated
    setUser(null)
    localStorage.removeItem('bn_user')
  }, [isClerkLoaded, isClerkSignedIn, clerkUser])

  const signOut = async () => {
    setUser(null)
    localStorage.removeItem('bn_user')
    if (isClerkSignedIn) {
      try {
        await clerkSignOut()
      } catch (e) {
        console.error('Error during Clerk signout:', e)
      }
    }
    window.location.href = '/'
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        isSignedIn: !!isClerkLoaded && !!isClerkSignedIn,
        isLoaded: isClerkLoaded,
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

function LoginModal({ onClose }: { onClose: () => void }) {
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
          maxWidth: 480,
          width: '100%',
          padding: '36px 40px',
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

        <p style={{ margin: '0 0 24px', fontSize: 14, color: 'var(--on-surface-variant)', lineHeight: 1.5 }}>
          Sign in with your account to access the dashboard and case tools.
        </p>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <a
            href="/sign-in"
            className="btn btn-primary"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '12px 20px',
              borderRadius: 100,
              fontSize: 14,
              fontWeight: 700,
              textDecoration: 'none',
              boxShadow: '0 0 14px var(--accent-light)',
            }}
          >
            Sign In
          </a>
          <a
            href="/sign-up"
            className="btn btn-secondary"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '12px 20px',
              borderRadius: 100,
              fontSize: 14,
              fontWeight: 700,
              textDecoration: 'none',
            }}
          >
            Create Account
          </a>
        </div>
      </div>
    </div>
  )
}
