import { createFileRoute, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store'
import { useWorkspaceStore } from '@/core/workspace/store'
import { SENTINEL_PROJECT_ID } from '@/core/workspace/constants'

export const Route = createFileRoute('/')({
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()

    if (!isAuthenticated) {
      throw redirect({ to: '/auth' })
    }

    const { lastActiveProjectId } = useWorkspaceStore.getState()

    throw redirect({
      to: '/$projectId',
      params: { projectId: lastActiveProjectId ?? SENTINEL_PROJECT_ID },
    })
  },
})
