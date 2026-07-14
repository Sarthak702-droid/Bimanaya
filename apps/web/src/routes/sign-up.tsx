import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useSignUp } from '@clerk/clerk-react'
import { Logo } from '../components/Logo'
import { User, Mail, Lock, ArrowRight, AlertCircle, KeyRound } from 'lucide-react'

export const Route = createFileRoute('/sign-up')({
  component: SignUpPage,
})

function SignUpPage() {
  const { isLoaded, signUp, setActive } = useSignUp()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fullName, setFullName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  
  // Verification step state
  const [pendingVerification, setPendingVerification] = useState(false)
  const [code, setCode] = useState('')
  
  const navigate = useNavigate()

  const handleSignUpSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isLoaded) return

    setLoading(true)
    setError('')

    const nameParts = fullName.split(' ')
    const firstName = nameParts[0] || ''
    const lastName = nameParts.slice(1).join(' ') || ''

    try {
      // Create user using Username, Email, and Password (absolutely no phone number!)
      await signUp.create({
        username,
        emailAddress: email,
        password,
        firstName,
        lastName,
      })

      // Send verification code to email
      await signUp.prepareEmailAddressVerification({
        strategy: 'email_code',
      })

      setPendingVerification(true)
    } catch (err: any) {
      console.error('Error during sign-up:', err)
      setError(err.errors?.[0]?.message || 'Registration failed. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const handleVerifySubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isLoaded) return

    setLoading(true)
    setError('')

    try {
      const completeSignUp = await signUp.attemptEmailAddressVerification({
        code,
      })

      if (completeSignUp.status === 'complete') {
        await setActive({ session: completeSignUp.createdSessionId })
        navigate({ to: '/dashboard' })
      } else {
        setError('Verification failed. Please double check the code.')
      }
    } catch (err: any) {
      console.error('Error during verification:', err)
      setError(err.errors?.[0]?.message || 'Invalid code. Please try again.')
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
            {pendingVerification ? 'Verify Your Email' : 'Create Account'}
          </h2>
          <p style={{ color: 'var(--on-surface-variant)', fontSize: 14, margin: 0 }}>
            {pendingVerification ? `Enter the code sent to ${email}` : 'Sign up using Username and Email only'}
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

          {!pendingVerification ? (
            /* ACCOUNT CREATION */
            <form onSubmit={handleSignUpSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div>
                <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 6, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Full Name
                </label>
                <div style={{ position: 'relative' }}>
                  <User
                    size={16}
                    style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                  />
                  <input
                    type="text"
                    required
                    placeholder="e.g. Rajesh Kumar"
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '10px 16px 10px 42px',
                      borderRadius: 10,
                      border: '1px solid var(--outline-variant)',
                      background: 'var(--surface-container)',
                      color: 'var(--on-surface)',
                      fontSize: 14,
                    }}
                  />
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 6, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Username
                </label>
                <div style={{ position: 'relative' }}>
                  <User
                    size={16}
                    style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                  />
                  <input
                    type="text"
                    required
                    placeholder="e.g. rajesh123"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '10px 16px 10px 42px',
                      borderRadius: 10,
                      border: '1px solid var(--outline-variant)',
                      background: 'var(--surface-container)',
                      color: 'var(--on-surface)',
                      fontSize: 14,
                    }}
                  />
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 6, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Email Address
                </label>
                <div style={{ position: 'relative' }}>
                  <Mail
                    size={16}
                    style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                  />
                  <input
                    type="email"
                    required
                    placeholder="name@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '10px 16px 10px 42px',
                      borderRadius: 10,
                      border: '1px solid var(--outline-variant)',
                      background: 'var(--surface-container)',
                      color: 'var(--on-surface)',
                      fontSize: 14,
                    }}
                  />
                </div>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 6, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
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
                    placeholder="Min. 8 characters"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '10px 16px 10px 42px',
                      borderRadius: 10,
                      border: '1px solid var(--outline-variant)',
                      background: 'var(--surface-container)',
                      color: 'var(--on-surface)',
                      fontSize: 14,
                    }}
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
                }}
              >
                {loading ? 'Creating...' : 'Sign Up'}
                {!loading && <ArrowRight size={16} />}
              </button>
            </form>
          ) : (
            /* VERIFICATION STEP */
            <form onSubmit={handleVerifySubmit} style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
              <div>
                <label style={{ display: 'block', fontSize: 12, fontWeight: 600, color: 'var(--on-surface)', marginBottom: 8, letterSpacing: '0.05em', textTransform: 'uppercase' }}>
                  Verification Code (OTP)
                </label>
                <div style={{ position: 'relative' }}>
                  <KeyRound
                    size={16}
                    style={{ position: 'absolute', left: 14, top: '50%', transform: 'translateY(-50%)', color: 'var(--on-surface-variant)' }}
                  />
                  <input
                    type="text"
                    required
                    placeholder="Enter 6-digit code"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    style={{
                      width: '100%',
                      padding: '12px 16px 12px 42px',
                      borderRadius: 12,
                      border: '1px solid var(--outline-variant)',
                      background: 'var(--surface-container)',
                      color: 'var(--on-surface)',
                      fontSize: 14,
                      textAlign: 'center',
                      letterSpacing: '0.2em',
                    }}
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
                  boxShadow: '0 0 16px var(--accent-light)',
                  opacity: loading ? 0.7 : 1,
                }}
              >
                {loading ? 'Verifying...' : 'Verify & Log In'}
                {!loading && <ArrowRight size={16} />}
              </button>
            </form>
          )}

          <div style={{ textAlign: 'center', marginTop: 24, fontSize: 13, color: 'var(--on-surface-variant)' }}>
            Already have an account?{' '}
            <Link to="/sign-in" style={{ color: 'var(--accent)', textDecoration: 'none', fontWeight: 600 }}>
              Sign In
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
