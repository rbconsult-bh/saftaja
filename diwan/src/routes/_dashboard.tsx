import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store'
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { AppSidebar } from '@/components/AppSidebar'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { useQuery } from '@connectrpc/connect-query'
import { getWorkspace } from '@/gen/saftaja/dashboard/workspace/v1/workspace-WorkspaceService_connectquery'
import { m } from '@/paraglide/messages'
import { AlertTriangle, Plus } from 'lucide-react'

export const Route = createFileRoute('/_dashboard')({
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (!isAuthenticated) throw redirect({ to: "/auth" })
  },
  component: DashboardLayout,
})

function DashboardLayout() {
  const { data, isPending, isError, refetch } = useQuery(getWorkspace)

  if (isPending) return <DashboardSkeleton />
  if (isError) return <DashboardError onRetry={() => refetch()} />
  if (!data || data.organizations.length === 0) return <NoOrganizations />
  if (data.organizations.flatMap(o => o.projects).length === 0) return <NoProjects />

  return (
    <SidebarProvider>
      <AppSidebar organizations={data.organizations} />
      <SidebarInset>
        <header className="flex h-16 items-center gap-2 border-b px-4">
          <SidebarTrigger />
          <Separator orientation="vertical" className="mr-2 h-4" />
        </header>
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}

function DashboardSkeleton() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="space-y-4 w-64">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    </div>
  )
}

function DashboardError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center space-y-4">
        <AlertTriangle className="mx-auto h-8 w-8 text-destructive" />
        <p className="text-muted-foreground">{m.common_error()}</p>
        <Button variant="outline" onClick={onRetry}>
          Retry
        </Button>
      </div>
    </div>
  )
}

function NoOrganizations() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center space-y-4 max-w-md px-4">
        <h2 className="text-xl font-semibold">Welcome to Saftaja</h2>
        <p className="text-muted-foreground">
          You don&apos;t belong to any organization yet.
          Please contact your administrator for an invitation.
        </p>
      </div>
    </div>
  )
}

function NoProjects() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center space-y-4 max-w-md px-4">
        <h2 className="text-xl font-semibold">No Projects Yet</h2>
        <p className="text-muted-foreground">
          Your organization doesn&apos;t have any projects. Create one to get started.
        </p>
        <Button disabled>
          <Plus className="mr-2 h-4 w-4" />
          {m.dashboard_workspace_create_project()}
        </Button>
      </div>
    </div>
  )
}
