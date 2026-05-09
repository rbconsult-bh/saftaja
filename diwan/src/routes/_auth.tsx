import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store'
import { useWorkspaceStore } from '@/core/workspace/store'
import { SENTINEL_PROJECT_ID } from '@/core/workspace/constants'

export const Route = createFileRoute('/_auth')({
  component: RouteComponent,
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (isAuthenticated) {
      const { lastActiveProjectId } = useWorkspaceStore.getState()
      throw redirect({
        to: '/$projectId',
        params: { projectId: lastActiveProjectId ?? SENTINEL_PROJECT_ID },
      })
    }
  },
})

function RouteComponent() {
  return (
    <div dir="auto" className="flex min-h-screen items-center justify-center bg-zinc-50 p-4 font-sans text-zinc-900">
      <div className="w-full max-w-md space-y-8 rounded-2xl bg-white p-8 shadow-sm border border-zinc-100">
        <Outlet />
      </div>
    </div>
  );
}
