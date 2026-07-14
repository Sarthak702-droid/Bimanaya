
interface LogoProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeMap = {
  sm: { height: 24 },
  md: { height: 32 },
  lg: { height: 48 },
}

export function Logo({ size = 'md', className = '' }: LogoProps) {
  const height = sizeMap[size].height
  return (
    <div 
      style={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 12,
        cursor: 'pointer',
        userSelect: 'none'
      }} 
      className={className}
    >
      <img
        src="/logo.png"
        alt="BimaNyaya Logo"
        style={{
          height: height,
          width: 'auto',
          objectFit: 'contain',
        }}
      />
      <span style={{
        fontFamily: 'var(--font-headline)',
        fontWeight: 300,
        fontSize: height * 0.6,
        color: 'var(--on-surface)',
        letterSpacing: '0.08em',
        textTransform: 'uppercase',
        lineHeight: 1,
      }}>
        BIMA<span style={{ fontWeight: 800, color: 'var(--accent)' }}>NYAYA</span>
      </span>
    </div>
  )
}
