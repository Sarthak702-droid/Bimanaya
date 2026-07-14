import { createFileRoute, Outlet } from '@tanstack/react-router'
import { DashboardLayout } from '../components/layout/DashboardLayout'

export const Route = createFileRoute('/dashboard')({
  component: DashboardRouteLayout,
})

function DashboardRouteLayout() {
  return (
    <DashboardLayout>
      <Outlet />
    </DashboardLayout>
  )
}
