import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@connectrpc/connect-query'
import { logout as logoutRpc } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'
import { useSessionStore } from '../../core/session/store'
import { Button } from "@/components/ui/button"
import { Loader2, LogOut } from "lucide-react"
import { m } from "@/paraglide/messages"

export const Route = createFileRoute('/_dashboard/project')({
  component: RouteComponent,
})

function RouteComponent() {
  const navigate = useNavigate()
  const storeLogout = useSessionStore((state) => state.actions.logout)
  const logoutMut = useMutation(logoutRpc)

  const handleLogout = async () => {
    logoutMut.mutate({}, {
      onSettled: () => {
        storeLogout()
        navigate({ to: '/auth' })
      }
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">
          {m.dashboard_project_title()}
        </h1>

        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          disabled={logoutMut.isPending}
          className="text-muted-foreground hover:text-destructive transition-colors"
        >
          {logoutMut.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <LogOut className="mr-2 h-4 w-4 rtl:rotate-180" />
          )}
          {m.dashboard_logout()}
        </Button>
      </div>

      <div className="rounded-xl border border-dashed p-20 flex items-center justify-center text-muted-foreground">
        {m.dashboard_project_welcome()}
      </div>
    </div>
  )
}
