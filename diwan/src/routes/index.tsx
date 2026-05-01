import { createFileRoute, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store';

export const Route = createFileRoute('/')({
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (isAuthenticated) {
      throw redirect({ to: "/project" });
    } else {
      throw redirect({ to: "/auth" });
    }
  },
})
