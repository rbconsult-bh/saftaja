import { createFileRoute, Outlet, useParams, useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useWorkspaceStore } from '@/core/workspace/store'
import { useWorkspace } from '@/core/workspace/WorkspaceContext'
import { Skeleton } from '@/components/ui/skeleton'

const PLACEHOLDER_PROJECT_ID = '_'

export const Route = createFileRoute('/_dashboard/$projectId')({
  component: ProjectLayout,
})

function ProjectLayout() {
  const { projectId } = useParams({ from: '/_dashboard/$projectId' })
  const setLastActive = useWorkspaceStore((state) => state.actions.setLastActiveProject)
  const { workspace } = useWorkspace()
  const navigate = useNavigate()

  useEffect(() => {
    const allProjects = workspace.organizations.flatMap((o) => o.projects)
    const projectExists = allProjects.some((p) => p.id === projectId)

    if (!projectExists || projectId === PLACEHOLDER_PROJECT_ID) {
      const firstProject = workspace.organizations[0]?.projects[0]
      if (firstProject) {
        navigate({
          to: '/$projectId',
          params: { projectId: firstProject.id },
          replace: true,
        })
      }
      return
    }

    setLastActive(projectId)
  }, [workspace, projectId, setLastActive, navigate])

  // Show skeleton while redirect is in flight (brief flash)
  const allProjects = workspace.organizations.flatMap((o) => o.projects)
  if (projectId === PLACEHOLDER_PROJECT_ID || !allProjects.some((p) => p.id === projectId)) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-96" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    )
  }

  return <Outlet />
}
