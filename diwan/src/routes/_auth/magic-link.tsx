import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@connectrpc/connect-query'
import { completeAuth } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'
import { useSessionStore } from '../../core/session/store'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Loader2 } from "lucide-react"
import { m } from "@/paraglide/messages"

type MagicLinkSearch = { token?: string }

export const Route = createFileRoute('/_auth/magic-link')({
  validateSearch: (search: Record<string, unknown>): MagicLinkSearch => ({
    token: search.token as string | undefined,
  }),
  component: RouteComponent,
})

function RouteComponent() {
  const { token } = Route.useSearch()
  const login = useSessionStore((state) => state.actions.login)
  const navigate = useNavigate()
  const completeAuthMut = useMutation(completeAuth)

  const handleLoginClick = () => {
    if (!token) return

    completeAuthMut.mutate(
      { token },
      {
        onSuccess: (data) => {
          login(data.accessToken, data.refreshToken)

          navigate({
            to: '/$projectId',
            params: { projectId: '_' },
          })
        },
      }
    )
  }

  if (!token) {
    return (
      <div className="text-center space-y-4">
        <h2 className="text-2xl font-bold">{m.auth_magic_link_invalid_title()}</h2>
        <p className="text-sm text-muted-foreground">{m.auth_magic_link_invalid_subtitle()}</p>
        <Button asChild variant="outline" className="w-full">
          <Link to="/auth">{m.auth_magic_link_return_button()}</Link>
        </Button>
      </div>
    )
  }

  return (
    <Card className="border-none shadow-none bg-transparent">
      <CardHeader className="text-center">
        <CardTitle className="text-2xl font-bold">
          {completeAuthMut.isError ? m.auth_magic_link_failed_title() : m.auth_magic_link_title()}
        </CardTitle>
        {!completeAuthMut.isError && (
          <CardDescription>
            {m.auth_magic_link_subtitle()}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {completeAuthMut.isError && (
          <Alert variant="destructive">
            <AlertDescription>
              {completeAuthMut.error?.message || m.common_error()}
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
      <CardFooter>
        {completeAuthMut.isError ? (
          <Button asChild className="w-full">
            <Link to="/auth">{m.auth_magic_link_failed_button()}</Link>
          </Button>
        ) : (
          <Button
            onClick={handleLoginClick}
            className="w-full h-11 text-base font-semibold"
            disabled={completeAuthMut.isPending || completeAuthMut.isSuccess}
          >
            {completeAuthMut.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin rtl:ml-2 rtl:mr-0" />
                {m.auth_magic_link_authenticating()}
              </>
            ) : m.auth_magic_link_button()}
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}
