import { createRootRoute, Outlet, redirect, useNavigate } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { useSessionStore } from '../core/session/store'
import { useEffect } from 'react';

const RootLayout = () => {
  const isAuthenticated = useSessionStore(s => s.isAuthenticated);
  const navigate = useNavigate();

  useEffect(() => {
    if (!isAuthenticated) {
      navigate({ to: "/auth" });
    }
  }, [isAuthenticated]);

  return (
    <>
      <Outlet />
      <TanStackRouterDevtools />
    </>
  )
}

export const Route = createRootRoute({
  component: RootLayout,
})
