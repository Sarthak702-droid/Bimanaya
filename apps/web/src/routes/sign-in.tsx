import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useSignIn } from '@clerk/clerk-react'
import { Logo } from '../components/Logo'
import { User, Lock, ArrowRight, AlertCircle } from 'lucide-react'

export const Route = createFileRoute('/sign-in')({
  component: SignInPage,
})

function SignInPage() {
  const { isLoaded, signIn, setActive } = useSignIn()
  const [identifier, setIdentifier] = useState('') // Can be Username or Email
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isLoaded) return

    setLoading(true)
    setError('')

    try {
      const result = await signIn.create({
        identifier, // Uses Username or Email directly
        password,
      })

      if (result.status === 'complete') {
        await setActive({ session: result.createdSessionId })
        navigate({ to: '/dashboard' })
      } else {
        setError('Extra verification steps required by Clerk server.')
      }
    } catch (err: any) {
      console.error('Error during sign-in:', err)
      setError(err.errors?.[0]?.message || 'Invalid username/email or password.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#060a09',
        position: 'relative',
        overflow: 'hidden',
        padding: 24,
      }}
    >
      {/* Glowing background */}
      <div
        style={{
          position: 'absolute',
          width: 500,
          height: 500,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(77, 255, 163, 0.06) 0%, transparent 70%)',
          top: '15%',
          left: '5%',
          pointerEvents: 'none',
        }}
      />

      <div style={{ position: 'relative', zIndex: 1, width: '100%', maxWidth: 420 }}>
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{ display: 'inline-block', marginBottom: 16 }}>
            <Logo size="lg" />
          </div>
          <h2 style={{ fontFamily: 'var(--font-headline)', fontSize: 24, fontWeight: 700, color: 'var(--on-surface)', margin: '0 0 8px' }}>
            Welcome Back
          </h2>
          <p style={{ color: 'var(--on-surface-variant)', fontSize: 14, margin: 0 }}>
            Sign in using your Username or Email
          </p>
        </div>

        {/* Card */}
        <div
          className="glass-card"
          style={{
            padding: '36px 40px',
            background: 'var(--surface-container-low)',
            border: '1px solid var(--glass-border)',
            borderRadius: 24,
            boxShadow: '0 20px 40px rgba(0,0,0,0.5)',
          }}
        >
          {error && (
            <div
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 10,
                padding: '12px 16px',
                background: 'rgba(255, 107, 107, 0.1)',
                border: '1px solid rgba(255, 107, 107, 0.2)',
                borderRadius: 12,
                color: '#ff8585',
                fontSize: 13,
                marginBottom: 24,
              }}
            >
              <AlertCircle size={18} style={{ flexShrink: 0, marginTop: 1 }} />
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 8, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                Username or Email
              </label>
              <div style={{ position: 'relative' }}>
                <User
                  size={16}
                  style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                />
                <input
                  type="text"
                  required
                  placeholder="Enter Username or Email"
                  value={identifier}
                  onChange={(e) => setIdentifier(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '12px 16px 12px 42px',
                    borderRadius: 12,
                    border: '1px solid var(--outline-variant)',
                    background: 'var(--surface-container)',
                    color: 'var(--on-surface)',
                    fontSize: 14,
                    transition: 'border-color 0.2s',
                  }}
                  onFocus={(e) => (e.target.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.target.style.borderColor = 'var(--outline-variant)')}
                />
              </div>
            </div>

            <div>
              <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 8, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                Password
              </label>
              <div style={{ position: 'relative' }}>
                <Lock
                  size={16}
                  style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                />
                <input
                  type="password"
                  required
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '12px 16px 12px 42px',
                    borderRadius: 12,
                    border: '1px solid var(--outline-variant)',
                    background: 'var(--surface-container)',
                    color: 'var(--on-surface)',
                    fontSize: 14,
                    transition: 'border-color 0.2s',
                  }}
                  onFocus={(e) => (e.target.style.borderColor = 'var(--accent)')}
                  onBlur={(e) => (e.target.style.borderColor = 'var(--outline-variant)')}
                />
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="btn btn-primary"
              style={{
                width: '100%',
                justifyContent: 'center',
                padding: '14px',
                borderRadius: 100,
                fontSize: 14,
                fontWeight: 700,
                letterSpacing: '0.05em',
                textTransform: 'uppercase',
                marginTop: 8,
                boxShadow: '0 0 16px var(--accent-light)',
                opacity: loading ? 0.7 : 1,
                cursor: loading ? 'not-allowed' : 'pointer',
              }}
            >
              {loading ? 'Signing In...' : 'Sign In'}
              {!loading && <ArrowRight size={16} />}
            </button>
          </form>

          <div style={{ textAlign: 'center', marginTop: 24, fontSize: 13, color: 'var(--on-surface-variant)' }}>
            Don't have an account?{' '}
            <Link to="/sign-up" style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: 600 }}>
              Sign Up
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
