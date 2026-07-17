import { createFileRoute, Outlet } from '@tanstack/react-router'
import { ReviewerLayout } from '../components/layout/ReviewerLayout'
import { RequireAuth } from '../components/auth/RequireAuth'

export const Route = createFileRoute('/reviewer')({
  component: ReviewerRouteLayout,
})

function ReviewerRouteLayout() {
  return (
    <RequireAuth roles={['REVIEWER', 'ADMIN']}>
      <ReviewerLayout>
        <Outlet />
      </ReviewerLayout>
    </RequireAuth>
  )
}
