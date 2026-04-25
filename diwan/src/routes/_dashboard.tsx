import { createFileRoute, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store';

export const Route = createFileRoute('/_dashboard')({
  component: RouteComponent,
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (!isAuthenticated) {
      throw redirect({ to: "/auth" });
    }
  },
})

function RouteComponent() {
  return <div>Hello "/_dashboard"!</div>
}
