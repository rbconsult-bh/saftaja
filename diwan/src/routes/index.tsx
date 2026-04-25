import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store';
import { useEffect } from 'react';

export const Route = createFileRoute('/')({
  component: RouteComponent,
})

function RouteComponent() {
  const isAuthenticated = useSessionStore(s => s.isAuthenticated);
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate({ to: "/auth" });
    } else {
      navigate({ to: "/project" });
    }
  }, [isAuthenticated]);

  return <></>;
}
