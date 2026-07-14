import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
import { ThemeProvider } from '../context/ThemeContext'
import { AuthProvider } from '../context/AuthContext'
import { ClerkProvider } from '@clerk/clerk-react'

const publishableKey = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY || 'pk_test_YWxlcnQtZ2hvc3QtNy5jbGVyay5hY2NvdW50cy5kZXYk'

import appCss from '../styles.css?url'

export const Route = createRootRoute({
  head: () => ({
    meta: [
      {
        charSet: 'utf-8',
      },
      {
        name: 'viewport',
        content: 'width=device-width, initial-scale=1',
      },
      {
        title: 'BimaNyaya — Insurance Dispute Resolution',
      },
      {
        name: 'description',
        content:
          'BimaNyaya leverages advanced AI and legal expertise to streamline complex insurance disputes, delivering definitive outcomes with mathematical precision.',
      },
    ],
    links: [
      {
        rel: 'stylesheet',
        href: appCss,
      },
      {
        rel: 'icon',
        href: '/logo.png',
        type: 'image/png',
      },
    ],
  }),
  component: RootComponent,
  shellComponent: RootDocument,
})

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" data-theme="dark">
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  )
}

function RootComponent() {
  return (
    <ThemeProvider>
      <ClerkProvider publishableKey={publishableKey}>
        <AuthProvider>
          <Outlet />
        </AuthProvider>
      </ClerkProvider>
    </ThemeProvider>
  )
}
