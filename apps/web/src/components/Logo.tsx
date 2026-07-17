interface LogoProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeMap = {
  sm: { height: 24, maxWidth: 125 },
  md: { height: 32, maxWidth: 167 },
  lg: { height: 48, maxWidth: 250 },
}

export function Logo({ size = 'md', className = '' }: LogoProps) {
  const { height, maxWidth } = sizeMap[size]
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        height,
        maxWidth,
        cursor: 'pointer',
        userSelect: 'none',
      }}
      className={className}
    >
      <img
        src="/bimanyaya-logo.svg"
        alt="BimaNyaya"
        style={{
          height: '100%',
          width: 'auto',
          maxWidth: '100%',
          objectFit: 'contain',
          display: 'block',
        }}
      />
    </div>
  )
}
