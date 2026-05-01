import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store'

export const Route = createFileRoute('/_auth')({
  component: RouteComponent,
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (isAuthenticated) {
      throw redirect({ to: "/project" });
    }
  },
})

function RouteComponent() {
  return <Outlet />;
}
