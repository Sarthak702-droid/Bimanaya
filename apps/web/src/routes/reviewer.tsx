import { createFileRoute, Outlet } from '@tanstack/react-router'
import { ReviewerLayout } from '../components/layout/ReviewerLayout'

export const Route = createFileRoute('/reviewer')({
  component: ReviewerRouteLayout,
})

function ReviewerRouteLayout() {
  return (
    <ReviewerLayout>
      <Outlet />
    </ReviewerLayout>
  )
}
