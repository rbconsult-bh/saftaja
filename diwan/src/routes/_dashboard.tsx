import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { useSessionStore } from '../core/session/store'
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { AppSidebar } from '@/components/AppSidebar'

export const Route = createFileRoute('/_dashboard')({
  beforeLoad: () => {
    const { isAuthenticated } = useSessionStore.getState()
    if (!isAuthenticated) throw redirect({ to: "/auth" })
  },
  component: DashboardLayout,
})

function DashboardLayout() {
  return (
    <SidebarProvider>

      <AppSidebar />

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
