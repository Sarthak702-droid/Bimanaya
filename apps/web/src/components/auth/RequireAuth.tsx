import { useAuth, useUser } from '@clerk/clerk-react'
import { Navigate } from '@tanstack/react-router'
import type { UserRole } from '../../context/AuthContext'

interface RequireAuthProps {
  children: React.ReactNode
  /** If set, user must have one of these roles in Clerk publicMetadata.role */
  roles?: UserRole[]
}

export function RequireAuth({ children, roles }: RequireAuthProps) {
  const { isLoaded, isSignedIn } = useAuth()
  const { user } = useUser()

  if (!isLoaded) {
    return (
      <div
        style={{
          minHeight: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#060a09',
          color: '#eafff4',
          fontFamily: 'var(--font-body, system-ui)',
        }}
      >
        Checking session…
      </div>
    )
  }

  if (!isSignedIn) {
    return <Navigate to="/sign-in" replace />
  }

  if (roles && roles.length > 0) {
    const role = (user?.publicMetadata?.role as UserRole | undefined) || 'POLICYHOLDER'
    if (!roles.includes(role)) {
      return <Navigate to="/dashboard" replace />
    }
  }

  return <>{children}</>
}
