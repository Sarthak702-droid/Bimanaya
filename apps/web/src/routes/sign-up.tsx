import { createFileRoute } from '@tanstack/react-router'
import { SignUp } from '@clerk/clerk-react'

export const Route = createFileRoute('/sign-up')({
  component: SignUpPage,
})

function SignUpPage() {
  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#060a09',
        position: 'relative',
        overflow: 'hidden',
        padding: 24,
      }}
    >
      {/* Dynamic glowing background circles */}
      <div
        style={{
          position: 'absolute',
          width: 500,
          height: 500,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(77, 255, 163, 0.08) 0%, transparent 70%)',
          top: '20%',
          left: '10%',
          pointerEvents: 'none',
        }}
      />

      <div style={{ position: 'relative', zIndex: 1 }}>
        <SignUp
          signInUrl="/sign-in"
          forceRedirectUrl="/dashboard"
          appearance={{
            variables: {
              colorPrimary: '#4dffa3',
              colorBackground: '#0b1412',
              colorInputBackground: '#080f0d',
              colorText: '#eafff4',
              colorTextSecondary: 'rgba(234, 255, 244, 0.7)',
              colorInputText: '#eafff4',
              colorBorder: 'rgba(234, 255, 244, 0.15)',
              borderRadius: '12px',
            },
            elements: {
              card: {
                border: '1px solid rgba(234, 255, 244, 0.15)',
                boxShadow: '0 20px 40px rgba(0,0,0,0.5)',
              },
              socialButtonsBlockButton: {
                border: '1px solid rgba(234, 255, 244, 0.15)',
                background: 'rgba(234, 255, 244, 0.03)',
                '&:hover': {
                  background: 'rgba(234, 255, 244, 0.08)',
                },
              },
              formButtonPrimary: {
                background: '#4dffa3',
                color: '#060a09',
                fontWeight: 700,
                '&:hover': {
                  background: '#2eff8e',
                },
              },
            },
          }}
        />
      </div>
    </div>
  )
}
