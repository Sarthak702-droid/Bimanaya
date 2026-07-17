import { createFileRoute, Outlet } from '@tanstack/react-router'
import { DashboardLayout } from '../components/layout/DashboardLayout'
import { RequireAuth } from '../components/auth/RequireAuth'

export const Route = createFileRoute('/dashboard')({
  component: DashboardRouteLayout,
})

function DashboardRouteLayout() {
  return (
    <RequireAuth>
      <DashboardLayout>
        <Outlet />
      </DashboardLayout>
    </RequireAuth>
  )
}
