import { createFileRoute, Outlet, useParams } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useWorkspaceStore } from '@/core/workspace/store'

export const Route = createFileRoute('/_dashboard/$projectId')({
  component: ProjectLayout,
})

function ProjectLayout() {
  const { projectId } = useParams({ from: '/_dashboard/$projectId' })
  const setLastActive = useWorkspaceStore((state) => state.actions.setLastActiveProject)

  useEffect(() => {
    if (projectId) {
      setLastActive(projectId)
    }
  }, [projectId, setLastActive])

  // TODO: call dashboard project metadata RPC here
  return <Outlet />
}
